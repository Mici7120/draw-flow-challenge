package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/config"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/httpapi"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/repository"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/runner"
	"github.com/Mici7120/draw-flow-challenge/backend/internal/service"
)

func main() {
	cfg := config.Load()

	var repo repository.ProgramRepository
	if cfg.StorageMode == "memory" {
		log.Println("Using in-memory program repository")
		repo = repository.NewMemoryRepository()
	} else {
		log.Printf("Using Dgraph repository at %s", cfg.DgraphHTTPURL)
		repo = repository.NewDgraphRepository(cfg.DgraphHTTPURL)
		initSchemaWithRetry(repo)
	}

	pyRunner := runner.NewPythonRunner(cfg.PythonBin, cfg.RunTimeout)
	svc := service.NewProgramService(repo, pyRunner)
	router := httpapi.NewRouter(svc, cfg.CORSOrigin)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Draw-Flow backend listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}

func initSchemaWithRetry(repo repository.ProgramRepository) {
	const maxAttempts = 10
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := repo.InitSchema(ctx)
		cancel()
		if err == nil {
			log.Println("Dgraph schema initialized")
			return
		}
		log.Printf("Dgraph schema init attempt %d/%d failed: %v", attempt, maxAttempts, err)
		time.Sleep(2 * time.Second)
	}
	log.Println("Warning: proceeding without confirmed Dgraph schema initialization")
}
