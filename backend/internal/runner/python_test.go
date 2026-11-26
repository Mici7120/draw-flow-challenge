package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPythonRunner_Success(t *testing.T) {
	r := NewPythonRunner("python3", 3*time.Second)
	res, err := r.RunPython(context.Background(), "x = 2 + 3\nprint(x)\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", res.ExitCode, res.Stderr)
	}
	if strings.TrimSpace(res.Stdout) != "5" {
		t.Fatalf("expected stdout '5', got %q", res.Stdout)
	}
	if res.TimedOut {
		t.Fatalf("expected TimedOut=false")
	}
}

func TestPythonRunner_RuntimeError(t *testing.T) {
	r := NewPythonRunner("python3", 3*time.Second)
	res, err := r.RunPython(context.Background(), "print(1 / 0)\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if !strings.Contains(res.Stderr, "ZeroDivisionError") {
		t.Fatalf("expected ZeroDivisionError in stderr, got %q", res.Stderr)
	}
}

func TestPythonRunner_Timeout(t *testing.T) {
	r := NewPythonRunner("python3", 300*time.Millisecond)
	res, err := r.RunPython(context.Background(), "while True:\n    pass\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.TimedOut {
		t.Fatalf("expected TimedOut=true")
	}
	if res.ExitCode != -1 {
		t.Fatalf("expected exit code -1, got %d", res.ExitCode)
	}
}
