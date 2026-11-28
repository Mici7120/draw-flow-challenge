package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/httpapi"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/sandbox"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/service"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/store"
)

func f64(v float64) *float64 { return &v }

func newTestRouter(t *testing.T, cfg sandbox.Config) http.Handler {
	t.Helper()
	now := time.Date(2026, 11, 28, 6, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		cur := now
		now = now.Add(time.Second)
		return cur
	}
	svc := service.NewProgramService(store.NewMemoryStore(), sandbox.NewRunner(cfg), clock)
	return httpapi.NewRouter(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if raw, ok := body.(string); ok {
			buf.WriteString(raw)
		} else if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHTTPAPI_EndToEndAcceptanceCriteria(t *testing.T) {
	t.Parallel()
	h := newTestRouter(t, sandbox.Config{Timeout: 2 * time.Second})

	t.Run("GET /healthz returns 200 ok", func(t *testing.T) {
		rec := doJSON(t, h, http.MethodGet, "/healthz", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("AC-1 draft run Add(2,3) -> Assign(x) -> Print(x)", func(t *testing.T) {
		graph := program.Graph{
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
		rec := doJSON(t, h, http.MethodPost, "/api/v1/programs/run", map[string]any{"graph": graph})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var res service.RunResult
		if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if res.Status != "succeeded" || res.PythonCode != "x = 2 + 3\nprint(x)" || res.Output != "5" || res.Variables["x"] != 5 {
			t.Fatalf("unexpected RunResult: %+v", res)
		}
	})

	t.Run("AC-2 assignment without output returns empty stdout and final variable", func(t *testing.T) {
		graph := program.Graph{
			Nodes: []program.Node{
				{ID: "s", Kind: program.KindStart},
				{ID: "mul", Kind: program.KindMath, Data: program.NodeData{Operator: "*", Left: f64(4), Right: f64(5)}},
				{ID: "asg", Kind: program.KindAssign, Data: program.NodeData{Name: "total"}},
			},
			Edges: []program.Edge{
				{ID: "e1", Source: "s", SourcePort: "next", Target: "asg", TargetPort: "in"},
				{ID: "e2", Source: "mul", SourcePort: "value", Target: "asg", TargetPort: "value"},
			},
		}
		rec := doJSON(t, h, http.MethodPost, "/api/v1/programs/run", map[string]any{"graph": graph})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var res service.RunResult
		_ = json.NewDecoder(rec.Body).Decode(&res)
		if res.Status != "succeeded" || res.Output != "" || res.Variables["total"] != 20 {
			t.Fatalf("unexpected RunResult: %+v", res)
		}
	})

	t.Run("AC-3 if-else branching and AC-4 inclusive for with nested if", func(t *testing.T) {
		for _, tc := range []struct {
			xVal    float64
			wantOut string
		}{{12, "1"}, {5, "0"}} {
			graph := program.Graph{
				Nodes: []program.Node{
					{ID: "s", Kind: program.KindStart},
					{ID: "ax", Kind: program.KindAssign, Data: program.NodeData{Name: "x", Value: f64(tc.xVal)}},
					{ID: "vx", Kind: program.KindVariable, Data: program.NodeData{Name: "x"}},
					{ID: "cond", Kind: program.KindIfElse, Data: program.NodeData{Operator: ">=", Right: f64(10)}},
					{ID: "t1", Kind: program.KindOutput, Data: program.NodeData{Value: f64(1)}},
					{ID: "f1", Kind: program.KindOutput, Data: program.NodeData{Value: f64(0)}},
				},
				Edges: []program.Edge{
					{ID: "e1", Source: "s", SourcePort: "next", Target: "ax", TargetPort: "in"},
					{ID: "e2", Source: "ax", SourcePort: "next", Target: "cond", TargetPort: "in"},
					{ID: "e3", Source: "vx", SourcePort: "value", Target: "cond", TargetPort: "left"},
					{ID: "e4", Source: "cond", SourcePort: "then", Target: "t1", TargetPort: "in"},
					{ID: "e5", Source: "cond", SourcePort: "else", Target: "f1", TargetPort: "in"},
				},
			}
			rec := doJSON(t, h, http.MethodPost, "/api/v1/programs/run", map[string]any{"graph": graph})
			var res service.RunResult
			_ = json.NewDecoder(rec.Body).Decode(&res)
			if res.Status != "succeeded" || res.Output != tc.wantOut {
				t.Errorf("x=%v: got status=%q out=%q, want %q", tc.xVal, res.Status, res.Output, tc.wantOut)
			}
		}

		// AC-4: for i in 1..4 accumulate total = total + i inside nested if
		loopGraph := program.Graph{
			Nodes: []program.Node{
				{ID: "s", Kind: program.KindStart},
				{ID: "init", Kind: program.KindAssign, Data: program.NodeData{Name: "total", Value: f64(0)}},
				{ID: "loop", Kind: program.KindFor, Data: program.NodeData{Name: "i", From: f64(1), To: f64(4)}},
				{ID: "vi1", Kind: program.KindVariable, Data: program.NodeData{Name: "i"}},
				{ID: "cond", Kind: program.KindIfElse, Data: program.NodeData{Operator: ">", Right: f64(0)}},
				{ID: "vt", Kind: program.KindVariable, Data: program.NodeData{Name: "total"}},
				{ID: "vi2", Kind: program.KindVariable, Data: program.NodeData{Name: "i"}},
				{ID: "add", Kind: program.KindMath, Data: program.NodeData{Operator: "+"}},
				{ID: "upd", Kind: program.KindAssign, Data: program.NodeData{Name: "total"}},
			},
			Edges: []program.Edge{
				{ID: "e1", Source: "s", SourcePort: "next", Target: "init", TargetPort: "in"},
				{ID: "e2", Source: "init", SourcePort: "next", Target: "loop", TargetPort: "in"},
				{ID: "e3", Source: "loop", SourcePort: "body", Target: "cond", TargetPort: "in"},
				{ID: "e4", Source: "vi1", SourcePort: "value", Target: "cond", TargetPort: "left"},
				{ID: "e5", Source: "cond", SourcePort: "then", Target: "upd", TargetPort: "in"},
				{ID: "e6", Source: "vt", SourcePort: "value", Target: "add", TargetPort: "left"},
				{ID: "e7", Source: "vi2", SourcePort: "value", Target: "add", TargetPort: "right"},
				{ID: "e8", Source: "add", SourcePort: "value", Target: "upd", TargetPort: "value"},
			},
		}
		rec := doJSON(t, h, http.MethodPost, "/api/v1/programs/run", map[string]any{"graph": loopGraph})
		var res service.RunResult
		_ = json.NewDecoder(rec.Body).Decode(&res)
		if res.Status != "succeeded" || res.Variables["total"] != 10 || res.Variables["i"] != 4 {
			t.Fatalf("AC-4 loop result = %+v, want total=10, i=4", res)
		}
	})

	t.Run("AC-6 & AC-7 save draft, list DESC, load, update, and run saved", func(t *testing.T) {
		draft := program.Graph{Nodes: []program.Node{{ID: "m1", Kind: program.KindMath, Data: program.NodeData{Operator: "+"}}}}
		rec1 := doJSON(t, h, http.MethodPost, "/api/v1/programs", map[string]any{"name": "First", "graph": draft})
		if rec1.Code != http.StatusCreated {
			t.Fatalf("POST draft status = %d, body = %s", rec1.Code, rec1.Body.String())
		}
		var p1 program.Program
		_ = json.NewDecoder(rec1.Body).Decode(&p1)
		if rec1.Header().Get("Location") != "/api/v1/programs/"+p1.ID {
			t.Errorf("Location header = %q", rec1.Header().Get("Location"))
		}

		// Running incomplete saved draft returns 422 validation_error with block_id.
		recBadRun := doJSON(t, h, http.MethodPost, "/api/v1/programs/"+p1.ID+"/run", nil)
		if recBadRun.Code != http.StatusUnprocessableEntity {
			t.Fatalf("run invalid draft status = %d, want 422", recBadRun.Code)
		}

		// Update saved program with runnable graph and run it.
		valid := program.Graph{
			Nodes: []program.Node{
				{ID: "s", Kind: program.KindStart},
				{ID: "a1", Kind: program.KindAssign, Data: program.NodeData{Name: "z", Value: f64(7)}},
				{ID: "vz", Kind: program.KindVariable, Data: program.NodeData{Name: "z"}},
				{ID: "o1", Kind: program.KindOutput},
			},
			Edges: []program.Edge{
				{ID: "e1", Source: "s", SourcePort: "next", Target: "a1", TargetPort: "in"},
				{ID: "e2", Source: "a1", SourcePort: "next", Target: "o1", TargetPort: "in"},
				{ID: "e3", Source: "vz", SourcePort: "value", Target: "o1", TargetPort: "value"},
			},
		}
		recUpd := doJSON(t, h, http.MethodPut, "/api/v1/programs/"+p1.ID, map[string]any{"name": "First Updated", "graph": valid})
		if recUpd.Code != http.StatusOK {
			t.Fatalf("PUT status = %d, body = %s", recUpd.Code, recUpd.Body.String())
		}

		recGet := doJSON(t, h, http.MethodGet, "/api/v1/programs/"+p1.ID, nil)
		if recGet.Code != http.StatusOK {
			t.Fatalf("GET status = %d", recGet.Code)
		}

		recList := doJSON(t, h, http.MethodGet, "/api/v1/programs", nil)
		if recList.Code != http.StatusOK {
			t.Fatalf("LIST status = %d", recList.Code)
		}

		recRun := doJSON(t, h, http.MethodPost, "/api/v1/programs/"+p1.ID+"/run", nil)
		if recRun.Code != http.StatusOK {
			t.Fatalf("RUN saved status = %d, body = %s", recRun.Code, recRun.Body.String())
		}
		var runRes service.RunResult
		_ = json.NewDecoder(recRun.Body).Decode(&runRes)
		if runRes.Status != "succeeded" || runRes.Output != "7" || runRes.Variables["z"] != 7 {
			t.Fatalf("saved run result = %+v", runRes)
		}
	})

	t.Run("AC-9 & AC-10 & AC-13 error envelopes, unknown ID, malformed JSON, 101 nodes, runtime div-by-0", func(t *testing.T) {
		// Unknown ID -> 404 structured error.
		rec404 := doJSON(t, h, http.MethodGet, "/api/v1/programs/nonexistent", nil)
		if rec404.Code != http.StatusNotFound || !strings.Contains(rec404.Body.String(), `"code":"not_found"`) {
			t.Errorf("404 response = %d %s", rec404.Code, rec404.Body.String())
		}

		// Malformed JSON -> 400 structured error.
		rec400 := doJSON(t, h, http.MethodPost, "/api/v1/programs", "{bad json")
		if rec400.Code != http.StatusBadRequest || !strings.Contains(rec400.Body.String(), `"code":"invalid_request"`) {
			t.Errorf("400 response = %d %s", rec400.Code, rec400.Body.String())
		}

		// Blank name -> 422 structured error.
		recBlank := doJSON(t, h, http.MethodPost, "/api/v1/programs", map[string]any{"name": "  ", "graph": program.Graph{}})
		if recBlank.Code != http.StatusUnprocessableEntity || !strings.Contains(recBlank.Body.String(), `"code":"validation_error"`) {
			t.Errorf("blank name response = %d %s", recBlank.Code, recBlank.Body.String())
		}

		// AC-13: 101 nodes rejected with 422.
		tooMany := make([]program.Node, program.MaxNodes+1)
		for i := range tooMany {
			tooMany[i] = program.Node{ID: fmt.Sprintf("n%d", i), Kind: program.KindNumber, Data: program.NodeData{Value: f64(1)}}
		}
		rec101 := doJSON(t, h, http.MethodPost, "/api/v1/programs", map[string]any{"name": "Big", "graph": program.Graph{Nodes: tooMany}})
		if rec101.Code != http.StatusUnprocessableEntity {
			t.Errorf("101 nodes status = %d, want 422", rec101.Code)
		}

		// AC-10: runtime division by zero returns 200 RunResult with sanitized error and zero host paths.
		divZero := program.Graph{
			Nodes: []program.Node{
				{ID: "s", Kind: program.KindStart},
				{ID: "o1", Kind: program.KindOutput, Data: program.NodeData{Value: f64(9)}},
				{ID: "div", Kind: program.KindMath, Data: program.NodeData{Operator: "/", Left: f64(10), Right: f64(0)}},
				{ID: "a1", Kind: program.KindAssign, Data: program.NodeData{Name: "x"}},
			},
			Edges: []program.Edge{
				{ID: "e1", Source: "s", SourcePort: "next", Target: "o1", TargetPort: "in"},
				{ID: "e2", Source: "o1", SourcePort: "next", Target: "a1", TargetPort: "in"},
				{ID: "e3", Source: "div", SourcePort: "value", Target: "a1", TargetPort: "value"},
			},
		}
		recDiv := doJSON(t, h, http.MethodPost, "/api/v1/programs/run", map[string]any{"graph": divZero})
		if recDiv.Code != http.StatusOK {
			t.Fatalf("div-by-zero status = %d, want 200", recDiv.Code)
		}
		bodyStr := recDiv.Body.String()
		if strings.Contains(bodyStr, "Traceback") || strings.Contains(bodyStr, "/tmp/") {
			t.Errorf("runtime error leaked internals: %s", bodyStr)
		}
		var runRes service.RunResult
		_ = json.Unmarshal([]byte(bodyStr), &runRes)
		if runRes.Status != "runtime_error" || runRes.Output != "9" || runRes.Error == nil || runRes.Error.Code != "division_by_zero" {
			t.Errorf("div-by-zero RunResult = %+v", runRes)
		}
	})
}

func TestHTTPAPI_HardeningAndErrorPaths(t *testing.T) {
	t.Parallel()
	h := newTestRouter(t, sandbox.Config{Timeout: 2 * time.Second})

	t.Run("unknown route returns 404 JSON envelope", func(t *testing.T) {
		rec := doJSON(t, h, http.MethodGet, "/api/v1/unknown", nil)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"code":"not_found"`) {
			t.Errorf("got %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("DELETE method returns 405 JSON envelope (out of scope v1)", func(t *testing.T) {
		rec := doJSON(t, h, http.MethodDelete, "/api/v1/programs/abc", nil)
		if rec.Code != http.StatusMethodNotAllowed || !strings.Contains(rec.Body.String(), `"code":"method_not_allowed"`) {
			t.Errorf("got %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("unknown JSON field and trailing tokens return 400", func(t *testing.T) {
		recUnknown := doJSON(t, h, http.MethodPost, "/api/v1/programs", `{"name":"x","graph":{},"extra":true}`)
		if recUnknown.Code != http.StatusBadRequest {
			t.Errorf("unknown field status = %d, want 400", recUnknown.Code)
		}
		recTrail := doJSON(t, h, http.MethodPost, "/api/v1/programs", `{"name":"x","graph":{}}{"more":1}`)
		if recTrail.Code != http.StatusBadRequest {
			t.Errorf("trailing token status = %d, want 400", recTrail.Code)
		}
	})

	t.Run("oversized body > 1 MiB returns 400", func(t *testing.T) {
		huge := `{"name":"` + strings.Repeat("a", (1<<20)+10) + `","graph":{}}`
		rec := doJSON(t, h, http.MethodPost, "/api/v1/programs", huge)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "1 MiB") {
			t.Errorf("oversized body response = %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("PUT on missing ID returns 404 and bad JSON on PUT/run returns 400", func(t *testing.T) {
		recMissing := doJSON(t, h, http.MethodPut, "/api/v1/programs/nope", map[string]any{"name": "X", "graph": program.Graph{}})
		if recMissing.Code != http.StatusNotFound {
			t.Errorf("PUT missing status = %d, want 404", recMissing.Code)
		}
		recPutBad := doJSON(t, h, http.MethodPut, "/api/v1/programs/nope", "{bad")
		if recPutBad.Code != http.StatusBadRequest {
			t.Errorf("PUT bad json status = %d, want 400", recPutBad.Code)
		}
		recRunBad := doJSON(t, h, http.MethodPost, "/api/v1/programs/run", "{bad")
		if recRunBad.Code != http.StatusBadRequest {
			t.Errorf("run draft bad json status = %d, want 400", recRunBad.Code)
		}
	})
}
