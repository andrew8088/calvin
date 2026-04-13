# Calvin TODOs

## Phase 1: `calvin stop` Command
Add `calvin stop` to the Phase 1 CLI. Read PID from calvin.pid, send SIGTERM,
wait up to 5s for graceful shutdown (in-flight hooks complete, SQLite WAL checkpoint),
clean up PID file. Essential for daemon lifecycle management.
**Depends on:** Signal handling in `calvin start` (SIGTERM/SIGINT handler with sync.WaitGroup).
**Source:** /plan-ceo-review outside voice, 2026-04-10.

## Phase 1: README FAQ — "Why not cron + gcalcli?"
Document why a purpose-built daemon beats cron + gcalcli: sub-second timer precision
at event boundaries, deduplication across restarts, crash recovery via hook_executions
table, structured event metadata on stdin, and the hook paradigm (convention-based,
zero config). One paragraph in the README FAQ section.
**Depends on:** README existing.
**Source:** /plan-ceo-review outside voice, 2026-04-10.

## Phase 3: Push Notifications
Replace polling with Google Calendar push notifications (calendarList.watch()).
Eliminates 60s polling latency for schedule changes, reduces API quota usage.
Requires public webhook endpoint (Cloudflare Tunnel or ngrok).
**Depends on:** Phase 1 validation confirming the core hook concept works.
**Source:** /plan-eng-review outside voice, 2026-04-10.

## Pre-Launch: OAuth App Verification
Google OAuth apps in testing mode limit to 100 manually-added test users.
For public distribution (brew install calvin), the app must pass Google's
verification process (requires privacy policy, homepage, takes days-weeks).
**Depends on:** Phase 1 validation. Deciding Calvin is worth open-sourcing.
**Source:** /office-hours learning (google-oauth-test-mode), 2026-04-10.

## Phase 2: Recompute Adjacent Events at Fire Time
previous_event and next_event are computed at timer schedule time, not fire time.
If events change between scheduling and firing, adjacent event data is stale.
Acceptable for Phase 1 (hooks don't use it). Critical for Phase 2 transition hooks.
**Depends on:** Phase 2 transition hook implementation.
**Source:** /plan-eng-review outside voice, 2026-04-10.

## Phase 2: Multi-Calendar Support
Watch multiple calendars (work + personal) with per-calendar hook filtering.
Unlocks "ignore personal events during work hooks" and "different hooks for
different calendars." Requires config schema for calendar list + filter rules.
Quota impact: each additional calendar doubles API usage per sync cycle.
**Depends on:** Phase 1 validation with primary calendar.
**Source:** /plan-eng-review outside voice, 2026-04-13.

## Phase 1.1: `calvin sync` Command
Force an immediate API poll + diff + timer reschedule. Useful after manually
changing your calendar when you don't want to wait up to 60s. Implementation
options: send SIGUSR1 to the daemon (read PID from calvin.pid) or use a Unix
domain socket for IPC. SIGUSR1 is simpler for Phase 1.
**Depends on:** Signal handling in `calvin start`.
**Source:** /plan-ceo-review reviewer concern #2, confirmed /plan-eng-review 2026-04-13.

## Phase 1.1: `--json` Output Mode
Add `--json` flag to `calvin events`, `calvin status`, and `calvin next` for
machine-parseable output. Enables piping to jq, scripting, and integration with
other tools. When `--json` is passed, output structured JSON instead of
human-formatted text. Disable color/symbols in JSON mode.
**Depends on:** Base commands implemented with human-readable output first.
**Source:** /plan-design-review, 2026-04-13.

## Phase 2+: macOS Shortcuts as Native Hook Runner
Allow hooks to trigger macOS Shortcuts by name via config (`runner = "shortcut"`).
Opens Calvin to non-shell automations (Focus mode, iMessage, Apple app integration).
Shell hooks can call `shortcuts run` directly as interim.
**Depends on:** Phase 1 validation of the shell hook model.
**Source:** /plan-ceo-review scope expansion #10, 2026-04-10.
