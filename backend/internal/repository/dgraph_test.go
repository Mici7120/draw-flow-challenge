package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/domain"
)

func TestDgraphRepository_HTTPFlow(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alter":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"code":"Success"}}`))
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"status":"healthy"}]`))
		case "/mutate":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"code":"Success"}}`))
		case "/query":
			var body struct {
				Variables map[string]string `json:"variables"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Variables["$id"] == "missing" {
				_, _ = w.Write([]byte(`{"data":{"programs":[]}}`))
				return
			}
			resp := map[string]any{
				"data": map[string]any{
					"programs": []map[string]any{
						{
							"uid":        "0x1",
							"program_id": "prog-1",
							"name":       "Sum Demo",
							"graph":      `{"drawflow":{"Home":{"data":{}}}}`,
							"code":       "x = 1 + 2\nprint(x)\n",
							"created_at": now.Format(time.RFC3339),
							"updated_at": now.Format(time.RFC3339),
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	repo := NewDgraphRepository(srv.URL)
	ctx := context.Background()

	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}
	if err := repo.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	saved, err := repo.Save(ctx, domain.Program{
		ID:        "prog-1",
		Name:      "Sum Demo",
		Graph:     json.RawMessage(`{"drawflow":{"Home":{"data":{}}}}`),
		Code:      "x = 1 + 2\nprint(x)\n",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil || saved.ID != "prog-1" {
		t.Fatalf("Save failed: %v", err)
	}

	list, err := repo.List(ctx)
	if err != nil || len(list) != 1 || list[0].Name != "Sum Demo" {
		t.Fatalf("List failed: %v, %+v", err, list)
	}

	got, err := repo.GetByID(ctx, "prog-1")
	if err != nil || got.Name != "Sum Demo" {
		t.Fatalf("GetByID failed: %v", err)
	}

	if _, err := repo.GetByID(ctx, "missing"); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := repo.Delete(ctx, "prog-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}
