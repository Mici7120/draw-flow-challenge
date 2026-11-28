---
type: review
title: Closing Review — program-model-and-execution
description: Whole-branch adversarial closing review against spec, constitution, and REVIEW.md prior to shipping PR.
tags: [sdd, review]
timestamp: 2026-11-28T05:16:00Z
topic: sdd
slug: program-model-and-execution
source_state: 335451c (clean)
---

# Closing Review — program-model-and-execution

> Summary: `0 findings: 0 blocker, 0 major, 0 nit` (after resolving 1 major finding `F-01` in commit `335451c`)
> Risk tier (`rsc risk --base main`): **Tier 1** (`default`, `docs`) · Always-human paths touched (`REVIEW.md`): `backend/internal/sandbox/*`, `backend/internal/store/dgraph.go`, `.rsc.json`
> Verdict: **`APPROVE`** (open PR for human review — do not self-merge due to `REVIEW.md` always-human paths and review budget)

## Attack list

1. **Spec fidelity (`AC-1..AC-13`):** Audited `specs/program-model-and-execution.md`, `plans/program-model-and-execution.md`, and `decisions.md` (`D-0004`/`D-0005`) against all 6 Go packages and HTTP end-to-end tests.
2. **Cross-package boundaries:** Audited `program` ↔ `codegen` ↔ `sandbox` ↔ `service` ↔ `httpapi` ↔ `store` contracts, edge/node validation, number formatting, empty JSON collections, and error mapping.
3. **Security boundary (`REVIEW.md` blockers #1–#5):** Audited `harness.py` AST allowlist, early fail-closed `rlimit` caps, `runner.go` `os.Pipe` + bounded chunk reader + synced pre-reap PGID kill, `dgraph.go` parameterized `$id` DQL + `SetJson` mutations, error sanitization (`400`/`404`/`405`/`422`/`500` + panic recovery), authorship, and zero committed secrets.
4. **Tests & verification:** Confirmed fresh uncached `backend/scripts/verify.sh` green across all 6 core packages (`83.4%`–`97.0%` coverage with `-race`).

## Findings & outcomes

| ID | Severity | Location | Finding | Outcome |
| --- | --- | --- | --- | --- |
| F-01 | major | `backend/internal/program/validate.go`, `backend/internal/sandbox/harness.py` | Identifier allowlist mismatch (`_*` and `builtins` passed `ValidateForRun` but failed sandbox AST check without `block_id`). | **fixed** in commit `335451c`: added `"builtins"` to `program.reservedIdentifiers`, aligned `harness.py` to `startswith("__")`, and added end-to-end parity tests in `TestHTTPAPI_ClosingReviewParity` (verified in `program-model-and-execution-2026-11-28-r2.md`). |

## Verdict: `APPROVE`
