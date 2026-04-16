package calendar

import (
	"testing"
	"time"
)

func TestDiff_NewEvent(t *testing.T) {
	db := []Event{}
	api := []Event{{ID: "1", Title: "Standup", Status: "confirmed"}}

	results := Diff(db, api, 1)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != DiffAdded {
		t.Errorf("expected DiffAdded, got %d", results[0].Type)
	}
	if results[0].Event.ID != "1" {
		t.Errorf("expected event ID '1', got %q", results[0].Event.ID)
	}
}

func TestDiff_DeletedEvent(t *testing.T) {
	db := []Event{{ID: "1", Title: "Standup", Status: "confirmed"}}
	api := []Event{{ID: "1", Title: "Standup", Status: "cancelled"}}

	results := Diff(db, api, 1)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != DiffDeleted {
		t.Errorf("expected DiffDeleted, got %d", results[0].Type)
	}
}

func TestDiff_CancelledEventNotInDB(t *testing.T) {
	db := []Event{}
	api := []Event{{ID: "1", Status: "cancelled"}}

	results := Diff(db, api, 1)

	if len(results) != 0 {
		t.Fatalf("expected 0 results for cancelled event not in DB, got %d", len(results))
	}
}

func TestDiff_ModifiedTitle(t *testing.T) {
	now := time.Now()
	db := []Event{{ID: "1", Title: "Standup", Start: now, End: now.Add(time.Hour), Status: "confirmed"}}
	api := []Event{{ID: "1", Title: "Standup v2", Start: now, End: now.Add(time.Hour), Status: "confirmed"}}

	results := Diff(db, api, 1)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != DiffModified {
		t.Errorf("expected DiffModified, got %d", results[0].Type)
	}
}

func TestDiff_ModifiedTime(t *testing.T) {
	now := time.Now()
	db := []Event{{ID: "1", Title: "Standup", Start: now, End: now.Add(time.Hour), Status: "confirmed"}}
	api := []Event{{ID: "1", Title: "Standup", Start: now.Add(30 * time.Minute), End: now.Add(time.Hour), Status: "confirmed"}}

	results := Diff(db, api, 1)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != DiffModified {
		t.Errorf("expected DiffModified, got %d", results[0].Type)
	}
}

func TestDiff_ModifiedAttendees(t *testing.T) {
	now := time.Now()
	db := []Event{{
		ID: "1", Title: "Standup", Start: now, End: now.Add(time.Hour), Status: "confirmed",
		Attendees: []Attendee{{Email: "a@b.com", Name: "Alice", Response: "accepted"}},
	}}
	api := []Event{{
		ID: "1", Title: "Standup", Start: now, End: now.Add(time.Hour), Status: "confirmed",
		Attendees: []Attendee{
			{Email: "a@b.com", Name: "Alice", Response: "accepted"},
			{Email: "c@d.com", Name: "Bob", Response: "needsAction"},
		},
	}}

	results := Diff(db, api, 1)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != DiffModified {
		t.Errorf("expected DiffModified, got %d", results[0].Type)
	}
}

func TestDiff_NoChange(t *testing.T) {
	now := time.Now()
	attendees := []Attendee{{Email: "a@b.com", Name: "Alice", Response: "accepted"}}
	db := []Event{{ID: "1", Title: "Standup", Start: now, End: now.Add(time.Hour), Status: "confirmed", Attendees: attendees}}
	api := []Event{{ID: "1", Title: "Standup", Start: now, End: now.Add(time.Hour), Status: "confirmed", Attendees: attendees}}

	results := Diff(db, api, 1)

	if len(results) != 0 {
		t.Fatalf("expected 0 results for unchanged event, got %d", len(results))
	}
}

func TestDiff_MultipleChanges(t *testing.T) {
	now := time.Now()
	db := []Event{
		{ID: "1", Title: "Keep", Start: now, End: now.Add(time.Hour), Status: "confirmed"},
		{ID: "2", Title: "Delete", Start: now, End: now.Add(time.Hour), Status: "confirmed"},
	}
	api := []Event{
		{ID: "1", Title: "Keep", Start: now, End: now.Add(time.Hour), Status: "confirmed"},
		{ID: "2", Status: "cancelled"},
		{ID: "3", Title: "New", Status: "confirmed"},
	}

	results := Diff(db, api, 1)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	types := map[DiffType]bool{}
	for _, r := range results {
		types[r.Type] = true
	}
	if !types[DiffDeleted] {
		t.Error("expected a DiffDeleted result")
	}
	if !types[DiffAdded] {
		t.Error("expected a DiffAdded result")
	}
}

func TestHashAttendees_Deterministic(t *testing.T) {
	a := []Attendee{{Email: "a@b.com", Name: "Alice", Response: "accepted"}}
	h1 := hashAttendees(a)
	h2 := hashAttendees(a)

	if h1 != h2 {
		t.Errorf("hash not deterministic: %q != %q", h1, h2)
	}
	if len(h1) != 16 {
		t.Errorf("expected 16 char hash, got %d", len(h1))
	}
}

func TestHashAttendees_DifferentOrder(t *testing.T) {
	a := []Attendee{
		{Email: "a@b.com", Name: "Alice", Response: "accepted"},
		{Email: "c@d.com", Name: "Bob", Response: "declined"},
	}
	b := []Attendee{
		{Email: "c@d.com", Name: "Bob", Response: "declined"},
		{Email: "a@b.com", Name: "Alice", Response: "accepted"},
	}

	if hashAttendees(a) == hashAttendees(b) {
		t.Error("different attendee order should produce different hashes")
	}
}

func TestHashAttendees_Empty(t *testing.T) {
	h := hashAttendees(nil)
	if h == "" {
		t.Error("hash of empty attendees should not be empty")
	}
}

func TestEventToPayload_Basic(t *testing.T) {
	now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	e := Event{
		ID:              "evt-1",
		Title:           "Standup",
		Start:           now,
		End:             now.Add(30 * time.Minute),
		Location:        "Room A",
		MeetingLink:     "https://meet.google.com/abc",
		MeetingProvider: "google_meet",
		Organizer:       "boss@co.com",
		Calendar:        "primary",
		Status:          "confirmed",
		Attendees:       []Attendee{{Email: "a@b.com", Name: "Alice", Response: "accepted"}},
	}

	p := EventToPayload(e, "on-event-start", nil, nil)

	if p.SchemaVersion != 1 {
		t.Errorf("expected schema version 1, got %d", p.SchemaVersion)
	}
	if p.ID != "evt-1" {
		t.Errorf("expected ID 'evt-1', got %q", p.ID)
	}
	if p.HookType != "on-event-start" {
		t.Errorf("expected hook type 'on-event-start', got %q", p.HookType)
	}
	if p.MeetingLink == nil || *p.MeetingLink != "https://meet.google.com/abc" {
		t.Error("expected meeting link to be set")
	}
	if p.Start != "2026-04-14T10:00:00Z" {
		t.Errorf("unexpected start: %q", p.Start)
	}
	if p.PreviousEvent != nil {
		t.Error("expected nil PreviousEvent")
	}
	if p.NextEvent != nil {
		t.Error("expected nil NextEvent")
	}
}

func TestEventToPayload_NoMeetingLink(t *testing.T) {
	e := Event{ID: "1", Start: time.Now(), End: time.Now().Add(time.Hour)}
	p := EventToPayload(e, "before-event-start", nil, nil)

	if p.MeetingLink != nil {
		t.Error("expected nil meeting link for empty string")
	}
}

func TestEventToPayload_WithAdjacent(t *testing.T) {
	now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	e := Event{ID: "2", Start: now, End: now.Add(time.Hour)}
	prev := &Event{ID: "1", Title: "Prev", Start: now.Add(-time.Hour), End: now, MeetingLink: "https://zoom.us/j/123"}
	next := &Event{ID: "3", Title: "Next", Start: now.Add(time.Hour), End: now.Add(2 * time.Hour)}

	p := EventToPayload(e, "on-event-start", prev, next)

	if p.PreviousEvent == nil {
		t.Fatal("expected PreviousEvent")
	}
	if p.PreviousEvent.ID != "1" {
		t.Errorf("expected prev ID '1', got %q", p.PreviousEvent.ID)
	}
	if p.PreviousEvent.MeetingLink != "https://zoom.us/j/123" {
		t.Errorf("expected prev meeting link, got %q", p.PreviousEvent.MeetingLink)
	}
	if p.NextEvent == nil {
		t.Fatal("expected NextEvent")
	}
	if p.NextEvent.ID != "3" {
		t.Errorf("expected next ID '3', got %q", p.NextEvent.ID)
	}
}

func TestEventToPayload_AllDay(t *testing.T) {
	start := time.Date(2026, 4, 14, 0, 0, 0, 0, time.Local)
	e := Event{
		ID:     "bday-1",
		Title:  "Birthday",
		Start:  start,
		End:    start.AddDate(0, 0, 1),
		AllDay: true,
		Status: "confirmed",
	}

	p := EventToPayload(e, "on-event-start", nil, nil)

	if !p.AllDay {
		t.Error("expected AllDay to be true in payload")
	}
	if p.Start != "2026-04-14" {
		t.Errorf("expected date-only start '2026-04-14', got %q", p.Start)
	}
	if p.End != "2026-04-15" {
		t.Errorf("expected date-only end '2026-04-15', got %q", p.End)
	}
}
