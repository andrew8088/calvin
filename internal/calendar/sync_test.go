package calendar

import (
	"os"
	"strings"
	"testing"

	googlecalendar "google.golang.org/api/calendar/v3"
)

func TestInferProvider(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://zoom.us/j/123456", "zoom"},
		{"https://meet.google.com/abc-defg-hij", "google_meet"},
		{"https://teams.microsoft.com/l/meetup-join/abc", "teams"},
		{"https://app.slack.com/huddle/T123/C456", "slack_huddle"},
		{"https://example.com/call", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := inferProvider(tt.url)
			if got != tt.expected {
				t.Errorf("inferProvider(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestExtractMeetingLink_HangoutLink(t *testing.T) {
	item := &googlecalendar.Event{
		HangoutLink: "https://meet.google.com/abc-defg-hij",
	}
	link, provider := extractMeetingLink(item)
	if link != "https://meet.google.com/abc-defg-hij" {
		t.Errorf("expected hangout link, got %q", link)
	}
	if provider != "google_meet" {
		t.Errorf("expected google_meet, got %q", provider)
	}
}

func TestExtractMeetingLink_ConferenceData(t *testing.T) {
	item := &googlecalendar.Event{
		ConferenceData: &googlecalendar.ConferenceData{
			EntryPoints: []*googlecalendar.EntryPoint{
				{EntryPointType: "phone", Uri: "tel:+1234567890"},
				{EntryPointType: "video", Uri: "https://zoom.us/j/999"},
			},
		},
	}
	link, provider := extractMeetingLink(item)
	if link != "https://zoom.us/j/999" {
		t.Errorf("expected zoom link, got %q", link)
	}
	if provider != "zoom" {
		t.Errorf("expected zoom, got %q", provider)
	}
}

func TestExtractMeetingLink_LocationFallback(t *testing.T) {
	tests := []struct {
		name     string
		location string
		wantLink string
		wantProv string
	}{
		{"zoom in location", "https://zoom.us/j/123", "https://zoom.us/j/123", "zoom"},
		{"teams in location", "https://teams.microsoft.com/l/meetup", "https://teams.microsoft.com/l/meetup", "teams"},
		{"no link", "Conference Room B", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &googlecalendar.Event{Location: tt.location}
			link, provider := extractMeetingLink(item)
			if link != tt.wantLink {
				t.Errorf("link = %q, want %q", link, tt.wantLink)
			}
			if provider != tt.wantProv {
				t.Errorf("provider = %q, want %q", provider, tt.wantProv)
			}
		})
	}
}

func TestExtractMeetingLink_HangoutTakesPriority(t *testing.T) {
	item := &googlecalendar.Event{
		HangoutLink: "https://meet.google.com/primary",
		ConferenceData: &googlecalendar.ConferenceData{
			EntryPoints: []*googlecalendar.EntryPoint{
				{EntryPointType: "video", Uri: "https://zoom.us/j/secondary"},
			},
		},
		Location: "https://teams.microsoft.com/l/tertiary",
	}
	link, provider := extractMeetingLink(item)
	if link != "https://meet.google.com/primary" {
		t.Errorf("expected hangout link to take priority, got %q", link)
	}
	if provider != "google_meet" {
		t.Errorf("expected google_meet, got %q", provider)
	}
}

func TestIsGoneError(t *testing.T) {
	tests := []struct {
		err    error
		expect bool
	}{
		{errors.New("googleapi: Error 410: Gone"), true},
		{errors.New("something 410 happened"), true},
		{errors.New("Gone with the wind"), true},
		{errors.New("404 not found"), false},
		{errors.New("unauthorized"), false},
	}

	for _, tt := range tests {
		got := isGoneError(tt.err)
		if got != tt.expect {
			t.Errorf("isGoneError(%q) = %v, want %v", tt.err, got, tt.expect)
		}
	}
}

func TestConvertEvent_AllDayEvent(t *testing.T) {
	item := &googlecalendar.Event{
		Id:      "1",
		Summary: "Birthday",
		Start:   &googlecalendar.EventDateTime{Date: "2026-04-14"},
		End:     &googlecalendar.EventDateTime{Date: "2026-04-15"},
		Status:  "confirmed",
	}
	event, ok := convertEvent(item, "primary")
	if !ok {
		t.Fatal("expected all-day event to convert successfully")
	}
	if !event.AllDay {
		t.Error("expected AllDay to be true")
	}
	if event.Title != "Birthday" {
		t.Errorf("Title = %q, want 'Birthday'", event.Title)
	}
	if event.Start.Format("2006-01-02") != "2026-04-14" {
		t.Errorf("Start date = %q, want '2026-04-14'", event.Start.Format("2006-01-02"))
	}
	if event.End.Format("2006-01-02") != "2026-04-15" {
		t.Errorf("End date = %q, want '2026-04-15'", event.End.Format("2006-01-02"))
	}
}

func TestConvertEvent_AllDayNoEnd(t *testing.T) {
	item := &googlecalendar.Event{
		Id:      "2",
		Summary: "Deadline",
		Start:   &googlecalendar.EventDateTime{Date: "2026-04-14"},
	}
	event, ok := convertEvent(item, "primary")
	if !ok {
		t.Fatal("expected all-day event without end to convert")
	}
	if !event.AllDay {
		t.Error("expected AllDay to be true")
	}
	if event.End.Format("2006-01-02") != "2026-04-15" {
		t.Errorf("End should default to next day, got %q", event.End.Format("2006-01-02"))
	}
}

func TestConvertEvent_AllDayMultiDay(t *testing.T) {
	item := &googlecalendar.Event{
		Id:      "3",
		Summary: "Vacation",
		Start:   &googlecalendar.EventDateTime{Date: "2026-04-14"},
		End:     &googlecalendar.EventDateTime{Date: "2026-04-18"},
	}
	event, ok := convertEvent(item, "primary")
	if !ok {
		t.Fatal("expected multi-day event to convert")
	}
	if event.End.Format("2006-01-02") != "2026-04-18" {
		t.Errorf("End = %q, want '2026-04-18'", event.End.Format("2006-01-02"))
	}
}

func TestConvertEvent_SkipsNilStart(t *testing.T) {
	item := &googlecalendar.Event{Id: "1", Summary: "No start"}
	_, ok := convertEvent(item, "primary")
	if ok {
		t.Error("expected event with nil start to be skipped")
	}
}

func TestConvertEvent_ValidEvent(t *testing.T) {
	item := &googlecalendar.Event{
		Id:      "evt-1",
		Summary: "Standup",
		Start:   &googlecalendar.EventDateTime{DateTime: "2026-04-14T10:00:00Z"},
		End:     &googlecalendar.EventDateTime{DateTime: "2026-04-14T10:30:00Z"},
		Attendees: []*googlecalendar.EventAttendee{
			{Email: "alice@co.com", DisplayName: "Alice", ResponseStatus: "accepted"},
		},
		Organizer: &googlecalendar.EventOrganizer{Email: "boss@co.com"},
		Location:  "Room A",
		Status:    "confirmed",
	}

	event, ok := convertEvent(item, "primary")
	if !ok {
		t.Fatal("expected successful conversion")
	}
	if event.ID != "evt-1" {
		t.Errorf("ID = %q, want 'evt-1'", event.ID)
	}
	if event.Title != "Standup" {
		t.Errorf("Title = %q, want 'Standup'", event.Title)
	}
	if event.Organizer != "boss@co.com" {
		t.Errorf("Organizer = %q, want 'boss@co.com'", event.Organizer)
	}
	if event.AllDay {
		t.Error("expected AllDay to be false for timed event")
	}
	if event.Calendar != "primary" {
		t.Errorf("Calendar = %q, want 'primary'", event.Calendar)
	}
	if len(event.Attendees) != 1 {
		t.Fatalf("expected 1 attendee, got %d", len(event.Attendees))
	}
	if event.Attendees[0].Email != "alice@co.com" {
		t.Errorf("attendee email = %q", event.Attendees[0].Email)
	}
}

func TestConvertEvent_DefaultStatus(t *testing.T) {
	item := &googlecalendar.Event{
		Id:    "1",
		Start: &googlecalendar.EventDateTime{DateTime: "2026-04-14T10:00:00Z"},
		End:   &googlecalendar.EventDateTime{DateTime: "2026-04-14T10:30:00Z"},
	}

	event, ok := convertEvent(item, "primary")
	if !ok {
		t.Fatal("expected successful conversion")
	}
	if event.Status != "confirmed" {
		t.Errorf("expected default status 'confirmed', got %q", event.Status)
	}
}

func TestSync_QueryParameters_SameForFullAndIncremental(t *testing.T) {
	// The Sync method must use identical base query parameters for both
	// initial (full) sync and incremental sync — only the syncToken differs.
	// Google Calendar API strongly requires this (same query params plus syncToken/maxResults
	// are allowed to differ between requests).
	//
	// The base parameters are:
	//   SingleEvents(true), OrderBy("startTime"), ShowDeleted(true)
	//
	// The incremental path adds: SyncToken(token)
	// Neither path uses: TimeMin, TimeMax
	//
	// This test documents that contract by verifying the query parameter
	// construction in the source code matches expectations.

	// Verify via source inspection that TimeMin/TimeMax are not used in Sync
	syncSrc, err := os.ReadFile("sync.go")
	if err != nil {
		t.Skip("cannot read sync.go, skipping source inspection")
	}

	src := string(syncSrc)

	if strings.Contains(src, "TimeMin") || strings.Contains(src, "TimeMax") {
		t.Error("Sync must not use TimeMin/TimeMax — they break sync-token compatibility with Google Calendar API")
	}

	if !strings.Contains(src, "SyncToken") {
		t.Error("Sync should use SyncToken for incremental refreshes")
	}

	if !strings.Contains(src, "SingleEvents(true)") {
		t.Error("Sync should use SingleEvents(true)")
	}

	if !strings.Contains(src, `OrderBy("startTime")`) {
		t.Error("Sync should use OrderBy(startTime)")
	}

	if !strings.Contains(src, "ShowDeleted(true)") {
		t.Error("Sync should use ShowDeleted(true)")
	}
}
