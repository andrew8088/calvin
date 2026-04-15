package cli

import (
	"testing"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
)

func TestGroupByDay(t *testing.T) {
	loc := time.UTC
	events := []calendar.Event{
		{ID: "e1", Title: "Morning", Start: time.Date(2026, 4, 14, 9, 0, 0, 0, loc)},
		{ID: "e2", Title: "Afternoon", Start: time.Date(2026, 4, 14, 14, 0, 0, 0, loc)},
		{ID: "e3", Title: "Tomorrow", Start: time.Date(2026, 4, 15, 10, 0, 0, 0, loc)},
		{ID: "e4", Title: "Next Wed", Start: time.Date(2026, 4, 16, 11, 0, 0, 0, loc)},
	}

	grouped := groupByDay(events, loc)

	if len(grouped["2026-04-14"]) != 2 {
		t.Errorf("expected 2 events on Apr 14, got %d", len(grouped["2026-04-14"]))
	}
	if len(grouped["2026-04-15"]) != 1 {
		t.Errorf("expected 1 event on Apr 15, got %d", len(grouped["2026-04-15"]))
	}
	if len(grouped["2026-04-16"]) != 1 {
		t.Errorf("expected 1 event on Apr 16, got %d", len(grouped["2026-04-16"]))
	}
	if len(grouped["2026-04-17"]) != 0 {
		t.Errorf("expected 0 events on Apr 17, got %d", len(grouped["2026-04-17"]))
	}
}

func TestGroupByDay_Empty(t *testing.T) {
	grouped := groupByDay(nil, time.UTC)
	if len(grouped) != 0 {
		t.Errorf("expected empty map, got %d keys", len(grouped))
	}
}

func TestGroupByDay_LocalTimezone(t *testing.T) {
	// An event at 23:00 UTC should appear on the next day in UTC+2
	utc := time.UTC
	loc := time.FixedZone("UTC+2", 2*60*60)

	events := []calendar.Event{
		{ID: "late", Title: "Late Night", Start: time.Date(2026, 4, 14, 23, 0, 0, 0, utc)},
	}

	groupedUTC := groupByDay(events, utc)
	groupedLocal := groupByDay(events, loc)

	if len(groupedUTC["2026-04-14"]) != 1 {
		t.Errorf("UTC: expected 1 event on Apr 14, got %d", len(groupedUTC["2026-04-14"]))
	}
	if len(groupedLocal["2026-04-15"]) != 1 {
		t.Errorf("UTC+2: expected 1 event on Apr 15, got %d", len(groupedLocal["2026-04-15"]))
	}
}
