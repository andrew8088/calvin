---
name: Deep Review
description: Deeply review Calvin for bugs, correctness issues, reliability risks, security issues, and missing tests. Use for repository-wide audits, pre-release reviews, or when investigating overall code quality.
---

## Purpose
Use this skill for a **deep, bug-focused review** of **Calvin**, a Go CLI for reading Google Calendar events and running hooks/scripts.

This skill is for:
- repository-wide bug hunting
- pre-release review
- risk assessment of the codebase
- finding correctness issues and missing regression coverage
- identifying operational and reliability risks

This skill is **not** for implementing fixes unless explicitly asked.

## Product assumptions
Assume the following unless the repository clearly shows otherwise:
- Calvin is a **read-only** Google Calendar CLI
- it reads calendar events and related metadata from Google Calendar
- it may evaluate event state transitions such as upcoming, active, ended, or changed
- it may trigger local hooks/scripts on those transitions
- it may store local state to avoid duplicate execution
- duplicate hook execution is a serious bug
- missed hook execution is a serious bug
- timezone and DST correctness are critical
- shell execution safety matters
- OAuth tokens and calendar data are sensitive

## Review mindset
Be skeptical, precise, and evidence-driven.

Optimize for:
- real bugs
- real operational risks
- high-confidence findings
- concrete evidence
- missing tests in risky areas

Do not optimize for:
- praise
- style nits
- speculative concerns without a plausible failure mode
- broad architectural commentary

## Review priorities

### 1. Event correctness
Inspect for:
- start/end boundary bugs
- timezone conversion mistakes
- DST handling bugs
- all-day event mishandling
- recurring event identity confusion
- duplicate or missed transition detection
- incorrect “active now” logic
- assumptions that Google timestamps always exist in one shape

### 2. Hook execution
Inspect for:
- duplicate firing
- missed firing
- command injection risk
- broken quoting/argument handling
- missing timeout/cancellation
- orphaned child processes
- blocking behavior
- poor stdout/stderr capture
- retries that duplicate side effects

### 3. API / OAuth behavior
Inspect for:
- token refresh bugs
- bad handling of expired credentials
- missing pagination
- missing request timeouts
- missing context propagation
- transient failure handling gaps
- malformed API response assumptions
- quota/rate-limit handling weaknesses

### 4. Persistence and idempotency
Inspect for:
- duplicate state rows
- stale state causing repeat or skipped hooks
- bad uniqueness assumptions
- transaction gaps
- interrupted-write inconsistency
- resource leaks
- schema upgrade fragility

### 5. Go-specific risks
Inspect for:
- goroutine leaks
- race conditions
- deadlocks
- channel misuse
- loop variable capture bugs
- nil interface traps
- dropped or shadowed errors
- defer-in-loop mistakes
- timer/ticker leaks
- shared mutable state without synchronization

### 6. CLI / config behavior
Inspect for:
- bad defaults
- poor handling of missing config
- unsafe assumptions about HOME or working directory
- cron/systemd/non-interactive environment issues
- misleading exit codes
- poor error messages that hide root causes
- invalid flag combinations not rejected

### 7. Security and privacy
Inspect for:
- command injection
- unsafe shell execution
- token leakage
- secrets in logs
- insecure permissions on token/config files
- over-logging of calendar contents
- unsafe temp file handling

## Workflow

### 1) Understand the codebase first
Identify:
- main entrypoints
- command structure
- config loading path
- auth/token loading and refresh
- Google Calendar fetch flow
- event normalization and evaluation
- hook execution path
- local persistence/state logic

### 2) Inspect the riskiest paths first
Prioritize:
- event timing logic
- hook invocation
- idempotency/state tracking
- API and auth boundaries
- timezone/DST handling
- shutdown/cancellation behavior

### 3) Validate where possible
Run useful commands such as:
```bash
go test ./...
go test -race ./...
go vet ./...
```

Also run any repo-specific lint/build/test commands you discover.

### 4) Prefer high-confidence findings
Prefer a smaller set of well-supported findings over many weak ones.

If uncertain, state:
- the assumption
- why it matters
- how to verify it

## Output format

### Findings

#### [Severity] Short title
- **Location:** `path/to/file.go` — `FuncName` (include line numbers if available)
- **Problem:** What is wrong
- **Why it matters:** Why this is a real risk in Calvin
- **Trigger scenario:** Exact runtime scenario
- **Impact:** What the user would observe
- **Evidence:** Code path, test evidence, or command output
- **Suggested fix direction:** High-level fix approach only
- **Missing test:** Specific test that should exist

Order findings by:
1. severity
2. confidence
3. user impact

### Open questions / assumptions
List only assumptions that materially affect confidence.

### Gaps in review
List anything you could not verify.

### Final assessment
Give a short overall judgment and identify the top 3 most concerning areas.

## Guardrails
Do not:
- make code changes unless explicitly asked
- rewrite the architecture
- lead with praise
- report style-only issues as bugs
- expose secrets or sensitive data in the review output

## Calvin-specific things to be extra skeptical about
Actively look for:
- one event causing two hook executions
- hook suppression because state is marked too early
- recurrence instances treated as unique events incorrectly
- all-day events treated like timed events
- DST causing early or late execution
- transient Google API failure causing permanent missed work
- token refresh failures producing misleading errors
- shell argument construction that breaks valid inputs or allows injection
- stale SQLite state causing repeated or skipped transitions
- logs leaking event names, attendees, descriptions, or tokens

## Definition of done
A deep review is complete when:
1. major code paths were inspected,
2. risky areas were checked first,
3. validation commands were run where possible,
4. findings are concrete and evidence-based,
5. missing tests and remaining uncertainty are clearly stated.
