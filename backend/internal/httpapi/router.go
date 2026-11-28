package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/service"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB

type apiErrorBody struct {
	Code    string                    `json:"code"`
	Message string                    `json:"message"`
	Details []program.ValidationIssue `json:"details,omitempty"`
}

type apiErrorEnvelope struct {
	Error apiErrorBody `json:"error"`
}

type saveProgramRequest struct {
	Name  string        `json:"name"`
	Graph program.Graph `json:"graph"`
}

type runDraftRequest struct {
	Graph program.Graph `json:"graph"`
}

type listProgramsResponse struct {
	Programs []program.ProgramSummary `json:"programs"`
}

type handler struct {
	svc    *service.ProgramService
	logger *slog.Logger
}

// NewRouter constructs the chi HTTP handler for the visual program REST API.
func NewRouter(svc *service.ProgramService, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &handler{svc: svc, logger: logger}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(h.recoverJSON)

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", nil)
	})

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1/programs", func(r chi.Router) {
		r.Post("/", h.createProgram)
		r.Get("/", h.listPrograms)
		r.Post("/run", h.runDraftProgram)
		r.Get("/{id}", h.getProgram)
		r.Put("/{id}", h.updateProgram)
		r.Post("/{id}/run", h.runSavedProgram)
	})

	return r
}

func (h *handler) recoverJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				h.logger.Error("http handler panic recovered", "method", r.Method, "path", r.URL.Path, "panic", rec)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (h *handler) createProgram(w http.ResponseWriter, r *http.Request) {
	var req saveProgramRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	created, err := h.svc.CreateProgram(r.Context(), req.Name, req.Graph)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/programs/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *handler) listPrograms(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListPrograms(r.Context())
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	if items == nil {
		items = []program.ProgramSummary{}
	}
	writeJSON(w, http.StatusOK, listProgramsResponse{Programs: items})
}

func (h *handler) getProgram(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.svc.GetProgram(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *handler) updateProgram(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req saveProgramRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	updated, err := h.svc.UpdateProgram(r.Context(), id, req.Name, req.Graph)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *handler) runSavedProgram(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.svc.RunSavedProgram(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *handler) runDraftProgram(w http.ResponseWriter, r *http.Request) {
	var req runDraftRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	res, err := h.svc.RunDraftProgram(r.Context(), req.Graph)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *program.ValidationError
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "program not found", nil)
	case errors.As(err, &verr):
		writeError(w, http.StatusUnprocessableEntity, "validation_error", verr.Error(), verr.Issues)
	case errors.Is(err, service.ErrPolicyViolation):
		writeError(w, http.StatusUnprocessableEntity, "sandbox_policy_violation", "program rejected by sandbox security policy", nil)
	default:
		h.logger.Error("internal handler error", "method", r.Method, "path", r.URL.Path, "err", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return errors.New("request body exceeds 1 MiB limit")
		}
		return errors.New("malformed or invalid JSON request body")
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string, details []program.ValidationIssue) {
	writeJSON(w, status, apiErrorEnvelope{
		Error: apiErrorBody{
			Code:    code,
			Message: msg,
			Details: details,
		},
	})
}
