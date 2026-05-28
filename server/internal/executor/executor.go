package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"learncode/internal/docker"
)

type Request struct {
	Code string `json:"code"`
}

type Result struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
}

type Executor struct {
	Docker *docker.Client
}

func NewExecutor(dockerClient *docker.Client) *Executor {
	return &Executor{Docker: dockerClient}
}

func (e *Executor) Execute(ctx context.Context, rc RuntimeConfig, code string) (*Result, error) {
	if e.Docker != nil && e.Docker.Available() && rc.Image != "" {
		if rc.Interpreter == "" {
			return nil, fmt.Errorf("interpreter is required for docker execution (image=%q)", rc.Image)
		}
		return e.runInDocker(ctx, rc, code)
	}

	if !rc.IsComplete() {
		return nil, fmt.Errorf("incomplete runtime config: type=%q interpreter=%q extension=%q run_cmd=%q — configure it in the version settings", rc.Type, rc.Interpreter, rc.Extension, rc.RunCmd)
	}
	return e.runOnHost(ctx, rc, code)
}

func (e *Executor) runInDocker(ctx context.Context, rc RuntimeConfig, code string) (*Result, error) {
	if rc.Extension == "" {
		rc.Extension = ".txt"
	}
	if rc.Type == "" || rc.Type == "unknown" {
		rc.Type = "interpreted"
	}
	if rc.RunCmd == "" {
		rc.RunCmd = "{interpreter} {file}"
	}

	opts := docker.ContainerOpts{
		Image:       rc.Image,
		Interpreter: rc.Interpreter,
		Code:        code,
		Extension:   rc.Extension,
		RunCmd:      rc.RunCmd,
		CompileCmd:  rc.CompileCmd,
		Type:        rc.Type,
		MemoryMB:    256,
		CPUs:        1.0,
		TimeoutSec:  30,
	}

	result, err := e.Docker.RunContainer(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("docker execution failed: %w", err)
	}

	return &Result{
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		ExitCode:   result.ExitCode,
		DurationMs: result.DurationMs,
	}, nil
}

func (e *Executor) runOnHost(ctx context.Context, rc RuntimeConfig, code string) (*Result, error) {
	dir, err := os.MkdirTemp("", "learncode-exec-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	srcPath := filepath.Join(dir, "main"+rc.Extension)
	if err := os.WriteFile(srcPath, []byte(code), 0o644); err != nil {
		return nil, fmt.Errorf("write temp file: %w", err)
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()

	if rc.Type == "compiled" {
		return runCompiled(execCtx, rc, dir, srcPath, start)
	}
	return runInterpreted(execCtx, rc, dir, srcPath, start)
}

func runInterpreted(ctx context.Context, rc RuntimeConfig, dir, srcPath string, start time.Time) (*Result, error) {
	interpreter := rc.FindInterpreter(altInterpreters(rc.Interpreter)...)
	if interpreter == "" {
		return nil, fmt.Errorf("interpreter %q not found in PATH", rc.Interpreter)
	}

	basename := strings.TrimSuffix(filepath.Base(srcPath), rc.Extension)

	runCmd := substitute(rc.RunCmd, map[string]string{
		"{interpreter}": interpreter,
		"{file}":        srcPath,
		"{basename}":    basename,
	})

	parts := strings.Fields(runCmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty run command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = dir

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	return buildResult(&stdout, &stderr, runErr, elapsed, ctx)
}

func runCompiled(ctx context.Context, rc RuntimeConfig, dir, srcPath string, start time.Time) (*Result, error) {
	outputPath := filepath.Join(dir, "main")

	basename := strings.TrimSuffix(filepath.Base(srcPath), rc.Extension)

	compileCmd := substitute(rc.CompileCmd, map[string]string{
		"{file}":     srcPath,
		"{output}":   outputPath,
		"{basename}": basename,
	})

	parts := strings.Fields(compileCmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty compile command")
	}

	compileStart := time.Now()
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = dir

	var compileOut strings.Builder
	cmd.Stdout = &compileOut
	cmd.Stderr = &compileOut

	if err := cmd.Run(); err != nil {
		elapsed := time.Since(compileStart).Milliseconds()
		if ctx.Err() == context.DeadlineExceeded {
			return &Result{
				Stderr:     "Compilation timed out after 30 seconds",
				ExitCode:   -1,
				DurationMs: elapsed,
			}, nil
		}
		return &Result{
			Stderr:     compileOut.String(),
			ExitCode:   -1,
			DurationMs: elapsed,
		}, nil
	}

	runCmd := substitute(rc.RunCmd, map[string]string{
		"{file}":      srcPath,
		"{output}":    outputPath,
		"{basename}":  basename,
		"{classname}": basename,
	})

	parts = strings.Fields(runCmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty run command")
	}

	cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = dir

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	return buildResult(&stdout, &stderr, runErr, elapsed, ctx)
}

func buildResult(stdout, stderr *strings.Builder, runErr error, elapsed int64, ctx context.Context) (*Result, error) {
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
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

func substitute(tmpl string, vars map[string]string) string {
	s := tmpl
	for k, v := range vars {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

func altInterpreters(primary string) []string {
	return []string{primary}
}
