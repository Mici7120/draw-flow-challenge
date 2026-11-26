# Draw-Flow — Visual Python AST Editor

Visual node-based programming editor built with **Vue 3 + Drawflow** (frontend), **Go + chi** (REST API backend), and **Dgraph** (persistence). Programs are built visually as graphs, parsed into an Abstract Syntax Tree (AST), transpiled in real time to **Python 3**, persisted in Dgraph, and executed on the backend.

## Architecture

- **`frontend/`** — Vue 3 + Vite + TypeScript + Drawflow
  - `src/ast/` — Drawflow JSON -> AST parser (`parser.ts`), cycle/identifier validator, and Python 3 code generator (`pythonGenerator.ts`).
  - `src/nodes/` — Node definitions (`nodeDefinitions.ts`) and starter example graphs (`presets.ts`).
  - `src/components/` — Node palette, live Python code viewer & execution console, saved programs modal, and keyboard quick-add palette (`Ctrl+K`).
- **`backend/`** — Go 1.24 + `chi` v5 + Dgraph HTTP DQL client + sandboxed Python 3 runner
  - `internal/repository/` — Dgraph DQL repository (`dgraph.go`) + in-memory repository (`memory.go`) for unit tests and offline local dev.
  - `internal/runner/` — Isolated `python3 -I` runner with timeout and stdout/stderr byte limits.
  - `internal/httpapi/` — REST API router with CORS, logging, recovery, and request size limits.
- **`docker-compose.yml`** — Orchestrates Dgraph standalone, Go backend, and nginx-served Vue frontend.

## Supported Visual Nodes

| Category | Node | Shortcut | Purpose |
| --- | --- | --- | --- |
| Control Flow | **Start** | `1` | Program entry point |
| Control Flow | **Code Block** | `2` | Groups sequential statements (`Body` + `Next`) |
| Control Flow | **If / Else** | `3` | Conditional branching (`Then`, `Else`, `Next`) |
| Control Flow | **For (From-To)** | `4` | Range loop (`inclusive` or `exclusive`, `Body` + `Next`) |
| Statements | **Assign** | `5` | Variable assignment (`var = expr`) |
| Statements | **Print** | `6` | Output expression to console |
| Expressions | **Math Op** | `7` | `+`, `-`, `*`, `/`, `%` |
| Expressions | **Compare** | `8` | `==`, `!=`, `<`, `<=`, `>`, `>=` |
| Expressions | **Number** | `9` | Numeric literal |
| Expressions | **Variable** | `0` | Variable reference |

### Keyboard Shortcuts

- `Ctrl+K` or `/` — Open Quick Add Node palette
- `1`–`9`, `0` — Insert node directly onto canvas
- `Ctrl+S` — Save or update program in backend
- `Ctrl+Enter` — Execute generated Python code
- `Delete` / `Backspace` — Delete selected node
- `Escape` — Close open dialogs

## REST API Endpoints

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/api/health` | Service and Dgraph health check |
| `POST` | `/api/programs` | Save a new program (`name`, `graph`, `code`) |
| `PUT` | `/api/programs/{id}` | Update an existing saved program |
| `GET` | `/api/programs` | List all saved programs |
| `GET` | `/api/programs/{id}` | Retrieve a saved program by ID |
| `DELETE` | `/api/programs/{id}` | Delete a saved program |
| `POST` | `/api/programs/run` | Execute submitted Python code |
| `POST` | `/api/programs/{id}/run` | Execute a saved program by ID |

## Running the Project

### Option 1: Docker Compose (Recommended — includes Dgraph)

```bash
docker compose up --build
```

- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8081`
- Dgraph HTTP: `http://localhost:8080`

### Option 2: Local Development

1. **Backend** (with Dgraph running on `http://localhost:8080`, or `STORAGE_MODE=memory` for quick local testing without Docker):

```bash
cd backend
STORAGE_MODE=memory PORT=8081 go run ./cmd/server
```

2. **Frontend**:

```bash
cd frontend
npm install
npm run dev
```

## Running Tests

```bash
# Backend unit & HTTP tests
cd backend && go test -v ./...

# Frontend AST compiler tests & production build
cd frontend && npm test && npm run build
```
