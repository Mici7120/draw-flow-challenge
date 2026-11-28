package store

import (
	"context"
	"errors"

	"github.com/dgraph-io/dgo/v2/protos/api"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
)

// ErrNotFound is returned when a requested program ID does not exist in storage.
var ErrNotFound = errors.New("program not found")

// ProgramStore defines persistence operations for visual programs.
type ProgramStore interface {
	Create(ctx context.Context, p program.Program) (program.Program, error)
	Update(ctx context.Context, p program.Program) (program.Program, error)
	Get(ctx context.Context, id string) (program.Program, error)
	List(ctx context.Context) ([]program.ProgramSummary, error)
}

// DgraphClient captures the minimal Dgraph transaction and schema operations used by DgraphStore.
type DgraphClient interface {
	Alter(ctx context.Context, op *api.Operation) error
	QueryWithVars(ctx context.Context, q string, vars map[string]string) (*api.Response, error)
	Mutate(ctx context.Context, mu *api.Mutation) (*api.Response, error)
}
