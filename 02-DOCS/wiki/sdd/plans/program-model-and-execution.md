---
type: plan
title: Plan — Program Model, Persistence & Sandboxed Execution
description: Structure-level implementation plan for Sub-spec 1 — visual program AST validation, Python 3.12 codegen, Dgraph persistence, sandboxed execution, and REST API contracts.
tags: [sdd, plan, go, python, dgraph, sandbox]
timestamp: 2026-11-28T04:45:00Z
topic: sdd
slug: program-model-and-execution
status: approved
---

# Plan — Program Model, Persistence & Sandboxed Execution

> Spec: [../specs/program-model-and-execution.md](../specs/program-model-and-execution.md) · Constitution: [../constitution.md](../constitution.md) · Status: approved (autopilot)
> Last updated: 2026-11-28 (reconciled against `analyze` findings)

## 0. Global Constraints

- **Backend stack & module:** Go 1.24 module `github.com/Mici7120/draw-flow-challenge/backend` with `chi` (`github.com/go-chi/chi/v5`) router and Dgraph (`github.com/dgraph-io/dgo/v2`) in `backend/` (`go.mod` and `go.sum` committed).
- **Target runtime:** Python 3.12 executed server-side inside an isolated subprocess.
- **Error envelope & HTTP body cap:** All non-2xx HTTP failures return `{ "error": { "code": "...", "message": "..." } }` (with optional `details` array for block-level validation errors). Raw stack traces, host filesystem paths, Go panics, and Dgraph internals are never returned to clients. HTTP request bodies are capped at `1 MiB` (`1 << 20` bytes).
- **Hard caps:** Max `100` nodes per program (reject `> 100` on save or run); trimmed non-empty program name max `120` characters; printed output capped at `64 KB` or `1,000` lines (whichever comes first), truncating cleanly with `output_truncated: true` and `truncation_notice`. Single trailing `\n` from Python `print()` is trimmed from `output` (so `print(5)` → `"5"`, `print(1); print(2)` → `"1\n2"`).
- **Supported v1 operators:** Math `+`, `-`, `*`, float `/` (`5 / 2 == 2.5`) only; comparisons `==`, `!=`, `<`, `<=`, `>`, `>=` only; inclusive `for (from..to)`; explicit `output`/`print` block plus final variable values.
- **Sandbox floor:** Isolated subprocess (`python3 -I -S`, cleared env, temp dir), default hard wall-clock timeout `<= 5s` with process-group kill, default memory cap `128 MB`, zero filesystem writes (`RLIMIT_FSIZE=0`), disabled network/file/dangerous builtins, and static Python AST allowlist rejecting `os`, `subprocess`, `socket`, `open`, `eval`, `exec`, `__import__`, `compile`, dunder attribute access, and arbitrary imports.
- **Performance budgets:** Graph-to-Python translation `<= 150 ms` for a 100-node program; execution aborts cleanly within `<= 5 s` wall-clock timeout.
- **Dgraph query safety:** All DQL queries and mutations use parameterized variables or structured JSON mutations — never string-interpolated untrusted input.
- **Quality & coverage floor:** `gofmt`, `go vet ./...`, `go build ./...`, and `go test -race ./...` pass with zero errors/warnings; `>= 80%` line coverage enforced on `internal/program`, `internal/codegen`, `internal/sandbox`, `internal/store`, `internal/service`, and `internal/httpapi`.
- **Git & authorship:** Work stays on a feature branch off `main` (never direct to `main`), commits use gitmoji + Conventional Commits, authored solely by the configured human git user (`Christian`), with zero AI trailers or footers.

## 1. Context & constraints

- **Acceptance criteria shaping the design:** spec §Acceptance `AC-1` through `AC-13` (math/assignment/output, assignment without output, `if-else`, inclusive `for` + nesting, empty/descending `for` & `pass` bodies, permissive draft save/update/list/load, run saved/draft, block-referenced validation errors including floating blocks, unknown ID & invalid save errors, clean runtime errors, sandbox/timeout/memory-cap enforcement, `<= 150 ms` 100-node translation, `> 100` node rejection & `64 KB`/`1,000`-line output truncation).
- **Non-functional bars:** `<= 150 ms` translation at 100 nodes; `<= 5 s` default execution timeout; `128 MB` default sandbox memory cap; `>= 80%` line coverage on core packages (constitution #6, #13, #15).
- **Constitution & decisions in play:** Stack canon (`constitution` #1–3, `D-0001`), workspace boundary `backend/` (`constitution` #7), JSON error shape (`constitution` #8), defense-in-depth Python subprocess sandbox (`constitution` #13, `D-0002`), no secrets (`constitution` #12), three-sub-spec decomposition (`D-0004`), run response semantics (`D-0005`).
- **Out of scope (must not creep in):** Visual canvas UI (Sub-spec 2), live code viewer/run panel UI (Sub-spec 3), program deletion, user auth/login, `//`, `%`, `string` values, HTTP blocks, `and`/`or`/`not`, `while`, functions, raw Python submission bypassing the visual graph.

## 2. Architecture

```text
[ HTTP Client / Editor ] (external)
         |  JSON over HTTP (REST, <= 1 MiB body)
         v
[ httpapi (chi router + DTOs) ] (internal)
         |
         v
[ service (ProgramService) ] (internal)
   |            |              |
   v            v              v
[ store ]   [ program ]    [ codegen ] ---> [ sandbox ]
(DgraphStore) (AST model    (Python 3.12     (AST check +
   |          + validator)   translator)      isolated python3)
   v                                              |
[ Dgraph ] (external)                    [ Python 3.12 subprocess ] (external)
```

- **`internal/program`** (internal) — Owns the canonical visual program model (`Graph`, `Node`, `Edge`, `Program`, `ProgramSummary`), draft-save checks (`name`, `<= 100` nodes), and full run-time graph validator (entry reachability, floating-block rejection, port completeness, cycle detection, identifier/keyword validation, static integer literal check for `for` bounds).
- **`internal/codegen`** (internal) — Deterministically translates a validated `Graph` into readable, properly indented Python 3.12 source code (`pass` for empty bodies, explicit parenthesization for nested math, inclusive `for`).
- **`internal/sandbox`** (internal) — Enforces static Python AST allowlist checks and executes generated Python inside an isolated Python 3.12 subprocess with wall-clock timeout + process-group kill, memory/file-size limits, restricted builtins, `64 KB` / `1,000`-line stdout truncation, side-channel final-variable capture (fd 3 pipe), and sanitized runtime/policy/timeout/resource-limit error mapping.
- **`internal/store`** (internal) — Declares the persistence contract (`ProgramStore`) and provides `DgraphStore` using parameterized DQL queries and structured JSON mutations (plus an in-memory implementation used by tests).
- **`internal/service`** (internal) — Orchestrates save, update, list, get, run-saved, and run-draft use-cases across `ProgramStore`, `program`, `codegen`, and a consumer-defined `CodeRunner` interface satisfied by `*sandbox.Runner`.
- **`internal/httpapi`** (internal) — Maps REST requests to `ProgramService` calls, enforces the `1 MiB` body cap, sets HTTP status codes, and formats the constitutional `{ "error": { "code", "message" } }` envelope.
- **`cmd/server`** (internal) — Reads environment configuration, initializes Dgraph schema + store, wires dependencies, and runs the HTTP server with explicit timeouts and graceful shutdown.

**Top architectural decision:** Validate the visual graph structurally and generate Python strictly server-side from a normalized graph contract (`program` → `codegen` → `sandbox`), rather than accepting opaque graphs and raw user-submitted Python. Chosen to satisfy `spec §What the user can do` (no raw Python bypass), `spec §What result they get` (block-referenced validation errors), and `constitution` #13 (narrow sandbox surface). Secondary decision (`D-0005`): post-start execution outcomes (`succeeded`, `runtime_error`, `timeout`, `resource_limit`) return `200 OK` with a structured `RunResult` carrying `python_code`, partial `output`, `output_truncated`, and `error: { code, message }`, while pre-execution rejections (validation failures and `sandbox_policy_violation`) return `422 Unprocessable Entity` with the standard `{ "error": { "code", "message" } }` envelope.

## 3. Interfaces & contracts

### Program Validator (`internal/program`)
```text
program.ValidateDraft(name string, g Graph) -> (trimmedName string, verr *ValidationError)
  - precondition: graph decoded from request payload
  - invariant: pure function; allows missing ports and floating blocks
  - enforces: trimmed non-empty name (<= 120 chars), len(g.Nodes) <= 100, unique non-empty node IDs, known node kinds

program.ValidateForRun(g Graph) -> (ValidatedAST, *ValidationError)
  - precondition: g decoded from request or loaded from store
  - invariant: pure, deterministic; every issue carries { BlockID, Code, Message }
  - enforces:
    * len(g.Nodes) <= 100
    * at most one `start` node (0 nodes or empty entry chain yields empty ValidatedAST)
    * zero floating/unreachable nodes from entry
    * all required expression/statement inputs connected and type-compatible
    * acyclic statement flow and acyclic expression DAG
    * valid Python identifiers (^[A-Za-z_][A-Za-z0-9_]*$), not Python keywords, not blocked builtins/dunders
    * math ops in {+, -, *, /}; comparisons in {==, !=, <, <=, >, >=}
    * literal `for` bounds (`from`/`to`) must be integers (non-integer literals fail validation)
```

### Python Code Generator (`internal/codegen`)
```text
codegen.Generate(ast program.ValidatedAST) -> (pythonCode string, err error)
  - precondition: ast passed program.ValidateForRun
  - invariant: pure, deterministic; <= 150 ms for 100 nodes
  - postcondition:
    * readable Python 3.12 with 4-space indentation
    * empty program produces "" (no statements)
    * empty if/else/for bodies produce `pass`
    * nested math expressions parenthesized to preserve AST precedence
    * inclusive for loop generates `for <var> in range(<from>, <to> + 1):`
```

### Sandboxed Python Runner (`internal/sandbox`)
```text
sandbox.NewRunner(cfg Config) -> *Runner
  - Config defaults: PythonBin="python3", Timeout=5s, MemoryLimitMB=128, MaxOutputBytes=65536, MaxOutputLines=1000

(*sandbox.Runner).Run(ctx context.Context, pythonCode string) -> Outcome
  - Outcome: { Status: "succeeded"|"runtime_error"|"timeout"|"resource_limit"|"policy_violation",
               Output: string, OutputTruncated: bool, TruncationNotice: string,
               Variables: map[string]float64, Error: *ErrorDetail }
  - invariant:
    * static Python AST allowlist runs BEFORE execution; disallowed constructs return Status="policy_violation" without executing user code
    * executes via isolated python3 subprocess (`-I -S`, empty env, temp workdir, memory cap, RLIMIT_FSIZE=0, restricted __builtins__)
    * hard timeout kills entire process group (SIGKILL) -> Status="timeout"
    * memory exhaustion -> Status="resource_limit"
    * stdout capped at MaxOutputBytes (64 KB) or MaxOutputLines (1000 lines), trimming single trailing `\n`, setting OutputTruncated=true
    * final user variables returned over dedicated side-channel pipe (fd 3), never parsed from stdout
    * runtime errors (ZeroDivisionError, NameError, non-integer range bound TypeError, OverflowError) sanitized -> Status="runtime_error", zero host paths or raw tracebacks
```

### Persistence Store (`internal/store`)
```text
ProgramStore:
  Create(ctx, p program.Program) -> (program.Program, error)
  Update(ctx, p program.Program) -> (program.Program, error | ErrNotFound)
  Get(ctx, id string) -> (program.Program, error | ErrNotFound)
  List(ctx) -> ([]program.ProgramSummary, error)   // ordered by updated_at DESC; empty slice when none
  - invariant: DgraphStore uses parameterized DQL variables and structured JSON mutations only
  - invariant: Update preserves id and created_at, replaces name and graph, updates updated_at (last-write-wins)
```

### Application Service (`internal/service`)
```text
CodeRunner (consumer interface in package service):
  Run(ctx context.Context, pythonCode string) sandbox.Outcome

service.NewProgramService(store store.ProgramStore, runner CodeRunner, clock func() time.Time) -> *ProgramService

(*ProgramService):
  CreateProgram(ctx, name, graph) -> (program.Program, error)          // *program.ValidationError
  UpdateProgram(ctx, id, name, graph) -> (program.Program, error)      // *program.ValidationError | ErrNotFound
  ListPrograms(ctx) -> ([]program.ProgramSummary, error)
  GetProgram(ctx, id) -> (program.Program, error)                      // ErrNotFound
  RunSavedProgram(ctx, id) -> (RunResult, error)                       // ErrNotFound | *program.ValidationError | ErrPolicyViolation
  RunDraftProgram(ctx, graph) -> (RunResult, error)                    // *program.ValidationError | ErrPolicyViolation
```

### REST HTTP API (`internal/httpapi`)
```text
POST   /api/v1/programs          { name, graph } -> 201 Program (+ Location header) | 400 | 422
GET    /api/v1/programs                          -> 200 { programs: ProgramSummary[] }
GET    /api/v1/programs/{id}                     -> 200 Program | 404
PUT    /api/v1/programs/{id}     { name, graph } -> 200 Program | 400 | 404 | 422
POST   /api/v1/programs/{id}/run                 -> 200 RunResult | 404 | 422
POST   /api/v1/programs/run      { graph }       -> 200 RunResult | 400 | 422
GET    /healthz                                  -> 200 { status: "ok" }

HTTP error mapping:
  - Malformed JSON / body > 1 MiB -> 400 { "error": { "code": "invalid_request", "message": "..." } }
  - ErrNotFound                   -> 404 { "error": { "code": "not_found", "message": "..." } }
  - *program.ValidationError      -> 422 { "error": { "code": "validation_error", "message": "...", "details": [...] } }
  - service.ErrPolicyViolation    -> 422 { "error": { "code": "sandbox_policy_violation", "message": "..." } }
  - Unexpected internal error     -> 500 { "error": { "code": "internal_error", "message": "internal server error" } }
  - Post-start execution outcomes (`succeeded`, `runtime_error`, `timeout`, `resource_limit`) -> 200 RunResult
```

## 4. Data model & flow

**Entities**

- **`Program`** — `id` (UUID string), `name` (trimmed string, 1..120 chars), `graph` (`Graph`), `created_at` (RFC 3339 UTC), `updated_at` (RFC 3339 UTC).
- **`ProgramSummary`** — `id`, `name`, `created_at`, `updated_at`.
- **`Graph`** — `nodes` (`[]Node`, max 100), `edges` (`[]Edge`).
- **`Node`** — `id` (unique non-empty string), `kind` (`start` | `code_block` | `number` | `variable` | `assign` | `math` | `if_else` | `for` | `output`), `data` (`NodeData`), optional `position` `{ x, y }` preserved verbatim for Sub-spec 2.
- **`NodeData` canonical JSON fields by `kind`:**
  - `start`, `code_block`: `{}` (no required fields).
  - `number`: `{ "value": <number> }`.
  - `variable`: `{ "name": "<identifier>" }`.
  - `assign`: `{ "name": "<identifier>", "value"?: <number> }` (`value` from edge or inline literal).
  - `math`: `{ "operator": "+"|"-"|"*"|"/", "left"?: <number>, "right"?: <number> }` (`left`/`right` from edge or inline literal).
  - `if_else`: `{ "operator": "=="|"!="|"<"|"<="|">"|">=", "left"?: <number>, "right"?: <number> }` (`left`/`right` from edge or inline literal).
  - `for`: `{ "name": "<loop_var_identifier>", "from"?: <number>, "to"?: <number> }` (`from`/`to` from edge or inline integer literal).
  - `output`: `{ "value"?: <number> }` (`value` from edge or inline literal).
- **`Edge` & port wiring:** `id`, `source`, `source_port`, `target`, `target_port`:
  - Statement flow: `start.next` → statement/`code_block.in`; statement.`next` → next statement/`code_block.in`.
  - Scoped bodies: `code_block.body`, `if_else.then`, `if_else.else`, and `for.body` each connect to either a `code_block.in` or directly to the first statement's `in` port; `code_block.next` continues after the block.
  - Expression values: `number.value`, `variable.value`, or `math.value` → target port (`value`, `left`, `right`, `from`, `to`).
- **`RunResult`** — `status` (`succeeded` | `runtime_error` | `timeout` | `resource_limit`), `python_code`, `output`, `output_truncated`, `truncation_notice` (optional), `variables` (`map[string]float64`), `error` (optional `{ code, message }`).

**Primary flow (`AC-1`: `Add(2, 3) → Assign(x) → Print(x)` run as draft)**

1. Client sends `POST /api/v1/programs/run` with `{ graph }`.
2. `httpapi` decodes JSON (enforcing `1 MiB` body cap) and calls `service.RunDraftProgram(ctx, graph)`.
3. `program.ValidateForRun(graph)` checks `<= 100` nodes, entry reachability, zero floating blocks, required ports, acyclicity, valid variable name `x`, and operator `+`; produces `ValidatedAST`.
4. `codegen.Generate(ast)` emits `"x = 2 + 3\nprint(x)"`.
5. `sandbox.Runner.Run(ctx, pythonCode)` parses Python against the AST allowlist, spawns isolated `python3 -I -S` with `5s` timeout, `128 MB` limit, `RLIMIT_FSIZE=0`, and fd-3 pipe; captures stdout `"5"` (single trailing `\n` trimmed) and fd-3 variables `{ "x": 5 }`.
6. `service` returns `RunResult{ Status: "succeeded", PythonCode: "x = 2 + 3\nprint(x)", Output: "5", OutputTruncated: false, Variables: {"x": 5} }` → `httpapi` writes `200 OK`.

- **Consistency boundaries:** Each `Create` / `Update` executes as a single Dgraph transaction (`CommitNow: true`); concurrent updates to the same ID use last-write-wins (`spec §Points to clarify`).
- **Migration impact:** Initial Dgraph schema predicates (`program.id` indexed exact, `program.name`, `program.graph_json`, `program.created_at`, `program.updated_at` indexed datetime) applied idempotently at startup; no destructive migration.

## 5. Testing strategy

| Acceptance criterion | Level | Asserts | Fakes / mocks |
| --- | --- | --- | --- |
| `AC-1` (Math + Assign + Output) | unit + integration | Draft run returns `x = 2 + 3\nprint(x)`, output `"5"`, `variables.x == 5` | Real `program`, `codegen`, real Python 3.12 `sandbox`; in-memory store for HTTP test |
| `AC-2` (Assign without Output) | unit + integration | `total = 4 * 5` succeeds with empty output and `variables.total == 20` | Real `program`, `codegen`, real Python 3.12 `sandbox` |
| `AC-3` (`If-Else` branching) | unit + integration | `x >= 10` prints `"1"` for `x=12` and `"0"` for `x=5`; all 6 comparisons tested | Real `program`, `codegen`, real Python 3.12 `sandbox` |
| `AC-4` (Inclusive `For` & nesting) | unit + integration | `i` from `1` to `4` with nested `if` visits `1,2,3,4` and `total == 10` | Real `program`, `codegen`, real Python 3.12 `sandbox` |
| `AC-5` (`For` `from > to` & empty branch) | unit + integration | `for 5..2`, `for 3..3`, empty `else`/`if`/`for` emit `pass` and succeed; empty program succeeds | Real `program`, `codegen`, real Python 3.12 `sandbox` |
| `AC-6` (Save, update, list, load) | unit + integration | Draft save (incomplete allowed), list DESC order, load exact graph, update keeps ID + refreshes timestamp, duplicate names allowed | In-memory `ProgramStore` for service/HTTP tests + parameterized-DQL mock/integration tests for `DgraphStore` |
| `AC-7` (Run saved program) | integration | `POST /api/v1/programs/{id}/run` returns Python, output, variables without mutating saved program | In-memory `ProgramStore` + real `program`/`codegen`/`sandbox` |
| `AC-8` (Validation error on run) | unit + integration | Missing input, floating block, math cycle, non-integer literal `for` bound, invalid/reserved variable name → `422` with `block_id`, sandbox never invoked | Real `program` validator; spy runner asserting zero invocations |
| `AC-9` (Unknown ID & invalid save) | integration | Unknown ID → `404`; blank or >120-char name → `422`; malformed JSON / >1 MiB body → `400`; zero host paths/tracebacks | In-memory `ProgramStore` + HTTP router |
| `AC-10` (Runtime error) | unit + integration | Division by `0`, unassigned variable, non-integer runtime `for` bound (e.g. `5 / 2`) → `200` `status: "runtime_error"`, partial stdout preserved, sanitized error | Real `codegen` + real Python 3.12 `sandbox` |
| `AC-11` (Sandbox, timeout & memory cap) | unit + security integration | Huge loop killed within timeout (`status: "timeout"`); memory bomb exceeding `128 MB` cap fails cleanly (`status: "resource_limit"`); crafted payloads (`os`, `subprocess`, `socket`, `open`, `eval`, `exec`, `__import__`, dunder chains) rejected by AST allowlist (`policy_violation` → HTTP `422`) with zero host/network effects | Real Python 3.12 `sandbox` (MUST NOT be mocked) |
| `AC-12` (Translation perf) | unit benchmark/test | 100-node graph translates in `<= 150 ms` | Real `program` + `codegen` |
| `AC-13` (Size & output caps) | unit + integration | 101-node graph rejected on save and run (`422`); >1,000-line or >64 KB print loop truncates cleanly with `output_truncated: true` and notice | Real `program` + real Python 3.12 `sandbox` |

- **Where the e2e line is:** Full HTTP-to-Python-subprocess tests via `httptest` cover every API endpoint + real validator + real codegen + real Python 3.12 sandbox; `DgraphStore` tests verify parameterized DQL variables and JSON mutation payloads against a fake Dgraph client (plus `docker-compose.yml` for live Dgraph runs).
- **What MUST be real:** The Python 3.12 subprocess sandbox (`internal/sandbox`) MUST execute real `python3` in tests — mocking the sandbox would make AC-1, AC-10, AC-11, and AC-13 meaningless.

## 6. Sequencing & dependencies

1. **Scaffold `backend/` Go module & quality gate** (`go.mod`, `scripts/verify.sh` with `>= 80%` coverage gate, local `docker-compose.yml` for Dgraph) — depends on: none — `[serial]`
2. **Domain model & graph validator (`internal/program`)** (entities, draft validation, full run validation, floating-block & cycle detection, identifier rules) — depends on: #1 — `[serial]`
3. **Python 3.12 code generator (`internal/codegen`)** (deterministic AST-to-Python emitter, `pass` bodies, nested precedence, inclusive `for`, 100-node `<= 150 ms` perf test) — depends on: #2 — `[parallelizable with #4 and #5]`
4. **Sandboxed Python 3.12 runner (`internal/sandbox`)** (static AST allowlist, isolated subprocess, timeout/pgid kill, `128 MB` memory cap + `RLIMIT_FSIZE=0`, `64 KB`/`1,000`-line truncation, fd-3 variable pipe, sanitized error mapping) — depends on: #1 — `[parallelizable with #3 and #5]`
5. **Persistence layer (`internal/store`)** (`ProgramStore` interface, thread-safe in-memory store, parameterized-DQL `DgraphStore` + schema setup) — depends on: #2 — `[parallelizable with #3 and #4]`
6. **Application service (`internal/service`)** (`ProgramService` wiring store, validator, codegen, and sandbox for all 6 operations) — depends on: #2, #3, #4, #5 — `[serial]`
7. **REST HTTP API & server entrypoint (`internal/httpapi`, `cmd/server`)** (`chi` routes, DTOs, error envelope, `1 MiB` body cap, HTTP integration tests covering all endpoints and ACs, graceful shutdown) — depends on: #6 — `[serial]`
8. **Recalibrate SDD runtime config (`02-DOCS/wiki/sdd/config.yaml`) & run full closing verification** — depends on: #7 — `[serial]`

- **Parallel candidates:** #3 (`internal/codegen`), #4 (`internal/sandbox`), and #5 (`internal/store`) touch disjoint packages and share no mutable state once #2 defines `internal/program` types.
- **Hard ordering constraints:** #2 must precede #3, #5, and #6 because all three depend on the `Graph` / `ValidatedAST` / `Program` types; #6 must precede #7 because HTTP handlers delegate to `ProgramService`.

## 7. Risks & open decisions

**Risks** (ranked by likelihood/impact)

| Risk | Trigger | Impact | Mitigation / spike to retire it |
| --- | --- | --- | --- |
| Sandbox escape or host leakage in Python runner | Crafted AST, dunder chain (`().__class__.__mro__`), or raw traceback leaking `/tmp/...` | Critical security breach (constitution #13, REVIEW blocker #1/#4) | Two-layer defense: strict Python `ast.parse` allowlist rejecting all imports, dunders, and attribute access BEFORE execution + isolated subprocess (`-I -S`, restricted `__builtins__`, memory limit, `RLIMIT_FSIZE=0`, process-group `SIGKILL`) + dedicated security test suite (`AC-11`). |
| `RLIMIT_AS` vs Python 3.12 virtual address space on Linux | Python 3.12 runtime mapping shared libs > 128 MB virtual address space | False-positive `MemoryError` on valid tiny programs | Test `RLIMIT_AS` (or `RLIMIT_DATA` + allocation cap) directly against the VM's Python 3.12 binary during T004; verify normal programs succeed and a memory-allocation bomb triggers `status: "resource_limit"`. |
| Subtle AST-to-Python semantics bugs (precedence, indentation, `for` runtime float bounds) | Nested math `(a + b) * c`, `for` bound from `x = 5 / 2`, empty `else` | Wrong execution results or invalid Python (`REVIEW` blocker #2) | Exhaustive table-driven unit tests + end-to-end Python execution tests for every block combination and nesting depth. |
| DQL query injection in `DgraphStore` | User-supplied program ID or name interpolated into query string | Security blocker (`REVIEW` blocker #3) | Use DQL `$id` query variables exclusively (`QueryWithVars`) and structured JSON mutations (`SetJson`); add regression test with DQL metacharacters in ID/name. |
| Sub-spec 2 DrawFlow JSON shape mismatch | DrawFlow's raw export format differs from normalized `Graph` | Adapter work in Sub-spec 2 | Keep `Graph` cleanly normalized (`nodes` + `edges` + opaque `position`) so Sub-spec 2 maps DrawFlow ↔ `Graph` at the frontend boundary without coupling backend AST logic to DrawFlow internals. |

**Open decisions**

- None blocking Sub-spec 1 — all spec ambiguities were resolved in `clarify`. Architectural choices taken here (REST route layout, `RunResult` `200 OK` for post-start `runtime_error`/`timeout`/`resource_limit` outcomes vs `422` for pre-execution validation or `sandbox_policy_violation`, normalized `Graph` model, fd-3 side-channel variable capture) are logged in [../decisions.md](../decisions.md) as `D-0005`.

## Tasks
<!-- generated by tasks on 2026-11-28; IDs are stable, do not renumber -->

| ID | [P] | Task | Done-check | Depends-on | Trace | Risk |
| --- | --- | --- | --- | --- | --- | --- |
| T001 |  | Initialize `backend/` Go 1.24 module (`github.com/Mici7120/draw-flow-challenge/backend`), `backend/scripts/verify.sh` with `>= 80%` coverage gate, and `docker-compose.yml` for local Dgraph | `cd backend && go build ./...` exits 0; `backend/scripts/verify.sh` coverage gate proven to fail on `<80%` threshold test and pass on `>=80%` | — | AC-1..AC-13 | 1 |
| T002 |  | Implement `internal/program` domain model, draft validator, and full run validator (TDD) | `cd backend && go test -race -cover ./internal/program/...` passes with `>= 80%` coverage; covers AC-1..AC-5 AST nodes (`code_block` + all 5 families + `output`), AC-8 floating/cycle/identifier/non-int-literal rejections, AC-9 blank/>120 name, AC-13 `>100` node cap | T001 | AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-8, AC-9, AC-13 | 1 |
| T003 | [P] | Implement `internal/codegen` deterministic Python 3.12 translator (TDD) | `cd backend && go test -race -cover ./internal/codegen/...` passes with `>= 80%` coverage; covers `x = 2 + 3\nprint(x)`, float `/`, `if-else`, inclusive `for`, nested precedence, `pass` empty bodies, and 100-node translation `<= 150 ms` | T002 | AC-1, AC-2, AC-3, AC-4, AC-5, AC-12 | 1 |
| T004 | [P] | Implement `internal/sandbox` static Python AST allowlist and isolated Python 3.12 runner (TDD) | `cd backend && go test -race -cover ./internal/sandbox/...` passes with `>= 80%` coverage against real `python3`; covers AC-1..AC-5 execution + fd-3 variables, AC-10 div-by-0/unassigned/non-int-for-bound runtime errors, AC-11 timeout kill + `128 MB` `resource_limit` memory bomb + unsafe import/builtin/dunder policy rejections, AC-13 64 KB / 1,000-line truncation | T001 | AC-1, AC-2, AC-3, AC-4, AC-5, AC-10, AC-11, AC-13 | 1 |
| T005 | [P] | Implement `internal/store` `ProgramStore` interface, in-memory store, and parameterized-DQL `DgraphStore` (TDD) | `cd backend && go test -race -cover ./internal/store/...` passes with `>= 80%` coverage; verifies create, update (keeps ID/created_at, refreshes updated_at), get, list DESC, duplicate names, `ErrNotFound`, and parameterized DQL queries with metacharacter safety | T002 | AC-6, AC-7, AC-9 | 1 |
| T006 |  | Implement `internal/service` `ProgramService` orchestrating persistence, validation, codegen, and sandbox (TDD) | `cd backend && go test -race -cover ./internal/service/...` passes with `>= 80%` coverage; verifies create/update/list/get, saved-run non-mutation (AC-7), draft-run, validation stopping before sandbox (AC-8), `ErrPolicyViolation`, and runtime/timeout/resource-limit error propagation | T002, T003, T004, T005 | AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7, AC-8, AC-9, AC-10, AC-11, AC-13 | 1 |
| T007 |  | Implement `internal/httpapi` `chi` REST router, JSON error envelope, `1 MiB` body cap, integration tests, and `cmd/server` entrypoint (TDD) | `cd backend && go test -race -cover ./internal/httpapi/... && go build ./...` passes with `>= 80%` coverage; verifies all 6 API endpoints + `/healthz`, `201/200/400/404/422` status codes, `{ "error": { "code", "message" } }` envelope, zero leaked paths/tracebacks, and end-to-end AC-1..AC-13 via `httptest` + real Python sandbox | T006 | AC-1..AC-13 | 1 |
| T008 |  | Recalibrate `02-DOCS/wiki/sdd/config.yaml` with detected Go stack, `strict_tdd: true`, and `backend/scripts/verify.sh` commands | `02-DOCS/wiki/sdd/config.yaml` records Go stack, `go test` runner, and `backend/scripts/verify.sh`; `npx @ericrisco/rsc doctor` reports healthy | T007 | Constitution #6 (SDD calibration) | 1 |
| T009 |  | Run full closing gate: all done-checks pass + `backend/scripts/verify.sh` green with `>= 80%` coverage | `cd backend && ./scripts/verify.sh` exits 0 with zero `[fail]` and zero coverage `[gap]`; all AC-1..AC-13 proven in verification record | T001..T008 | AC-1..AC-13 | — |

### Per-task Interfaces

**T001 — Interfaces**
- Consumes: Go 1.24 toolchain, `constitution.md` stack canon (`chi`, `dgo`).
- Produces: `backend/go.mod` (`module github.com/Mici7120/draw-flow-challenge/backend`, `go 1.24`), `backend/go.sum`, executable `backend/scripts/verify.sh` enforcing `gofmt`, `go vet`, `go test -race`, and `>= 80%` coverage across core packages, and root `docker-compose.yml` for local Dgraph (`dgraph/standalone`).

**T002 — Interfaces**
- Consumes: Go module `github.com/Mici7120/draw-flow-challenge/backend` (T001).
- Produces:
  - Types in package `program`: `Program{ID, Name, Graph, CreatedAt, UpdatedAt}`, `ProgramSummary{ID, Name, CreatedAt, UpdatedAt}`, `Graph{Nodes []Node, Edges []Edge}`, `Node{ID, Kind, Data NodeData, Position *Position}`, `NodeData{Value *float64, Name string, Operator string, Left *float64, Right *float64, From *float64, To *float64}`, `Edge{ID, Source, SourcePort, Target, TargetPort}`, `ValidationIssue{BlockID, Code, Message}`, `ValidationError{Issues []ValidationIssue}`, `ValidatedAST{Statements []Stmt}` (`AssignStmt`, `IfElseStmt`, `ForStmt`, `OutputStmt`, `NumberExpr`, `VarExpr`, `MathExpr`, `CompareExpr`).
  - Functions: `ValidateDraft(name string, g Graph) (string, *ValidationError)` and `ValidateForRun(g Graph) (ValidatedAST, *ValidationError)`.

**T003 — Interfaces**
- Consumes: `program.ValidatedAST` and statement/expression types (T002).
- Produces: `codegen.Generate(ast program.ValidatedAST) (string, error)` returning deterministic Python 3.12 code (`""` for empty AST, `pass` for empty bodies, inclusive `for` `range(<from>, <to> + 1)`).

**T004 — Interfaces**
- Consumes: Go module (T001) and system `python3` (3.12).
- Produces:
  - `sandbox.Config{PythonBin string, Timeout time.Duration, MemoryLimitMB int, MaxOutputBytes int, MaxOutputLines int}` with defaults `("python3", 5*time.Second, 128, 65536, 1000)`.
  - `sandbox.ErrorDetail{Code string, Message string}`.
  - `sandbox.Outcome{Status string, Output string, OutputTruncated bool, TruncationNotice string, Variables map[string]float64, Error *ErrorDetail}` where `Status` is `"succeeded" | "runtime_error" | "timeout" | "resource_limit" | "policy_violation"`.
  - `sandbox.Runner` struct with `NewRunner(cfg Config) *Runner` and `(*Runner).Run(ctx context.Context, pythonCode string) Outcome`.

**T005 — Interfaces**
- Consumes: `program.Program`, `program.ProgramSummary`, `program.Graph` (T002).
- Produces:
  - `store.ErrNotFound` sentinel error.
  - `store.ProgramStore` interface: `Create(ctx, program.Program) (program.Program, error)`, `Update(ctx, program.Program) (program.Program, error)`, `Get(ctx, id string) (program.Program, error)`, `List(ctx) ([]program.ProgramSummary, error)`.
  - `store.NewMemoryStore() *MemoryStore`, `store.NewDgraphStore(client DgraphClient) *DgraphStore`, `(*DgraphStore).ApplySchema(ctx) error`, and `store.NewGRPCClient(addr string) (DgraphClient, func() error, error)`.

**T006 — Interfaces**
- Consumes: `program.ValidateDraft`, `program.ValidateForRun`, `*program.ValidationError` (T002); `codegen.Generate` (T003); `sandbox.Outcome`, `*sandbox.Runner` (T004); `store.ProgramStore`, `store.ErrNotFound` (T005).
- Produces:
  - `service.CodeRunner` consumer interface: `Run(ctx context.Context, pythonCode string) sandbox.Outcome`.
  - `service.ErrNotFound`, `service.ErrPolicyViolation`, `service.ErrorDetail{Code, Message}`, `service.RunResult{Status, PythonCode, Output, OutputTruncated, TruncationNotice, Variables, Error}`.
  - `service.NewProgramService(st store.ProgramStore, runner CodeRunner, now func() time.Time) *ProgramService` with `CreateProgram`, `UpdateProgram`, `ListPrograms`, `GetProgram`, `RunSavedProgram`, `RunDraftProgram` (returning `*program.ValidationError`, `ErrNotFound`, or `ErrPolicyViolation` on failures).

**T007 — Interfaces**
- Consumes: `*service.ProgramService`, `service.ErrNotFound`, `service.ErrPolicyViolation` (T006); `*program.ValidationError` (T002); `sandbox.NewRunner`, `sandbox.Config` (T004); `store.NewDgraphStore`, `store.NewGRPCClient`, `store.NewMemoryStore` (T005).
- Produces: `httpapi.NewRouter(svc *service.ProgramService, logger *slog.Logger) http.Handler` serving `POST /api/v1/programs`, `GET /api/v1/programs`, `GET /api/v1/programs/{id}`, `PUT /api/v1/programs/{id}`, `POST /api/v1/programs/{id}/run`, `POST /api/v1/programs/run`, `GET /healthz` with `1 MiB` body cap and `{ "error": { "code", "message", "details"? } }` error mapping, plus runnable `backend/cmd/server/main.go`.

## Review Workload Forecast

| Dimension | Forecast | Why |
| --- | --- | --- |
| Estimated changed lines | ~1,600–2,200 lines (incl. comprehensive unit/integration tests) | 6 Go packages (`program`, `codegen`, `sandbox`, `store`, `service`, `httpapi`), `cmd/server`, `verify.sh`, `docker-compose.yml`, plus test suites covering AC-1..AC-13 |
| Files / areas | ~20–24 files under `backend/`, `docker-compose.yml`, and `02-DOCS/wiki/sdd/` | Backend domain, translator, Python sandbox, Dgraph/memory store, REST API, server entrypoint, SDD artifacts |
| Risk | Max tier: `1` (0 tier-2 tasks forecast by `rsc risk`); high security sensitivity on `internal/sandbox` and `internal/store` | Each task gets a fresh-context tier-1 review (re-checking `rsc risk` on the real diff per task), and `internal/sandbox` + `internal/store` receive explicit security-boundary scrutiny per `REVIEW.md` |
| Suggested delivery | `single-pr` (stacked on PR #5 `chore/sdd-init` or targeting `main` with `Depends on: #5`) | Exceeds the 400-line review budget due to initial backend scaffolding + full test coverage; delivered cleanly as per-task atomic commits on one feature branch with full verification record and closing review for human inspection |
