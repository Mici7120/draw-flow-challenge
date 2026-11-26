package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/domain"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	programs map[string]domain.Program
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		programs: make(map[string]domain.Program),
	}
}

func (r *MemoryRepository) InitSchema(_ context.Context) error {
	return nil
}

func (r *MemoryRepository) Ping(_ context.Context) error {
	return nil
}

func (r *MemoryRepository) Save(_ context.Context, program domain.Program) (domain.Program, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.programs[program.ID] = program
	return program, nil
}

func (r *MemoryRepository) Update(_ context.Context, program domain.Program) (domain.Program, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.programs[program.ID]
	if !ok {
		return domain.Program{}, domain.ErrNotFound
	}
	program.CreatedAt = existing.CreatedAt
	r.programs[program.ID] = program
	return program, nil
}

func (r *MemoryRepository) List(_ context.Context) ([]domain.ProgramSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]domain.ProgramSummary, 0, len(r.programs))
	for _, p := range r.programs {
		list = append(list, domain.ProgramSummary{
			ID:        p.ID,
			Name:      p.Name,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})
	return list, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id string) (domain.Program, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.programs[id]
	if !ok {
		return domain.Program{}, domain.ErrNotFound
	}
	return p, nil
}

func (r *MemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.programs[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.programs, id)
	return nil
}
