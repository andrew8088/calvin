package calendar

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
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

func TestBuildSyncQueryPlanFullSync(t *testing.T) {
	now := time.Date(2026, 4, 18, 15, 30, 0, 0, time.UTC)

	plan := buildSyncQueryPlan(now, "")
	spec := plan.requestSpec("")
	pageSpec := plan.requestSpec("page-2")

	if !plan.fullSync {
		t.Fatal("expected full sync plan")
	}
	if spec.syncToken != "" {
		t.Fatalf("syncToken = %q, want empty", spec.syncToken)
	}
	if spec.orderBy != "startTime" {
		t.Fatalf("orderBy = %q, want startTime", spec.orderBy)
	}
	if spec.timeMin != now.Format(time.RFC3339) {
		t.Fatalf("timeMin = %q, want %q", spec.timeMin, now.Format(time.RFC3339))
	}
	wantMax := now.Add(7 * 24 * time.Hour).Format(time.RFC3339)
	if spec.timeMax != wantMax {
		t.Fatalf("timeMax = %q, want %q", spec.timeMax, wantMax)
	}
	if pageSpec.pageToken != "page-2" {
		t.Fatalf("pageToken = %q, want page-2", pageSpec.pageToken)
	}
	if pageSpec.timeMin != spec.timeMin || pageSpec.timeMax != spec.timeMax || pageSpec.orderBy != spec.orderBy {
		t.Fatal("paginated full-sync request should preserve the original filters")
	}
}

func TestBuildSyncQueryPlanIncrementalSync(t *testing.T) {
	now := time.Date(2026, 4, 18, 15, 30, 0, 0, time.UTC)

	plan := buildSyncQueryPlan(now, "stored-sync-token")
	spec := plan.requestSpec("page-2")

	if plan.fullSync {
		t.Fatal("expected incremental sync plan")
	}
	if spec.syncToken != "stored-sync-token" {
		t.Fatalf("syncToken = %q, want stored-sync-token", spec.syncToken)
	}
	if spec.orderBy != "" {
		t.Fatalf("orderBy = %q, want empty", spec.orderBy)
	}
	if spec.timeMin != "" || spec.timeMax != "" {
		t.Fatalf("incremental sync should not set time bounds, got timeMin=%q timeMax=%q", spec.timeMin, spec.timeMax)
	}
	if !spec.singleEvents || !spec.showDeleted {
		t.Fatal("incremental sync should keep singleEvents and showDeleted enabled")
	}
	if spec.pageToken != "page-2" {
		t.Fatalf("pageToken = %q, want page-2", spec.pageToken)
	}
}

func TestSyncPreservesIncrementalQueryAcrossPages(t *testing.T) {
	syncer := NewSyncer(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}))

	var seenQueries []string
	syncer.newService = func(ctx context.Context, _ oauth2.TokenSource) (*googlecalendar.Service, error) {
		client := &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				query := req.URL.Query()
				seenQueries = append(seenQueries, req.URL.RawQuery)

				switch query.Get("pageToken") {
				case "":
					if got := query.Get("syncToken"); got != "stored-sync-token" {
						t.Errorf("page 1 syncToken = %q, want stored-sync-token", got)
					}
					if got := query.Get("orderBy"); got != "" {
						t.Errorf("page 1 orderBy = %q, want empty", got)
					}
					if got := query.Get("timeMin"); got != "" {
						t.Errorf("page 1 timeMin = %q, want empty", got)
					}
					if got := query.Get("timeMax"); got != "" {
						t.Errorf("page 1 timeMax = %q, want empty", got)
					}
					return jsonResponse(req, `{
						"items": [
							{
								"id": "evt-1",
								"summary": "Planning",
								"start": {"dateTime": "2026-04-18T10:00:00Z"},
								"end": {"dateTime": "2026-04-18T11:00:00Z"}
							}
						],
						"nextPageToken": "page-2"
					}`), nil
				case "page-2":
					if got := query.Get("syncToken"); got != "stored-sync-token" {
						t.Errorf("page 2 syncToken = %q, want stored-sync-token", got)
					}
					if got := query.Get("orderBy"); got != "" {
						t.Errorf("page 2 orderBy = %q, want empty", got)
					}
					if got := query.Get("timeMin"); got != "" {
						t.Errorf("page 2 timeMin = %q, want empty", got)
					}
					if got := query.Get("timeMax"); got != "" {
						t.Errorf("page 2 timeMax = %q, want empty", got)
					}
					return jsonResponse(req, `{
						"items": [
							{
								"id": "evt-2",
								"summary": "Retro",
								"start": {"dateTime": "2026-04-18T12:00:00Z"},
								"end": {"dateTime": "2026-04-18T13:00:00Z"}
							}
						],
						"nextSyncToken": "fresh-sync-token"
					}`), nil
				default:
					t.Fatalf("unexpected pageToken %q", query.Get("pageToken"))
					return nil, nil
				}
			}),
		}

		return googlecalendar.NewService(
			ctx,
			option.WithHTTPClient(client),
			option.WithEndpoint("https://calendar.test/"),
		)
	}

	events, nextSyncToken, fullSync, err := syncer.Sync(context.Background(), "work@company.com", "stored-sync-token")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if fullSync {
		t.Fatal("expected incremental sync")
	}
	if nextSyncToken != "fresh-sync-token" {
		t.Fatalf("nextSyncToken = %q, want fresh-sync-token", nextSyncToken)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Calendar != "work@company.com" || events[1].Calendar != "work@company.com" {
		t.Fatalf("events should preserve calendar ID, got %+v", events)
	}
	if len(seenQueries) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(seenQueries))
	}
	if !strings.Contains(seenQueries[1], "pageToken=page-2") {
		t.Fatalf("second request should include pageToken, got %q", seenQueries[1])
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(req *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body:    io.NopCloser(strings.NewReader(body)),
		Request: req,
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
