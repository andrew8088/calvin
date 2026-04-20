---
name: Bug Fix
description: Investigate, reproduce, verify, and minimally fix bugs in Calvin. Use for targeted bug reports involving Google Calendar event logic, hooks, OAuth, SQLite state, CLI behavior, or reliability issues.
---

## Purpose
Use this skill when fixing a **specific bug** in **Calvin**, a Go CLI for reading Google Calendar events and running hooks/scripts.

This skill is for:
- diagnosing reported bugs
- reproducing failures
- tracing root causes
- making minimal targeted fixes
- adding focused regression tests
- validating the repair

This skill is **not** for broad refactors unless the bug cannot be fixed safely otherwise.

## Product assumptions
Assume the following unless the repo clearly shows otherwise:
- Calvin is a **read-only** Google Calendar CLI
- it may poll or read Google Calendar events and evaluate event state
- it may trigger local scripts/hooks on event transitions
- it may persist local state to prevent duplicate triggers
- duplicate hook execution is a serious bug
- missed hook execution is a serious bug
- timezone and DST bugs are serious
- shell execution safety matters
- secrets and tokens must never leak

## Working style
Be conservative, evidence-driven, and minimal.

Prefer:
- reproducing before editing
- tracing the exact failing path
- the smallest correct fix
- preserving current behavior outside the bug
- adding regression coverage

Avoid:
- cleanup-only edits
- drive-by refactors
- changing unrelated interfaces
- speculative fixes without evidence
- silent behavior changes without tests

## Bug-fix workflow

### 1) Understand the bug report
Identify:
- the user-visible symptom
- expected behavior
- actual behavior
- the narrowest testable failure statement
- the likely subsystem:
  - event timing/state logic
  - recurring event handling
  - all-day event handling
  - timezone/DST logic
  - OAuth/token refresh
  - Google API fetch/parsing
  - hook execution
  - SQLite/local state/idempotency
  - CLI/config parsing
  - concurrency/shutdown

If the report is vague, infer the smallest concrete behavior you can test.

### 2) Inspect before editing
Read the relevant path end-to-end:
- CLI entrypoint
- config loading
- auth setup
- API fetch
- event normalization
- state evaluation
- deduplication/idempotency
- hook execution
- persistence updates
- error/logging path

Do not patch the first visible symptom if the true bug begins upstream.

### 3) Reproduce the bug
Whenever possible, reproduce using:
- an existing failing test
- a new focused failing test
- a minimal fixture
- a CLI invocation
- package-level testing
- race detection
- targeted logs/debugging

Useful commands:
```bash
go test ./...
go test -race ./...
go vet ./...
```

Use the narrowest command that exercises the issue first.

### 4) Calvin-specific failure modes to check
Be especially suspicious of:

#### Event and time logic
- UTC vs local-time confusion
- DST transitions
- start/end off-by-one bugs
- all-day event interpretation
- recurring instance identity mistakes
- “active now” comparisons at exact boundaries

#### Hook behavior
- one event firing twice
- state marked complete before hook success
- no timeout/cancellation
- broken argument escaping
- unexpected shell expansion
- child process leakage
- blocking forever on hook execution
- transient failures causing permanent skips

#### Google Calendar/API behavior
- missing pagination
- nil or unexpected timestamp shapes
- poor token refresh handling
- misleading auth errors
- missing request timeouts
- missing context propagation
- retries that duplicate side effects

#### SQLite/state behavior
- event marked processed too early
- duplicate rows or broken uniqueness
- stale state causing repeat or skipped hooks
- inconsistent writes after interruption
- missing transactions around related updates
- DB/file handle leaks

#### Go-specific behavior
- loop variable capture
- dropped errors
- nil interface confusion
- defer-in-loop
- goroutine leaks
- timer/ticker leaks
- data races
- deadlocks
- unsynchronized shared state

### 5) Fix minimally
Once the root cause is identified:
- change the smallest correct surface area
- keep style and structure consistent with the repo
- avoid adjacent refactors
- preserve behavior outside the fix
- improve errors only when it materially helps diagnosis

### 6) Add regression coverage
Every real bug fix should usually include a regression test unless infeasible.

Prefer tests that are:
- narrow
- deterministic
- inexpensive
- clearly named for the failure mode
- explicit about expected behavior

Good Calvin test targets include:
- timezone conversion
- DST boundary handling
- all-day event behavior
- recurring event identity
- duplicate hook prevention
- missed-hook regression
- token refresh failure handling
- SQLite idempotency
- hook timeout/cancellation behavior

### 7) Validate after the fix
After editing:
- run targeted tests first
- then broader package tests
- then `go test ./...`
- then `go test -race ./...` when relevant
- then `go vet ./...`

If anything cannot be validated, state exactly what remains unverified.

### 8) Commit the changes
Every bug fix should be committed with the tests that validate it
- include a description of the bug and why the fix works in the commit

## Output format

### Bug summary
- observed symptom
- expected behavior
- root cause
- affected subsystem
- severity

### Changes made
- file-by-file summary
- why each change was necessary

### Validation
- commands run
- tests added or updated
- result of each check

### Remaining risk
- anything not fully verified
- edge cases still worth watching

## Guardrails
Do not:
- print or persist OAuth secrets or tokens
- add verbose logging of private calendar data unless necessary
- switch to shell-based execution if direct exec is safer
- introduce retries that can duplicate side effects without idempotency protection
- mark a hook complete before success is actually achieved
- swallow errors that affect observable behavior
- broaden the task into a refactor unless required

## Decision rules
- If the bug can be reproduced, reproduce first.
- If it cannot be reproduced, inspect the most likely path and state assumptions clearly.
- If the fix is uncertain, prefer a failing test or better instrumentation over a speculative patch.
- If a broader change seems necessary, explain why the narrow fix would be unsafe or incorrect.

## Definition of done
A bug fix is complete only when:
1. the root cause is identified,
2. the code change is minimal and targeted,
3. a regression test exists when feasible,
4. validation has been run,
5. any remaining uncertainty is clearly stated,
6. the change is committed with a clear description of the solution.
