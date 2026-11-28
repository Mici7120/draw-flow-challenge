package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/dgo/v2/protos/api"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/store"
)

func f64(v float64) *float64 { return &v }

func TestMemoryStore_CRUDAndOrdering(t *testing.T) {
	t.Parallel()
	st := store.NewMemoryStore()

	// Empty list returns non-nil empty slice.
	empty, err := st.List(t.Context())
	if err != nil {
		t.Fatalf("List empty error: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("List empty = %#v, want non-nil empty slice", empty)
	}

	// Unknown ID returns ErrNotFound.
	if _, err := st.Get(t.Context(), "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get missing err = %v, want ErrNotFound", err)
	}
	if _, err := st.Update(t.Context(), program.Program{ID: "missing", Name: "X"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Update missing err = %v, want ErrNotFound", err)
	}

	t1 := time.Date(2026, 11, 28, 1, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Minute)
	t3 := t2.Add(time.Minute)

	p1 := program.Program{
		ID:   "id-1",
		Name: "Test",
		Graph: program.Graph{
			Nodes: []program.Node{{ID: "n1", Kind: program.KindMath, Data: program.NodeData{Operator: "+"}}},
		},
		CreatedAt: t1,
		UpdatedAt: t1,
	}
	p2 := program.Program{
		ID:        "id-2",
		Name:      "Test", // duplicate names allowed
		Graph:     program.Graph{Nodes: []program.Node{{ID: "s", Kind: program.KindStart}}},
		CreatedAt: t2,
		UpdatedAt: t2,
	}

	if _, err := st.Create(t.Context(), p1); err != nil {
		t.Fatalf("Create p1: %v", err)
	}
	if _, err := st.Create(t.Context(), p2); err != nil {
		t.Fatalf("Create p2: %v", err)
	}

	// Verify list order DESC (p2 newer than p1).
	list, err := st.List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 || list[0].ID != "id-2" || list[1].ID != "id-1" {
		t.Fatalf("List order = %+v, want [id-2, id-1]", list)
	}

	// Update p1 preserves created_at and moves p1 to the top of List DESC.
	updatedInput := program.Program{
		ID:   "id-1",
		Name: "Renamed",
		Graph: program.Graph{
			Nodes: []program.Node{{ID: "a1", Kind: program.KindAssign, Data: program.NodeData{Name: "x", Value: f64(42)}}},
		},
		UpdatedAt: t3,
	}
	gotUpdated, err := st.Update(t.Context(), updatedInput)
	if err != nil {
		t.Fatalf("Update p1: %v", err)
	}
	if !gotUpdated.CreatedAt.Equal(t1) || !gotUpdated.UpdatedAt.Equal(t3) || gotUpdated.Name != "Renamed" {
		t.Fatalf("Updated = %+v, want CreatedAt=%v UpdatedAt=%v Name=Renamed", gotUpdated, t1, t3)
	}

	loaded, err := st.Get(t.Context(), "id-1")
	if err != nil {
		t.Fatalf("Get id-1: %v", err)
	}
	if len(loaded.Graph.Nodes) != 1 || loaded.Graph.Nodes[0].ID != "a1" {
		t.Fatalf("Loaded graph = %+v, want node a1", loaded.Graph)
	}

	listAfter, err := st.List(t.Context())
	if err != nil {
		t.Fatalf("List after update: %v", err)
	}
	if listAfter[0].ID != "id-1" || listAfter[1].ID != "id-2" {
		t.Fatalf("List after update = %+v, want [id-1, id-2]", listAfter)
	}
}

type fakeDgraphClient struct {
	alterFn  func(ctx context.Context, op *api.Operation) error
	queryFn  func(ctx context.Context, q string, vars map[string]string) (*api.Response, error)
	mutateFn func(ctx context.Context, mu *api.Mutation) (*api.Response, error)
}

func (f *fakeDgraphClient) Alter(ctx context.Context, op *api.Operation) error {
	if f.alterFn != nil {
		return f.alterFn(ctx, op)
	}
	return nil
}

func (f *fakeDgraphClient) QueryWithVars(ctx context.Context, q string, vars map[string]string) (*api.Response, error) {
	return f.queryFn(ctx, q, vars)
}

func (f *fakeDgraphClient) Mutate(ctx context.Context, mu *api.Mutation) (*api.Response, error) {
	return f.mutateFn(ctx, mu)
}

func TestDgraphStore_ParameterizedDQLAndMutations(t *testing.T) {
	t.Parallel()

	t.Run("ApplySchema installs Program schema", func(t *testing.T) {
		t.Parallel()
		var gotSchema string
		fc := &fakeDgraphClient{
			alterFn: func(_ context.Context, op *api.Operation) error {
				gotSchema = op.Schema
				return nil
			},
		}
		ds := store.NewDgraphStore(fc)
		if err := ds.ApplySchema(t.Context()); err != nil {
			t.Fatalf("ApplySchema: %v", err)
		}
		if !strings.Contains(gotSchema, "program.id: string @index(exact)") || !strings.Contains(gotSchema, "type Program") {
			t.Errorf("unexpected schema:\n%s", gotSchema)
		}
	})

	t.Run("Get uses parameterized $id variable and never interpolates untrusted input", func(t *testing.T) {
		t.Parallel()
		maliciousID := `") { uid } evil(func: has(dgraph.type)) {`
		t1 := time.Date(2026, 11, 28, 2, 0, 0, 0, time.UTC)

		fc := &fakeDgraphClient{
			queryFn: func(_ context.Context, q string, vars map[string]string) (*api.Response, error) {
				if strings.Contains(q, maliciousID) {
					t.Fatalf("DQL injection: untrusted ID interpolated directly into query:\n%s", q)
				}
				if vars["$id"] != maliciousID {
					t.Errorf("vars[$id] = %q, want %q", vars["$id"], maliciousID)
				}
				payload := map[string]any{
					"programs": []map[string]any{
						{
							"uid":                "0x1",
							"program.id":         maliciousID,
							"program.name":       "Safe",
							"program.graph_json": `{"nodes":[],"edges":[]}`,
							"program.created_at": t1.Format(time.RFC3339Nano),
							"program.updated_at": t1.Format(time.RFC3339Nano),
						},
					},
				}
				b, _ := json.Marshal(payload)
				return &api.Response{Json: b}, nil
			},
		}

		ds := store.NewDgraphStore(fc)
		got, err := ds.Get(t.Context(), maliciousID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.ID != maliciousID || got.Name != "Safe" {
			t.Errorf("Get = %+v", got)
		}
	})

	t.Run("Create, Update, Get not-found, and List DESC", func(t *testing.T) {
		t.Parallel()
		t1 := time.Date(2026, 11, 28, 3, 0, 0, 0, time.UTC)
		t2 := t1.Add(time.Minute)

		var lastMutate []byte
		fc := &fakeDgraphClient{
			mutateFn: func(_ context.Context, mu *api.Mutation) (*api.Response, error) {
				if !mu.CommitNow {
					t.Error("expected CommitNow = true")
				}
				lastMutate = mu.SetJson
				return &api.Response{}, nil
			},
			queryFn: func(_ context.Context, q string, vars map[string]string) (*api.Response, error) {
				if vars != nil && vars["$id"] == "missing" {
					return &api.Response{Json: []byte(`{"programs":[]}`)}, nil
				}
				if vars != nil && vars["$id"] == "p1" {
					b, _ := json.Marshal(map[string]any{
						"programs": []map[string]any{{
							"uid":                "0x42",
							"program.id":         "p1",
							"program.name":       "Old",
							"program.graph_json": `{"nodes":[],"edges":[]}`,
							"program.created_at": t1.Format(time.RFC3339Nano),
							"program.updated_at": t1.Format(time.RFC3339Nano),
						}},
					})
					return &api.Response{Json: b}, nil
				}
				if strings.Contains(q, "orderdesc: program.updated_at") {
					b, _ := json.Marshal(map[string]any{
						"programs": []map[string]any{{
							"program.id":         "p1",
							"program.name":       "New",
							"program.created_at": t1.Format(time.RFC3339Nano),
							"program.updated_at": t2.Format(time.RFC3339Nano),
						}},
					})
					return &api.Response{Json: b}, nil
				}
				return &api.Response{Json: []byte(`{"programs":[]}`)}, nil
			},
		}

		ds := store.NewDgraphStore(fc)
		p := program.Program{ID: "p1", Name: "Old", CreatedAt: t1, UpdatedAt: t1}
		if _, err := ds.Create(t.Context(), p); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if !strings.Contains(string(lastMutate), `"dgraph.type":["Program"]`) {
			t.Errorf("Create mutation missing Program dgraph.type: %s", string(lastMutate))
		}

		if _, err := ds.Get(t.Context(), "missing"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Get missing err = %v, want ErrNotFound", err)
		}
		if _, err := ds.Update(t.Context(), program.Program{ID: "missing"}); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Update missing err = %v, want ErrNotFound", err)
		}

		upd, err := ds.Update(t.Context(), program.Program{ID: "p1", Name: "New", UpdatedAt: t2})
		if err != nil {
			t.Fatalf("Update p1: %v", err)
		}
		if !upd.CreatedAt.Equal(t1) || !upd.UpdatedAt.Equal(t2) || !strings.Contains(string(lastMutate), `"uid":"0x42"`) {
			t.Errorf("Update preserved created_at / uid failed: upd=%+v mut=%s", upd, string(lastMutate))
		}

		list, err := ds.List(t.Context())
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) != 1 || list[0].ID != "p1" || list[0].Name != "New" {
			t.Errorf("List = %+v", list)
		}
	})
}

func TestStore_ErrorPathsAndDeepCopyIsolation(t *testing.T) {
	t.Parallel()

	t.Run("MemoryStore isolates caller mutations to returned Graph", func(t *testing.T) {
		t.Parallel()
		st := store.NewMemoryStore()
		t1 := time.Date(2026, 11, 28, 4, 0, 0, 0, time.UTC)
		orig := program.Program{
			ID:   "iso-1",
			Name: "Isolated",
			Graph: program.Graph{
				Nodes: []program.Node{{
					ID:       "n1",
					Kind:     program.KindNumber,
					Data:     program.NodeData{Value: f64(10)},
					Position: &program.Position{X: 1, Y: 2},
				}},
				Edges: []program.Edge{{ID: "e1", Source: "n1", SourcePort: "value", Target: "n2", TargetPort: "in"}},
			},
			CreatedAt: t1,
			UpdatedAt: t1,
		}
		got, err := st.Create(t.Context(), orig)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		*got.Graph.Nodes[0].Data.Value = 999
		got.Graph.Nodes[0].Position.X = 500
		got.Graph.Edges[0].Target = "mutated"

		reloaded, err := st.Get(t.Context(), "iso-1")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if *reloaded.Graph.Nodes[0].Data.Value != 10 || reloaded.Graph.Nodes[0].Position.X != 1 || reloaded.Graph.Edges[0].Target != "n2" {
			t.Errorf("caller mutation leaked into store: %+v", reloaded.Graph)
		}
	})

	t.Run("DgraphStore propagates client and decode errors cleanly", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("dgraph unavailable")
		fc := &fakeDgraphClient{
			alterFn: func(_ context.Context, _ *api.Operation) error { return boom },
			queryFn: func(_ context.Context, _ string, _ map[string]string) (*api.Response, error) {
				return nil, boom
			},
			mutateFn: func(_ context.Context, _ *api.Mutation) (*api.Response, error) {
				return nil, boom
			},
		}
		ds := store.NewDgraphStore(fc)
		if err := ds.ApplySchema(t.Context()); !errors.Is(err, boom) {
			t.Errorf("ApplySchema err = %v, want boom", err)
		}
		if _, err := ds.Create(t.Context(), program.Program{ID: "p1"}); !errors.Is(err, boom) {
			t.Errorf("Create err = %v, want boom", err)
		}
		if _, err := ds.Get(t.Context(), "p1"); !errors.Is(err, boom) {
			t.Errorf("Get err = %v, want boom", err)
		}
		if _, err := ds.List(t.Context()); !errors.Is(err, boom) {
			t.Errorf("List err = %v, want boom", err)
		}

		badJSON := &fakeDgraphClient{
			queryFn: func(_ context.Context, _ string, _ map[string]string) (*api.Response, error) {
				return &api.Response{Json: []byte(`{not-json`)}, nil
			},
		}
		dsBad := store.NewDgraphStore(badJSON)
		if _, err := dsBad.Get(t.Context(), "p1"); err == nil {
			t.Error("expected Get decode error on malformed JSON")
		}
		if _, err := dsBad.List(t.Context()); err == nil {
			t.Error("expected List decode error on malformed JSON")
		}
	})
}
