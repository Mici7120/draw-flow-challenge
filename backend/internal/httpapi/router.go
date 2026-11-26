package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/domain"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const maxBodyBytes = 2 << 20 // 2 MiB

type Server struct {
	svc        *service.ProgramService
	corsOrigin string
}

func NewRouter(svc *service.ProgramService, corsOrigin string) http.Handler {
	if corsOrigin == "" {
		corsOrigin = "*"
	}
	s := &Server{
		svc:        svc,
		corsOrigin: corsOrigin,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(s.corsMiddleware)

	r.Route("/api", func(api chi.Router) {
		api.Get("/health", s.handleHealth)

		api.Route("/programs", func(p chi.Router) {
			p.Get("/", s.handleListPrograms)
			p.Post("/", s.handleSaveProgram)
			p.Post("/run", s.handleRunCode)

			p.Route("/{id}", func(item chi.Router) {
				item.Get("/", s.handleGetProgram)
				item.Put("/", s.handleUpdateProgram)
				item.Delete("/", s.handleDeleteProgram)
				item.Post("/run", s.handleRunProgramByID)
			})
		})
	})

	return r
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.corsOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.Health(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "degraded",
			"error":  err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListPrograms(w http.ResponseWriter, r *http.Request) {
	programs, err := s.svc.ListPrograms(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"programs": programs})
}

func (s *Server) handleSaveProgram(w http.ResponseWriter, r *http.Request) {
	var input domain.SaveProgramInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	program, err := s.svc.SaveProgram(r.Context(), input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, program)
}

func (s *Server) handleGetProgram(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	program, err := s.svc.GetProgram(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, program)
}

func (s *Server) handleUpdateProgram(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input domain.SaveProgramInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	program, err := s.svc.UpdateProgram(r.Context(), id, input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, program)
}

func (s *Server) handleDeleteProgram(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.svc.DeleteProgram(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRunCode(w http.ResponseWriter, r *http.Request) {
	var input domain.RunCodeInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := s.svc.RunCode(r.Context(), input.Code)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRunProgramByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, err := s.svc.RunProgramByID(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid JSON request body")
	}
	return nil
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
