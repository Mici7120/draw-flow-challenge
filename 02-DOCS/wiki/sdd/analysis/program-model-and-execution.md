---
type: analysis
title: Analysis — Program Model, Persistence & Sandboxed Execution
description: Pre-implementation consistency gate report cross-reading constitution, spec, plan, and tasks for Sub-spec 1.
tags: [sdd, analyze, consistency]
timestamp: 2026-11-28T04:46:00Z
topic: sdd
slug: program-model-and-execution
---

# Analysis — Program Model, Persistence & Sandboxed Execution

> Verdict: **`GATE: PASS`** (0 CRITICAL, 0 HIGH, 0 MEDIUM, 0 LOW remaining after reconciling 8 initial findings)
> Artifacts checked: [constitution.md](../constitution.md) · [program-model-and-execution.md](../specs/program-model-and-execution.md) · [program-model-and-execution.md](../plans/program-model-and-execution.md) · [decisions.md](../decisions.md)

## Requirement coverage map

| REQ-ID | Spec requirement (short) | Plan section | Task(s) | Status |
| --- | --- | --- | --- | --- |
| AC-1 | Math + assignment + output (`x = 2 + 3\nprint(x)`) | §3, §4, §5 | T002, T003, T004, T006, T007 | covered |
| AC-2 | Assignment without output (`total = 4 * 5`) | §3, §4, §5 | T002, T003, T004, T006, T007 | covered |
| AC-3 | `If-Else` numeric comparisons (`==, !=, <, <=, >, >=`) | §3, §4, §5 | T002, T003, T004, T006, T007 | covered |
| AC-4 | Inclusive `For` (`1..4`) & nested `if-else` | §3, §4, §5 | T002, T003, T004, T006, T007 | covered |
| AC-5 | `For` boundary (`5..2`), empty bodies (`pass`), empty program | §3, §4, §5 | T002, T003, T004, T006, T007 | covered |
| AC-6 | Save draft, update, list DESC, load exact structure, duplicate names | §3, §4, §5 | T002, T005, T006, T007 | covered |
| AC-7 | Run saved program without mutating stored record | §3, §5 | T005, T006, T007 | covered |
| AC-8 | Validation error on run (missing port, floating block, cycle, non-int literal `for`, bad ident) | §3, §5 | T002, T006, T007 | covered |
| AC-9 | Unknown ID (`404`) & invalid save (`422`/`400`), zero leaked internals | §0, §3, §5 | T002, T005, T006, T007 | covered |
| AC-10 | Runtime error (div/0, unassigned var, non-int `for` expr) with partial stdout | §3, §5 | T004, T006, T007 | covered |
| AC-11 | Sandbox AST allowlist (`policy_violation`), timeout kill, `128 MB` `resource_limit` | §0, §3, §5, §7 | T004, T006, T007 | covered |
| AC-12 | 100-node translation `<= 150 ms` | §0, §3, §5 | T003, T007 | covered |
| AC-13 | `> 100` nodes rejected; `64 KB` / `1,000`-line stdout truncation with notice | §0, §3, §5 | T002, T004, T006, T007 | covered |

## Six-analysis check & resolved findings

Initial independent adversarial review surfaced **8 findings** (0 CRITICAL, 2 HIGH, 2 MEDIUM, 4 LOW), all resolved in their owning artifacts before opening Stage 4 (`implement`):

| # | Severity | Type | Artifacts | Finding | Resolution |
| --- | --- | --- | --- | --- | --- |
| 1 | HIGH | Contradiction / Carrier | Plan §3 vs `T002`, `T004`, `T006` | `ValidateDraft`, `ValidateForRun`, and `sandbox.Run` signatures diverged between §3 and Per-task Interfaces; `sandbox.CodeRunner` not produced by `T004`. | Aligned §3 and Per-task Interfaces verbatim; declared `service.CodeRunner` as a consumer interface in `T006` satisfied by `*sandbox.Runner`. |
| 2 | HIGH | Contradiction | Spec §What result vs Plan §2–§4, §7, `D-0005` | Spec grouped pre-execution sandbox policy failure with runtime/timeout partial-output responses; Plan mapped `policy_violation` to `422` error envelope and omitted `resource_limit` in §2/`D-0005`. | Separated pre-execution validation/policy rejection (`422` `{ error }`) from post-start execution failures (`200` `RunResult` with `runtime_error` / `timeout` / `resource_limit`) across Spec, Plan §2–§4/§7, `T007`, and `D-0005`. |
| 3 | MEDIUM | Carrier completeness | Plan `Per-task Interfaces` | Missing `T001` Go module path carrier, `T006` constructor signature, and `T007` wiring dependencies (`store`, `sandbox`, `*program.ValidationError`). | Added `**T001 — Interfaces**` pinning `github.com/Mici7120/draw-flow-challenge/backend` and completed `T006`/`T007` `Consumes`/`Produces`. |
| 4 | MEDIUM | Ambiguity | Plan §4 & `T002` | `Node.Data` listed both `name` and `var_name`, and `code_block` port wiring was unspecified. | Standardized on `name` for all variable identifiers and documented exact `NodeData` fields and `code_block` / `if_else` / `for` port wiring in §4 and `T002`. |
| 5 | LOW | Ambiguity | Spec AC-1 vs Plan §4 | Trailing `\n` handling on Python `print()` stdout unspecified. | Pinned single trailing `\n` trimming in Plan §0 and §3 (`print(5)` → `"5"`). |
| 6 | LOW | Coverage | Plan §5 & `T004` | `128 MB` `resource_limit` test absent from §5 (`AC-11`) and `T004` Done-check. | Added explicit `128 MB` memory-bomb `resource_limit` assertion to §5 and `T004`. |
| 7 | LOW | Ambiguity | Plan §0, §3, `T007` | HTTP request body cap had no concrete byte limit. | Pinned `1 MiB` (`1 << 20` bytes) cap in §0, §3, and `T007`. |
| 8 | LOW | Duplication / Trace | Plan §0 vs `T005`, `T008` | §0 omitted `internal/store` from coverage list; `T008` traced to `AC-1..AC-13`. | Added `internal/store` to §0 coverage list and marked `T008` trace as `Constitution #6 (SDD calibration)`. |

## Routing

- **`GATE: PASS`** — Proceed to ASDD Stage 4 (`implement`) starting at `T001`.
