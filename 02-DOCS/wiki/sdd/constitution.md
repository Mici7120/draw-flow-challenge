---
type: constitution
title: draw-flow-challenge — Constitution
description: The non-negotiable principles every rsc-sdd phase obeys.
tags: [sdd, constitution]
timestamp: 2026-11-27T20:44:00Z
topic: sdd
version: v1.0.0
---

# draw-flow-challenge — Constitution

> Version: v1.0.0 · Ratified: 2026-11-27 · Last amended: 2026-11-27
> The non-negotiable principles every rsc-sdd phase obeys. Concrete stack mechanics live in
> `02-DOCS/wiki/stack/*`; this file ratifies the bar and links the detail.

## Known facts (from onboarding)

- Goal: Build the draw-flow visual node editor challenge (Vue.js + DrawFlow, Go/chi/Dgraph backend, Python code generation and evaluation).
- Project kind: software · software scope: growing.
- How code changes land: branches closed with a pull request; `main` is closed to direct work (`--workflow branches`).

## 1. Stack canon

1. **Frontend stack fixed:** Vue 3 + TypeScript + Vite with DrawFlow (`jerosoler/Drawflow`), managed with `pnpm` on Node 22 (single committed `pnpm-lock.yaml`). Changing the framework or language is a MAJOR amendment.
2. **Backend stack fixed:** Go 1.24 with `chi` router (`go.mod` and `go.sum` committed) and Dgraph as the database. Changing the language, router, or database is a MAJOR amendment.
3. **Target runtime fixed:** Visual graphs compile to Python 3.12 code and evaluate server-side through the Go API.

## 2. Quality bar

4. **Formatter & linter clean:** Frontend (`eslint` + `prettier`) and backend (`gofmt` + `go vet` / `golangci-lint`) pass with zero errors and zero warnings before merge.
5. **Strict types required:** TypeScript runs with `"strict": true` (`vue-tsc --noEmit`) and Go compiles cleanly (`go build ./...`) with zero errors before merge.
6. **Tests & coverage gate merges:** Unit and API integration tests pass on every PR; line coverage is ≥ 80% on AST conversion, Python code generation, sandbox runner, and REST API handler packages.

## 3. Conventions

7. **Workspace boundaries:** Frontend lives in `frontend/` and backend lives in `backend/`. Shared API contracts are documented before both sides implement them.
8. **Consistent API & error shape:** REST endpoints (save, list, get, run) return JSON with accurate HTTP status codes and structured `{ "error": { "code": "...", "message": "..." } }` failures. Raw stack traces and host filesystem paths are never returned to clients.
9. **Commit format:** Commits use gitmoji + Conventional Commits (e.g. `✨ feat(editor): add if-else node`), enforced by `.rsc/gitmoji-guard.mjs`.

## 4. Branching & shipping

10. **Branch-and-PR workflow:** All changes happen on a branch off `main` and merge via pull request. Direct commits or pushes to `main` are blocked (`.rsc/branch-guard.mjs`).
11. **Git authorship is the human's:** Commits and PRs are authored only by the configured git user. No `Co-Authored-By` AI trailers and no "generated with" footers (enforced at `ship`).

## 5. Security & privacy floor

12. **No committed secrets:** Credentials, tokens, and `.env` files are never committed. Provider secrets load from `01-TOOLS/<provider>/.env` (gitignored). Baseline: `security-audit`.
13. **Sandboxed Python execution:** Generated Python executes only inside an isolated subprocess enforcing a strict wall-clock timeout, memory cap, disabled network and filesystem writes, and an AST allowlist that blocks unsafe imports and builtins (`os`, `subprocess`, `socket`, `open`, `eval`, `exec`, `__import__`).

## 6. UX & accessibility floor

14. **Keyboard & mouse operability:** The node editor supports both keyboard and mouse programming, visible focus indicators, and WCAG 2.2 AA contrast on core controls and code output.

## 7. Performance budgets

15. **Responsive preview & bounded runs:** Graph-to-Python code generation completes in ≤ 150 ms for graphs up to 100 nodes; Python execution endpoints enforce a hard wall-clock timeout (default ≤ 5 s) and return a clear timeout error.

## 8. Knowledge & decisions

16. **Decisions are logged:** Every significant technical or workflow decision is appended to `02-DOCS/wiki/sdd/decisions.md` with date, options considered, and rationale. The constitution is the highest-order decision record.

## Definition of Done (the merge bar `verify` runs against)

A change ships only when ALL hold:

- [ ] Formatter and linter pass with zero errors/warnings (principle 4).
- [ ] Strict TypeScript and Go builds pass cleanly (principle 5).
- [ ] Tests pass and ≥ 80% coverage holds on core AST, codegen, runner, and API packages (principle 6).
- [ ] Workspace layout, JSON error shape, and gitmoji commit format followed (principles 7–9).
- [ ] Changes live on a branch off `main`, merged via PR, authored solely by the human (principles 10–11).
- [ ] Zero secrets committed; Python sandbox limits and AST allowlist verified if runner code changed (principles 12–13).
- [ ] Keyboard/mouse operability and performance budgets checked where applicable (principles 14–15).
- [ ] Significant decisions appended to `02-DOCS/wiki/sdd/decisions.md` (principle 16).

## Amendment log (append-only)

| Date | Version | Change | Why |
|------|---------|--------|-----|
| 2026-11-27 | draft | Draft generated by `rsc onboard`. | The SDD floor needs a constitution; ratification is the `constitution` phase's. |
| 2026-11-27 | v1.0.0 | Ratified initial constitution (principles 1–16 and Definition of Done). | Project kickoff — explicitly ratified by user. |
