# Code Review Guide

> Ratified alongside `02-DOCS/wiki/sdd/constitution.md`. Every automated and human review reads this file first.

## Summary format

Every review **must** start with a single count line before any detail:
`<total> findings: <N> blocker, <N> major, <N> nit`

## Nit cap

Report at most **3 nits** per review. Drop lower-priority style preferences once the cap is reached.

## What failure looks like here

Flag these defect classes as **blockers**:

1. **Sandbox escapes or unbounded Python execution** — running generated Python without timeout, memory limit, filesystem/network isolation, or AST allowlist checks (constitution principle 13).
2. **Broken AST-to-Python semantics** — visual blocks (assignment, math, if-else, from-to loop) generating invalid indentation, wrong evaluation order, or silently dropping edges/nodes.
3. **Unvalidated Dgraph queries or mutations** — interpolating untrusted user input directly into DQL strings instead of using parameterized queries/variables.
4. **Leaked internals in API errors** — returning raw Go panics, Python tracebacks containing host paths, or Dgraph internals to HTTP callers (principle 8).
5. **Committed secrets or broken harness guards** — `.env` files, API keys, AI `Co-Authored-By` trailers, or direct changes to `main` (principles 10–12).

## Paths to skip

- Lockfiles (`pnpm-lock.yaml`, `go.sum`) unless reviewing dependency additions/security advisories.
- Generated build artifacts (`frontend/dist/`, coverage reports).
- Machine-local harness state (`.rsc/`, `.antigravity/rsc/`, `.claude/skills/`).

## Always-human paths

Never self-merge changes touching these paths without explicit human review:

- Python execution sandbox and AST validator (`backend/**/runner/**`, `backend/**/sandbox/**`)
- Database schema and migrations (`backend/**/dgraph/**`, `backend/**/schema/**`)
- Cloud environment & harness configuration (`.agents/environment.json`, `.rsc.json`, `.claude/settings.json`)
- Constitution and review rules (`02-DOCS/wiki/sdd/constitution.md`, `REVIEW.md`)

> Note: Listing paths here informs reviewers. To enforce elevated risk automatically via `rsc risk`, mirror these globs into `.rsc/sello-config.json` under `risk.raise` when `rsc sello` is enabled.
