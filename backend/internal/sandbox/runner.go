package sandbox

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

//go:embed harness.py
var harnessScript string

const (
	StatusSucceeded       = "succeeded"
	StatusRuntimeError    = "runtime_error"
	StatusTimeout         = "timeout"
	StatusResourceLimit   = "resource_limit"
	StatusPolicyViolation = "policy_violation"

	defaultTimeout        = 5 * time.Second
	defaultMemoryLimitMB  = 128
	defaultMaxOutputBytes = 64 * 1024
	defaultMaxOutputLines = 1000
	maxSideChannelBytes   = 1 << 20
)

// Config tunes isolated Python 3.12 execution limits.
type Config struct {
	PythonBin      string
	Timeout        time.Duration
	MemoryLimitMB  int
	MaxOutputBytes int
	MaxOutputLines int
}

// ErrorDetail carries a sanitized, user-facing failure code and message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Outcome represents the complete result of a sandboxed Python execution attempt.
type Outcome struct {
	Status           string             `json:"status"`
	Output           string             `json:"output"`
	OutputTruncated  bool               `json:"output_truncated"`
	TruncationNotice string             `json:"truncation_notice,omitempty"`
	Variables        map[string]float64 `json:"variables"`
	Error            *ErrorDetail       `json:"error,omitempty"`
}

// Runner executes validated Python 3.12 code inside an isolated subprocess.
type Runner struct {
	cfg Config
}

// NewRunner constructs a Runner applying safe defaults for zero-valued fields.
func NewRunner(cfg Config) *Runner {
	if cfg.PythonBin == "" {
		cfg.PythonBin = "python3"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.MemoryLimitMB <= 0 {
		cfg.MemoryLimitMB = defaultMemoryLimitMB
	}
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = defaultMaxOutputBytes
	}
	if cfg.MaxOutputLines <= 0 {
		cfg.MaxOutputLines = defaultMaxOutputLines
	}
	return &Runner{cfg: cfg}
}

type harnessPayload struct {
	Status    string             `json:"status"`
	Variables map[string]float64 `json:"variables"`
	Error     *ErrorDetail       `json:"error,omitempty"`
}

// Run checks the Python AST allowlist and executes code inside an isolated subprocess.
func (r *Runner) Run(ctx context.Context, pythonCode string) Outcome {
	if strings.TrimSpace(pythonCode) == "" {
		return Outcome{
			Status:    StatusSucceeded,
			Output:    "",
			Variables: map[string]float64{},
		}
	}

	tmpDir, err := os.MkdirTemp("", "df-sandbox-*")
	if err != nil {
		return internalFailure()
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Explicit os.Pipe for stdout avoids the os/exec.StdoutPipe + cmd.Wait() close race.
	outR, outW, err := os.Pipe()
	if err != nil {
		return internalFailure()
	}
	defer func() { _ = outR.Close() }()

	varR, varW, err := os.Pipe()
	if err != nil {
		_ = outW.Close()
		return internalFailure()
	}
	defer func() { _ = varR.Close() }()

	runCtx, cancel := context.WithTimeout(ctx, r.cfg.Timeout)
	defer cancel()

	cmd := exec.Command(r.cfg.PythonBin, "-I", "-S", "-c", harnessScript, strconv.Itoa(r.cfg.MemoryLimitMB))
	cmd.Dir = tmpDir
	cmd.Env = []string{}
	cmd.Stdin = strings.NewReader(pythonCode)
	cmd.Stdout = outW
	cmd.Stderr = io.Discard
	cmd.ExtraFiles = []*os.File{varW}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		_ = outW.Close()
		_ = varW.Close()
		return internalFailure()
	}
	// Close parent copies of write ends so readers observe EOF cleanly when child exits.
	_ = outW.Close()
	_ = varW.Close()

	pid := cmd.Process.Pid
	var (
		reaped          atomic.Bool
		killMu          sync.Mutex
		killedByTimeout atomic.Bool
		killerWG        sync.WaitGroup
	)
	done := make(chan struct{})
	killerWG.Add(1)
	go func() {
		defer killerWG.Done()
		select {
		case <-runCtx.Done():
			killMu.Lock()
			if !reaped.Load() {
				if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
					killedByTimeout.Store(true)
				}
				_ = syscall.Kill(-pid, syscall.SIGKILL)
			}
			killMu.Unlock()
		case <-done:
		}
	}()

	var (
		capturedOut string
		truncated   bool
		sideBytes   []byte
		ioWG        sync.WaitGroup
	)
	ioWG.Add(2)
	go func() {
		defer ioWG.Done()
		capturedOut, truncated = readBoundedOutput(outR, r.cfg.MaxOutputBytes, r.cfg.MaxOutputLines)
	}()
	go func() {
		defer ioWG.Done()
		sideBytes, _ = io.ReadAll(io.LimitReader(varR, maxSideChannelBytes))
	}()

	waitErr := cmd.Wait()
	// Mark process reaped under lock before allowing any further SIGKILL, then stop killer goroutine.
	killMu.Lock()
	reaped.Store(true)
	killMu.Unlock()
	close(done)
	killerWG.Wait()
	ioWG.Wait()
	cancel()

	out := Outcome{
		Output:          strings.TrimSuffix(capturedOut, "\n"),
		OutputTruncated: truncated,
		Variables:       map[string]float64{},
	}
	if truncated {
		out.TruncationNotice = fmt.Sprintf("output truncated: exceeded limit (%d bytes or %d lines)", r.cfg.MaxOutputBytes, r.cfg.MaxOutputLines)
	}

	if killedByTimeout.Load() {
		out.Status = StatusTimeout
		out.Error = &ErrorDetail{
			Code:    "execution_timeout",
			Message: "program execution exceeded the time limit",
		}
		return out
	}
	if ctx.Err() != nil {
		out.Status = StatusRuntimeError
		out.Error = &ErrorDetail{
			Code:    "execution_cancelled",
			Message: "program execution was cancelled",
		}
		return out
	}

	if len(sideBytes) > 0 {
		var payload harnessPayload
		if err := json.Unmarshal(sideBytes, &payload); err == nil && payload.Status != "" {
			out.Status = payload.Status
			if payload.Variables != nil {
				out.Variables = payload.Variables
			}
			out.Error = payload.Error
			return out
		}
	}

	if waitErr != nil {
		out.Status = StatusResourceLimit
		out.Error = &ErrorDetail{
			Code:    "resource_limit_exceeded",
			Message: "program exceeded sandbox resource limits",
		}
		return out
	}

	return internalFailure()
}

// readBoundedOutput reads fixed-size chunks up to maxBytes or maxLines, then drains via io.Discard
// without allocating additional memory in the Go host process.
func readBoundedOutput(r io.Reader, maxBytes, maxLines int) (string, bool) {
	var buf bytes.Buffer
	chunk := make([]byte, 4096)
	lines := 0
	truncated := false

	for {
		n, err := r.Read(chunk)
		if n > 0 {
			for i := 0; i < n; i++ {
				b := chunk[i]
				if buf.Len() >= maxBytes {
					truncated = true
					break
				}
				if lines >= maxLines {
					truncated = true
					break
				}
				buf.WriteByte(b)
				if b == '\n' {
					lines++
				}
			}
			if truncated {
				_, _ = io.Copy(io.Discard, r)
				break
			}
		}
		if err != nil {
			break
		}
	}
	return buf.String(), truncated
}

func internalFailure() Outcome {
	return Outcome{
		Status:    StatusRuntimeError,
		Variables: map[string]float64{},
		Error: &ErrorDetail{
			Code:    "sandbox_internal_error",
			Message: "sandbox runner failed to execute program",
		},
	}
}
