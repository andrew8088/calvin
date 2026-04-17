# Changelog

## Unreleased

### Added

- `calvin free` to print today's free time between cached meetings in a pipe-friendly format, with `--json` support.
- `calvin match` and `calvin ignore` commands for in-script hook filtering using inferred event context (`CALVIN_EVENT_FILE`).

### Docs

- Updated the command list to include `calvin week` and `calvin free`, and corrected the documented `--json` coverage.

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
