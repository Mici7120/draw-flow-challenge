package repository

import (
	"context"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/domain"
)

type ProgramRepository interface {
	InitSchema(ctx context.Context) error
	Ping(ctx context.Context) error
	Save(ctx context.Context, program domain.Program) (domain.Program, error)
	Update(ctx context.Context, program domain.Program) (domain.Program, error)
	List(ctx context.Context) ([]domain.ProgramSummary, error)
	GetByID(ctx context.Context, id string) (domain.Program, error)
	Delete(ctx context.Context, id string) error
}
