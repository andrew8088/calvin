package cli

import (
	"testing"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
)

func TestFormatFreeSlotsText(t *testing.T) {
	loc := time.FixedZone("PDT", -7*60*60)
	slots := []calendar.FreeSlot{{
		Start:           time.Date(2026, 4, 14, 16, 0, 0, 0, time.UTC),
		End:             time.Date(2026, 4, 14, 17, 30, 0, 0, time.UTC),
		DurationSeconds: 5400,
	}}

	lines := formatFreeSlotsText(slots, loc)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	want := "2026-04-14T09:00:00-07:00\t2026-04-14T10:30:00-07:00\t5400"
	if lines[0] != want {
		t.Fatalf("line = %q, want %q", lines[0], want)
	}
}

func TestFormatFreeSlotsText_Empty(t *testing.T) {
	lines := formatFreeSlotsText(nil, time.UTC)
	if len(lines) != 0 {
		t.Fatalf("expected no lines, got %d", len(lines))
	}
}
