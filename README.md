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
| `pre_event`   | N minutes before the event starts    |
| `event_start` | When the event starts                |
| `event_end`   | When the event ends                  |

Hooks receive event data as JSON on stdin:

```bash
#!/usr/bin/env bash
TITLE=$(jq -r '.title' < /dev/stdin)
LINK=$(jq -r '.meeting_link // empty' < /dev/stdin)
osascript -e "display notification \"$TITLE\" with title \"Calvin\""
```

Run `calvin hooks schema` to see all available JSON fields.

## Commands

```
calvin init          Scaffold config and example hooks
calvin auth          Authenticate with Google Calendar
calvin start         Start the daemon (--background for detached)
calvin stop          Stop the daemon
calvin status        Show daemon health dashboard
calvin next          Show next upcoming event with countdown
calvin events        List today's events
calvin events <id>   Show event detail with hook execution log
calvin hooks list    List all discovered hooks
calvin hooks new     Create a new hook from template
calvin hooks schema  Print the JSON input schema
calvin test <hook>   Test a hook with real or mock event data
calvin doctor        Run health checks
calvin logs          Show daemon logs (--hook, --level, --event filters)
calvin version       Print version
```

## Creating hooks

```bash
calvin hooks new pre_event my-notifier
```

This creates a hook at `~/.config/calvin/hooks/pre_event/my-notifier` with a starter template. Edit it, and Calvin picks it up on the next sync cycle (no restart needed).

Hooks must be executable (`chmod +x`). Calvin discovers hooks by scanning the hooks directory every sync cycle.

## Example hooks

### Desktop notification before meetings

```bash
#!/usr/bin/env bash
TITLE=$(jq -r '.title' < /dev/stdin)
LINK=$(jq -r '.meeting_link // empty' < /dev/stdin)
MSG="Meeting in 5 min: $TITLE"
[ -n "$LINK" ] && MSG="$MSG\n$LINK"
osascript -e "display notification \"$MSG\" with title \"Calvin\""
```

### Auto-open meeting links

```bash
#!/usr/bin/env bash
LINK=$(jq -r '.meeting_link // empty' < /dev/stdin)
[ -n "$LINK" ] && open "$LINK"
```

### Slack status updater

```bash
#!/usr/bin/env bash
TITLE=$(jq -r '.title' < /dev/stdin)
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
# pre_event: enable DND
shortcuts run "Turn On Focus"
```

### Prepare meeting notes

```bash
#!/usr/bin/env bash
TITLE=$(jq -r '.title' < /dev/stdin)
ATTENDEES=$(jq -r '.attendees[].name' < /dev/stdin | tr '\n' ', ')
DATE=$(jq -r '.start' < /dev/stdin | cut -dT -f1)
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

## License

MIT
