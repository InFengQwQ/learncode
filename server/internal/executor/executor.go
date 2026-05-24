package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Request is a code execution request.
type Request struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

// Result is the output of a code execution.
type Result struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
}

// Config holds language-specific execution configuration.
type Config struct {
	Interpreter string // e.g., "python3", "node"
	Extension   string // e.g., ".py", ".js"
	Args        []string
}

// configFor returns the execution config for a language slug.
// Only Python3 and Node.js are supported in this initial demo.
func configFor(langSlug string) (Config, bool) {
	switch strings.ToLower(langSlug) {
	case "python":
		return Config{Interpreter: "python3", Extension: ".py"}, true
	case "javascript", "js", "node", "nodejs":
		return Config{Interpreter: "node", Extension: ".js"}, true
	default:
		return Config{}, false
	}
}

// Execute runs the code in a subprocess with a 30-second timeout.
func Execute(ctx context.Context, req Request) (*Result, error) {
	cfg, ok := configFor(req.Language)
	if !ok {
		return nil, fmt.Errorf("unsupported language: %q (only python and javascript are supported in this demo)", req.Language)
	}

	// Write code to a temporary file so we can pass it to the interpreter.
	dir, err := os.MkdirTemp("", "learncode-exec-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	srcPath := filepath.Join(dir, "main"+cfg.Extension)
	if err := os.WriteFile(srcPath, []byte(req.Code), 0o644); err != nil {
		return nil, fmt.Errorf("write temp file: %w", err)
	}

	// 30-second timeout to prevent infinite loops from hanging the server.
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	args := append(cfg.Args, srcPath)
	cmd := exec.CommandContext(execCtx, cfg.Interpreter, args...)
	cmd.Dir = dir

	start := time.Now()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if execCtx.Err() == context.DeadlineExceeded {
			return &Result{
				Stdout:     stdout.String(),
				Stderr:     "Execution timed out after 30 seconds",
				ExitCode:   -1,
				DurationMs: elapsed,
			}, nil
		} else {
			return nil, fmt.Errorf("execution failed: %w", runErr)
		}
	}

	return &Result{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   exitCode,
		DurationMs: elapsed,
	}, nil
}
