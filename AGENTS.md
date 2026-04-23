# AGENTS.md

Calvin is a Go CLI for reading Google Calendar events and running local hooks/scripts.

## Purpose

This is a Go CLI project. Agents working in this repo should favor:

- modern, idiomatic Go
- simplicity over cleverness
- standard library first
- test-driven development where practical
- a **functional core, imperative shell** architecture
- small, reviewable changes
- clear error handling and predictable CLI behavior

When in doubt, optimize for maintainability, correctness, and ease of testing.

---

## Project assumptions

- Language: Go
- Minimum Go version: **1.23**
- Application type: **CLI**
- CLI framework: **Cobra**
- Config format: **TOML**
- Local storage: **SQLite**
- External integrations: Google APIs / OAuth2
- Logging may use lumberjack-backed file rotation where appropriate

## Priorities

- correctness over cleverness
- minimal, targeted changes
- strong regression coverage
- no secret leakage
- no duplicate hook firing
- no missed hook execution
- careful timezone and DST handling
- safe hook/script execution
- clear error reporting

## Skill routing

- Use **Deep Review** for repository-wide audits, pre-release reviews, bug hunting, and identifying missing tests or systemic risks.
- Use **Bug Fix** for a specific reported bug, failing test, or concrete malfunction that needs diagnosis, reproduction, a targeted fix, and validation.

## Calvin-specific skepticism
Be especially cautious about: - duplicate hook execution
- missed hook execution
- recurring event identity bugs
- all-day event handling
- DST and timezone edge cases
- token refresh failures
- Google API pagination and transient failures
- SQLite idempotency/state bugs
- command injection and shell quoting
- logs that expose calendar details or credentials

## Working norms
- Reproduce before editing when feasible.
- Prefer the smallest correct fix.
- Add regression tests for real bugs when feasible.
- Avoid broad refactors unless required for correctness.
- State assumptions and unverified areas explicitly.
- If a task came from the tracker (`TODO.yaml` or equivalent), update the tracker entry when the work is complete before finishing or committing.

---

## Operating principles

### 1. Prefer the standard library

Use the Go standard library unless there is a strong, concrete reason not to.

Before introducing a new dependency, ask:

- Does the standard library already solve this well enough?
- Is the dependency actively maintained and widely trusted?
- Does it reduce complexity rather than hide it?
- Is it worth the additional API surface, upgrade risk, and transitive dependencies?

Do not add dependencies for minor convenience.

### 2. Keep the code idiomatic and lightweight

Follow normal Go conventions:

- simple packages
- small interfaces
- concrete types by default
- explicit control flow
- composition over inheritance-style abstraction
- minimal magic
- avoid framework-like internal architectures

Prefer readable code over heavily abstracted code.

### 3. Small, focused changes

Agents should make the smallest change that fully solves the problem.

Avoid bundling unrelated refactors into feature work. If refactoring is necessary, keep it tightly scoped to enabling the change.

### 4. Leave the repo better than you found it

When touching code, improve nearby clarity where inexpensive:

- rename confusing identifiers
- remove dead code
- strengthen tests
- simplify branching
- improve doc comments where needed

Do not perform broad cleanup unless explicitly asked.

---

## Architecture: functional core, imperative shell

This repo should bias toward a **functional core, imperative shell** design.

### Functional core

Put domain behavior in code that is:

- deterministic
- side-effect free where possible
- driven by explicit inputs
- returning values rather than mutating hidden state
- easy to unit test without files, databases, network, clocks, or environment variables

Examples:

- parsing domain-specific inputs into validated internal representations
- planning sync operations
- merge/reconciliation rules
- filtering, transformation, and decision logic
- formatting domain output models before rendering

### Imperative shell

Keep side effects at the edges:

- Cobra command wiring
- filesystem access
- environment/config loading
- clock access
- random generation
- network calls
- OAuth flows
- database I/O
- terminal output
- logging

The shell should:

- gather inputs
- call pure or mostly-pure core logic
- perform side effects
- translate errors into user-facing CLI behavior

### Practical rule

If a behavior can be written as a pure function, do that first.
If a behavior cannot be pure, isolate the impure parts behind a narrow seam.

---

## Testing policy

### 1. Prefer TDD where practical

Agents should use **test-driven development where practical**:

1. write or update a failing test
2. implement the smallest change to make it pass
3. refactor while keeping tests green

TDD is especially encouraged for:

- parsing
- business rules
- synchronization logic
- configuration resolution
- output shaping
- bug fixes

It may be less useful for thin glue code, generated code, or trivial wiring.

### 2. Test behavior, not implementation details

Tests should focus on observable behavior:

- inputs and outputs
- returned errors
- persisted effects
- emitted user-visible text when relevant

Avoid brittle tests tied to private implementation details.

### 3. Prefer table-driven tests

Use table-driven tests for logic with multiple scenarios. Name cases clearly.

### 4. Keep unit tests fast and isolated

Unit tests should not require:

- network access
- real OAuth credentials
- external services
- shared mutable global state
- sleeping for timing coordination

When I/O is necessary, prefer:

- `t.TempDir()`
- in-memory substitutes where reasonable
- temporary SQLite databases created per test
- explicit dependency injection

### 5. Add regression tests for bugs

Every bug fix should include a regression test when practical.

### 6. Integration tests should be deliberate

Use integration tests for boundaries that matter, such as:

- SQLite persistence behavior
- config file loading
- Cobra command execution
- Google API client integration seams

Keep them fewer and more targeted than unit tests.

### 7. Control time and nondeterminism

Do not call `time.Now()`, random generators, or environment lookups directly inside core logic when testability matters.

Inject these as dependencies or pass them in explicitly.

---

## Project layout guidance

Keep layout **lightweight and idiomatic**, not ceremonial.

### Preferred tendencies

- keep packages cohesive and small
- use `internal/` for non-public application code that should not be imported externally
- keep Cobra command definitions near CLI entry behavior
- keep domain logic separate from command wiring and I/O
- avoid deep package trees without clear value

### Suggested shape

This is guidance, not a rigid template:

- `cmd/` for CLI entrypoints and Cobra setup
- `internal/...` for application internals
- a small set of focused packages for domain logic, config, storage, and integrations

Avoid creating packages with only one file or extremely vague names like `utils`, `common`, or `helpers`.

---

## CLI design rules

### 1. CLI behavior must be predictable

Commands should:

- have clear names
- use consistent flag semantics
- provide useful help text
- return meaningful exit behavior
- print stable, understandable output

### 2. Separate command wiring from command logic

Cobra commands should mostly do the following:

- bind flags
- load dependencies
- call a function or service
- map returned errors to user-facing output

Do not bury business logic directly inside `Run` or `RunE` bodies.

### 3. Be explicit about stdout vs stderr

Use:

- `stdout` for primary command output
- `stderr` for errors, warnings, and diagnostics

This keeps commands script-friendly.

### 4. Treat user-facing text as part of the interface

Error messages and help text should be:

- concise
- actionable
- specific
- free of internal jargon where possible

Good error messages explain what failed and what the user can do next.

---

## Error handling

### 1. Return errors; do not hide them

Prefer returning errors up the call stack over logging-and-continuing or panicking.

### 2. Wrap errors with context

Use `%w` and add useful context:

```go
return fmt.Errorf("open config %q: %w", path, err)
```

Context should help a future reader or user understand what operation failed.

### 3. Avoid panic for normal failures

Do not panic for expected runtime errors such as:

- invalid config
- missing files
- database errors
- API failures
- invalid user input

Reserve panic for truly unrecoverable programmer errors.

### 4. Keep user-facing errors clean

Internal layers may return rich contextual errors. The CLI boundary should decide how much detail to show the user.

For known user mistakes, prefer friendly messages over raw stack-like chains.

### 5. Use sentinel errors sparingly

Use sentinel errors only when callers genuinely need to branch on them. Prefer typed or wrapped errors over large sentinel sets.

---

## API and package design

Even though this is a CLI, package APIs should be designed carefully.

### 1. Accept interfaces only when needed

Do not introduce interfaces prematurely. Accept and return concrete types unless an interface clearly improves:

- testing
- decoupling at a real boundary
- substitutability across implementations you actually need

Define interfaces close to the consumer, not the producer.

### 2. Keep interfaces small

Prefer narrow interfaces with one or a few methods.

### 3. Make zero values useful when sensible

Design types so zero values are either valid or clearly impossible to misuse.

### 4. Pass `context.Context` correctly

Use `context.Context` for request-scoped cancellation and deadlines, especially around:

- network calls
- database operations
- long-running command operations

Do not store context in structs. Pass it explicitly.

### 5. Avoid global mutable state

Do not rely on package globals for config, clients, clocks, or caches.

Prefer explicit construction and dependency passing.

---

## Configuration

### 1. Make config resolution explicit

Configuration should have a clear precedence order if multiple sources exist, for example:

- flags
- environment
- config file
- defaults

If such precedence exists, preserve it consistently and test it.

### 2. Validate early

Parse and validate config near startup so failures happen early and clearly.

### 3. Keep config models distinct from domain models

Do not let raw config structs become the application's core domain objects unless that is genuinely the same concept.

---

## Database and persistence guidance

SQLite is part of this project. Agents should keep persistence logic disciplined.

### 1. Keep SQL and domain logic separate

Persistence code should handle:

- queries
- scanning
- transactions
- mapping between rows and domain models

Business rules should not be embedded deeply inside SQL-facing code unless the rule truly belongs in the database layer.

### 2. Prefer explicit transactions when correctness depends on them

When multiple writes must succeed or fail together, use explicit transactions.

### 3. Test persistence at the repository/store boundary

Use integration-style tests for SQL behavior that matters:

- schema assumptions
- constraints
- transaction behavior
- query correctness

### 4. Handle migrations deliberately

If the repo has schema migration logic, changes must be:

- forward-safe
- tested
- idempotent where appropriate
- clearly documented

### 5. Do not over-abstract the database layer

A small, clear store/repository package is better than a generic abstraction that hides SQL semantics.

---

## External integrations

This project uses OAuth2 and Google APIs.

### 1. Wrap third-party APIs at the boundary

Keep vendor-specific types and logic near integration edges.

Translate them into project-owned types as they enter the core.

### 2. Do not leak external schemas everywhere

Avoid spreading Google API response types throughout the codebase.

### 3. Make integrations testable

Where practical, isolate integration calls behind narrow adapter layers so core behavior can be tested without live API calls.

### 4. Be conservative with retries and backoff

If retry behavior exists, make it explicit and bounded.

### 5. Protect secrets

Never hardcode credentials, tokens, or secret material.

Be careful not to log:

- access tokens
- refresh tokens
- authorization headers
- full credential payloads

---

## Logging and observability

### 1. Logs should help operators, not narrate every line

Log meaningful lifecycle events and failures, not noise.

### 2. Prefer structured or at least consistent logs

If the repo has a logging pattern, follow it consistently.

### 3. Do not log secrets or sensitive user data

Redact where needed.

### 4. Keep logging out of the functional core

Core logic should usually return values and errors, not emit logs directly.

Logging belongs in orchestration layers unless there is a strong reason otherwise.

### 5. Add instrumentation only where it clarifies behavior

Do not add observability complexity without a concrete benefit.

---

## Concurrency

Use concurrency only when it clearly improves responsiveness, throughput, or structure.

### Rules

- prefer simple synchronous code by default
- use goroutines deliberately, not decoratively
- coordinate with contexts, channels, and `sync` primitives carefully
- avoid shared mutable state where possible
- make shutdown and cancellation explicit
- test concurrency-sensitive behavior carefully

If concurrency makes code harder to reason about without strong benefit, do not add it.

---

## Code style

### Follow standard Go style

- run `gofmt`
- write clear names
- keep functions focused
- avoid excessive comments that restate the code
- add doc comments for exported identifiers when useful

### Prefer clarity over cleverness

Avoid:

- unnecessary generic abstractions
- deeply nested branching when a simpler shape exists
- hidden control flow
- helper piles with vague names

### Function size guidance

A function need not be tiny, but if it is hard to scan, hard to name, or hard to test, split it.

---

## Tooling expectations

Agents should use and respect modern Go tooling.

### Required baseline

At minimum, changes should pass:

```sh
go test ./...
go vet ./...
```

### Formatting

Always format code with:

```sh
gofmt -w .
```

If this repo adopts `gofumpt`, follow it consistently. Do not assume it is required unless the repo already uses it.

### Linting

If `golangci-lint` is configured in the repo, run it and fix issues relevant to your change.

If not, at least ensure code is clean under `go vet` and common Go best practices.

### Vulnerability checking

When touching dependencies or security-sensitive code, prefer running:

```sh
govulncheck ./...
```

if available in the development environment.

### Dependency hygiene

When modifying dependencies:

- keep `go.mod` and `go.sum` tidy
- do not add unnecessary modules
- prefer stable releases
- explain why a new dependency is needed

---

## How agents should implement changes

When asked to implement something, follow this default workflow:

1. understand the user-visible behavior required
2. identify the core logic and the side-effecting shell
3. add or update tests first where practical
4. implement the smallest correct change
5. run formatting and tests
6. do a brief cleanup pass for clarity
7. summarize what changed and any notable tradeoffs

---

## What to avoid

Agents should avoid:

- adding dependencies without strong justification
- putting business logic directly into Cobra command handlers
- tightly coupling core behavior to filesystem, network, DB, or terminal I/O
- introducing broad interfaces before they are needed
- using global mutable state
- writing tests that depend on the network or real credentials
- mixing unrelated refactors into feature work
- panics for expected runtime failures
- logging secrets or tokens
- creating vague packages like `util` or `helpers` as a dumping ground

---

## Decision rules for ambiguous cases

If multiple valid approaches exist, prefer the option that is:

1. more idiomatic in Go
2. easier to test
3. simpler to explain to a future maintainer
4. lower in dependency and abstraction cost
5. more aligned with functional core / imperative shell

---

## Definition of done

A change is not complete until, where applicable:

- tests cover the new or changed behavior
- code is formatted
- `go test ./...` passes
- `go vet ./...` passes
- errors are contextual and user-facing behavior is sensible
- side effects are kept at the edges
- the design remains idiomatic and standard-library-first

---

## Notes for future contributors and agents

This repo values pragmatic engineering discipline over architecture astronautics.

Write Go that a strong Go developer would consider:

- boring in a good way
- easy to test
- easy to debug
- easy to change

Favor a clean core of deterministic behavior surrounded by a thin shell of CLI and integration code.
