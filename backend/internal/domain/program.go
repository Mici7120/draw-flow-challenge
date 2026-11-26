package domain

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("program not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Program struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Graph     json.RawMessage `json:"graph"`
	Code      string          `json:"code"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type ProgramSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SaveProgramInput struct {
	Name  string          `json:"name"`
	Graph json.RawMessage `json:"graph"`
	Code  string          `json:"code"`
}

type RunCodeInput struct {
	Code string `json:"code"`
}

type RunResult struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exitCode"`
	TimedOut   bool   `json:"timedOut"`
	DurationMs int64  `json:"durationMs"`
}
