---
type: progress
title: Progress — Program Model, Persistence & Sandboxed Execution
description: Append-only ASDD Stage 4 implementation ledger and decision queue for Sub-spec 1.
tags: [sdd, progress, asdd]
timestamp: 2026-11-28T04:47:00Z
topic: sdd
slug: program-model-and-execution
---

# Progress — Program Model, Persistence & Sandboxed Execution

> Spec: [../specs/program-model-and-execution.md](../specs/program-model-and-execution.md) · Plan: [../plans/program-model-and-execution.md](../plans/program-model-and-execution.md)
> Mode: Autopilot (user approved spec + clarify in gated Stage 2, then authorized autopilot for Stage 3–5)

## Decision queue

- [review-after] Q1 — Plan `02-DOCS/wiki/sdd/plans/program-model-and-execution.md` and architectural decision `D-0005` were approved under autopilot after `analyze` passed with 0 open findings, `02-DOCS/wiki/sdd/plans/program-model-and-execution.md`, Stage 3 (`plan`/`tasks`/`analyze`)
- [review-after] Q2 — Combined Sub-spec 1 backend PR exceeds the 400-line review budget (~1,600–2,200 lines including full unit/integration test suites for initial greenfield backend); left open for human review rather than self-merging, `02-DOCS/wiki/sdd/plans/program-model-and-execution.md`, Stage 3 (`tasks`)

## Task ledger

## T001 — 2026-11-28
- status: complete
- red: `backend/scripts/verify.sh` failed closed on empty module and failed negative coverage control (`MIN_COVERAGE=99.9` vs `92.3%`)
- green: `backend/go.mod` (`go 1.24.3`), `backend/go.sum`, `backend/scripts/verify.sh`, `docker-compose.yml` committed (`0f51789`); positive control (`MIN_COVERAGE=80.0`) passed
- files: `backend/go.mod`, `backend/go.sum`, `backend/scripts/verify.sh`, `docker-compose.yml`
- risk: tier 1 (`rsc risk --base 47bda56`)
- review: 2 reported → 2 unique → 0 blocker, 0 major, 2 nits (VERDICT: pass; nits noted for `verify.sh` cleanup in T008)
- controls: tests ✅ build ✅ security ✅ traceability ✅ (AC-1..AC-13 foundation)
- selected_skills: `go`, `docker` (compact rules: Go 1.24 fixed, read-only verify gate with hard >=80% floor, pinned Dgraph standalone image)

## T003 — 2026-11-28
- status: complete
- red: `go test ./internal/codegen/...` failed on `not implemented` assertions
- green: `go test -race -cover ./internal/codegen/...` passed (`90.6%` coverage >= `80%` floor); 100-node translation benchmark completed in `< 1 ms` (<= `150 ms` budget)
- triangulation: added tests for expression-based `for` bounds (`(b + 2) + 1`) and malformed/nil AST error paths
- files: `backend/internal/codegen/python.go`, `backend/internal/codegen/python_test.go`
- risk: tier 1 (`rsc risk --base 3b18dd4`, commit `fc46eb4`)
- review: 0 reported → 0 unique → 0 confirmed (VERDICT: pass, round 1)
- controls: tests ✅ build ✅ security ✅ traceability ✅ (AC-1, AC-2, AC-3, AC-4, AC-5, AC-12)
- selected_skills: `go`, `testing-go`

## T002 — 2026-11-28
- status: complete
- red: `go test ./internal/program/...` failed on `not implemented` assertions; initial implementation caught statement-cycle `stmtStack` bug via TDD
- green: `go test -race -cover ./internal/program/...` passed (`93.5%` coverage >= `80%` floor)
- triangulation: added edge cases for missing/multiple `start`, bad edges, ambiguous ports, diamond expression DAG DoS (`expression_too_complex`), dangling empty-graph edges, miswired unused port edges, all 4 math ops, all 6 comparison ops, populated `else`, and `code_block.next`
- files: `backend/internal/program/model.go`, `backend/internal/program/validate.go`, `backend/internal/program/validate_test.go`
- risk: tier 1 (`rsc risk --base 0f51789`, commits `3b18dd4` + fix `e3650fb`)
- review: round 1 found 2 blockers (diamond DAG complexity, silently dropped edges) + 1 nit → all fixed in `e3650fb`; round 2 returned 0 findings (VERDICT: pass)
- controls: tests ✅ build ✅ security ✅ traceability ✅ (AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-8, AC-9, AC-12, AC-13)
- selected_skills: `go`, `testing-go`

## T006 — 2026-11-28
- status: complete
- red: `go test ./internal/service/...` failed on `not implemented` assertions
- green: `go test -race -cover ./internal/service/...` passed (`86.0%` coverage >= `80%` floor)
- files: `backend/internal/service/service.go`, `backend/internal/service/service_test.go`
- risk: tier 1 (`rsc risk --base 4723340`, commit `b8cb625`)
- review: 2 minor/nit suggestions → 0 blocker, 0 major (VERDICT: pass, round 1)
- controls: tests ✅ build ✅ security ✅ traceability ✅ (AC-1..AC-11, AC-13)
- selected_skills: `go`, `testing-go`

## T005 — 2026-11-28
- status: complete
- red: `go test ./internal/store/...` failed on `not implemented` assertions
- green: `go test -race -cover ./internal/store/...` passed (`83.4%` coverage >= `80%` floor)
- triangulation: added MemoryStore deep-copy isolation, DgraphStore error/decode paths, same-hour DgraphStore.List DESC sorting, pre-Mutate UID/CreatedAt guard, and 8-goroutine concurrent `-race` test
- files: `backend/internal/store/store.go`, `backend/internal/store/memory.go`, `backend/internal/store/dgraph.go`, `backend/internal/store/store_test.go`
- risk: tier 1 (`rsc risk --base 6ec37e5`, commits `4723340` + fix `be553bf`)
- review: round 1 found 1 major (sub-hour DgraphStore.List sort) + 2 nits → all fixed in `be553bf`; round 2 returned 0 findings (VERDICT: pass)
- controls: tests ✅ build ✅ security ✅ (parameterized DQL `$id` verified) traceability ✅ (AC-6, AC-7, AC-9)
- selected_skills: `go`, `testing-go`

## T008 — 2026-11-28
- status: complete
- red: `backend/scripts/verify.sh` negative control (`MIN_COVERAGE=99.9`) failed all 6 core packages as expected
- green: `02-DOCS/wiki/sdd/config.yaml` recalibrated (`stacks: [go]`, `strict_tdd: true`, `verify: [cd backend && ./scripts/verify.sh]`, `models.enabled: false` preserved); single-pass `verify.sh` passed all 6 core packages; `rsc doctor` reports healthy
- files: `02-DOCS/wiki/sdd/config.yaml`, `backend/scripts/verify.sh`
- risk: tier 1 (commit `48c04d2`)
- controls: tests ✅ build ✅ security ✅ traceability ✅ (Constitution #6 SDD calibration)
- selected_skills: `sdd-init`, `go`

## T004 — 2026-11-28
- status: complete
- red: `go test ./internal/sandbox/...` failed on `not implemented` assertions
- green: `go test -race -cover ./internal/sandbox/...` passed against real Python 3.12 (`87.0%` coverage >= `80%` floor)
- triangulation: added byte-cap truncation (`64 B`), 1,000-line truncation, 10 AST security policy payloads, timeout kill, 128 MB memory-bomb `resource_limit`, and hardened `print`/`range` AST abuse tests
- files: `backend/internal/sandbox/harness.py`, `backend/internal/sandbox/runner.go`, `backend/internal/sandbox/runner_test.go`
- risk: tier 1 (`rsc risk --base e3650fb`, commits `6ec37e5` + hardening `8cf7515`)
- review: round 1 found 3 should-fix (`StdoutPipe`/`Wait` race, unbounded `ReadBytes`, late/fail-open `setrlimit`) + 2 nits (`print`/`range` AST context+arity, post-reap PGID kill) → all 5 fixed in `8cf7515`; round 2 returned 0 findings (VERDICT: pass)
- controls: tests ✅ build ✅ security ✅ (real Python 3.12 AST allowlist + isolated subprocess + rlimits verified) traceability ✅ (AC-1..AC-5, AC-10, AC-11, AC-13)
- selected_skills: `go`, `testing-go`, `security-audit`

## T007 — 2026-11-28
- status: complete
- red: `go test ./internal/httpapi/...` failed with 404 on unimplemented routes
- green: `go test -race -cover ./internal/httpapi/...` passed (`97.0%` coverage >= `80%` floor); `go build ./...` compiles `cmd/server` cleanly
- triangulation: added tests for unknown route `404`, unsupported `DELETE` `405`, unknown JSON field / trailing token / `>1 MiB` body `400`, `AC-5` descending `for` + `pass`, `AC-6`/`AC-8` decoded payloads + `block_id: m1` details, `AC-11` timeout + `422 sandbox_policy_violation`, `AC-13` output truncation, sanitized `500` internal error, and panic recovery
- files: `backend/internal/httpapi/router.go`, `backend/internal/httpapi/router_test.go`, `backend/cmd/server/main.go`, `backend/go.mod`, `backend/go.sum`
- risk: tier 1 (`rsc risk --base be553bf`, commit `915266e` + review fix commit)
- review: round 1 found 0 blocker, 3 major test-coverage gaps + 2 nits (`http.ErrAbortHandler`, `WriteTimeout` derived from `RUN_TIMEOUT`) → all 5 fixed
- controls: tests ✅ build ✅ security ✅ (zero leaked paths/panics/Dgraph internals) traceability ✅ (AC-1..AC-13)
- selected_skills: `go`, `testing-go`, `api-design`
