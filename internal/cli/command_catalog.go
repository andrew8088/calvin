package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type flagMetadata struct {
	Name  string `json:"name"`
	Scope string `json:"scope,omitempty"`
	Type  string `json:"type"`
}

type commandMetadata struct {
	Path         string         `json:"path"`
	Summary      string         `json:"summary"`
	Args         []string       `json:"args,omitempty"`
	Flags        []flagMetadata `json:"flags,omitempty"`
	EnvVars      []string       `json:"env_vars,omitempty"`
	ExitCodes    map[int]string `json:"exit_codes,omitempty"`
	OutputModes  []string       `json:"output_modes,omitempty"`
	MutatesState bool           `json:"mutates_state"`
	SchemaRefs   []string       `json:"schema_refs,omitempty"`
	Examples     []string       `json:"examples,omitempty"`
}

type commandAnnotations struct {
	Args         []string
	EnvVars      []string
	ExitCodes    map[int]string
	OutputModes  []string
	MutatesState bool
	SchemaRefs   []string
}

var commandCatalogAnnotations = map[string]commandAnnotations{
	"commands":   {OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"describe":   {Args: []string{"command-path"}, OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 2: "unknown command path"}},
	"schema":     {Args: []string{"schema-name"}, OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 2: "unknown schema"}, SchemaRefs: []string{"hook-payload", "command-result"}},
	"auth":       {OutputModes: []string{"text", "json"}, MutatesState: true, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"doctor":     {OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"events":     {Args: []string{"event-id"}, OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"free":       {OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"hooks list": {OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}, SchemaRefs: []string{"command-result"}},
	"hooks new": {
		Args:         []string{"type", "name"},
		OutputModes:  []string{"text", "json"},
		MutatesState: true,
		ExitCodes:    map[int]string{0: "success", 1: "failure", 2: "usage error"},
		SchemaRefs:   []string{"command-result"},
	},
	"hooks schema": {OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}, SchemaRefs: []string{"hook-payload"}},
	"ignore": {
		OutputModes: []string{"text", "json"},
		EnvVars:     []string{"CALVIN_EVENT_FILE"},
		ExitCodes:   map[int]string{0: "matched", 1: "not matched", 2: "usage or context error"},
	},
	"init":       {OutputModes: []string{"text", "json"}, MutatesState: true, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"logs":       {OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"match":      {OutputModes: []string{"text", "json"}, EnvVars: []string{"CALVIN_EVENT_FILE"}, ExitCodes: map[int]string{0: "matched", 1: "not matched", 2: "usage or context error"}},
	"next":       {OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"start":      {OutputModes: []string{"text", "json"}, MutatesState: true, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"status":     {OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"stop":       {OutputModes: []string{"text", "json"}, MutatesState: true, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"sync":       {OutputModes: []string{"text", "json"}, MutatesState: true, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"test":       {Args: []string{"hook-name"}, OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"version":    {OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success"}},
	"week":       {OutputModes: []string{"text", "json"}, ExitCodes: map[int]string{0: "success", 1: "failure"}},
	"completion": {Args: []string{"shell"}, OutputModes: []string{"text"}, ExitCodes: map[int]string{0: "success", 1: "failure"}},
}

func buildCommandCatalog(root *cobra.Command) map[string]commandMetadata {
	entries := map[string]commandMetadata{}
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			if child.Hidden || child.Name() == "help" {
				continue
			}
			path := strings.TrimPrefix(child.CommandPath(), root.Name()+" ")
			annotations := commandCatalogAnnotations[path]
			entries[path] = commandMetadata{
				Path:         path,
				Summary:      child.Short,
				Args:         append([]string(nil), annotations.Args...),
				Flags:        collectFlagMetadata(child),
				EnvVars:      append([]string(nil), annotations.EnvVars...),
				ExitCodes:    copyExitCodes(annotations.ExitCodes),
				OutputModes:  append([]string(nil), annotations.OutputModes...),
				MutatesState: annotations.MutatesState,
				SchemaRefs:   append([]string(nil), annotations.SchemaRefs...),
				Examples:     splitExamples(child.Example),
			}
			walk(child)
		}
	}
	walk(root)
	return entries
}

func collectFlagMetadata(cmd *cobra.Command) []flagMetadata {
	seen := map[string]bool{}
	flags := make([]flagMetadata, 0)
	cmd.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
		if seen[flag.Name] {
			return
		}
		seen[flag.Name] = true
		flags = append(flags, flagMetadata{Name: flag.Name, Scope: "inherited", Type: flag.Value.Type()})
	})
	cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		if seen[flag.Name] {
			return
		}
		seen[flag.Name] = true
		flags = append(flags, flagMetadata{Name: flag.Name, Scope: "local", Type: flag.Value.Type()})
	})
	sort.Slice(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	return flags
}

func copyExitCodes(in map[int]string) map[int]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[int]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func splitExamples(example string) []string {
	if example == "" {
		return nil
	}
	parts := strings.Split(example, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
