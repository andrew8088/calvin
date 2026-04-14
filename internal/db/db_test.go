package db

import (
	"context"
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

	token, err := d.GetSyncToken()
	if err != nil {
		t.Fatalf("GetSyncToken: %v", err)
	}
	if token != "" {
		t.Errorf("expected empty initial token, got %q", token)
	}

	d.SetSyncToken("abc123")
	token, _ = d.GetSyncToken()
	if token != "abc123" {
		t.Errorf("expected 'abc123', got %q", token)
	}

	d.SetSyncToken("def456")
	token, _ = d.GetSyncToken()
	if token != "def456" {
		t.Errorf("expected 'def456' after update, got %q", token)
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

func TestDeleteStaleSyncGeneration(t *testing.T) {
	d := openTestDB(t)

	d.UpsertEvent(testEvent("old"), 1)
	e2 := testEvent("new")
	e2.Start = time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	e2.End = time.Date(2026, 4, 15, 11, 0, 0, 0, time.UTC)
	d.UpsertEvent(e2, 2)

	deleted, err := d.DeleteStaleSyncGeneration(2)
	if err != nil {
		t.Fatalf("DeleteStaleSyncGeneration: %v", err)
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
}

func TestHookExecution(t *testing.T) {
	d := openTestDB(t)

	err := d.RecordHookExecution("evt-1", "notify", "pre_event", "success", "out", "err", 150)
	if err != nil {
		t.Fatalf("RecordHookExecution: %v", err)
	}

	executed, _ := d.HasHookExecuted("evt-1", "notify", "pre_event")
	if !executed {
		t.Error("expected hook to show as executed")
	}

	executed, _ = d.HasHookExecuted("evt-1", "notify", "event_start")
	if executed {
		t.Error("different hook type should not match")
	}

	executed, _ = d.HasHookExecuted("evt-2", "notify", "pre_event")
	if executed {
		t.Error("different event should not match")
	}
}

func TestHookExecution_FailedNotDeduped(t *testing.T) {
	d := openTestDB(t)

	d.RecordHookExecution("evt-1", "notify", "pre_event", "failed", "", "error", 50)

	executed, _ := d.HasHookExecuted("evt-1", "notify", "pre_event")
	if executed {
		t.Error("failed executions should not count as dedup")
	}
}

func TestGetHookExecutions(t *testing.T) {
	d := openTestDB(t)

	d.RecordHookExecution("evt-1", "notify", "pre_event", "success", "hello", "", 100)
	d.RecordHookExecution("evt-1", "slack", "event_start", "failed", "", "boom", 50)

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

	err := d.WithTransaction(context.Background(), func() error {
		return d.UpsertEvent(testEvent("txn-evt"), 1)
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

	err := d.WithTransaction(context.Background(), func() error {
		d.UpsertEvent(testEvent("rollback-evt"), 1)
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

func TestAttendeesHash_Consistent(t *testing.T) {
	a := []calendar.Attendee{{Email: "a@b.com", Name: "Alice", Response: "accepted"}}
	h1 := attendeesHash(a)
	h2 := attendeesHash(a)
	if h1 != h2 {
		t.Errorf("hash not deterministic: %q != %q", h1, h2)
	}
}
