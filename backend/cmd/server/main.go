package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/httpapi"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/sandbox"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/service"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	port := envOr("PORT", "8081")
	dgraphAddr := envOr("DGRAPH_ADDR", "localhost:9080")
	pythonBin := envOr("PYTHON_BIN", "python3")
	runTimeout := envDuration("RUN_TIMEOUT", 5*time.Second)
	runMemoryMB := envInt("RUN_MEMORY_MB", 128)

	dgClient, closeFn, err := store.NewGRPCClient(dgraphAddr)
	if err != nil {
		logger.Error("failed to connect to dgraph", "addr", dgraphAddr, "err", err)
		os.Exit(1)
	}
	defer func() { _ = closeFn() }()

	dgStore := store.NewDgraphStore(dgClient)
	schemaCtx, cancelSchema := context.WithTimeout(context.Background(), 10*time.Second)
	if err := dgStore.ApplySchema(schemaCtx); err != nil {
		cancelSchema()
		logger.Error("failed to apply dgraph schema", "addr", dgraphAddr, "err", err)
		os.Exit(1)
	}
	cancelSchema()

	runner := sandbox.NewRunner(sandbox.Config{
		PythonBin:     pythonBin,
		Timeout:       runTimeout,
		MemoryLimitMB: runMemoryMB,
	})
	svc := service.NewProgramService(dgStore, runner, nil)
	router := httpapi.NewRouter(svc, logger)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting draw-flow backend", "addr", srv.Addr, "dgraph", dgraphAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutting down gracefully")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "err", err)
			os.Exit(1)
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
