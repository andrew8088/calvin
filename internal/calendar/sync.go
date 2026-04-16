package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Syncer struct {
	tokenSource oauth2.TokenSource
}

func NewSyncer(ts oauth2.TokenSource) *Syncer {
	return &Syncer{tokenSource: ts}
}

func (s *Syncer) Sync(ctx context.Context, calendarID, syncToken string) ([]Event, string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	srv, err := googlecalendar.NewService(ctx, option.WithTokenSource(s.tokenSource))
	if err != nil {
		return nil, "", false, fmt.Errorf("creating calendar service: %w", err)
	}

	var allEvents []Event
	var nextSyncToken string
	pageToken := ""
	fullSync := syncToken == ""

	for {
		call := srv.Events.List(calendarID).
			SingleEvents(true).
			OrderBy("startTime").
			ShowDeleted(true)

		if syncToken != "" && pageToken == "" {
			call = call.SyncToken(syncToken)
		} else if fullSync && pageToken == "" {
			now := time.Now()
			call = call.TimeMin(now.Format(time.RFC3339)).
				TimeMax(now.Add(7 * 24 * time.Hour).Format(time.RFC3339))
		}

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		result, err := call.Do()
		if err != nil {
			if isGoneError(err) {
				return s.Sync(ctx, calendarID, "")
			}
			return nil, "", false, fmt.Errorf("fetching events: %w", err)
		}

		for _, item := range result.Items {
			event, ok := convertEvent(item, calendarID)
			if !ok {
				continue
			}
			allEvents = append(allEvents, event)
		}

		if result.NextPageToken != "" {
			pageToken = result.NextPageToken
			continue
		}

		nextSyncToken = result.NextSyncToken
		break
	}

	return allEvents, nextSyncToken, fullSync, nil
}

func (s *Syncer) FetchNextEvent(ctx context.Context, calendarID string) (*Event, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	srv, err := googlecalendar.NewService(ctx, option.WithTokenSource(s.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("creating calendar service: %w", err)
	}

	now := time.Now()
	result, err := srv.Events.List(calendarID).
		SingleEvents(true).
		OrderBy("startTime").
		TimeMin(now.Format(time.RFC3339)).
		MaxResults(1).
		Do()

	if err != nil {
		return nil, fmt.Errorf("fetching next event: %w", err)
	}

	for _, item := range result.Items {
		event, ok := convertEvent(item, calendarID)
		if !ok {
			continue
		}
		return &event, nil
	}
	return nil, nil
}

func (s *Syncer) CheckAPIAccess(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	srv, err := googlecalendar.NewService(ctx, option.WithTokenSource(s.tokenSource))
	if err != nil {
		return fmt.Errorf("creating calendar service: %w", err)
	}

	_, err = srv.CalendarList.List().MaxResults(1).Do()
	if err != nil {
		return fmt.Errorf("accessing calendar API: %w", err)
	}
	return nil
}

func convertEvent(item *googlecalendar.Event, calendarID string) (Event, bool) {
	if item.Start == nil {
		return Event{}, false
	}

	var start, end time.Time
	var allDay bool

	if item.Start.DateTime != "" {
		var err error
		start, err = time.Parse(time.RFC3339, item.Start.DateTime)
		if err != nil {
			return Event{}, false
		}
		end, _ = time.Parse(time.RFC3339, item.End.DateTime)
	} else if item.Start.Date != "" {
		allDay = true
		var err error
		start, err = time.ParseInLocation("2006-01-02", item.Start.Date, time.Local)
		if err != nil {
			return Event{}, false
		}
		if item.End != nil && item.End.Date != "" {
			end, _ = time.ParseInLocation("2006-01-02", item.End.Date, time.Local)
		} else {
			end = start.AddDate(0, 0, 1)
		}
	} else {
		return Event{}, false
	}

	var attendees []Attendee
	for _, a := range item.Attendees {
		attendees = append(attendees, Attendee{
			Email:    a.Email,
			Name:     a.DisplayName,
			Response: a.ResponseStatus,
		})
	}

	organizer := ""
	if item.Organizer != nil {
		organizer = item.Organizer.Email
	}

	link, provider := extractMeetingLink(item)

	status := "confirmed"
	if item.Status != "" {
		status = item.Status
	}

	return Event{
		ID:              item.Id,
		Title:           item.Summary,
		Start:           start,
		End:             end,
		AllDay:          allDay,
		Location:        item.Location,
		Description:     item.Description,
		MeetingLink:     link,
		MeetingProvider: provider,
		Attendees:       attendees,
		Organizer:       organizer,
		Calendar:        calendarID,
		Status:          status,
	}, true
}

func extractMeetingLink(item *googlecalendar.Event) (string, string) {
	if item.HangoutLink != "" {
		return item.HangoutLink, "google_meet"
	}

	if item.ConferenceData != nil {
		for _, ep := range item.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" && ep.Uri != "" {
				return ep.Uri, inferProvider(ep.Uri)
			}
		}
	}

	if item.Location != "" {
		if strings.Contains(item.Location, "zoom.us") {
			return item.Location, "zoom"
		}
		if strings.Contains(item.Location, "teams.microsoft.com") {
			return item.Location, "teams"
		}
	}

	return "", "unknown"
}

func inferProvider(url string) string {
	switch {
	case strings.Contains(url, "zoom.us"):
		return "zoom"
	case strings.Contains(url, "meet.google.com"):
		return "google_meet"
	case strings.Contains(url, "teams.microsoft.com"):
		return "teams"
	case strings.Contains(url, "slack.com/huddle"):
		return "slack_huddle"
	default:
		return "unknown"
	}
}

func isGoneError(err error) bool {
	return strings.Contains(err.Error(), "410") || strings.Contains(err.Error(), "Gone")
}
