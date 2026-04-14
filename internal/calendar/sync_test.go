package calendar

import (
	"errors"
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

func TestConvertEvent_SkipsAllDay(t *testing.T) {
	item := &googlecalendar.Event{
		Id:      "1",
		Summary: "All day",
		Start:   &googlecalendar.EventDateTime{Date: "2026-04-14"},
	}
	_, ok := convertEvent(item)
	if ok {
		t.Error("expected all-day event to be skipped")
	}
}

func TestConvertEvent_SkipsNilStart(t *testing.T) {
	item := &googlecalendar.Event{Id: "1", Summary: "No start"}
	_, ok := convertEvent(item)
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

	event, ok := convertEvent(item)
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

	event, ok := convertEvent(item)
	if !ok {
		t.Fatal("expected successful conversion")
	}
	if event.Status != "confirmed" {
		t.Errorf("expected default status 'confirmed', got %q", event.Status)
	}
}
