---
type: spec
title: Spec — Program Model, Persistence & Sandboxed Execution
description: WHAT and WHY for Sub-spec 1 of draw-flow-challenge — visual program contract, Python translation, program persistence, and sandboxed execution.
tags: [sdd, spec, ast, python, sandbox]
timestamp: 2026-11-28T04:19:55Z
topic: sdd
slug: program-model-and-execution
branch: feat/program-model-and-execution
status: planned
---

# Spec — Program Model, Persistence & Sandboxed Execution

> Slug: `program-model-and-execution` · Status: planned · Created: 2026-11-28
> Inherits: [constitution](../constitution.md) · Sub-spec 1 of 3 (followed by Visual Node Editor and Live Code View & Run UI)

## Problem & why
Beginners learning programming struggle with text syntax before they grasp core logic like variables, math, branches, and loops. Static flowcharts remove syntax friction, but they cannot run or show whether the logic actually works. Without a shared, validated contract defining how visual blocks translate into runnable Python and persist across sessions, a visual editor and execution service cannot reliably turn learner programs into predictable results or beginner-friendly feedback.

## Cost of not building it
Learners have no way to verify whether their visual logic works, save progress, or compare visual blocks against real Python code. The challenge's core requirement—turning visual AST programs into runnable, saved Python programs—remains unmet, and Sub-specs 2 and 3 (the visual editor and live code view) have no stable behavioral contract to build against.

## The cheapest alternative
Let learners draw flowcharts in a generic diagram tool and manually rewrite them in a standalone Python interpreter or notebook. That avoids building persistence and sandboxed execution, but breaks the learning loop: learners still hit raw text syntax errors, diagrams drift out of sync with executed code, and there is no unified save, load, and run workflow.

## Goals
- Express simple numeric programs using connected visual blocks covering code blocks, assignment, basic math (`+`, `-`, `*`, `/`), `if-else` comparisons (`==`, `!=`, `<`, `<=`, `>`, `>=`), inclusive `for` (`from..to`) loops, and Output/Print.
- Deterministically translate any valid visual program into readable, equivalent Python 3.12 code.
- Save work-in-progress or complete programs, update existing saved programs, list saved programs, and load a saved program back intact.
- Safely execute either a saved program or an unsaved draft and return its generated Python, printed output, and final variable values—or clear, block-referenced feedback when validation, runtime, timeout, or sandbox checks fail.

## Non-goals / out of scope
- **Visual canvas UI and keyboard/mouse editing** — covered by Sub-spec 2.
- **Live code viewer and run-output UI panel** — covered by Sub-spec 3.
- **Optional challenge extras** — `string` data type, HTTP request blocks, and target languages other than Python.
- **Additional language features** — integer division (`//`), modulo (`%`), compound boolean operators (`and`, `or`, `not`), `while` loops, user-defined functions, recursion, collections (lists/dicts), or interactive standard input (`input()`).
- **Program deletion, user accounts, login, or permissions** — single shared workspace without delete for v1.
- **Unrequested extras** — accepting raw arbitrary Python text bypassing the visual program contract, program version history/diffs, shareable public links, real-time multi-user collaboration, or a step-by-step interactive debugger.

## Users & context
- **Programming learner:** Someone practicing variables, arithmetic, branching, and loops who assembles visual programs, saves incomplete drafts mid-work, reloads them later, and runs them to verify behavior against real Python.
- **Challenge evaluator:** A reviewer testing program persistence (save, update, list, load), Python translation accuracy, beginner-friendly validation errors, and sandboxed execution safety (including invalid inputs and runaway loops).

## What the user can do
Any user in the single shared workspace (no login required) can:
- **Save a new program** with a non-empty name and a visual program definition (complete or incomplete work-in-progress draft).
- **Update an existing saved program** by its identifier, replacing its name and/or visual program definition.
- **List saved programs** to browse their identifiers, names, and last-updated timestamps.
- **Load a saved program** by its identifier and retrieve its saved visual structure intact.
- **Run a saved program** by its identifier (when the saved program passes validation).
- **Run an unsaved program draft** directly without saving or modifying any stored program.

No user is allowed to:
- Submit raw Python text directly to bypass the visual program contract.
- Access host files, network, environment variables, or unsafe Python capabilities during execution.
- Delete saved programs (deferred out of v1).

## What result they get
- **Save / Update:** Receives the program's unique identifier, name, timestamps, and stored visual structure (incomplete drafts are preserved verbatim).
- **List:** Receives all saved program summaries (identifier, name, timestamps) ordered most-recently-updated first, or an empty list when none exist.
- **Load:** Receives the exact visual program last saved for that identifier, or a clear "not found" error for an unknown identifier.
- **Run (human-approved reference example):** For a visual program that adds `2` and `3`, assigns the result to `x`, and prints `x`, the user gets:
  - **Generated Python:**
    ```python
    x = 2 + 3
    print(x)
    ```
  - **Printed output:** `5` (with `output_truncated: false`; if output exceeds 64 KB or 1,000 lines, it is truncated cleanly with `output_truncated: true` and a truncation notice)
  - **Final variables:** `x = 5`
  - **Status:** succeeded
- **Run (invalid, incomplete, or policy-blocked program):** Receives a structured validation or sandbox-policy failure (naming offending block identifier(s) for graph errors) and a plain-language reason; execution does not start.
- **Run (runtime, timeout, or resource-limit failure):** Receives the generated Python, any printed output emitted before failure, and a clean structured error (never host paths, raw server stack traces, or internal persistence errors).

## What it must respect
Inherited from [constitution.md](../constitution.md) (v1.0.0), [decisions.md](../decisions.md), and [REVIEW.md](../../../../REVIEW.md):
- **Python 3.12 server-side evaluation** ([constitution.md](../constitution.md) principle 3) under the ratified project stack ([D-0001](../decisions.md)); technical architecture and transport design belong to `plan`.
- **Structured, safe error contract** ([constitution.md](../constitution.md) principle 8): every failure uses a consistent structured error shape (`code` and `message`) with accurate status semantics and zero leaked host paths, raw tracebacks, or internal persistence details.
- **Defense-in-depth sandbox** ([constitution.md](../constitution.md) principle 13, [D-0002](../decisions.md)): isolated execution with strict wall-clock timeout, memory cap, disabled network and filesystem writes, and static checks rejecting unsafe imports and builtins (`os`, `subprocess`, `socket`, `open`, `eval`, `exec`, `__import__`).
- **Performance budgets** ([constitution.md](../constitution.md) principle 15): graph-to-Python translation completes in ≤ 150 ms for graphs up to 100 nodes; execution enforces a hard wall-clock timeout (default ≤ 5 s).
- **Injection & identifier safety** ([REVIEW.md](../../../../REVIEW.md)): user-supplied program names, variable identifiers, and numeric literals are strictly validated so untrusted input cannot inject code or persistence queries.
- **Quality & coverage floor** ([constitution.md](../constitution.md) principles 4–6): ≥ 80% line coverage across translation, sandbox execution, and program-service logic.

## How it proves it
- Automated unit and integration tests exercising every acceptance criterion (AC-1 through AC-13): all 5 mandatory block families + Output/Print, nested `if-else` and inclusive `for`, permissive draft save/update/list/load round-trips, saved and draft runs, invalid-graph rejections, size/output caps, runtime errors, and sandbox/timeout probes.
- A timing test verifying translation completes in ≤ 150 ms on a 100-node program and an oversized loop terminates within the ≤ 5 s timeout.
- Coverage report proving ≥ 80% line coverage across translation, sandbox execution, and program-service packages.
- Persistent evidence: test execution logs, timing numbers, coverage summary, and reference-program run output recorded in `02-DOCS/wiki/sdd/verifications/program-model-and-execution-<date>.md`.

## Behaviour
- **Main paths:**
  - A visual program follows connected statements in sequence from its entry point, grouping nested bodies inside code blocks.
  - **Assignment** binds a valid variable name to a numeric literal, another variable, or a math expression.
  - **Basic math** evaluates `+`, `-`, `*`, and standard Python float division `/` (e.g. `5 / 2 == 2.5`) on two numeric inputs (literals, variables, or nested math blocks), preserving precedence cleanly when nested.
  - **`If-Else`** evaluates one numeric comparison (`==`, `!=`, `<`, `<=`, `>`, `>=`) and executes the `if` block when true or the optional `else` block when false.
  - **Inclusive `For` (`from..to`)** iterates a loop variable over integers from `from` through `to` inclusive (`1` to `5` executes with `1, 2, 3, 4, 5`), where `from` and `to` each accept an integer literal, a variable, or a math expression, and executes its body block on each iteration.
  - **Output / Print** evaluates a numeric input and appends its value as a line of printed output.
  - Translation deterministically produces readable, properly indented Python 3.12 matching the exact visual block order and nesting.
  - Saving a new program remembers its name and visual structure across sessions and assigns a unique identifier.
  - Updating an existing saved program replaces its name and/or visual structure under the same identifier and refreshes its last-updated timestamp.
  - Listing reveals summaries (identifier, name, timestamps) of all saved programs ordered most-recently-updated first.
  - Loading by identifier returns the exact visual structure last saved.
  - Running a valid saved program or unsaved draft translates it to Python, evaluates it inside the sandbox, and returns the generated Python, all printed output lines in order, and the final values of assigned variables.
- **Edge cases:**
  - **Incomplete draft on save vs run:** Saving a program with missing required inputs or unconnected floating blocks succeeds and loads back intact; running that draft fails validation before execution starts.
  - **Empty branches or empty program:** An empty `if`, `else`, or `for` body translates to a valid Python no-op (`pass`) without syntax errors; running a program with zero executable statements succeeds with empty printed output and no variables.
  - **`For` boundary cases:** `from == to` runs once with that value; `from > to` runs zero times without error.
  - **Nesting:** Nested `if-else` inside `for`, `for` inside `if-else`, and nested math expressions preserve exact indentation and evaluation order.
  - **Variable updates:** Reassigning a variable updates its value for all later statements and reports its final value when execution ends.
  - **No Output/Print block:** A program with only `Add → Assign` succeeds with empty printed output and still reports the assigned variable's final value.
  - **Duplicate names:** Saving two programs with the same name keeps both under distinct identifiers.
  - **Empty catalog:** Listing when no programs are saved returns an empty list, not an error.
  - **Printed output cap:** When a run emits more than 64 KB or 1,000 lines of printed output (whichever comes first), execution truncates the captured output cleanly and marks `output_truncated: true` with a clear truncation notice rather than discarding earlier lines.
- **Error paths:**
  - **Invalid save input:** A blank name, a name over 120 characters, a program with more than 100 nodes, or a malformed program structure is rejected with a clear validation error.
  - **Unknown identifier:** Loading, updating, or running a non-existent program identifier returns a clear "not found" error without affecting other saved programs.
  - **Invalid visual graph on run:** Programs exceeding 100 nodes, disconnected required inputs, unconnected floating blocks detached from the entry path, cyclic math connections, invalid or reserved variable names (including Python keywords or non-identifier characters), or non-integer literal `for` bounds fail validation before execution and identify the offending block(s).
  - **Runtime failure:** Division by zero, reading an unassigned variable, or a `for` bound expression evaluating to a non-integer stops execution cleanly and returns the generated Python, any output printed before failure, and a plain-language runtime error.
  - **Timeout or resource limit:** A runaway loop or excessive memory use stops within the ≤ 5 s / memory budget and returns a clear timeout or resource-limit error.
  - **Sandbox policy rejection:** Any attempt to inject unsafe names, imports, or disallowed builtins is blocked before execution and reported as a policy violation.
  - **Never revealed:** Host file paths, raw server-side Python tracebacks, internal persistence errors, or environment details never appear in any response.

## Acceptance criteria
- **AC-1 (Math + assignment + output):** Given a valid visual program that adds `2` and `3`, assigns the result to `x`, and prints `x`, When run as a draft, Then it returns generated Python `x = 2 + 3\nprint(x)`, printed output `5`, and final variable `x = 5`.
- **AC-2 (Assignment without output):** Given a valid visual program that multiplies `4` and `5` and assigns the result to `total` with no Output/Print block, When run, Then execution succeeds with empty printed output and final variable `total = 20`.
- **AC-3 (`If-Else` branching):** Given a valid visual program with an `if-else` comparing `x >= 10` (true branch prints `1`, false branch prints `0`), When run with `x = 12` and with `x = 5`, Then printed output is `1` and `0` respectively.
- **AC-4 (Inclusive `For` & nesting):** Given a valid visual program with a `for` loop for `i` from `1` to `4` that accumulates `total = total + i` (with a nested `if` check inside the loop), When run, Then `i` visits `1, 2, 3, 4` inclusively and final `total` equals `10`.
- **AC-5 (`For` boundary & empty branch):** Given a valid visual program with `for` from `5` to `2` or an empty `else` block, When translated and run, Then it generates valid Python (using `pass` for empty bodies) and completes without errors.
- **AC-6 (Save, update, list, load):** Given a user saves a new program (including an incomplete draft), When they list programs, load it by identifier, and later update its name and blocks, Then the list orders it most-recently-updated first, load returns the exact saved structure, and update preserves the same identifier with the updated content.
- **AC-7 (Run saved program):** Given a valid saved program, When run by its identifier, Then it returns the generated Python, printed output, and final variable values without modifying the saved program.
- **AC-8 (Validation error on run):** Given a saved or draft program with a disconnected required input, an unconnected floating block detached from the entry path, a cyclic math connection, a non-integer literal `for` bound, or an invalid/reserved variable name, When run, Then execution does not start and a structured validation error identifies the offending block(s).
- **AC-9 (Unknown ID & invalid save):** Given a non-existent program identifier or a save with a blank name, When loaded/updated/run or saved, Then a structured error (`not found` or `validation`) is returned with zero leaked paths or stack traces.
- **AC-10 (Runtime error):** Given a valid visual program that divides by `0`, reads an unassigned variable, or evaluates a non-integer `for` bound expression, When run, Then execution stops cleanly and returns the generated Python, any output printed before failure, and a user-facing runtime error free of host paths or raw tracebacks.
- **AC-11 (Sandbox & timeout):** Given a visual program whose loop exceeds the ≤ 5 s timeout or a crafted payload attempting unsafe imports/builtins (`os`, `subprocess`, `socket`, `open`, `eval`, `exec`, `__import__`), When run, Then execution is blocked or aborted within budget with a structured policy or timeout error and zero host or network side effects.
- **AC-12 (Translation performance):** Given a valid 100-node visual program, When translated to Python, Then translation finishes in ≤ 150 ms.
- **AC-13 (Size & output caps):** Given a program with 101 nodes, When saved or run, Then it is rejected with a structured validation error before execution; and Given a valid program whose loop prints more than 1,000 lines (or > 64 KB), When run, Then output is truncated cleanly at the cap with `output_truncated: true` and a truncation notice.

## Points to clarify
- **suposición tomada** — Program names allow duplicates because identity is by unique program identifier. *Base:* Learners often save multiple drafts with names like "Test" or "Exercise 1". *Riesgo:* If names must be unique, save validation and conflict error criteria change.
- **suposición tomada** — Running a program with zero executable statements succeeds with empty printed output and empty variables. *Base:* Matches running an empty Python script cleanly. *Riesgo:* If empty runs should fail validation instead, error paths and AC-5 change.
- **suposición tomada** — Numeric literals and math support floats; `for` bounds (`from`/`to`) accept integer literals, variables, or math expressions that must evaluate to integers (non-integer literals fail validation; non-integer runtime results fail with a runtime error). *Base:* Lets learners loop up to a variable while honoring Python `range` integer bounds. *Riesgo:* If v1 restricts `for` bounds to integer literals only, the `for` block input contract simplifies and runtime bound errors disappear.
- **suposición tomada** — Concurrent updates to the same saved program use last-write-wins semantics. *Base:* Single shared workspace without accounts or collaborative editing in v1. *Riesgo:* If multi-tab conflict detection is needed, update semantics and conflict error criteria change.
- **suposición tomada** — Program names are trimmed, non-empty, and capped at 120 characters. *Base:* Prevents unbounded metadata strings while covering any readable title. *Riesgo:* If longer titles are required, save validation thresholds change.
- **suposición tomada** — Variable scoping matches Python 3.12 flat module scope (variables assigned inside `if-else` or `for`, including the `for` loop variable after at least one iteration, appear in final variables). *Base:* Keeps final variable inspection 1-to-1 with standard Python execution. *Riesgo:* If loop variables should be hidden from final variables, output filtering changes.
- **suposición tomada** — Default sandbox memory cap is 128 MB per run. *Base:* Leaves headroom for Python 3.12 startup plus simple numeric loops while stopping memory bombs per constitution principle 13. *Riesgo:* If the host runtime requires a tighter or looser cap, sandbox limits adjust.
- **decisión diferida** — Program deletion, integer division (`//`), modulo (`%`), `string` values, HTTP request blocks, compound boolean operators (`and`/`or`/`not`), and target languages beyond Python; intentionally deferred out of v1.

## Clarifications
- **2026-11-28 — Q1 (Floating/unconnected blocks on run):** Reject unconnected blocks detached from the entry path at run time with a block-referenced validation error (still allowed when saving drafts). *Why:* Prevents beginners from silently losing unconnected blocks and removes ambiguity from multiple disconnected chains.
- **2026-11-28 — Q2 (Division & extra math operators):** Keep strictly `+`, `-`, `*`, and standard Python float division (`/`, e.g. `5 / 2 == 2.5`) in v1; defer `//` and `%` to a later cycle under Non-goals. *Why:* Matches basic math scope cleanly without bloating the initial operator set.
- **2026-11-28 — Q3 (Node count & printed output caps):** Cap programs at 100 nodes (reject > 100 on save or run) and cap printed output at 64 KB or 1,000 lines (whichever comes first), truncating cleanly with `output_truncated: true` and a notice. *Why:* Aligns with constitution principle 15's 100-node budget and lets learners inspect early loop output instead of losing it on overflow.
