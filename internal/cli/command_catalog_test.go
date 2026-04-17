package cli

import "testing"

func TestCommandCatalogCoversNestedCommands(t *testing.T) {
	entries := buildCommandCatalog(rootCmd)
	for _, path := range []string{"hooks list", "hooks new", "hooks schema", "match", "ignore"} {
		if _, ok := entries[path]; !ok {
			t.Fatalf("missing command metadata for %q", path)
		}
	}
}

func TestCommandCatalogIncludesAIAnnotations(t *testing.T) {
	entries := buildCommandCatalog(rootCmd)
	match := entries["match"]
	if match.ExitCodes[1] == "" {
		t.Fatal("match exit code annotations missing")
	}
	hooksNew := entries["hooks new"]
	if !hooksNew.MutatesState {
		t.Fatal("hooks new mutates_state should be true")
	}
	if len(hooksNew.Args) != 2 || hooksNew.Args[0] != "type" || hooksNew.Args[1] != "name" {
		t.Fatalf("hooks new args = %#v", hooksNew.Args)
	}
	if !hasCatalogFlag(hooksNew.Flags, "json") || !hasCatalogFlag(hooksNew.Flags, "output") {
		t.Fatalf("hooks new flags = %#v", hooksNew.Flags)
	}
}

func hasCatalogFlag(flags []flagMetadata, name string) bool {
	for _, flag := range flags {
		if flag.Name == name {
			return true
		}
	}
	return false
}
