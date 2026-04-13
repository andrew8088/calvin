# Privacy Policy

Calvin is a local-only command-line tool that watches your Google Calendar and runs shell scripts you define on your own machine.

## What data Calvin accesses

Calvin requests read-only access to your Google Calendar (`calendar.readonly` scope). It reads event metadata: titles, times, locations, meeting links, attendees, and descriptions.

## How your data is used

Calvin syncs your calendar events to a local SQLite database on your computer. This data is used solely to trigger shell scripts (hooks) that you create and control. Examples include desktop notifications before meetings, auto-opening meeting links, and logging.

## Where your data is stored

All data stays on your machine. Calvin stores:
- An OAuth refresh token at `~/.local/share/calvin/token.json`
- A SQLite database of synced events at `~/.local/share/calvin/events.db`
- Log files at `~/.local/state/calvin/calvin.log`

## What Calvin does NOT do

- Calvin never sends your calendar data to any third-party server
- Calvin never creates, modifies, or deletes calendar events
- Calvin never shares your data with anyone
- Calvin has no analytics, telemetry, or tracking

## Revoking access

Run `calvin auth --revoke` to delete your stored credentials. You can also revoke access from your Google Account at https://myaccount.google.com/permissions.

## Contact

For questions or concerns, open an issue at https://github.com/andrew8088/calvin/issues.
