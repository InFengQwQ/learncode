package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"

	"learncode/internal/docker"
	"learncode/internal/executor"
	"learncode/internal/model"
	"learncode/internal/repo"
)

type VersionInitResult struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Verified bool   `json:"verified"`
	ImageRef string `json:"image_ref"`
}

type InitService struct {
	VersionRepo *repo.VersionRepo
	LangRepo    *repo.LanguageRepo
	Docker      *docker.Client
}

func (s *InitService) Initialize(ctx context.Context, versionID string) (*VersionInitResult, error) {
	version, err := s.VersionRepo.GetByID(ctx, versionID)
	if err != nil {
		return nil, fmt.Errorf("version not found: %w", err)
	}

	lang, err := s.LangRepo.GetByID(ctx, version.LanguageID)
	if err != nil {
		return nil, fmt.Errorf("language not found: %w", err)
	}

	rc, err := executor.ParseRuntimeConfig(version.RuntimeConfig)
	if err != nil || !rc.IsComplete() {
		rc = executor.DefaultRuntimeConfig(lang.Slug)
	}

	image := rc.Image
	if image == "" && len(version.SourceURLs) > 0 {
		var src struct {
			ImageTag string `json:"image_tag"`
		}
		if json.Unmarshal(version.SourceURLs, &src) == nil && src.ImageTag != "" {
			image = src.ImageTag
			rc.Image = image
		}
	}

	if s.Docker != nil && s.Docker.Available() && image != "" {
		return s.initWithDocker(ctx, version, lang, rc, image)
	}

	return s.initOnHost(ctx, version, rc)
}

func (s *InitService) initWithDocker(ctx context.Context, version *model.LanguageVersion, lang *model.Language, rc executor.RuntimeConfig, image string) (*VersionInitResult, error) {
	result := &VersionInitResult{ImageRef: image}

	pullResult, err := s.Docker.PullImage(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("pull image %s: %w", image, err)
	}
	if pullResult.Already {
		slog.Info("image already present", "image", image)
	} else {
		slog.Info("image pulled", "image", image)
	}

	rc.Image = image
	if rc.Interpreter == "" {
		rc.Interpreter = lang.Slug
	}

	versionOutput, verifyErr := s.Docker.VerifyInterpreter(ctx, image, rc.Interpreter)
	if verifyErr != nil {
		slog.Warn("interpreter verification failed — image may not contain the expected runtime",
			"image", image, "interpreter", rc.Interpreter, "error", verifyErr)
		result.Status = "unavailable"
		result.Message = fmt.Sprintf("image %s pulled but interpreter %q not found", image, rc.Interpreter)
		result.Verified = false
	} else {
		slog.Info("interpreter verified", "image", image, "interpreter", rc.Interpreter, "version", versionOutput)
		result.Status = "success"
		result.Message = fmt.Sprintf("image %s ready (%s)", image, versionOutput)
		result.Verified = true
	}

	if err := s.markInitialized(ctx, version.ID, image, rc); err != nil {
		return nil, fmt.Errorf("mark initialized: %w", err)
	}

	return result, nil
}

func (s *InitService) initOnHost(ctx context.Context, version *model.LanguageVersion, rc executor.RuntimeConfig) (*VersionInitResult, error) {
	interpreter := rc.Interpreter
	if interpreter == "" {
		return nil, fmt.Errorf("Docker unavailable and no interpreter set in runtime config")
	}
	if path, err := exec.LookPath(interpreter); err == nil {
		image := version.Image
		if err := s.markInitialized(ctx, version.ID, image, rc); err != nil {
			return nil, err
		}
		return &VersionInitResult{
			Status:   "host_mode",
			Message:  fmt.Sprintf("host interpreter found: %s (%s)", interpreter, path),
			Verified: true,
			ImageRef: image,
		}, nil
	}

	return nil, fmt.Errorf("Docker unavailable and interpreter %q not found in PATH", interpreter)
}

func (s *InitService) markInitialized(ctx context.Context, versionID string, image string, rc executor.RuntimeConfig) error {
	rcBytes := rc.Marshal()
	return s.VersionRepo.MarkInitialized(ctx, versionID, image, rcBytes)
}
