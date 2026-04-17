package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema <name>",
	Short: "Print machine-readable schemas",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSchema(args[0])
	},
}

func runSchema(name string) error {
	schema, err := namedSchema(name)
	if err != nil {
		return err
	}
	return printJSON(schema)
}

func namedSchema(name string) (any, error) {
	switch name {
	case "hook-payload":
		return hookPayloadSchema(), nil
	case "command-result":
		return commandResultSchema(), nil
	default:
		return nil, newExitError(2, "schema", "unknown_schema", fmt.Sprintf("unknown schema: %s", name), nil, nil)
	}
}

func hookPayloadSchema() map[string]any {
	return map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title":   "Calvin Hook Payload",
		"type":    "object",
		"properties": map[string]any{
			"schema_version":   map[string]any{"type": "integer"},
			"id":               map[string]any{"type": "string"},
			"title":            map[string]any{"type": "string"},
			"start":            map[string]any{"type": "string"},
			"end":              map[string]any{"type": "string"},
			"all_day":          map[string]any{"type": "boolean"},
			"location":         map[string]any{"type": "string"},
			"description":      map[string]any{"type": "string"},
			"meeting_link":     map[string]any{"type": []string{"string", "null"}},
			"meeting_provider": map[string]any{"type": "string"},
			"attendees": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"email":    map[string]any{"type": "string"},
						"name":     map[string]any{"type": "string"},
						"response": map[string]any{"type": "string"},
					},
				},
			},
			"organizer":      map[string]any{"type": "string"},
			"calendar":       map[string]any{"type": "string"},
			"status":         map[string]any{"type": "string"},
			"hook_type":      map[string]any{"type": "string"},
			"previous_event": adjacentEventSchema(),
			"next_event":     adjacentEventSchema(),
		},
		"required": []string{"schema_version", "id", "title", "start", "end", "all_day", "location", "description", "meeting_provider", "attendees", "organizer", "calendar", "status", "hook_type"},
	}
}

func adjacentEventSchema() map[string]any {
	return map[string]any{
		"type": []string{"object", "null"},
		"properties": map[string]any{
			"id":           map[string]any{"type": "string"},
			"title":        map[string]any{"type": "string"},
			"start":        map[string]any{"type": "string"},
			"end":          map[string]any{"type": "string"},
			"meeting_link": map[string]any{"type": "string"},
		},
	}
}

func commandResultSchema() map[string]any {
	return map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title":   "Calvin Command Result",
		"type":    "object",
		"properties": map[string]any{
			"ok":      map[string]any{"type": "boolean"},
			"command": map[string]any{"type": "string"},
			"data":    map[string]any{"type": []string{"object", "array", "string", "number", "boolean", "null"}},
			"warnings": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
		},
		"required": []string{"ok", "command"},
	}
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
