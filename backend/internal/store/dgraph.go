package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Mici7120/draw-flow-challenge/backend/internal/program"
)

const dgraphSchema = `
program.id: string @index(exact) .
program.name: string .
program.graph_json: string .
program.created_at: datetime .
program.updated_at: datetime @index(hour) .

type Program {
	program.id
	program.name
	program.graph_json
	program.created_at
	program.updated_at
}
`

const queryGetProgramByID = `query getProgram($id: string) {
	programs(func: eq(program.id, $id)) {
		uid
		program.id
		program.name
		program.graph_json
		program.created_at
		program.updated_at
	}
}`

const queryListPrograms = `{
	programs(func: type(Program), orderdesc: program.updated_at) {
		program.id
		program.name
		program.created_at
		program.updated_at
	}
}`

type dgraphRecord struct {
	UID       string   `json:"uid,omitempty"`
	DType     []string `json:"dgraph.type,omitempty"`
	ID        string   `json:"program.id"`
	Name      string   `json:"program.name"`
	GraphJSON string   `json:"program.graph_json"`
	CreatedAt string   `json:"program.created_at"`
	UpdatedAt string   `json:"program.updated_at"`
}

type dgraphQueryResult struct {
	Programs []dgraphRecord `json:"programs"`
}

// DgraphStore persists visual programs in Dgraph using parameterized DQL queries and JSON mutations.
type DgraphStore struct {
	client DgraphClient
}

// NewDgraphStore constructs a DgraphStore backed by the provided DgraphClient.
func NewDgraphStore(client DgraphClient) *DgraphStore {
	return &DgraphStore{client: client}
}

// ApplySchema idempotently installs the Program Dgraph schema predicates and type.
func (d *DgraphStore) ApplySchema(ctx context.Context) error {
	if err := d.client.Alter(ctx, &api.Operation{Schema: dgraphSchema}); err != nil {
		return fmt.Errorf("apply dgraph schema: %w", err)
	}
	return nil
}

// Create inserts a new Program node into Dgraph via a structured JSON mutation.
func (d *DgraphStore) Create(ctx context.Context, p program.Program) (program.Program, error) {
	graphBytes, err := json.Marshal(p.Graph)
	if err != nil {
		return program.Program{}, fmt.Errorf("marshal program graph: %w", err)
	}
	rec := dgraphRecord{
		UID:       "_:program",
		DType:     []string{"Program"},
		ID:        p.ID,
		Name:      p.Name,
		GraphJSON: string(graphBytes),
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	payload, err := json.Marshal(rec)
	if err != nil {
		return program.Program{}, fmt.Errorf("marshal dgraph mutation: %w", err)
	}
	if _, err := d.client.Mutate(ctx, &api.Mutation{SetJson: payload, CommitNow: true}); err != nil {
		return program.Program{}, fmt.Errorf("dgraph create program: %w", err)
	}
	return cloneProgram(p), nil
}

// Update replaces an existing Program's name and graph while preserving its ID and CreatedAt.
func (d *DgraphStore) Update(ctx context.Context, p program.Program) (program.Program, error) {
	existing, err := d.findRecordByID(ctx, p.ID)
	if err != nil {
		return program.Program{}, err
	}
	if existing.UID == "" {
		return program.Program{}, fmt.Errorf("dgraph update program: stored record missing uid")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, existing.CreatedAt)
	if err != nil {
		return program.Program{}, fmt.Errorf("parse stored created_at: %w", err)
	}
	graphBytes, err := json.Marshal(p.Graph)
	if err != nil {
		return program.Program{}, fmt.Errorf("marshal program graph: %w", err)
	}
	rec := dgraphRecord{
		UID:       existing.UID,
		DType:     []string{"Program"},
		ID:        existing.ID,
		Name:      p.Name,
		GraphJSON: string(graphBytes),
		CreatedAt: existing.CreatedAt,
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	payload, err := json.Marshal(rec)
	if err != nil {
		return program.Program{}, fmt.Errorf("marshal dgraph update: %w", err)
	}
	if _, err := d.client.Mutate(ctx, &api.Mutation{SetJson: payload, CommitNow: true}); err != nil {
		return program.Program{}, fmt.Errorf("dgraph update program: %w", err)
	}
	out := cloneProgram(p)
	out.ID = existing.ID
	out.CreatedAt = createdAt
	return out, nil
}

// Get fetches a Program by ID using a parameterized DQL query.
func (d *DgraphStore) Get(ctx context.Context, id string) (program.Program, error) {
	rec, err := d.findRecordByID(ctx, id)
	if err != nil {
		return program.Program{}, err
	}
	var g program.Graph
	if rec.GraphJSON != "" {
		if err := json.Unmarshal([]byte(rec.GraphJSON), &g); err != nil {
			return program.Program{}, fmt.Errorf("decode stored graph: %w", err)
		}
	}
	if g.Nodes == nil {
		g.Nodes = []program.Node{}
	}
	if g.Edges == nil {
		g.Edges = []program.Edge{}
	}
	createdAt, err := time.Parse(time.RFC3339Nano, rec.CreatedAt)
	if err != nil {
		return program.Program{}, fmt.Errorf("parse stored created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, rec.UpdatedAt)
	if err != nil {
		return program.Program{}, fmt.Errorf("parse stored updated_at: %w", err)
	}
	return program.Program{
		ID:        rec.ID,
		Name:      rec.Name,
		Graph:     g,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// List returns all saved program summaries ordered by exact UpdatedAt descending.
func (d *DgraphStore) List(ctx context.Context) ([]program.ProgramSummary, error) {
	resp, err := d.client.QueryWithVars(ctx, queryListPrograms, nil)
	if err != nil {
		return nil, fmt.Errorf("dgraph list programs: %w", err)
	}
	var qr dgraphQueryResult
	if err := json.Unmarshal(resp.Json, &qr); err != nil {
		return nil, fmt.Errorf("decode dgraph list response: %w", err)
	}
	out := make([]program.ProgramSummary, 0, len(qr.Programs))
	for _, rec := range qr.Programs {
		createdAt, err := time.Parse(time.RFC3339Nano, rec.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse stored created_at: %w", err)
		}
		updatedAt, err := time.Parse(time.RFC3339Nano, rec.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse stored updated_at: %w", err)
		}
		out = append(out, program.ProgramSummary{
			ID:        rec.ID,
			Name:      rec.Name,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
	sortSummariesDesc(out)
	return out, nil
}

func (d *DgraphStore) findRecordByID(ctx context.Context, id string) (dgraphRecord, error) {
	resp, err := d.client.QueryWithVars(ctx, queryGetProgramByID, map[string]string{"$id": id})
	if err != nil {
		return dgraphRecord{}, fmt.Errorf("dgraph query program: %w", err)
	}
	var qr dgraphQueryResult
	if err := json.Unmarshal(resp.Json, &qr); err != nil {
		return dgraphRecord{}, fmt.Errorf("decode dgraph get response: %w", err)
	}
	if len(qr.Programs) == 0 {
		return dgraphRecord{}, ErrNotFound
	}
	return qr.Programs[0], nil
}

type grpcDgraphClient struct {
	dg *dgo.Dgraph
}

func (c *grpcDgraphClient) Alter(ctx context.Context, op *api.Operation) error {
	return c.dg.Alter(ctx, op)
}

func (c *grpcDgraphClient) QueryWithVars(ctx context.Context, q string, vars map[string]string) (*api.Response, error) {
	txn := c.dg.NewReadOnlyTxn()
	return txn.QueryWithVars(ctx, q, vars)
}

func (c *grpcDgraphClient) Mutate(ctx context.Context, mu *api.Mutation) (*api.Response, error) {
	txn := c.dg.NewTxn()
	defer func() { _ = txn.Discard(ctx) }()
	return txn.Mutate(ctx, mu)
}

// NewGRPCClient opens a Dgraph gRPC client connection to addr (e.g. "localhost:9080").
func NewGRPCClient(addr string) (DgraphClient, func() error, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("dial dgraph %q: %w", addr, err)
	}
	dg := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	return &grpcDgraphClient{dg: dg}, conn.Close, nil
}
