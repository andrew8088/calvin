# Changelog

## Unreleased

## v0.9.0 — 2026-04-17

### Added

- `calvin logs --follow` / `-f` to stream new matching daemon log entries in text mode.

### Fixed

- Ongoing all-day events are now diffed against the pre-sync database state so they stop being reported as newly created on every refresh.
- SQLite access now runs through a serialized DB layer to prevent concurrent hook execution and adjacent-event lookups from crashing.

## v0.8.0 — 2026-04-17

### Breaking

- **Unified JSON contract.** Agent-relevant commands now return a consistent top-level JSON envelope under `--json`, instead of the older mix of raw arrays, raw objects, and text-only responses. If you were parsing the previous raw `--json` shapes from `events`, `next`, `week`, `free`, or `status`, update your automation to read the new `{ "ok", "command", "data" }` structure.

### Added

- `calvin free` to print today's free time between cached meetings in a pipe-friendly format, with `--json` support.
- `calvin match` and `calvin ignore` commands for in-script hook filtering using inferred event context (`CALVIN_EVENT_FILE`).
- `calvin commands`, `calvin describe`, and `calvin schema` for AI-first runtime discovery.
- `CALVIN_OUTPUT=json` and `--output json` to make structured output explicit.
- Structured JSON stderr errors for JSON-mode command, usage, and parse failures.

### Docs

- Updated the command list to include `calvin commands`, `calvin describe`, and `calvin schema`, and documented the new JSON contract.

## v0.7.0 — 2026-04-16

### Improved

- GitHub Actions now run on Node 24.

## v0.6.0 — 2026-04-16

### Improved

- `scripts/release.sh` now waits for the GitHub Actions `release.yml` run and surfaces the workflow URL while the release is publishing.

## v0.5.0 — 2026-04-16

### Added

- `calvin week` to show the next 7 days of events.
- All-day event support across sync, storage, and CLI output.
- `calvin free` to print today's free time between cached meetings.

### Improved

- Added local release automation scripts for tagged GitHub Releases.

## v0.4.0 — 2026-04-15

### Breaking

- **Hook directory names renamed.** `pre_event` is now `before-event-start`, `event_start` is now `on-event-start`, `event_end` is now `on-event-end`. Rename your hooks directories to match:
  ```bash
  cd ~/.config/calvin/hooks
  mv pre_event before-event-start
  mv event_start on-event-start
  mv event_end on-event-end
  ```

### Added

- **Multi-calendar support.** Watch multiple calendars by adding `[[calendars]]` entries to `config.toml`. Hooks receive the `calendar` field in their JSON payload for filtering.
- **Adjacent events in hook payloads.** `previous_event` and `next_event` are now populated at fire time, enabling transition-aware hooks.
- **`calvin sync` command.** Force an immediate calendar sync instead of waiting up to 60s. Sends SIGUSR1 to the running daemon.
- **`--json` flag** on `calvin events`, `calvin next`, and `calvin status` for machine-readable output.

### Improved

- **`calvin stop`** now waits for the daemon to exit (5s timeout, falls back to SIGKILL).

### Fixed

- Hook examples that read stdin multiple times now buffer correctly.

## v0.3.0 — 2026-04-15

### Added

- Multi-calendar sync infrastructure (per-calendar sync tokens, DB migration).
- Adjacent event recomputation at hook fire time.
- `calvin sync` and improved `calvin stop`.
- `--json` output mode.
- README FAQ section.

### Fixed

- Stdin consumed on first `jq` call in hook examples, leaving subsequent fields empty.

## v0.2.0 — 2026-04-14

### Added

- 93 unit and integration tests across calendar, config, db, hooks, logging, and scheduler packages.
- CI test workflow (GitHub Actions).

### Fixed

- SQLite concurrent access crash (mutex on Executor).
- Timeout pipe leak (process group kill + WaitDelay).
- XDG env setup in integration tests.

## v0.1.0 — 2026-04-14

Initial release. Local Go daemon that watches Google Calendar and fires user-defined shell scripts at event boundaries.

- 13 CLI commands
- Google Calendar sync (incremental, pagination, 410 recovery)
- Hook discovery, execution, semaphore, dedup, timeout
- Structured JSON logging with rotation
- OAuth2 flow with bundled credentials
- SQLite storage (WAL mode, CGO-free)
- Homebrew tap, goreleaser builds (macOS + Linux, amd64/arm64)
