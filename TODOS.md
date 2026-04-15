# Calvin TODOs

## Phase 3: Push Notifications
Replace polling with Google Calendar push notifications (calendarList.watch()).
Eliminates 60s polling latency for schedule changes, reduces API quota usage.
Requires public webhook endpoint (Cloudflare Tunnel or ngrok).
**Depends on:** Phase 1 validation confirming the core hook concept works.
**Source:** /plan-eng-review outside voice, 2026-04-10.

## Pre-Launch: OAuth App Verification
Google OAuth verification submitted 2026-04-15. Waiting on Google review (1-3 weeks).
calendar.readonly is Sensitive (not Restricted), no CASA audit needed.
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

## Phase 2+: macOS Shortcuts as Native Hook Runner
Allow hooks to trigger macOS Shortcuts by name via config (`runner = "shortcut"`).
Opens Calvin to non-shell automations (Focus mode, iMessage, Apple app integration).
Shell hooks can call `shortcuts run` directly as interim.
**Depends on:** Phase 1 validation of the shell hook model.
**Source:** /plan-ceo-review scope expansion #10, 2026-04-10.
