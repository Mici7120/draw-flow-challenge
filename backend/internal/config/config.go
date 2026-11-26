package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port          string
	DgraphHTTPURL string
	StorageMode   string
	PythonBin     string
	RunTimeout    time.Duration
	CORSOrigin    string
}

func Load() Config {
	timeoutSec := 5
	if raw := os.Getenv("RUN_TIMEOUT_SECONDS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 30 {
			timeoutSec = parsed
		}
	}

	return Config{
		Port:          getEnv("PORT", "8081"),
		DgraphHTTPURL: getEnv("DGRAPH_HTTP_URL", "http://localhost:8080"),
		StorageMode:   getEnv("STORAGE_MODE", "dgraph"),
		PythonBin:     getEnv("PYTHON_BIN", "python3"),
		RunTimeout:    time.Duration(timeoutSec) * time.Second,
		CORSOrigin:    getEnv("CORS_ORIGIN", "*"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
