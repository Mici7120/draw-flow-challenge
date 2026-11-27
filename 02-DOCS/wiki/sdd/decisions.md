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
