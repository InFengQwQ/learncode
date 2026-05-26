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

// Client wraps Docker operations for building and running code in containers.
// If the Docker daemon is unavailable, Available() returns false and callers
// should fall back to host execution.
type Client struct {
	available bool
}

// NewClient creates a Docker client. If the Docker daemon is unreachable,
// the client is created in unavailable mode — callers should check Available().
func NewClient() (*Client, error) {
	c := &Client{}

	// Try to reach the Docker daemon with a quick ping.
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

// Available returns true if the Docker daemon is reachable.
func (c *Client) Available() bool {
	return c.available
}

// PullResult contains the outcome of an image pull.
type PullResult struct {
	ImageRef string
	Already  bool // true if image was already present locally
}

// PullImage pulls a Docker image. If the image already exists locally, it
// returns immediately with Already=true. Context can be used for cancellation.
func (c *Client) PullImage(ctx context.Context, ref string) (*PullResult, error) {
	if !c.available {
		return nil, fmt.Errorf("docker daemon not available")
	}

	// Check if image exists locally first.
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

// ImageExists checks if a Docker image is available locally.
func (c *Client) ImageExists(ctx context.Context, ref string) (bool, error) {
	if !c.available {
		return false, fmt.Errorf("docker daemon not available")
	}

	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", ref)
	if err := cmd.Run(); err != nil {
		return false, nil // image does not exist
	}
	return true, nil
}

// ContainerOpts configures a container execution.
type ContainerOpts struct {
	Image       string  // Docker image ref (e.g. "python:3.12-slim")
	Interpreter string  // Interpreter binary name inside the container (e.g. "python", "node")
	Code        string  // Source code to write and execute
	Extension   string  // File extension (e.g. ".py")
	RunCmd      string  // Command template with {file}, {interpreter}, {basename}, {output}, {classname}
	CompileCmd  string  // Command template for compiled languages (empty for interpreted)
	Type        string  // "interpreted" or "compiled"
	MemoryMB    int64   // Memory limit in MB (default 256)
	CPUs        float64 // CPU limit (default 1.0)
	TimeoutSec  int     // Execution timeout (default 30)
}

// ContainerResult holds the output of a container execution.
type ContainerResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}

// RunContainer executes code inside a Docker container with security constraints.
// Uses stdin piping to pass source code into the container — avoids volume mount
// issues in Docker-in-Docker scenarios where the container's filesystem paths
// don't map to the host Docker daemon's filesystem.
func (c *Client) RunContainer(ctx context.Context, opts ContainerOpts) (*ContainerResult, error) {
	if !c.available {
		return nil, fmt.Errorf("docker daemon not available")
	}

	if opts.Image == "" {
		return nil, fmt.Errorf("container image is required")
	}

	// Defaults
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

	// Build shell script: write code from stdin to file, then compile/run.
	// For interpreted:  cat > file && run
	// For compiled:     cat > file && compile && run
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
		"-i",                        // Keep stdin open for piping code
		"--rm",                      // Remove container after execution
		"--network", "none",         // No network access
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

// BuildImage builds a Docker image from a Dockerfile in the given context directory.
// Returns build log lines for debugging.
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

// VerifyImage runs a simple hello-world verification inside a container to
// confirm the language runtime is working. It writes a minimal program and
// checks that it produces output.
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