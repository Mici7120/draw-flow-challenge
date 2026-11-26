package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/domain"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/repository"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/runner"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/service"
)

func setupTestRouter() http.Handler {
	repo := repository.NewMemoryRepository()
	pyRunner := runner.NewPythonRunner("python3", 3*time.Second)
	svc := service.NewProgramService(repo, pyRunner)
	return NewRouter(svc, "*")
}

func TestProgramsCRUDAndRunEndpoints(t *testing.T) {
	router := setupTestRouter()

	// 1. Health check
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for health, got %d", rec.Code)
	}

	// 2. Save a program
	saveBody := `{
		"name": "Loop Sum",
		"graph": {"drawflow":{"Home":{"data":{}}}},
		"code": "total = 0\nfor i in range(1, 5):\n    total = total + i\nprint(total)\n"
	}`
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/programs", bytes.NewBufferString(saveBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for save, got %d: %s", rec.Code, rec.Body.String())
	}

	var created domain.Program
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode created program: %v", err)
	}
	if created.ID == "" || created.Name != "Loop Sum" {
		t.Fatalf("unexpected created program: %+v", created)
	}

	// 3. List programs
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/programs", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for list, got %d", rec.Code)
	}
	var listResp struct {
		Programs []domain.ProgramSummary `json:"programs"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode list: %v", err)
	}
	if len(listResp.Programs) != 1 || listResp.Programs[0].ID != created.ID {
		t.Fatalf("unexpected list response: %+v", listResp.Programs)
	}

	// 4. Get program by ID
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/programs/"+created.ID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for get, got %d", rec.Code)
	}

	// 5. Run saved program by ID
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/programs/"+created.ID+"/run", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for run by id, got %d: %s", rec.Code, rec.Body.String())
	}
	var runRes domain.RunResult
	if err := json.NewDecoder(rec.Body).Decode(&runRes); err != nil {
		t.Fatalf("failed to decode run result: %v", err)
	}
	if runRes.ExitCode != 0 || strings.TrimSpace(runRes.Stdout) != "10" {
		t.Fatalf("expected stdout '10' with exit 0, got %+v", runRes)
	}

	// 6. Run ad-hoc code
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/programs/run", bytes.NewBufferString(`{"code":"print(6 * 7)\n"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for ad-hoc run, got %d", rec.Code)
	}
	runRes = domain.RunResult{}
	_ = json.NewDecoder(rec.Body).Decode(&runRes)
	if strings.TrimSpace(runRes.Stdout) != "42" {
		t.Fatalf("expected stdout '42', got %q", runRes.Stdout)
	}

	// 7. Update program
	updateBody := `{
		"name": "Loop Sum v2",
		"graph": {"drawflow":{"Home":{"data":{}}}},
		"code": "print(99)\n"
	}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/programs/"+created.ID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for update, got %d", rec.Code)
	}

	// 8. Delete program
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/programs/"+created.ID, nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for delete, got %d", rec.Code)
	}

	// 9. Get deleted returns 404
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/programs/"+created.ID, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rec.Code)
	}
}
