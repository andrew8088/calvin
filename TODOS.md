# Calvin TODOs

## Pre-Launch: OAuth App Verification
Google OAuth verification submitted 2026-04-15. Waiting on Google review (1-3 weeks).
calendar.readonly is Sensitive (not Restricted), no CASA audit needed.
**Source:** /office-hours learning (google-oauth-test-mode), 2026-04-10.

## Phase 3: Push Notifications
Replace polling with Google Calendar push notifications (calendarList.watch()).
Eliminates 60s polling latency for schedule changes, reduces API quota usage.
Requires public webhook endpoint (Cloudflare Tunnel or ngrok).
**Source:** /plan-eng-review outside voice, 2026-04-10.

## Ideas

1. **Event filtering rules** — Per-hook `match`/`ignore` patterns in config (glob on title, calendar, organizer, attendee count) so hooks don't each need their own jq guard.
2. **`on-event-change` hook type** — Fire hooks when events are modified (rescheduled, retitled, attendees changed). Pass both old and new payloads.
3. **`calvin week` command** — Show the next 7 days of events, not just today. (Or `calvin events --range=week`.)
4. **Hook retry with backoff** — `hook_max_retries` and `hook_retry_delay_seconds` in config. Re-queue failed/timed-out hooks with exponential backoff.
5. **`calvin replay <event-id> [hook-type]`** — Re-fire hooks for a real past event using the stored payload, distinct from `calvin test` which uses mock data.
6. **Hook failure notifications** — Optional `[notifications]` config section (webhook URL or command) that Calvin calls whenever a hook fails, so failures don't go unnoticed.
7. **`calvin free` command** — Show free/busy gaps between today's meetings. Pipe-friendly for scheduling scripts.
8. **Per-hook `pre_event_minutes`** — Allow overrides via sidecar `.toml` file or comment directive (`# calvin: pre_event_minutes=15`) instead of one global value.
9. **`calvin export`** — Export the local event cache to `.ics`, `.csv`, or `.json` for feeding into time-tracking, billing, or analytics tools.
10. **All-day event support** — `convertEvent` currently skips all-day events. Fall back to `item.Start.Date` so birthdays, deadlines, and OOO markers can trigger hooks.
11. **`calvin hooks validate`** — Pre-flight check: run shellcheck, verify shebang, detect common mistakes (reading stdin twice, missing jq), confirm clean exit with schema payload.
12. **`calvin stats` command** — Aggregate hook_executions into a dashboard: hooks fired per day, average duration, failure rate, slowest hooks, most active events.
