---
type: verification
title: Verification — program-model-and-execution — 2026-11-28 (r2)
description: Post-closing-review fresh verification receipt at commit 335451c confirming F-01 parity fix and all 6 core packages >= 80%.
tags: [sdd, verification]
timestamp: 2026-11-28T05:15:30Z
topic: sdd
slug: program-model-and-execution
source_state: 335451c (clean)
---

# Verification — program-model-and-execution — 2026-11-28 (r2)

> Re-run after fixing Closing Review finding `F-01` (commit `335451c`); supersedes `program-model-and-execution-2026-11-28.md` (`de75786`) append-only.

## Stack gate (`go clean -testcache && ./scripts/verify.sh`)

- `gofmt -l .`: **PASS**
- `go vet ./...`: **PASS**
- `go build ./...`: **PASS**
- `go test -race -cover ./...`: **PASS** (all 6 core packages green with `-race`, including `TestHTTPAPI_ClosingReviewParity`)
  - `./internal/program`: **93.5%** `>= 80.0%`
  - `./internal/codegen`: **90.6%** `>= 80.0%`
  - `./internal/sandbox`: **87.0%** `>= 80.0%`
  - `./internal/store`: **83.4%** `>= 80.0%`
  - `./internal/service`: **86.0%** `>= 80.0%`
  - `./internal/httpapi`: **97.0%** `>= 80.0%`
- `npx @ericrisco/rsc spec gate 02-DOCS/wiki/sdd/specs/program-model-and-execution.md`: **PASS** (`8/8` typed)

## Delta verified since r1 (`de75786..335451c`)

- [x] **F-01 (`AC-8` validator ↔ sandbox identifier parity):** `builtins` added to `program.reservedIdentifiers` (fails validation with `422 validation_error` and `block_id` before sandbox execution); `sandbox/harness.py` aligned to reject dunders (`startswith("__")`) while allowing benign single-underscore identifiers (`_x`) per Plan §3. Proven end-to-end by `TestHTTPAPI_ClosingReviewParity`.

## Rollback plan

- `git revert <merge-sha>`; Dgraph schema is additive/idempotent, no feature flags.

## Capas no ejecutadas

- **HERRAMIENTA-AUSENTE** — `staticcheck`, `golangci-lint`, `govulncheck` not installed on this VM.
- **SUSTITUIDA** — `DgraphStore` verified via fake `DgraphClient` (parameterized `$id` DQL, JSON mutations, sub-hour sorting, UID guards); live Dgraph container run via `docker compose up dgraph` remains for runtime smoke testing.
- **NO-APLICA** — Frontend lint/typecheck/a11y (`frontend/` belongs to Sub-specs 2 & 3).

## Hallazgos descartados

- None.

## Verdict: PASS — ready to ship via PR for human review.
