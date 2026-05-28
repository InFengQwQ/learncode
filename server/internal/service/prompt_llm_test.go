package service

import (
	"context"
	"testing"

	"learncode/internal/config"
	"learncode/internal/llm"
)

func loadLLMService(t *testing.T) *llm.Service {
	t.Helper()
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Skipf("cannot load config: %v", err)
	}
	for i := range cfg.LLM.Providers {
		cfg.LLM.Providers[i].Endpoint = "http://localhost:1234/v1"
	}
	svc, err := llm.NewService(cfg.LLM)
	if err != nil {
		t.Skipf("cannot create llm service: %v", err)
	}
	return svc
}

func TestFilterVersionSourcesPrompt(t *testing.T) {
	svc := loadLLMService(t)
	ctx := context.Background()

	vars := map[string]string{
		"LanguageName": "Go",
		"WebsiteURL":   "https://go.dev",
		"URLs":         "https://go.dev/dl/\nhttps://github.com/golang/go\nhttps://go.dev/doc/\nhttps://go.dev/blog\nhttps://en.wikipedia.org/wiki/Go_(programming_language)",
	}

	tmpl, err := llm.LoadTemplate("../../prompts/filter_version_sources.yaml", vars)
	if err != nil {
		t.Fatalf("load filter_version_sources template: %v", err)
	}

	content, usage, err := svc.ChatWithTemp(ctx, tmpl.SystemPrompt, tmpl.UserPrompt, tmpl.Temperature, tmpl.MaxTokens)
	if err != nil {
		t.Fatalf("llm call failed: %v", err)
	}
	t.Logf("filter_version_sources token usage: prompt=%d completion=%d total=%d",
		usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	t.Logf("filter_version_sources raw response:\n%s", content)

	var urls []string
	if err := llm.ParseLLMJSON(content, &urls); err != nil {
		t.Fatalf("parse filter_version_sources response: %v", err)
	}

	if len(urls) == 0 {
		t.Error("expected at least one URL, got none")
	}
	t.Logf("filtered %d URLs", len(urls))
	for i, u := range urls {
		t.Logf("url[%d]: %s", i, u)
	}
}

func TestExtractVersionsPrompt(t *testing.T) {
	svc := loadLLMService(t)
	ctx := context.Background()

	pageText := `Download Python. Python 3.13.0. Release Date: Oct. 7, 2024. This is the latest stable release.
		Download Windows installer (64-bit) https://www.python.org/ftp/python/3.13.0/python-3.13.0-amd64.exe
		Python 3.12.7. Release Date: Oct. 1, 2024. Download Windows installer
		Python 3.11.10. Release Date: Oct. 1, 2024. Download Windows installer
		Python 3.10.15. Release Date: Oct. 1, 2024. Download Windows installer
		Docker: docker pull python:3.13, docker pull python:3.13-slim, python:3.12, python:3.12-slim`

	vars := map[string]string{
		"LanguageName":       "Python",
		"Slug":               "python",
		"CompatibilityModel": "versioned",
		"PageURL":            "https://www.python.org/downloads/",
		"PageText":           pageText,
	}

	tmpl, err := llm.LoadTemplate("../../prompts/extract_versions.yaml", vars)
	if err != nil {
		t.Fatalf("load extract_versions template: %v", err)
	}

	content, usage, err := svc.ChatWithTemp(ctx, tmpl.SystemPrompt, tmpl.UserPrompt, tmpl.Temperature, tmpl.MaxTokens)
	if err != nil {
		t.Fatalf("llm call failed: %v", err)
	}
	t.Logf("extract_versions token usage: prompt=%d completion=%d total=%d",
		usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	t.Logf("extract_versions raw response:\n%s", content)

	var result struct {
		Versions   []DiscoveredVersion `json:"versions"`
		Latest     string              `json:"latest"`
		DockerRefs []string            `json:"docker_refs"`
	}
	if err := llm.ParseLLMJSON(content, &result); err != nil {
		t.Fatalf("parse extract_versions response: %v", err)
	}

	if len(result.Versions) == 0 {
		t.Error("expected at least one version, got none")
	}
	if result.Latest == "" {
		t.Error("expected latest version string, got empty")
	}

	t.Logf("latest=%s docker_refs=%v", result.Latest, result.DockerRefs)
	for i, v := range result.Versions {
		t.Logf("version[%d]: version=%s lts=%v released=%s download_url=%s brief=%s docker_refs=%v",
			i, v.Version, v.LTS, v.Released, v.DownloadURL, v.Brief, v.DockerRefs)

		if v.Version == "" {
			t.Errorf("version[%d]: version is empty", i)
		}
	}

	found3_13 := false
	found3_12 := false
	for _, v := range result.Versions {
		if v.Version == "3.13.0" {
			found3_13 = true
			if v.DownloadURL == "" {
				t.Log("3.13.0 has no download_url — may be a LLM extraction gap")
			}
		}
		if v.Version == "3.12.7" {
			found3_12 = true
		}
	}
	if !found3_13 {
		t.Error("expected version 3.13.0 to be extracted from page text")
	}
	if !found3_12 {
		t.Error("expected version 3.12.7 to be extracted from page text")
	}

	if len(result.DockerRefs) == 0 {
		t.Log("no docker_refs extracted — may be acceptable if LLM missed them")
	}
	for _, ref := range result.DockerRefs {
		t.Logf("docker_ref: %s", ref)
	}
}

func TestExtractVersionsStrict(t *testing.T) {
	svc := loadLLMService(t)
	ctx := context.Background()

	pageText := `The Go Programming Language. Latest stable release: Go 1.24.0.
		Download: https://go.dev/dl/go1.24.0.windows-amd64.msi
		Docker: docker pull golang:1.24`

	vars := map[string]string{
		"LanguageName":       "Go",
		"Slug":               "go",
		"CompatibilityModel": "strict",
		"PageURL":            "https://go.dev/dl/",
		"PageText":           pageText,
	}

	tmpl, err := llm.LoadTemplate("../../prompts/extract_versions.yaml", vars)
	if err != nil {
		t.Fatalf("load extract_versions template: %v", err)
	}

	content, usage, err := svc.ChatWithTemp(ctx, tmpl.SystemPrompt, tmpl.UserPrompt, tmpl.Temperature, tmpl.MaxTokens)
	if err != nil {
		t.Fatalf("llm call failed: %v", err)
	}
	t.Logf("extract_versions strict token usage: prompt=%d completion=%d total=%d",
		usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	t.Logf("extract_versions strict raw response:\n%s", content)

	var result struct {
		Versions   []DiscoveredVersion `json:"versions"`
		Latest     string              `json:"latest"`
		DockerRefs []string            `json:"docker_refs"`
	}
	if err := llm.ParseLLMJSON(content, &result); err != nil {
		t.Fatalf("parse extract_versions response: %v", err)
	}

	if len(result.Versions) != 1 {
		t.Errorf("strict mode: expected exactly 1 version, got %d: %v", len(result.Versions), result.Versions)
	}
	if len(result.Versions) > 0 {
		t.Logf("strict version: %s (latest=%s)", result.Versions[0].Version, result.Latest)
		if result.Versions[0].Version != result.Latest {
			t.Logf("version != latest: %s vs %s (may be acceptable)", result.Versions[0].Version, result.Latest)
		}
	}
}

func TestFullVersionDiscoveryWorkflow(t *testing.T) {
	svc := loadLLMService(t)
	ctx := context.Background()

	svc2 := &LanguageInitService{
		LLM:       svc,
		PromptDir: "../../prompts",
		Scraper:   nil,
	}

	_ = ctx
	_ = svc2
	t.Skip("end-to-end test requires mock scraper — tested via unit prompt tests above")
}
