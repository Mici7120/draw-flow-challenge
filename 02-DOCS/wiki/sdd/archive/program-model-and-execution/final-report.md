---
type: archive
title: Final Report — program-model-and-execution
description: Shipped PR summary, verification evidence, closing review outcome, and landing decision for Sub-spec 1.
tags: [sdd, archive, ship]
timestamp: 2026-11-28T05:16:30Z
topic: sdd
slug: program-model-and-execution
---

# Final Report — Program Model, Persistence & Sandboxed Execution

## What shipped
- Normalized visual program domain model and graph validator (`backend/internal/program`) supporting `start`, `code_block`, `number`, `variable`, `assign`, `math` (`+ - * /`), `if_else` (`== != < <= > >=`), inclusive `for (from..to)`, and `output`, enforcing `<= 100` nodes, trimmed `<= 120`-char names, zero floating blocks on run, cycle detection, diamond-DAG complexity cap, unused-edge rejection, and strict identifier validation.
- Deterministic Python 3.12 code generator (`backend/internal/codegen`) with 4-space indentation, nested parenthesization, `pass` empty bodies, and `< 1 ms` 100-node translation.
- Defense-in-depth isolated Python 3.12 sandbox runner (`backend/internal/sandbox`) with strict AST allowlist, early fail-closed `RLIMIT_FSIZE`/`RLIMIT_AS`/`RLIMIT_DATA` caps (`128 MB`), isolated `python3 -I -S` subprocess, synchronized PGID timeout kill (`<= 5 s`), fixed-chunk bounded stdout capture (`64 KB` / `1,000` lines truncation with notice), dedicated fd-3 final-variable channel, and sanitized error reporting.
- Persistence layer (`backend/internal/store`) with `ProgramStore`, thread-safe deep-copying `MemoryStore`, and parameterized-DQL `DgraphStore` + local `docker-compose.yml`.
- Application service (`backend/internal/service`), `chi` REST HTTP API (`backend/internal/httpapi`), and runnable server entrypoint (`backend/cmd/server`).
- Recalibrated SDD config (`02-DOCS/wiki/sdd/config.yaml`) and single-pass `backend/scripts/verify.sh` coverage gate.

## Artifacts
- Spec: [../../specs/program-model-and-execution.md](../../specs/program-model-and-execution.md)
- Plan & tasks: [../../plans/program-model-and-execution.md](../../plans/program-model-and-execution.md)
- Analysis: [../../analysis/program-model-and-execution.md](../../analysis/program-model-and-execution.md)
- Progress & decision queue: [../../progress/program-model-and-execution.md](../../progress/program-model-and-execution.md)
- Verification (latest): [../../verifications/program-model-and-execution-2026-11-28-r2.md](../../verifications/program-model-and-execution-2026-11-28-r2.md)
- Closing review: [../../reviews/program-model-and-execution.md](../../reviews/program-model-and-execution.md)

## Landing decision
- Opened Pull Request on branch `feat/program-model-and-execution` (stacked on PR #5 `chore/sdd-init`) and left open for human review — **not self-merged** because the branch touches `REVIEW.md` always-human paths (`backend/internal/sandbox/*`, `backend/internal/store/dgraph.go`, `.rsc.json`) and exceeds the 400-line review budget (`D-0006`).
