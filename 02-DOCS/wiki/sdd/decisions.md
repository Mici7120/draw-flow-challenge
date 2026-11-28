# SDD Decisions Log (append-only)

> One entry per significant architectural or workflow decision. Never edit or delete past entries; supersede by appending.

---

## D-0001 — Frontend stack and quality bar
- **date:** 2026-11-27
- **context:** `README.md` specifies Vue.js + DrawFlow for the visual editor and Go + chi + Dgraph for the backend, without pinning Vue version, TypeScript, package manager, or test thresholds.
- **options considered:**
  1. Vue 3 + TypeScript + Vite + `pnpm`, strict lint/types, ≥ 80% coverage on core AST/API logic.
  2. Vue 3 + TypeScript + Vite + `npm`, strict lint/types, no fixed coverage floor.
  3. Vue 3 + JavaScript + Vite + `npm`, basic lint and smoke tests.
- **decision:** Option 1 — Vue 3 + TypeScript + Vite + `pnpm`, strict lint/types, and ≥ 80% line coverage on AST conversion, Python codegen, sandbox runner, and REST API handler packages.
- **why:** Strong typing catches AST graph schema mismatches early; `pnpm` is fast and preinstalled; an 80% floor protects the core compiler and API surface without forcing brittle UI snapshot tests.
- **supersedes:** none

---

## D-0002 — Python execution sandbox baseline
- **date:** 2026-11-27
- **context:** The Go backend evaluates user-authored Python code generated from visual nodes, which is an untrusted code execution surface.
- **options considered:**
  1. Isolated Python subprocess with strict timeout, memory cap, disabled network/filesystem writes, and an AST allowlist.
  2. Dedicated Docker container per execution with resource limits.
  3. Unsandboxed local Python subprocess with timeout only.
- **decision:** Option 1 — Isolated subprocess + resource caps + AST allowlist.
- **why:** Provides defense-in-depth (static AST allowlist + runtime isolation and limits) while staying fast and easy to run in both local dev and CI without requiring Docker-in-Docker.
- **supersedes:** none

---

## D-0003 — Cloud environment auto-sync scope
- **date:** 2026-11-27
- **context:** `.rsc/` and managed skill symlinks are gitignored, so fresh clones and cloud VMs need `rsc sync` at the exact `catalogVersion` pinned in `.rsc.json`.
- **options considered:**
  1. Repo-scoped `.agents/environment.json` shared via PR.
  2. Project-scoped `/home/antigravity/env-setup.sh` for this cloud project only.
  3. Manual sync per session (no script).
- **decision:** Option 1 — Repo-scoped `.agents/environment.json` running `npx --yes @ericrisco/rsc@<catalogVersion> sync`.
- **why:** Versioned in git so every collaborator and fresh cloud VM gets the exact pinned harness version automatically without drifting to `@latest`.
- **supersedes:** none

---

## D-0004 — Three-sub-spec decomposition and Sub-spec 1 contract shape
- **date:** 2026-11-28
- **context:** `README.md` describes three connected subsystems (visual node editor, live code/run view, and backend persistence + Python evaluation) plus optional stretch goals (`string`, HTTP requests, extra languages).
- **options considered:**
  1. Decompose into 3 sequential sub-specs starting with Sub-spec 1 (`program-model-and-execution`), using a validated visual program contract with permissive draft saving and strict validation on run; defer optional extras and delete out of v1.
  2. Single unified MVP spec covering all three parts in one pass.
  3. Opaque visual program save with direct raw-Python execution.
- **decision:** Option 1 — 3 sequential sub-specs, starting with `02-DOCS/wiki/sdd/specs/program-model-and-execution.md` (Approach 1: validated visual program contract, permissive draft save, strict validated run, explicit Output/Print + final variables, inclusive `for`, 6 numeric comparisons, create+update save, single shared workspace).
- **why:** Prevents oversized specs with hidden contradictions, keeps one source of truth between visual blocks and executed Python, avoids exposing learners to raw syntax tracebacks, and narrows the sandbox attack surface.
- **supersedes:** none

---

## D-0005 — Sub-spec 1 backend architecture, REST run error semantics, and sandbox variable capture
- **date:** 2026-11-28
- **context:** Planning `02-DOCS/wiki/sdd/plans/program-model-and-execution.md` required settling how the Go backend separates graph validation, Python codegen, sandbox execution, and Dgraph persistence, how runtime execution errors return partial stdout + generated Python, and how final variable values are extracted safely from the Python subprocess.
- **options considered:**
  1. Modular Go packages (`internal/program`, `internal/codegen`, `internal/sandbox`, `internal/store`, `internal/service`, `internal/httpapi`, `cmd/server`); normalized `Graph` AST decoupled from DrawFlow UI format; `200 OK` structured `RunResult` for post-start execution outcomes (`succeeded`, `runtime_error`, `timeout`, `resource_limit` — carrying `python_code`, partial `output`, `output_truncated`, and `error: { code, message }`) vs `400`/`404`/`422` `{ error }` envelope for malformed, missing, validation-failed, or pre-execution `sandbox_policy_violation` requests; dedicated fd-3 pipe for final variable JSON capture alongside static Python AST allowlist + isolated `python3 -I -S` subprocess.
  2. Returning `400`/`500` on runtime errors without `python_code` or partial stdout, and parsing final variables from a sentinel printed to stdout.
- **decision:** Option 1 — clean layered packages, normalized `Graph` contract, `200 OK` `RunResult` for post-start execution outcomes (`runtime_error`, `timeout`, `resource_limit`) and `422` `{ error }` for pre-execution validation or `sandbox_policy_violation`, plus a dedicated fd-3 pipe for final variables.
- **why:** Satisfies `spec §What result they get` (always returning generated Python and partial output on runtime/timeout failures), prevents user `print()` output from spoofing variable capture, decouples backend AST validation from frontend DrawFlow layout details, and allows parallel implementation and ≥ 80% unit coverage per package.
- **supersedes:** none

---

## D-0006 — Land Sub-spec 1 via PR awaiting human review (no self-merge)
- **date:** 2026-11-28
- **context:** `feat/program-model-and-execution` passed closing verification (`r2`, commit `335451c`) and closing review (`APPROVE`), with `rsc risk --base main` classifying the branch as Tier 1. However, the branch touches `REVIEW.md` always-human paths (`backend/internal/sandbox/*`, `backend/internal/store/dgraph.go`, `.rsc.json`), exceeds the 400-line review budget, and stacks on PR #5 (`chore/sdd-init`).
- **options considered:**
  1. Open PR against `main` (noting `Depends on: #5`) and leave it open for human review without self-merging.
  2. Self-merge the PR automatically under autopilot because `rsc risk` is Tier 1.
- **decision:** Option 1 — open PR and leave open for human review.
- **why:** Respects `REVIEW.md`'s always-human paths for sandbox/Dgraph/harness changes, respects `config.yaml`'s `ask-on-risk` delivery strategy above the 400-line budget, and gives the human owner final sign-off on the greenfield backend architecture.
- **supersedes:** none
