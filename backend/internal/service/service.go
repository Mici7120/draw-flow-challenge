package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/codegen"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/sandbox"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/store"
)

var (
	// ErrNotFound signals that a requested program identifier does not exist.
	ErrNotFound = store.ErrNotFound
	// ErrPolicyViolation signals that the sandbox AST allowlist rejected the program before execution.
	ErrPolicyViolation = errors.New("sandbox policy violation")
)

// CodeRunner executes generated Python 3.12 code inside a sandbox.
type CodeRunner interface {
	Run(ctx context.Context, pythonCode string) sandbox.Outcome
}

// ErrorDetail carries a sanitized, user-facing failure code and message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RunResult is the structured execution result returned for saved or draft runs.
type RunResult struct {
	Status           string             `json:"status"`
	PythonCode       string             `json:"python_code"`
	Output           string             `json:"output"`
	OutputTruncated  bool               `json:"output_truncated"`
	TruncationNotice string             `json:"truncation_notice,omitempty"`
	Variables        map[string]float64 `json:"variables"`
	Error            *ErrorDetail       `json:"error,omitempty"`
}

// ProgramService coordinates persistence, graph validation, codegen, and sandbox execution.
type ProgramService struct {
	store  store.ProgramStore
	runner CodeRunner
	now    func() time.Time
}

// NewProgramService wires dependencies into a ProgramService.
func NewProgramService(st store.ProgramStore, runner CodeRunner, now func() time.Time) *ProgramService {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &ProgramService{store: st, runner: runner, now: now}
}

// CreateProgram validates draft invariants and persists a new visual program.
func (s *ProgramService) CreateProgram(ctx context.Context, name string, g program.Graph) (program.Program, error) {
	trimmedName, verr := program.ValidateDraft(name, g)
	if verr != nil {
		return program.Program{}, verr
	}
	ts := s.now().UTC()
	p := program.Program{
		ID:        uuid.NewString(),
		Name:      trimmedName,
		Graph:     g,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	created, err := s.store.Create(ctx, p)
	if err != nil {
		return program.Program{}, fmt.Errorf("create program: %w", err)
	}
	return created, nil
}

// UpdateProgram validates draft invariants and updates an existing saved program in place.
func (s *ProgramService) UpdateProgram(ctx context.Context, id, name string, g program.Graph) (program.Program, error) {
	trimmedName, verr := program.ValidateDraft(name, g)
	if verr != nil {
		return program.Program{}, verr
	}
	p := program.Program{
		ID:        id,
		Name:      trimmedName,
		Graph:     g,
		UpdatedAt: s.now().UTC(),
	}
	updated, err := s.store.Update(ctx, p)
	if err != nil {
		return program.Program{}, fmt.Errorf("update program: %w", err)
	}
	return updated, nil
}

// ListPrograms returns summaries of all saved programs ordered most-recently-updated first.
func (s *ProgramService) ListPrograms(ctx context.Context) ([]program.ProgramSummary, error) {
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list programs: %w", err)
	}
	return items, nil
}

// GetProgram retrieves a saved program by ID.
func (s *ProgramService) GetProgram(ctx context.Context, id string) (program.Program, error) {
	p, err := s.store.Get(ctx, id)
	if err != nil {
		return program.Program{}, fmt.Errorf("get program: %w", err)
	}
	return p, nil
}

// RunSavedProgram loads a saved program by ID and runs it without modifying the stored record.
func (s *ProgramService) RunSavedProgram(ctx context.Context, id string) (RunResult, error) {
	p, err := s.store.Get(ctx, id)
	if err != nil {
		return RunResult{}, fmt.Errorf("load program for run: %w", err)
	}
	return s.runGraph(ctx, p.Graph)
}

// RunDraftProgram validates, translates, and executes an unsaved visual program directly.
func (s *ProgramService) RunDraftProgram(ctx context.Context, g program.Graph) (RunResult, error) {
	return s.runGraph(ctx, g)
}

func (s *ProgramService) runGraph(ctx context.Context, g program.Graph) (RunResult, error) {
	ast, verr := program.ValidateForRun(g)
	if verr != nil {
		return RunResult{}, verr
	}
	pyCode, err := codegen.Generate(ast)
	if err != nil {
		return RunResult{}, fmt.Errorf("generate python: %w", err)
	}
	outcome := s.runner.Run(ctx, pyCode)
	if outcome.Status == sandbox.StatusPolicyViolation {
		return RunResult{}, ErrPolicyViolation
	}
	vars := outcome.Variables
	if vars == nil {
		vars = map[string]float64{}
	}
	res := RunResult{
		Status:           outcome.Status,
		PythonCode:       pyCode,
		Output:           outcome.Output,
		OutputTruncated:  outcome.OutputTruncated,
		TruncationNotice: outcome.TruncationNotice,
		Variables:        vars,
	}
	if outcome.Error != nil {
		res.Error = &ErrorDetail{
			Code:    outcome.Error.Code,
			Message: outcome.Error.Message,
		}
	}
	return res, nil
}
