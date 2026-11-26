package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/domain"
)

const maxOutputBytes = 64 * 1024

type CodeRunner interface {
	RunPython(ctx context.Context, code string) (domain.RunResult, error)
}

type PythonRunner struct {
	pythonBin string
	timeout   time.Duration
}

func NewPythonRunner(pythonBin string, timeout time.Duration) *PythonRunner {
	if pythonBin == "" {
		pythonBin = "python3"
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &PythonRunner{
		pythonBin: pythonBin,
		timeout:   timeout,
	}
}

func (r *PythonRunner) RunPython(ctx context.Context, code string) (domain.RunResult, error) {
	tmpDir, err := os.MkdirTemp("", "drawflow-run-*")
	if err != nil {
		return domain.RunResult{}, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "program.py")
	if err := os.WriteFile(scriptPath, []byte(code), 0o600); err != nil {
		return domain.RunResult{}, fmt.Errorf("failed to write python script: %w", err)
	}

	execCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, r.pythonBin, "-I", scriptPath)
	cmd.Dir = tmpDir
	cmd.Env = []string{
		"PYTHONIOENCODING=utf-8",
		"PYTHONDONTWRITEBYTECODE=1",
	}

	stdout := &limitedBuffer{limit: maxOutputBytes}
	stderr := &limitedBuffer{limit: maxOutputBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	start := time.Now()
	runErr := cmd.Run()
	durationMs := time.Since(start).Milliseconds()

	result := domain.RunResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   0,
		DurationMs: durationMs,
	}

	if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		result.TimedOut = true
		result.ExitCode = -1
		if result.Stderr != "" {
			result.Stderr += "\n"
		}
		result.Stderr += fmt.Sprintf("Execution timed out after %s", r.timeout)
		return result, nil
	}

	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return domain.RunResult{}, fmt.Errorf("failed to execute python: %w", runErr)
	}

	return result, nil
}

type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		if len(p) <= remaining {
			b.buf.Write(p)
		} else {
			b.buf.Write(p[:remaining])
			b.truncated = true
		}
	} else {
		b.truncated = true
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	if b.truncated {
		return b.buf.String() + "\n... [output truncated]"
	}
	return b.buf.String()
}
