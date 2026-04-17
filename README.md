# Calvin

Programmable calendar hooks for your Mac. Calvin watches your Google Calendar and fires shell scripts at event lifecycle moments.

Drop scripts in your hooks directory, and Calvin handles the rest.

## Install

```bash
# Homebrew
brew install andrew8088/tap/calvin

# Or download from GitHub Releases
# https://github.com/andrew8088/calvin/releases/latest
```

## Quick start

```bash
calvin init                    # Scaffold config and example hooks
calvin auth                    # Authenticate with Google Calendar
calvin test example-notify     # Test an example hook
calvin start --background      # Start the daemon
```

Calvin requests read-only access to your calendar. It never creates, modifies, or deletes events.

## How it works

Calvin runs as a local daemon that syncs your Google Calendar every 60 seconds. When events approach, start, or end, it fires shell scripts you define.

Three hook types:

| Hook Type     | When it fires                        |
|---------------|--------------------------------------|
| `before-event-start`   | N minutes before the event starts    |
| `on-event-start` | When the event starts                |
| `on-event-end`   | When the event ends                  |

Hooks receive event data as JSON on stdin:

```bash
#!/usr/bin/env bash
calvin match --calendar "primary" --title "*standup*" || exit 0
INPUT=$(cat /dev/stdin)
TITLE=$(echo "$INPUT" | jq -r '.title')
LINK=$(echo "$INPUT" | jq -r '.meeting_link // empty')
osascript -e "display notification \"$TITLE\" with title \"Calvin\""
```

Hooks also receive `previous_event` and `next_event` with adjacent event details (or `null`). Run `calvin hooks schema` to see all available JSON fields.

## Commands

```
calvin init          Scaffold config and example hooks
calvin auth          Authenticate with Google Calendar
calvin start         Start the daemon (--background for detached)
calvin stop          Stop the daemon
calvin sync          Force an immediate calendar sync
calvin status        Show daemon health dashboard
calvin next          Show next upcoming event with countdown
calvin events        List today's events
calvin events <id>   Show event detail with hook execution log
calvin week          Show the next 7 days of events
calvin free          Show today's free time between meetings
calvin commands      List runtime command metadata
calvin describe      Describe a command path for agents
calvin schema        Print machine-readable schemas
calvin hooks list    List all discovered hooks
calvin hooks new     Create a new hook from template
calvin hooks schema  Print the JSON input schema
calvin match         Assert hook event matches filters
calvin ignore        Assert hook event matches ignore filters
calvin test <hook>   Test a hook with real or mock event data
calvin doctor        Run health checks
calvin logs          Show daemon logs (--hook, --level, --event filters)
calvin version       Print version
```

## Agent Usage

Calvin now exposes a structured CLI surface for agent workflows.

- Use `--json` for structured command output.
- Use `--output json` if you want the output mode spelled explicitly.
- Use `CALVIN_OUTPUT=json` to make JSON the default for a session.
- Use `calvin commands`, `calvin describe`, and `calvin schema` instead of scraping `--help`.

Examples:

```bash
calvin commands --json
calvin describe hooks new --json
calvin schema hook-payload
calvin version --json
calvin events --json
calvin events abc123 --json
calvin logs --json --since 2026-04-17T17:30:00Z
calvin test example-notify --json
```

Structured command output now uses a consistent top-level contract:

```json
{
  "ok": true,
  "command": "version",
  "data": {
    "version": "dev",
    "commit": "none"
  }
}
```

Errors in JSON mode are written to `stderr` as structured JSON with a non-zero exit code:

```json
{
  "ok": false,
  "command": "hooks new",
  "error": {
    "code": "invalid_hook_type",
    "message": "invalid hook type: before-start"
  }
}
```

Notes:

- `match` and `ignore` keep their exit codes: `0` matched, `1` not matched, `2` usage or context error.
- `help` and `completion` remain human-oriented in v1. Agents should use `commands`, `describe`, and `schema` instead.
- `auth --json` does not support the interactive browser OAuth flow yet. It supports structured revoke output and structured failures.
- `start --json` currently reports structured preflight failures only.

`calvin free` prints one free slot per line as `start<TAB>end<TAB>duration_seconds`, using local RFC3339 timestamps.

## Creating hooks

```bash
calvin hooks new before-event-start my-notifier
```

This creates a hook at `~/.config/calvin/hooks/before-event-start/my-notifier` with a starter template. Edit it, and Calvin picks it up on the next sync cycle (no restart needed).

Hooks must be executable (`chmod +x`). Calvin discovers hooks by scanning the hooks directory every sync cycle.

### In-script filtering (no extra config)

Use `calvin match` and `calvin ignore` at the top of your hook script:

```bash
#!/usr/bin/env bash
calvin match --calendar "Work*" --title "*1:1*" || exit 0
calvin ignore --title "*OOO*" && exit 0

INPUT=$(cat /dev/stdin)
TITLE=$(echo "$INPUT" | jq -r '.title')
echo "Running for: $TITLE"
```

Both commands infer event context automatically inside hooks through `CALVIN_EVENT_FILE`. They return exit code `0` when matched, `1` when not matched, and `2` for usage/context errors.

## Example hooks

### Desktop notification before meetings

```bash
#!/usr/bin/env bash
INPUT=$(cat /dev/stdin)
TITLE=$(echo "$INPUT" | jq -r '.title')
LINK=$(echo "$INPUT" | jq -r '.meeting_link // empty')
MSG="Meeting in 5 min: $TITLE"
[ -n "$LINK" ] && MSG="$MSG\n$LINK"
osascript -e "display notification \"$MSG\" with title \"Calvin\""
```

### Auto-open meeting links

```bash
#!/usr/bin/env bash
INPUT=$(cat /dev/stdin)
LINK=$(echo "$INPUT" | jq -r '.meeting_link // empty')
[ -n "$LINK" ] && open "$LINK"
```

### Slack status updater

```bash
#!/usr/bin/env bash
INPUT=$(cat /dev/stdin)
TITLE=$(echo "$INPUT" | jq -r '.title')
curl -s -X POST https://slack.com/api/users.profile.set \
  -H "Authorization: Bearer $SLACK_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"profile\":{\"status_text\":\"In: $TITLE\",\"status_emoji\":\":calendar:\"}}"
```

### Log meetings to a file

```bash
#!/usr/bin/env bash
EVENT=$(cat /dev/stdin)
TITLE=$(echo "$EVENT" | jq -r '.title')
START=$(echo "$EVENT" | jq -r '.start')
END=$(echo "$EVENT" | jq -r '.end')
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $TITLE ($START - $END)" >> ~/meeting-log.txt
```

### Focus mode (DND toggle)

```bash
#!/usr/bin/env bash
# before-event-start: enable DND
shortcuts run "Turn On Focus"
```

### Prepare meeting notes

```bash
#!/usr/bin/env bash
INPUT=$(cat /dev/stdin)
TITLE=$(echo "$INPUT" | jq -r '.title')
ATTENDEES=$(echo "$INPUT" | jq -r '.attendees[].name' | tr '\n' ', ')
DATE=$(echo "$INPUT" | jq -r '.start' | cut -dT -f1)
FILE=~/notes/$DATE-$(echo "$TITLE" | tr ' ' '-' | tr '[:upper:]' '[:lower:]').md
mkdir -p ~/notes
cat > "$FILE" << EOF
# $TITLE
Date: $DATE
Attendees: $ATTENDEES

## Agenda

## Notes

## Action Items
EOF
open "$FILE"
```

## Configuration

Config lives at `~/.config/calvin/config.toml`:

```toml
sync_interval_seconds = 60
pre_event_minutes = 5
hook_timeout_seconds = 30
max_concurrent_hooks = 10
hook_output_max_bytes = 65536
hook_execution_retention_days = 30
```

### Multiple calendars

By default Calvin watches your primary calendar. To watch additional calendars, add them to `config.toml`:

```toml
[[calendars]]
id = "primary"

[[calendars]]
id = "personal@gmail.com"
```

Each hook receives a `calendar` field in its JSON payload, and can filter by calendar without jq guards:

```bash
calvin match --calendar "primary" || exit 0
```

## File locations

Calvin follows the XDG Base Directory Specification:

| Purpose | Path |
|---------|------|
| Config  | `~/.config/calvin/` |
| Data    | `~/.local/share/calvin/` |
| State   | `~/.local/state/calvin/` |
| Hooks   | `~/.config/calvin/hooks/` |
| Logs    | `~/.local/state/calvin/calvin.log` |
| DB      | `~/.local/share/calvin/events.db` |

## FAQ

**Why not cron + gcalcli?**

Calvin fires hooks at sub-second precision on event boundaries — something cron's minute granularity can't match. It deduplicates across restarts (hooks won't re-fire after a crash), recovers from failures via the `hook_executions` table, and delivers structured event JSON on stdin so your scripts don't need to parse CLI output. The hook convention (drop a script in a directory, done) means zero per-hook configuration. And setup is simpler: Calvin ships its own OAuth credentials, so you don't need to create a Google Cloud project or configure your own API keys.

## Privacy

Calvin requests read-only access to your calendar. All data stays on your machine. Read the full [privacy policy](PRIVACY).

## License

[MIT](LICENSE)
