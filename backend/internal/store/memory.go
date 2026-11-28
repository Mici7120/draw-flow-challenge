package store

import (
	"context"
	"sort"
	"sync"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
)

// MemoryStore is a thread-safe in-memory ProgramStore implementation used in tests.
type MemoryStore struct {
	mu       sync.RWMutex
	programs map[string]program.Program
}

// NewMemoryStore initializes an empty in-memory ProgramStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		programs: make(map[string]program.Program),
	}
}

// Create stores a new program record.
func (m *MemoryStore) Create(_ context.Context, p program.Program) (program.Program, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := cloneProgram(p)
	m.programs[stored.ID] = stored
	return cloneProgram(stored), nil
}

// Update replaces an existing program's name and graph while preserving its ID and CreatedAt.
func (m *MemoryStore) Update(_ context.Context, p program.Program) (program.Program, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.programs[p.ID]
	if !ok {
		return program.Program{}, ErrNotFound
	}
	updated := cloneProgram(p)
	updated.ID = existing.ID
	updated.CreatedAt = existing.CreatedAt
	m.programs[existing.ID] = updated
	return cloneProgram(updated), nil
}

// Get retrieves a program by ID or returns ErrNotFound.
func (m *MemoryStore) Get(_ context.Context, id string) (program.Program, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.programs[id]
	if !ok {
		return program.Program{}, ErrNotFound
	}
	return cloneProgram(p), nil
}

// List returns all saved program summaries ordered by UpdatedAt descending.
func (m *MemoryStore) List(_ context.Context) ([]program.ProgramSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]program.ProgramSummary, 0, len(m.programs))
	for _, p := range m.programs {
		out = append(out, program.ProgramSummary{
			ID:        p.ID,
			Name:      p.Name,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func cloneProgram(p program.Program) program.Program {
	cp := p
	if p.Graph.Nodes != nil {
		cp.Graph.Nodes = make([]program.Node, len(p.Graph.Nodes))
		for i, n := range p.Graph.Nodes {
			nc := n
			nc.Data = cloneNodeData(n.Data)
			if n.Position != nil {
				pos := *n.Position
				nc.Position = &pos
			}
			cp.Graph.Nodes[i] = nc
		}
	} else {
		cp.Graph.Nodes = []program.Node{}
	}
	if p.Graph.Edges != nil {
		cp.Graph.Edges = make([]program.Edge, len(p.Graph.Edges))
		copy(cp.Graph.Edges, p.Graph.Edges)
	} else {
		cp.Graph.Edges = []program.Edge{}
	}
	return cp
}

func cloneNodeData(d program.NodeData) program.NodeData {
	cp := d
	cp.Value = cloneFloatPtr(d.Value)
	cp.Left = cloneFloatPtr(d.Left)
	cp.Right = cloneFloatPtr(d.Right)
	cp.From = cloneFloatPtr(d.From)
	cp.To = cloneFloatPtr(d.To)
	return cp
}

func cloneFloatPtr(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
