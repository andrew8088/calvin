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
