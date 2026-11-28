package sandbox_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/sandbox"
)

func TestRunner_SuccessCases(t *testing.T) {
	t.Parallel()
	r := sandbox.NewRunner(sandbox.Config{})

	tests := []struct {
		name     string
		code     string
		wantOut  string
		wantVars map[string]float64
	}{
		{
			name:     "empty program succeeds with empty output and variables",
			code:     "",
			wantOut:  "",
			wantVars: map[string]float64{},
		},
		{
			name:     "AC-1 add 2 + 3 assign x and print x",
			code:     "x = 2 + 3\nprint(x)",
			wantOut:  "5",
			wantVars: map[string]float64{"x": 5},
		},
		{
			name:     "AC-2 assignment without print",
			code:     "total = 4 * 5",
			wantOut:  "",
			wantVars: map[string]float64{"total": 20},
		},
		{
			name:     "float division 5 / 2 == 2.5",
			code:     "q = 5 / 2\nprint(q)",
			wantOut:  "2.5",
			wantVars: map[string]float64{"q": 2.5},
		},
		{
			name:     "AC-4 inclusive for with nested if",
			code:     "total = 0\nfor i in range(1, 4 + 1):\n    if i > 0:\n        total = total + i",
			wantOut:  "",
			wantVars: map[string]float64{"total": 10, "i": 4},
		},
		{
			name:     "AC-5 descending for 5..2 runs zero times with pass",
			code:     "for i in range(5, 2 + 1):\n    pass",
			wantOut:  "",
			wantVars: map[string]float64{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := r.Run(t.Context(), tc.code)
			if out.Status != sandbox.StatusSucceeded {
				t.Fatalf("Status = %q, error = %+v, want %q", out.Status, out.Error, sandbox.StatusSucceeded)
			}
			if out.Output != tc.wantOut {
				t.Errorf("Output = %q, want %q", out.Output, tc.wantOut)
			}
			if out.OutputTruncated {
				t.Errorf("OutputTruncated = true, want false")
			}
			if len(out.Variables) != len(tc.wantVars) {
				t.Fatalf("Variables = %v, want %v", out.Variables, tc.wantVars)
			}
			for k, wantV := range tc.wantVars {
				if gotV, ok := out.Variables[k]; !ok || gotV != wantV {
					t.Errorf("Variables[%q] = %v (present=%v), want %v", k, gotV, ok, wantV)
				}
			}
		})
	}
}

func TestRunner_RuntimeErrors(t *testing.T) {
	t.Parallel()
	r := sandbox.NewRunner(sandbox.Config{})

	tests := []struct {
		name           string
		code           string
		wantPartialOut string
		wantErrCode    string
	}{
		{
			name:           "AC-10 division by zero preserves partial output and hides tracebacks",
			code:           "print(1)\nx = 10 / 0\nprint(2)",
			wantPartialOut: "1",
			wantErrCode:    "division_by_zero",
		},
		{
			name:           "AC-10 unassigned variable error",
			code:           "x = y + 1",
			wantPartialOut: "",
			wantErrCode:    "undefined_variable",
		},
		{
			name:           "AC-10 non-integer runtime for bound",
			code:           "n = 5 / 2\nfor i in range(1, n + 1):\n    print(i)",
			wantPartialOut: "",
			wantErrCode:    "non_integer_bound",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := r.Run(t.Context(), tc.code)
			if out.Status != sandbox.StatusRuntimeError {
				t.Fatalf("Status = %q, want %q (error=%+v)", out.Status, sandbox.StatusRuntimeError, out.Error)
			}
			if out.Output != tc.wantPartialOut {
				t.Errorf("Output = %q, want %q", out.Output, tc.wantPartialOut)
			}
			if out.Error == nil {
				t.Fatal("expected non-nil Error")
			}
			if out.Error.Code != tc.wantErrCode {
				t.Errorf("Error.Code = %q, want %q", out.Error.Code, tc.wantErrCode)
			}
			if strings.Contains(out.Error.Message, "Traceback") || strings.Contains(out.Error.Message, "/tmp/") || strings.Contains(out.Error.Message, ".py") {
				t.Errorf("Error.Message leaked internals: %q", out.Error.Message)
			}
		})
	}
}

func TestRunner_SecurityPolicyViolations(t *testing.T) {
	t.Parallel()
	r := sandbox.NewRunner(sandbox.Config{})

	payloads := []struct {
		name string
		code string
	}{
		{name: "import os", code: "import os\nos.system('id')"},
		{name: "from subprocess import Popen", code: "from subprocess import Popen"},
		{name: "import socket", code: "import socket"},
		{name: "open file", code: "f = open('/etc/passwd')"},
		{name: "eval call", code: "x = eval('1+1')"},
		{name: "exec call", code: "exec('x=1')"},
		{name: "dunder __import__", code: "m = __import__('os')"},
		{name: "dunder attribute chain", code: "x = ().__class__.__bases__[0].__subclasses__()"},
		{name: "while loop blocked", code: "while 1:\n    pass"},
		{name: "function def blocked", code: "def f():\n    pass"},
	}

	for _, tc := range payloads {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := r.Run(t.Context(), tc.code)
			if out.Status != sandbox.StatusPolicyViolation {
				t.Errorf("Status = %q, want %q (error=%+v)", out.Status, sandbox.StatusPolicyViolation, out.Error)
			}
			if out.Error == nil || out.Error.Code != "sandbox_policy_violation" {
				t.Errorf("Error = %+v, want code sandbox_policy_violation", out.Error)
			}
		})
	}
}

func TestRunner_TimeoutResourceLimitAndOutputTruncation(t *testing.T) {
	t.Parallel()

	t.Run("AC-11 timeout aborts runaway loop within budget", func(t *testing.T) {
		t.Parallel()
		r := sandbox.NewRunner(sandbox.Config{Timeout: 300 * time.Millisecond})
		start := time.Now()
		out := r.Run(t.Context(), "x = 0\nfor i in range(1, 1000000000 + 1):\n    x = x + i")
		elapsed := time.Since(start)
		if out.Status != sandbox.StatusTimeout {
			t.Fatalf("Status = %q, want %q (error=%+v)", out.Status, sandbox.StatusTimeout, out.Error)
		}
		if elapsed > 3*time.Second {
			t.Errorf("timeout took too long: %v", elapsed)
		}
		if out.Error == nil || out.Error.Code != "execution_timeout" {
			t.Errorf("Error = %+v, want execution_timeout", out.Error)
		}
	})

	t.Run("AC-11 memory bomb triggers resource_limit under 128 MB cap", func(t *testing.T) {
		t.Parallel()
		r := sandbox.NewRunner(sandbox.Config{Timeout: 4 * time.Second, MemoryLimitMB: 128})
		// Doubling bit-length in 35 iterations creates an integer exceeding 128 MB.
		out := r.Run(t.Context(), "x = 2\nfor i in range(1, 35 + 1):\n    x = x * x")
		if out.Status != sandbox.StatusResourceLimit {
			t.Fatalf("Status = %q, want %q (error=%+v)", out.Status, sandbox.StatusResourceLimit, out.Error)
		}
		if out.Error == nil || out.Error.Code != "resource_limit_exceeded" {
			t.Errorf("Error = %+v, want resource_limit_exceeded", out.Error)
		}
	})

	t.Run("AC-13 output exceeding 1000 lines truncates cleanly with notice", func(t *testing.T) {
		t.Parallel()
		r := sandbox.NewRunner(sandbox.Config{Timeout: 3 * time.Second, MaxOutputLines: 1000, MaxOutputBytes: 64 * 1024})
		out := r.Run(t.Context(), "for i in range(1, 1500 + 1):\n    print(i)")
		if out.Status != sandbox.StatusSucceeded {
			t.Fatalf("Status = %q, want %q (error=%+v)", out.Status, sandbox.StatusSucceeded, out.Error)
		}
		if !out.OutputTruncated {
			t.Fatal("OutputTruncated = false, want true")
		}
		if out.TruncationNotice == "" {
			t.Error("expected non-empty TruncationNotice")
		}
		lines := strings.Split(out.Output, "\n")
		if len(lines) > 1000 {
			t.Errorf("got %d lines, want <= 1000", len(lines))
		}
		if lines[0] != "1" {
			t.Errorf("first line = %q, want \"1\"", lines[0])
		}
	})
}

func TestRunner_ByteCapTruncation(t *testing.T) {
	t.Parallel()
	r := sandbox.NewRunner(sandbox.Config{Timeout: 3 * time.Second, MaxOutputBytes: 64, MaxOutputLines: 1000})
	out := r.Run(t.Context(), "for i in range(1, 50 + 1):\n    print(1234567890)")
	if out.Status != sandbox.StatusSucceeded {
		t.Fatalf("Status = %q, want %q (error=%+v)", out.Status, sandbox.StatusSucceeded, out.Error)
	}
	if !out.OutputTruncated {
		t.Fatal("expected OutputTruncated = true when exceeding byte cap")
	}
	if len(out.Output) > 64 {
		t.Errorf("len(Output) = %d, want <= 64", len(out.Output))
	}
}
