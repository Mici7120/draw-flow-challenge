package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/sandbox"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/service"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/store"
)

func f64(v float64) *float64 { return &v }

type spyRunner struct {
	calls   int
	lastPy  string
	outcome sandbox.Outcome
}

func (s *spyRunner) Run(_ context.Context, py string) sandbox.Outcome {
	s.calls++
	s.lastPy = py
	return s.outcome
}

func validAddAssignPrintGraph() program.Graph {
	return program.Graph{
		Nodes: []program.Node{
			{ID: "s", Kind: program.KindStart},
			{ID: "n2", Kind: program.KindNumber, Data: program.NodeData{Value: f64(2)}},
			{ID: "n3", Kind: program.KindNumber, Data: program.NodeData{Value: f64(3)}},
			{ID: "add", Kind: program.KindMath, Data: program.NodeData{Operator: "+"}},
			{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "x"}},
			{ID: "vx", Kind: program.KindVariable, Data: program.NodeData{Name: "x"}},
			{ID: "out", Kind: program.KindOutput},
		},
		Edges: []program.Edge{
			{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
			{ID: "e2", Source: "n2", SourcePort: "value", Target: "add", TargetPort: "left"},
			{ID: "e3", Source: "n3", SourcePort: "value", Target: "add", TargetPort: "right"},
			{ID: "e4", Source: "add", SourcePort: "value", Target: "asg", TargetPort: "value"},
			{ID: "e5", Source: "asg", SourcePort: "next", Target: "out", TargetPort: "in"},
			{ID: "e6", Source: "vx", SourcePort: "value", Target: "out", TargetPort: "value"},
		},
	}
}

func TestProgramService_CRUDAndRun(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 28, 5, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		cur := now
		now = now.Add(time.Minute)
		return cur
	}

	st := store.NewMemoryStore()
	runner := sandbox.NewRunner(sandbox.Config{})
	svc := service.NewProgramService(st, runner, clock)

	// AC-6: Save incomplete draft succeeds and loads verbatim.
	draftGraph := program.Graph{
		Nodes: []program.Node{{ID: "m1", Kind: program.KindMath, Data: program.NodeData{Operator: "+"}}},
	}
	created, err := svc.CreateProgram(t.Context(), "  Work In Progress  ", draftGraph)
	if err != nil {
		t.Fatalf("CreateProgram draft: %v", err)
	}
	if created.ID == "" || created.Name != "Work In Progress" {
		t.Fatalf("created = %+v", created)
	}

	// Blank name rejected with *program.ValidationError.
	if _, err := svc.CreateProgram(t.Context(), "   ", draftGraph); err == nil {
		t.Fatal("expected validation error for blank name")
	} else {
		var verr *program.ValidationError
		if !errors.As(err, &verr) {
			t.Errorf("err type = %T, want *program.ValidationError", err)
		}
	}

	// Update saved program to a valid graph, preserving ID and CreatedAt.
	updated, err := svc.UpdateProgram(t.Context(), created.ID, "Adder", validAddAssignPrintGraph())
	if err != nil {
		t.Fatalf("UpdateProgram: %v", err)
	}
	if updated.ID != created.ID || !updated.CreatedAt.Equal(created.CreatedAt) || !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("updated timestamps/id mismatch: created=%+v updated=%+v", created, updated)
	}

	// List & Get.
	list, err := svc.ListPrograms(t.Context())
	if err != nil || len(list) != 1 || list[0].Name != "Adder" {
		t.Fatalf("ListPrograms = %+v, err=%v", list, err)
	}
	loaded, err := svc.GetProgram(t.Context(), created.ID)
	if err != nil || loaded.Name != "Adder" {
		t.Fatalf("GetProgram = %+v, err=%v", loaded, err)
	}

	// AC-7: Run saved program succeeds and does NOT mutate UpdatedAt or stored record.
	res, err := svc.RunSavedProgram(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("RunSavedProgram: %v", err)
	}
	if res.Status != sandbox.StatusSucceeded || res.PythonCode != "x = 2 + 3\nprint(x)" || res.Output != "5" || res.Variables["x"] != 5 {
		t.Fatalf("RunSavedProgram result = %+v", res)
	}
	afterRun, err := svc.GetProgram(t.Context(), created.ID)
	if err != nil || !afterRun.UpdatedAt.Equal(updated.UpdatedAt) {
		t.Fatalf("RunSavedProgram mutated stored program: before=%v after=%v", updated.UpdatedAt, afterRun.UpdatedAt)
	}

	// Unknown ID returns ErrNotFound.
	if _, err := svc.RunSavedProgram(t.Context(), "does-not-exist"); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("RunSavedProgram missing err = %v, want ErrNotFound", err)
	}
}

func TestProgramService_ValidationStopsBeforeSandboxAndPolicyViolation(t *testing.T) {
	t.Parallel()

	t.Run("AC-8 invalid draft does not invoke sandbox", func(t *testing.T) {
		t.Parallel()
		spy := &spyRunner{}
		svc := service.NewProgramService(store.NewMemoryStore(), spy, nil)
		badGraph := program.Graph{
			Nodes: []program.Node{{ID: "m1", Kind: program.KindMath, Data: program.NodeData{Operator: "+"}}},
		}
		_, err := svc.RunDraftProgram(t.Context(), badGraph)
		var verr *program.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("err = %v, want *program.ValidationError", err)
		}
		if spy.calls != 0 {
			t.Errorf("sandbox invoked %d times on invalid graph, want 0", spy.calls)
		}
	})

	t.Run("policy_violation maps to ErrPolicyViolation", func(t *testing.T) {
		t.Parallel()
		spy := &spyRunner{
			outcome: sandbox.Outcome{
				Status: sandbox.StatusPolicyViolation,
				Error:  &sandbox.ErrorDetail{Code: "sandbox_policy_violation", Message: "blocked"},
			},
		}
		svc := service.NewProgramService(store.NewMemoryStore(), spy, nil)
		_, err := svc.RunDraftProgram(t.Context(), validAddAssignPrintGraph())
		if !errors.Is(err, service.ErrPolicyViolation) {
			t.Fatalf("err = %v, want ErrPolicyViolation", err)
		}
	})

	t.Run("runtime error preserves python_code, partial output, and error detail", func(t *testing.T) {
		t.Parallel()
		spy := &spyRunner{
			outcome: sandbox.Outcome{
				Status: sandbox.StatusRuntimeError,
				Output: "1",
				Error:  &sandbox.ErrorDetail{Code: "division_by_zero", Message: "division by zero"},
			},
		}
		svc := service.NewProgramService(store.NewMemoryStore(), spy, nil)
		res, err := svc.RunDraftProgram(t.Context(), validAddAssignPrintGraph())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != sandbox.StatusRuntimeError || res.PythonCode == "" || res.Output != "1" || res.Error == nil || res.Error.Code != "division_by_zero" {
			t.Fatalf("res = %+v", res)
		}
	})
}
