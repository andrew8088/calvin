package calendar

import (
	"testing"
	"time"
)

func TestFreeSlotsForWindow_EmptyDay(t *testing.T) {
	dayStart := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	slots := FreeSlotsForWindow(dayStart, dayEnd, nil)
	if len(slots) != 1 {
		t.Fatalf("expected 1 free slot, got %d", len(slots))
	}
	if !slots[0].Start.Equal(dayStart) {
		t.Fatalf("expected slot to start at %s, got %s", dayStart, slots[0].Start)
	}
	if !slots[0].End.Equal(dayEnd) {
		t.Fatalf("expected slot to end at %s, got %s", dayEnd, slots[0].End)
	}
	if slots[0].DurationSeconds != 24*60*60 {
		t.Fatalf("expected full-day duration, got %d", slots[0].DurationSeconds)
	}
}

func TestFreeSlotsForWindow_MergesOverlappingBusyEvents(t *testing.T) {
	dayStart := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	events := []Event{
		{ID: "carry", Start: dayStart.Add(-30 * time.Minute), End: dayStart.Add(30 * time.Minute)},
		{ID: "one", Start: dayStart.Add(9 * time.Hour), End: dayStart.Add(10 * time.Hour)},
		{ID: "two", Start: dayStart.Add(9*time.Hour + 30*time.Minute), End: dayStart.Add(11 * time.Hour)},
		{ID: "three", Start: dayStart.Add(11 * time.Hour), End: dayStart.Add(12 * time.Hour)},
	}

	slots := FreeSlotsForWindow(dayStart, dayEnd, events)
	if len(slots) != 2 {
		t.Fatalf("expected 2 free slots, got %d", len(slots))
	}
	assertSlot(t, slots[0], dayStart.Add(30*time.Minute), dayStart.Add(9*time.Hour))
	assertSlot(t, slots[1], dayStart.Add(12*time.Hour), dayEnd)
}

func TestFreeSlotsForWindow_AllDayBusy(t *testing.T) {
	dayStart := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	events := []Event{{ID: "all-day", AllDay: true, Start: dayStart, End: dayEnd}}

	slots := FreeSlotsForWindow(dayStart, dayEnd, events)
	if len(slots) != 0 {
		t.Fatalf("expected no free slots, got %d", len(slots))
	}
}

func assertSlot(t *testing.T, slot FreeSlot, wantStart, wantEnd time.Time) {
	t.Helper()
	if !slot.Start.Equal(wantStart) {
		t.Fatalf("slot start = %s, want %s", slot.Start, wantStart)
	}
	if !slot.End.Equal(wantEnd) {
		t.Fatalf("slot end = %s, want %s", slot.End, wantEnd)
	}
	if slot.DurationSeconds != int64(wantEnd.Sub(wantStart).Seconds()) {
		t.Fatalf("slot duration = %d, want %d", slot.DurationSeconds, int64(wantEnd.Sub(wantStart).Seconds()))
	}
}
