# Google Sync Handoff

## Context

The repeated `New event: Home (...)` log spam is fixed in code by diffing against the pre-upsert DB state instead of diffing after the sync write.

That fix does not explain why `calvin` appears to fetch the full event set on every sync. This doc is the handoff for that remaining investigation.

## Observed Symptoms

- `calvin logs` showed `Synced 52/53 events across 1 calendars` every minute instead of a small incremental change set.
- `calvin logs` repeatedly logged the same event as new before the diff fix.
- `calvin status` showed `last sync: never`, which means `GetSyncToken("primary")` returned an empty token.

## Relevant Code

- `internal/calendar/sync.go`
  - `Sync(...)` builds the Google Calendar `Events.List` request.
  - Full sync path uses `TimeMin(now)` and `TimeMax(now + 7 days)`.
  - Incremental sync path uses `SyncToken(syncToken)`.
  - The request always sets `SingleEvents(true)`, `OrderBy("startTime")`, and `ShowDeleted(true)`.
- `internal/cli/start_cmd.go`
  - `runForeground(...)` reads the stored sync token, calls `syncer.Sync(...)`, then persists `newToken` if non-empty.
- `internal/db/db.go`
  - `GetSyncToken(...)` and `SetSyncToken(...)` own token persistence.
- `internal/cli/status_cmd.go`
  - `runStatus()` only checks `GetSyncToken("primary")`.

## Strong Hypotheses

### 1. `status` may be misleading for non-primary calendars

If the configured calendar is not `primary`, `calvin status` can still print `last sync: never` even when another calendar has a stored token.

This does not explain the repeated full-sized sync responses by itself, but it does mean the current status output is not reliable evidence unless the synced calendar is actually `primary`.

## 2. The incremental request shape likely violates Google sync-token rules

Google's docs say `syncToken` cannot be combined with `orderBy`, `timeMin`, or `timeMax`, and they also say the initial and incremental requests should otherwise use the same query parameters.

Current code in `internal/calendar/sync.go` does this:

- full sync: `SingleEvents(true)`, `OrderBy("startTime")`, `ShowDeleted(true)`, `TimeMin(...)`, `TimeMax(...)`
- incremental sync: `SingleEvents(true)`, `OrderBy("startTime")`, `ShowDeleted(true)`, `SyncToken(...)`

That is suspicious in two ways:

- `OrderBy("startTime")` is still present on the incremental path.
- The full sync uses a bounded 7-day window, while the incremental path cannot legally reuse `timeMin` or `timeMax` with `syncToken`.

This may mean the current query design is fundamentally incompatible with Google incremental sync, even if the API does not fail loudly in the current logs.

## 3. `nextSyncToken` may never be arriving or never being saved

If `result.NextSyncToken` is empty on the final page, `start_cmd.go` will never persist a token because it only calls `SetSyncToken(...)` when `newToken != ""`.

That could come from:

- a request shape that does not produce `nextSyncToken`
- a paging edge case
- saving the token under a calendar ID that `status` does not inspect

## Suggested Investigation Order

### 1. Check what tokens actually exist in SQLite

Stop the daemon first, then inspect `sync_state`.

```sh
calvin stop
sqlite3 "$HOME/.local/share/calvin/events.db" "select calendar_id, length(token), updated_at from sync_state;"
```

Questions to answer:

- Is there a token row at all?
- Is the row for `primary` or another calendar ID?
- Is the token empty or non-empty?

### 2. Add temporary logging around token flow

Instrument `internal/cli/start_cmd.go` and log, per calendar:

- calendar ID
- whether the input sync token is empty
- whether `fullSync` is true
- whether `newToken` is empty
- `len(events)` returned by `syncer.Sync(...)`

This should make it obvious whether Calvin is:

- never receiving a `nextSyncToken`
- receiving it but saving under a different calendar ID
- receiving a 410 path and resetting into full sync repeatedly

### 3. Inspect the Google request builder in `internal/calendar/sync.go`

The first code change to test is small and likely correct:

- do not set `OrderBy("startTime")` when `syncToken != ""`

After that, reassess the bigger design issue:

- a bounded initial sync with `TimeMin` and `TimeMax` may not compose cleanly with Google incremental sync tokens

If incremental sync requires a stable query shape that cannot include the rolling window, the likely fix is:

- do one unbounded initial sync for token establishment
- keep local scheduling and display bounded in the app layer, not in the Google sync query

## Expected Good Behavior After Fix

- Most sync cycles should return zero or a very small number of events.
- `calvin logs` should stop showing the entire upcoming week every minute.
- `sync_state` should contain a non-empty token for each synced calendar.
- `calvin status` should report sync state per configured calendar, not only `primary`.

## Notes

- The duplicate `Home` log bug was real and is already fixed separately.
- The Google sync-token issue still needs root-cause confirmation. The strongest lead is the request shape in `internal/calendar/sync.go`.
