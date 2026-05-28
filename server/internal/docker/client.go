package docker

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	available bool
}

func NewClient() (*Client, error) {
	c := &Client{}

	cmd := exec.Command("docker", "info")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		slog.Warn("docker daemon not reachable, falling back to host execution", "error", err)
		return c, nil
	}

	c.available = true
	slog.Info("docker client initialized successfully")
	return c, nil
}

func (c *Client) Available() bool {
	return c.available
}

type PullResult struct {
	ImageRef string
	Already  bool
}

func (c *Client) PullImage(ctx context.Context, ref string) (*PullResult, error) {
	if !c.available {
		return nil, fmt.Errorf("docker daemon not available")
	}

	if exists, _ := c.ImageExists(ctx, ref); exists {
		slog.Info("image already exists locally", "image", ref)
		return &PullResult{ImageRef: ref, Already: true}, nil
	}

	slog.Info("pulling docker image", "image", ref)
	start := time.Now()

	cmd := exec.CommandContext(ctx, "docker", "pull", ref)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker pull %s: %w\n%s", ref, err, string(output))
	}

	slog.Info("image pulled", "image", ref, "duration", time.Since(start).Round(time.Second))
	return &PullResult{ImageRef: ref, Already: false}, nil
}

func (c *Client) ImageExists(ctx context.Context, ref string) (bool, error) {
	if !c.available {
		return false, fmt.Errorf("docker daemon not available")
	}

	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", ref)
	if err := cmd.Run(); err != nil {
		return false, nil
	}
	return true, nil
}

type ContainerOpts struct {
	Image       string
	Interpreter string
	Code        string
	Extension   string
	RunCmd      string
	CompileCmd  string
	Type        string
	MemoryMB    int64
	CPUs        float64
	TimeoutSec  int
}

type ContainerResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}

func (c *Client) RunContainer(ctx context.Context, opts ContainerOpts) (*ContainerResult, error) {
	if !c.available {
		return nil, fmt.Errorf("docker daemon not available")
	}

	if opts.Image == "" {
		return nil, fmt.Errorf("container image is required")
	}

	if opts.MemoryMB == 0 {
		opts.MemoryMB = 256
	}
	if opts.CPUs == 0 {
		opts.CPUs = 1.0
	}
	if opts.TimeoutSec == 0 {
		opts.TimeoutSec = 30
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(opts.TimeoutSec)*time.Second)
	defer cancel()

	start := time.Now()

	filename := "main" + opts.Extension
	basename := "main"

	vars := map[string]string{
		"{interpreter}": opts.Interpreter,
		"{file}":        filename,
		"{basename}":    basename,
		"{classname}":   basename,
		"{output}":      "./main.out",
	}

	var script string
	if opts.Type == "compiled" && opts.CompileCmd != "" {
		compileCmd := substitute(opts.CompileCmd, vars)
		runCmd := substitute(opts.RunCmd, vars)
		script = fmt.Sprintf("cat > %s && %s && %s", filename, compileCmd, runCmd)
	} else {
		runCmd := substitute(opts.RunCmd, vars)
		script = fmt.Sprintf("cat > %s && %s", filename, runCmd)
	}

	args := []string{
		"run",
		"-i",
		"--rm",
		"--network", "none",
		"--memory", fmt.Sprintf("%dm", opts.MemoryMB),
		"--cpus", fmt.Sprintf("%.1f", opts.CPUs),
		"--pids-limit", "64",
	}
	args = append(args, opts.Image, "sh", "-c", script)

	cmd := exec.CommandContext(execCtx, "docker", args...)
	cmd.Stdin = strings.NewReader(opts.Code)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	result := &ContainerResult{
		DurationMs: elapsed,
	}

	if runErr != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result.Stderr = fmt.Sprintf("Execution timed out after %d seconds", opts.TimeoutSec)
			result.ExitCode = -1
			return result, nil
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("docker run failed: %w", runErr)
		}
	}

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	return result, nil
}

func (c *Client) BuildImage(ctx context.Context, tag string, contextDir string) ([]string, error) {
	if !c.available {
		return nil, fmt.Errorf("docker daemon not available")
	}

	slog.Info("building docker image", "tag", tag, "context", contextDir)

	cmd := exec.CommandContext(ctx, "docker", "build", "-t", tag, contextDir)
	output, err := cmd.CombinedOutput()
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	if err != nil {
		return lines, fmt.Errorf("docker build failed: %w\n%s", err, string(output))
	}

	slog.Info("docker image built successfully", "tag", tag)
	return lines, nil
}

func (c *Client) VerifyInterpreter(ctx context.Context, image, interpreter string) (string, error) {
	if !c.available {
		return "", fmt.Errorf("docker daemon not available")
	}

	verifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(verifyCtx, "docker", "run", "--rm",
		"--entrypoint", interpreter, image, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("interpreter %q not found in image %s: %w\n%s", interpreter, image, err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func (c *Client) VerifyImage(ctx context.Context, opts ContainerOpts, expectedOutput string) error {
	result, err := c.RunContainer(ctx, opts)
	if err != nil {
		return fmt.Errorf("verification run failed: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("verification exited with code %d: %s", result.ExitCode, result.Stderr)
	}

	if expectedOutput != "" && !strings.Contains(result.Stdout, expectedOutput) {
		return fmt.Errorf("verification output mismatch: expected %q, got %q", expectedOutput, result.Stdout)
	}

	return nil
}

func substitute(tmpl string, vars map[string]string) string {
	s := tmpl
	for k, v := range vars {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}
