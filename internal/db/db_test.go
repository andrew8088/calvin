package db

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(":memory:", false)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func testEvent(id string) calendar.Event {
	return calendar.Event{
		ID:              id,
		Title:           "Test Event " + id,
		Start:           time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
		End:             time.Date(2026, 4, 14, 11, 0, 0, 0, time.UTC),
		Location:        "Room A",
		Description:     "A test event",
		MeetingLink:     "https://meet.google.com/abc",
		MeetingProvider: "google_meet",
		Organizer:       "boss@co.com",
		Calendar:        "primary",
		Status:          "confirmed",
		Attendees: []calendar.Attendee{
			{Email: "alice@co.com", Name: "Alice", Response: "accepted"},
		},
	}
}

func TestUpsertAndGet(t *testing.T) {
	d := openTestDB(t)
	e := testEvent("evt-1")

	if err := d.UpsertEvent(e, 1); err != nil {
		t.Fatalf("UpsertEvent: %v", err)
	}

	got, err := d.GetEvent("evt-1")
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if got == nil {
		t.Fatal("expected event, got nil")
	}
	if got.Title != "Test Event evt-1" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.MeetingLink != "https://meet.google.com/abc" {
		t.Errorf("MeetingLink = %q", got.MeetingLink)
	}
	if got.Organizer != "boss@co.com" {
		t.Errorf("Organizer = %q", got.Organizer)
	}
	if len(got.Attendees) != 1 {
		t.Fatalf("expected 1 attendee, got %d", len(got.Attendees))
	}
	if got.Attendees[0].Email != "alice@co.com" {
		t.Errorf("attendee email = %q", got.Attendees[0].Email)
	}
}

func TestUpsertUpdate(t *testing.T) {
	d := openTestDB(t)
	e := testEvent("evt-1")
	d.UpsertEvent(e, 1)

	e.Title = "Updated Title"
	d.UpsertEvent(e, 2)

	got, _ := d.GetEvent("evt-1")
	if got.Title != "Updated Title" {
		t.Errorf("expected updated title, got %q", got.Title)
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	d := openTestDB(t)
	got, err := d.GetEvent("nonexistent")
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent event")
	}
}

func TestListEventsForDay(t *testing.T) {
	d := openTestDB(t)

	today := testEvent("today")
	today.Start = time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	today.End = time.Date(2026, 4, 14, 11, 0, 0, 0, time.UTC)
	d.UpsertEvent(today, 1)

	tomorrow := testEvent("tomorrow")
	tomorrow.Start = time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	tomorrow.End = time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC)
	d.UpsertEvent(tomorrow, 1)

	events, err := d.ListEventsForDay(time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ListEventsForDay: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != "today" {
		t.Errorf("expected 'today', got %q", events[0].ID)
	}
}

func TestListEventsForDay_ExcludesCancelled(t *testing.T) {
	d := openTestDB(t)

	e := testEvent("cancelled")
	e.Status = "cancelled"
	d.UpsertEvent(e, 1)

	events, _ := d.ListEventsForDay(time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC))
	if len(events) != 0 {
		t.Errorf("expected 0 events (cancelled excluded), got %d", len(events))
	}
}

func TestListEventsBetween(t *testing.T) {
	d := openTestDB(t)

	// Insert events across several days
	for i := 0; i < 10; i++ {
		e := testEvent(fmt.Sprintf("evt-%d", i))
		e.Start = time.Date(2026, 4, 14+i, 10, 0, 0, 0, time.UTC)
		e.End = e.Start.Add(time.Hour)
		d.UpsertEvent(e, 1)
	}

	start := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC) // 7-day range

	events, err := d.ListEventsBetween(start, end)
	if err != nil {
		t.Fatalf("ListEventsBetween: %v", err)
	}
	if len(events) != 7 {
		t.Fatalf("expected 7 events in 7-day range, got %d", len(events))
	}

	// Verify chronological order
	for i := 1; i < len(events); i++ {
		if events[i].Start.Before(events[i-1].Start) {
			t.Errorf("events not in chronological order at index %d", i)
		}
	}
}

func TestListEventsBetween_ExcludesCancelled(t *testing.T) {
	d := openTestDB(t)

	e := testEvent("active")
	e.Start = time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	e.End = time.Date(2026, 4, 14, 11, 0, 0, 0, time.UTC)
	d.UpsertEvent(e, 1)

	cancelled := testEvent("cancelled")
	cancelled.Start = time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	cancelled.End = time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC)
	cancelled.Status = "cancelled"
	d.UpsertEvent(cancelled, 1)

	start := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)

	events, err := d.ListEventsBetween(start, end)
	if err != nil {
		t.Fatalf("ListEventsBetween: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event (cancelled excluded), got %d", len(events))
	}
	if events[0].ID != "active" {
		t.Errorf("expected 'active', got %q", events[0].ID)
	}
}

func TestListEventsBetween_Empty(t *testing.T) {
	d := openTestDB(t)

	start := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)

	events, err := d.ListEventsBetween(start, end)
	if err != nil {
		t.Fatalf("ListEventsBetween: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestListEventsOverlapping_IncludesCarryoverAndOrdersByStart(t *testing.T) {
	d := openTestDB(t)

	windowStart := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	windowEnd := windowStart.Add(24 * time.Hour)

	carryover := testEvent("carryover")
	carryover.Start = windowStart.Add(-30 * time.Minute)
	carryover.End = windowStart.Add(30 * time.Minute)
	d.UpsertEvent(carryover, 1)

	morning := testEvent("morning")
	morning.Start = windowStart.Add(9 * time.Hour)
	morning.End = morning.Start.Add(time.Hour)
	d.UpsertEvent(morning, 1)

	endsAtWindowStart := testEvent("ends-at-start")
	endsAtWindowStart.Start = windowStart.Add(-2 * time.Hour)
	endsAtWindowStart.End = windowStart
	d.UpsertEvent(endsAtWindowStart, 1)

	startsAtWindowEnd := testEvent("starts-at-end")
	startsAtWindowEnd.Start = windowEnd
	startsAtWindowEnd.End = windowEnd.Add(time.Hour)
	d.UpsertEvent(startsAtWindowEnd, 1)

	events, err := d.ListEventsOverlapping(windowStart, windowEnd)
	if err != nil {
		t.Fatalf("ListEventsOverlapping: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 overlapping events, got %d", len(events))
	}
	if events[0].ID != "carryover" {
		t.Fatalf("expected first event to be carryover, got %q", events[0].ID)
	}
	if events[1].ID != "morning" {
		t.Fatalf("expected second event to be morning, got %q", events[1].ID)
	}
}

func TestListEventsOverlapping_ExcludesCancelled(t *testing.T) {
	d := openTestDB(t)

	windowStart := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	windowEnd := windowStart.Add(24 * time.Hour)

	active := testEvent("active")
	active.Start = windowStart.Add(10 * time.Hour)
	active.End = active.Start.Add(time.Hour)
	d.UpsertEvent(active, 1)

	cancelled := testEvent("cancelled")
	cancelled.Start = windowStart.Add(11 * time.Hour)
	cancelled.End = cancelled.Start.Add(time.Hour)
	cancelled.Status = "cancelled"
	d.UpsertEvent(cancelled, 1)

	events, err := d.ListEventsOverlapping(windowStart, windowEnd)
	if err != nil {
		t.Fatalf("ListEventsOverlapping: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 overlapping event, got %d", len(events))
	}
	if events[0].ID != "active" {
		t.Fatalf("expected active event, got %q", events[0].ID)
	}
}

func TestListEventsOverlapping_IncludesAllDay(t *testing.T) {
	d := openTestDB(t)

	windowStart := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	windowEnd := windowStart.Add(24 * time.Hour)

	allDay := testEvent("all-day")
	allDay.AllDay = true
	allDay.Start = windowStart
	allDay.End = windowEnd
	d.UpsertEvent(allDay, 1)

	events, err := d.ListEventsOverlapping(windowStart, windowEnd)
	if err != nil {
		t.Fatalf("ListEventsOverlapping: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 overlapping event, got %d", len(events))
	}
	if !events[0].AllDay {
		t.Fatal("expected all-day event to be preserved")
	}
}

func TestListUpcomingEvents(t *testing.T) {
	d := openTestDB(t)

	for i := 0; i < 5; i++ {
		e := testEvent("evt-" + string(rune('A'+i)))
		e.Start = time.Date(2026, 4, 14+i, 10, 0, 0, 0, time.UTC)
		e.End = e.Start.Add(time.Hour)
		d.UpsertEvent(e, 1)
	}

	events, err := d.ListUpcomingEvents(time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC), 3)
	if err != nil {
		t.Fatalf("ListUpcomingEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestSyncToken(t *testing.T) {
	d := openTestDB(t)

	token, err := d.GetSyncToken("primary")
	if err != nil {
		t.Fatalf("GetSyncToken: %v", err)
	}
	if token != "" {
		t.Errorf("expected empty initial token, got %q", token)
	}

	d.SetSyncToken("primary", "abc123")
	token, _ = d.GetSyncToken("primary")
	if token != "abc123" {
		t.Errorf("expected 'abc123', got %q", token)
	}

	d.SetSyncToken("primary", "def456")
	token, _ = d.GetSyncToken("primary")
	if token != "def456" {
		t.Errorf("expected 'def456' after update, got %q", token)
	}
}

func TestSyncToken_PerCalendar(t *testing.T) {
	d := openTestDB(t)

	d.SetSyncToken("primary", "token-primary")
	d.SetSyncToken("work@company.com", "token-work")

	tok1, _ := d.GetSyncToken("primary")
	tok2, _ := d.GetSyncToken("work@company.com")

	if tok1 != "token-primary" {
		t.Errorf("primary token = %q, want token-primary", tok1)
	}
	if tok2 != "token-work" {
		t.Errorf("work token = %q, want token-work", tok2)
	}
}

func TestSyncGeneration(t *testing.T) {
	d := openTestDB(t)

	gen, _ := d.GetSyncGeneration()
	if gen != 0 {
		t.Errorf("expected initial gen 0, got %d", gen)
	}

	d.UpsertEvent(testEvent("evt-1"), 5)
	gen, _ = d.GetSyncGeneration()
	if gen != 5 {
		t.Errorf("expected gen 5, got %d", gen)
	}
}

func TestDeleteStaleSyncGenerationForCalendar(t *testing.T) {
	d := openTestDB(t)

	old := testEvent("old")
	old.Calendar = "primary"
	d.UpsertEvent(old, 1)

	e2 := testEvent("new")
	e2.Start = time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	e2.End = time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC)
	e2.Calendar = "primary"
	d.UpsertEvent(e2, 2)

	other := testEvent("other-calendar")
	other.Calendar = "work@company.com"
	d.UpsertEvent(other, 1)

	deleted, err := d.DeleteStaleSyncGenerationForCalendar("primary", 2)
	if err != nil {
		t.Fatalf("DeleteStaleSyncGenerationForCalendar: %v", err)
	}
	if len(deleted) != 1 || deleted[0] != "old" {
		t.Errorf("expected [old] deleted, got %v", deleted)
	}

	got, _ := d.GetEvent("old")
	if got != nil {
		t.Error("stale event should be deleted")
	}
	got, _ = d.GetEvent("new")
	if got == nil {
		t.Error("current gen event should remain")
	}
	got, _ = d.GetEvent("other-calendar")
	if got == nil {
		t.Error("event from another calendar should remain")
	}
}

func TestHookExecution(t *testing.T) {
	d := openTestDB(t)

	err := d.RecordHookExecution("evt-1", "notify", "before-event-start", "success", "out", "err", 150)
	if err != nil {
		t.Fatalf("RecordHookExecution: %v", err)
	}

	executed, _ := d.HasHookExecuted("evt-1", "notify", "before-event-start")
	if !executed {
		t.Error("expected hook to show as executed")
	}

	executed, _ = d.HasHookExecuted("evt-1", "notify", "on-event-start")
	if executed {
		t.Error("different hook type should not match")
	}

	executed, _ = d.HasHookExecuted("evt-2", "notify", "before-event-start")
	if executed {
		t.Error("different event should not match")
	}
}

func TestHookExecution_FailedNotDeduped(t *testing.T) {
	d := openTestDB(t)

	d.RecordHookExecution("evt-1", "notify", "before-event-start", "failed", "", "error", 50)

	executed, _ := d.HasHookExecuted("evt-1", "notify", "before-event-start")
	if executed {
		t.Error("failed executions should not count as dedup")
	}
}

func TestGetHookExecutions(t *testing.T) {
	d := openTestDB(t)

	d.RecordHookExecution("evt-1", "notify", "before-event-start", "success", "hello", "", 100)
	d.RecordHookExecution("evt-1", "slack", "on-event-start", "failed", "", "boom", 50)

	execs, err := d.GetHookExecutions("evt-1")
	if err != nil {
		t.Fatalf("GetHookExecutions: %v", err)
	}
	if len(execs) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(execs))
	}
}

func TestIntegrityCheck(t *testing.T) {
	d := openTestDB(t)
	if err := d.IntegrityCheck(); err != nil {
		t.Errorf("IntegrityCheck: %v", err)
	}
}

func TestWithTransaction(t *testing.T) {
	d := openTestDB(t)

	err := d.WithTransaction(context.Background(), func(tx *Tx) error {
		return tx.UpsertEvent(testEvent("txn-evt"), 1)
	})
	if err != nil {
		t.Fatalf("WithTransaction: %v", err)
	}

	got, _ := d.GetEvent("txn-evt")
	if got == nil {
		t.Error("event should exist after committed transaction")
	}
}

func TestWithTransaction_Rollback(t *testing.T) {
	d := openTestDB(t)

	err := d.WithTransaction(context.Background(), func(tx *Tx) error {
		tx.UpsertEvent(testEvent("rollback-evt"), 1)
		return context.Canceled
	})
	if err == nil {
		t.Error("expected error from rolled-back transaction")
	}

	got, _ := d.GetEvent("rollback-evt")
	if got != nil {
		t.Error("event should not exist after rollback")
	}
}

func TestAttendeesJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    []calendar.Attendee
		expected string
	}{
		{"empty", nil, "[]"},
		{"single", []calendar.Attendee{
			{Email: "a@b.com", Name: "Alice", Response: "accepted"},
		}, `[{"email":"a@b.com","name":"Alice","response":"accepted"}]`},
		{"multiple", []calendar.Attendee{
			{Email: "a@b.com", Name: "Alice", Response: "accepted"},
			{Email: "c@d.com", Name: "Bob", Response: "declined"},
		}, `[{"email":"a@b.com","name":"Alice","response":"accepted"},{"email":"c@d.com","name":"Bob","response":"declined"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := attendeesJSON(tt.input)
			if got != tt.expected {
				t.Errorf("attendeesJSON() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseAttendees(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"empty array", "[]", 0},
		{"single", `[{"email":"a@b.com","name":"Alice","response":"accepted"}]`, 1},
		{"multiple", `[{"email":"a@b.com","name":"Alice","response":"accepted"},{"email":"c@d.com","name":"Bob","response":"declined"}]`, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAttendees(tt.input)
			if len(got) != tt.want {
				t.Errorf("parseAttendees() returned %d attendees, want %d", len(got), tt.want)
			}
		})
	}
}

func TestParseAttendees_RoundTrip(t *testing.T) {
	original := []calendar.Attendee{
		{Email: "alice@co.com", Name: "Alice", Response: "accepted"},
		{Email: "bob@co.com", Name: "Bob", Response: "needsAction"},
	}

	jsonStr := attendeesJSON(original)
	parsed := parseAttendees(jsonStr)

	if len(parsed) != len(original) {
		t.Fatalf("round-trip: got %d attendees, want %d", len(parsed), len(original))
	}
	for i, a := range parsed {
		if a.Email != original[i].Email {
			t.Errorf("[%d] Email = %q, want %q", i, a.Email, original[i].Email)
		}
		if a.Name != original[i].Name {
			t.Errorf("[%d] Name = %q, want %q", i, a.Name, original[i].Name)
		}
		if a.Response != original[i].Response {
			t.Errorf("[%d] Response = %q, want %q", i, a.Response, original[i].Response)
		}
	}
}

func TestGetAdjacentEvents(t *testing.T) {
	d := openTestDB(t)

	e1 := testEvent("evt-1")
	e1.Title = "Morning Standup"
	e1.Start = time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC)
	e1.End = time.Date(2026, 4, 14, 9, 30, 0, 0, time.UTC)
	d.UpsertEvent(e1, 1)

	e2 := testEvent("evt-2")
	e2.Title = "Design Review"
	e2.Start = time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	e2.End = time.Date(2026, 4, 14, 11, 0, 0, 0, time.UTC)
	d.UpsertEvent(e2, 1)

	e3 := testEvent("evt-3")
	e3.Title = "Lunch"
	e3.Start = time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	e3.End = time.Date(2026, 4, 14, 13, 0, 0, 0, time.UTC)
	d.UpsertEvent(e3, 1)

	prev, next, err := d.GetAdjacentEvents("evt-2", e2.Start, e2.End)
	if err != nil {
		t.Fatalf("GetAdjacentEvents: %v", err)
	}
	if prev == nil {
		t.Fatal("expected previous event, got nil")
	}
	if prev.ID != "evt-1" {
		t.Errorf("previous event = %q, want evt-1", prev.ID)
	}
	if next == nil {
		t.Fatal("expected next event, got nil")
	}
	if next.ID != "evt-3" {
		t.Errorf("next event = %q, want evt-3", next.ID)
	}
}

func TestGetAdjacentEvents_NoPrev(t *testing.T) {
	d := openTestDB(t)

	e := testEvent("only")
	e.Start = time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	e.End = time.Date(2026, 4, 14, 11, 0, 0, 0, time.UTC)
	d.UpsertEvent(e, 1)

	prev, next, err := d.GetAdjacentEvents("only", e.Start, e.End)
	if err != nil {
		t.Fatalf("GetAdjacentEvents: %v", err)
	}
	if prev != nil {
		t.Errorf("expected nil prev, got %q", prev.ID)
	}
	if next != nil {
		t.Errorf("expected nil next, got %q", next.ID)
	}
}

func TestGetAdjacentEvents_SkipsCancelled(t *testing.T) {
	d := openTestDB(t)

	cancelled := testEvent("cancelled")
	cancelled.Start = time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC)
	cancelled.End = time.Date(2026, 4, 14, 9, 30, 0, 0, time.UTC)
	cancelled.Status = "cancelled"
	d.UpsertEvent(cancelled, 1)

	e := testEvent("current")
	e.Start = time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	e.End = time.Date(2026, 4, 14, 11, 0, 0, 0, time.UTC)
	d.UpsertEvent(e, 1)

	prev, _, err := d.GetAdjacentEvents("current", e.Start, e.End)
	if err != nil {
		t.Fatalf("GetAdjacentEvents: %v", err)
	}
	if prev != nil {
		t.Errorf("expected nil prev (cancelled should be skipped), got %q", prev.ID)
	}
}

func TestConcurrentAccessDoesNotCrash(t *testing.T) {
	d := openTestDB(t)

	e1 := testEvent("evt-1")
	e1.Start = time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC)
	e1.End = time.Date(2026, 4, 14, 9, 30, 0, 0, time.UTC)
	if err := d.UpsertEvent(e1, 1); err != nil {
		t.Fatalf("UpsertEvent evt-1: %v", err)
	}

	e2 := testEvent("evt-2")
	e2.Start = time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	e2.End = time.Date(2026, 4, 14, 11, 0, 0, 0, time.UTC)
	if err := d.UpsertEvent(e2, 1); err != nil {
		t.Fatalf("UpsertEvent evt-2: %v", err)
	}

	e3 := testEvent("evt-3")
	e3.Start = time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	e3.End = time.Date(2026, 4, 14, 13, 0, 0, 0, time.UTC)
	if err := d.UpsertEvent(e3, 1); err != nil {
		t.Fatalf("UpsertEvent evt-3: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 16)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if _, _, err := d.GetAdjacentEvents("evt-2", e2.Start, e2.End); err != nil {
					errCh <- fmt.Errorf("GetAdjacentEvents: %w", err)
					return
				}
				if _, err := d.ListUpcomingEvents(e1.Start.Add(-time.Hour), 10); err != nil {
					errCh <- fmt.Errorf("ListUpcomingEvents: %w", err)
					return
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent database access timed out")
	}

	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestAttendeesHash_Consistent(t *testing.T) {
	a := []calendar.Attendee{{Email: "a@b.com", Name: "Alice", Response: "accepted"}}
	h1 := attendeesHash(a)
	h2 := attendeesHash(a)
	if h1 != h2 {
		t.Errorf("hash not deterministic: %q != %q", h1, h2)
	}
}
