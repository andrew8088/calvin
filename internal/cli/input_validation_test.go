package cli

import (
	"testing"

	"github.com/andrew8088/calvin/internal/logging"
)

func TestValidateSinceTimestampRejectsGarbage(t *testing.T) {
	if err := validateSinceTimestamp("tomorrow-ish"); err == nil {
		t.Fatal("expected invalid timestamp to be rejected")
	}
}

func TestLogsSinceFilterParsesTimeInsteadOfComparingStrings(t *testing.T) {
	entries := []logging.Entry{
		{Timestamp: "2026-04-17T10:00:00-07:00"},
		{Timestamp: "2026-04-17T18:00:00Z"},
	}
	filtered, err := filterLogsSince(entries, "2026-04-17T17:30:00Z")
	if err != nil {
		t.Fatalf("filterLogsSince: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("filtered = %d, want 1", len(filtered))
	}
}
