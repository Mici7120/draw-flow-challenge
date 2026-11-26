package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/domain"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/repository"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/runner"
	"github.com/google/uuid"
)

type ProgramService struct {
	repo   repository.ProgramRepository
	runner runner.CodeRunner
}

func NewProgramService(repo repository.ProgramRepository, codeRunner runner.CodeRunner) *ProgramService {
	return &ProgramService{
		repo:   repo,
		runner: codeRunner,
	}
}

func (s *ProgramService) Health(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *ProgramService) SaveProgram(ctx context.Context, input domain.SaveProgramInput) (domain.Program, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Program{}, fmt.Errorf("%w: program name is required", domain.ErrInvalidInput)
	}
	if len(input.Graph) == 0 || !json.Valid(input.Graph) {
		return domain.Program{}, fmt.Errorf("%w: valid graph JSON is required", domain.ErrInvalidInput)
	}

	now := time.Now().UTC()
	program := domain.Program{
		ID:        uuid.NewString(),
		Name:      name,
		Graph:     input.Graph,
		Code:      input.Code,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.repo.Save(ctx, program)
}

func (s *ProgramService) UpdateProgram(ctx context.Context, id string, input domain.SaveProgramInput) (domain.Program, error) {
	id = strings.TrimSpace(id)
	name := strings.TrimSpace(input.Name)
	if id == "" || name == "" {
		return domain.Program{}, fmt.Errorf("%w: program id and name are required", domain.ErrInvalidInput)
	}
	if len(input.Graph) == 0 || !json.Valid(input.Graph) {
		return domain.Program{}, fmt.Errorf("%w: valid graph JSON is required", domain.ErrInvalidInput)
	}

	program := domain.Program{
		ID:        id,
		Name:      name,
		Graph:     input.Graph,
		Code:      input.Code,
		UpdatedAt: time.Now().UTC(),
	}
	return s.repo.Update(ctx, program)
}

func (s *ProgramService) ListPrograms(ctx context.Context) ([]domain.ProgramSummary, error) {
	return s.repo.List(ctx)
}

func (s *ProgramService) GetProgram(ctx context.Context, id string) (domain.Program, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Program{}, fmt.Errorf("%w: program id is required", domain.ErrInvalidInput)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ProgramService) DeleteProgram(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: program id is required", domain.ErrInvalidInput)
	}
	return s.repo.Delete(ctx, id)
}

func (s *ProgramService) RunCode(ctx context.Context, code string) (domain.RunResult, error) {
	if strings.TrimSpace(code) == "" {
		return domain.RunResult{}, fmt.Errorf("%w: code is required", domain.ErrInvalidInput)
	}
	return s.runner.RunPython(ctx, code)
}

func (s *ProgramService) RunProgramByID(ctx context.Context, id string) (domain.RunResult, error) {
	program, err := s.GetProgram(ctx, id)
	if err != nil {
		return domain.RunResult{}, err
	}
	return s.RunCode(ctx, program.Code)
}
