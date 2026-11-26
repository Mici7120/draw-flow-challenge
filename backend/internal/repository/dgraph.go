package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/domain"
)

const dgraphSchema = `
program_id: string @index(exact) @upsert .
name: string @index(term) .
graph: string .
code: string .
created_at: datetime @index(hour) .
updated_at: datetime @index(hour) .

type Program {
	program_id
	name
	graph
	code
	created_at
	updated_at
}
`

type DgraphRepository struct {
	baseURL    string
	httpClient *http.Client
}

func NewDgraphRepository(baseURL string) *DgraphRepository {
	return &DgraphRepository{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (r *DgraphRepository) InitSchema(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/alter", strings.NewReader(dgraphSchema))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dgraph alter request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("dgraph alter failed with status %d: %s", resp.StatusCode, string(body))
	}

	var parsed dgraphEnvelope
	if err := json.Unmarshal(body, &parsed); err == nil && len(parsed.Errors) > 0 {
		return fmt.Errorf("dgraph alter error: %s", parsed.Errors[0].Message)
	}
	return nil
}

func (r *DgraphRepository) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("dgraph health returned status %d", resp.StatusCode)
	}
	return nil
}

type dgraphProgramNode struct {
	UID       string    `json:"uid,omitempty"`
	DType     []string  `json:"dgraph.type,omitempty"`
	ProgramID string    `json:"program_id,omitempty"`
	Name      string    `json:"name,omitempty"`
	Graph     string    `json:"graph,omitempty"`
	Code      string    `json:"code,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type dgraphError struct {
	Message string `json:"message"`
}

type dgraphEnvelope struct {
	Data   json.RawMessage `json:"data"`
	Errors []dgraphError   `json:"errors"`
}

func (r *DgraphRepository) Save(ctx context.Context, program domain.Program) (domain.Program, error) {
	node := dgraphProgramNode{
		UID:       "_:new_program",
		DType:     []string{"Program"},
		ProgramID: program.ID,
		Name:      program.Name,
		Graph:     string(program.Graph),
		Code:      program.Code,
		CreatedAt: program.CreatedAt.UTC(),
		UpdatedAt: program.UpdatedAt.UTC(),
	}

	payload := map[string]any{
		"set": []dgraphProgramNode{node},
	}
	if err := r.mutate(ctx, payload); err != nil {
		return domain.Program{}, err
	}
	return program, nil
}

func (r *DgraphRepository) Update(ctx context.Context, program domain.Program) (domain.Program, error) {
	existingNode, err := r.findNodeByID(ctx, program.ID)
	if err != nil {
		return domain.Program{}, err
	}

	program.CreatedAt = existingNode.CreatedAt
	node := dgraphProgramNode{
		UID:       existingNode.UID,
		DType:     []string{"Program"},
		ProgramID: program.ID,
		Name:      program.Name,
		Graph:     string(program.Graph),
		Code:      program.Code,
		CreatedAt: existingNode.CreatedAt.UTC(),
		UpdatedAt: program.UpdatedAt.UTC(),
	}

	payload := map[string]any{
		"set": []dgraphProgramNode{node},
	}
	if err := r.mutate(ctx, payload); err != nil {
		return domain.Program{}, err
	}
	return program, nil
}

func (r *DgraphRepository) List(ctx context.Context) ([]domain.ProgramSummary, error) {
	const query = `{
		programs(func: type(Program), orderdesc: updated_at) {
			program_id
			name
			created_at
			updated_at
		}
	}`

	raw, err := r.query(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Programs []dgraphProgramNode `json:"programs"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to decode dgraph list response: %w", err)
	}

	summaries := make([]domain.ProgramSummary, 0, len(result.Programs))
	for _, p := range result.Programs {
		summaries = append(summaries, domain.ProgramSummary{
			ID:        p.ProgramID,
			Name:      p.Name,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return summaries, nil
}

func (r *DgraphRepository) GetByID(ctx context.Context, id string) (domain.Program, error) {
	node, err := r.findNodeByID(ctx, id)
	if err != nil {
		return domain.Program{}, err
	}

	graphBytes := json.RawMessage(node.Graph)
	if len(bytes.TrimSpace(graphBytes)) == 0 {
		graphBytes = json.RawMessage(`{}`)
	}

	return domain.Program{
		ID:        node.ProgramID,
		Name:      node.Name,
		Graph:     graphBytes,
		Code:      node.Code,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
	}, nil
}

func (r *DgraphRepository) Delete(ctx context.Context, id string) error {
	node, err := r.findNodeByID(ctx, id)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"delete": []map[string]string{
			{"uid": node.UID},
		},
	}
	return r.mutate(ctx, payload)
}

func (r *DgraphRepository) findNodeByID(ctx context.Context, id string) (dgraphProgramNode, error) {
	const query = `query findProgram($id: string) {
		programs(func: eq(program_id, $id)) @filter(type(Program)) {
			uid
			program_id
			name
			graph
			code
			created_at
			updated_at
		}
	}`

	raw, err := r.query(ctx, query, map[string]string{"$id": id})
	if err != nil {
		return dgraphProgramNode{}, err
	}

	var result struct {
		Programs []dgraphProgramNode `json:"programs"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return dgraphProgramNode{}, fmt.Errorf("failed to decode dgraph get response: %w", err)
	}
	if len(result.Programs) == 0 {
		return dgraphProgramNode{}, domain.ErrNotFound
	}
	return result.Programs[0], nil
}

func (r *DgraphRepository) query(ctx context.Context, dql string, vars map[string]string) (json.RawMessage, error) {
	reqPayload := map[string]any{
		"query": dql,
	}
	if len(vars) > 0 {
		reqPayload["variables"] = vars
	}

	encoded, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/query", bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dgraph query failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dgraph query returned status %d: %s", resp.StatusCode, string(body))
	}

	var env dgraphEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("failed to parse dgraph query envelope: %w", err)
	}
	if len(env.Errors) > 0 {
		return nil, fmt.Errorf("dgraph query error: %s", env.Errors[0].Message)
	}
	return env.Data, nil
}

func (r *DgraphRepository) mutate(ctx context.Context, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/mutate?commitNow=true", bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dgraph mutate failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("dgraph mutate returned status %d: %s", resp.StatusCode, string(body))
	}

	var env dgraphEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("failed to parse dgraph mutate envelope: %w", err)
	}
	if len(env.Errors) > 0 {
		return fmt.Errorf("dgraph mutate error: %s", env.Errors[0].Message)
	}
	return nil
}
