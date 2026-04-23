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
	now         func() time.Time
	newService  func(context.Context, oauth2.TokenSource) (*googlecalendar.Service, error)
}

func NewSyncer(ts oauth2.TokenSource) *Syncer {
	return &Syncer{
		tokenSource: ts,
		now:         time.Now,
		newService:  newCalendarService,
	}
}

type syncQueryPlan struct {
	fullSync  bool
	orderBy   string
	syncToken string
	timeMin   string
	timeMax   string
}

type syncRequestSpec struct {
	singleEvents bool
	showDeleted  bool
	orderBy      string
	syncToken    string
	timeMin      string
	timeMax      string
	pageToken    string
}

func buildSyncQueryPlan(now time.Time, syncToken string) syncQueryPlan {
	plan := syncQueryPlan{
		fullSync:  syncToken == "",
		syncToken: syncToken,
	}
	if syncToken != "" {
		return plan
	}

	plan.orderBy = "startTime"
	plan.timeMin = now.Format(time.RFC3339)
	plan.timeMax = now.Add(7 * 24 * time.Hour).Format(time.RFC3339)
	return plan
}

func (p syncQueryPlan) requestSpec(pageToken string) syncRequestSpec {
	return syncRequestSpec{
		singleEvents: true,
		showDeleted:  true,
		orderBy:      p.orderBy,
		syncToken:    p.syncToken,
		timeMin:      p.timeMin,
		timeMax:      p.timeMax,
		pageToken:    pageToken,
	}
}

func applySyncRequestSpec(call *googlecalendar.EventsListCall, spec syncRequestSpec) *googlecalendar.EventsListCall {
	call = call.SingleEvents(spec.singleEvents).ShowDeleted(spec.showDeleted)
	if spec.orderBy != "" {
		call = call.OrderBy(spec.orderBy)
	}
	if spec.syncToken != "" {
		call = call.SyncToken(spec.syncToken)
	}
	if spec.timeMin != "" {
		call = call.TimeMin(spec.timeMin)
	}
	if spec.timeMax != "" {
		call = call.TimeMax(spec.timeMax)
	}
	if spec.pageToken != "" {
		call = call.PageToken(spec.pageToken)
	}
	return call
}

func newCalendarService(ctx context.Context, ts oauth2.TokenSource) (*googlecalendar.Service, error) {
	return googlecalendar.NewService(ctx, option.WithTokenSource(ts))
}

func (s *Syncer) currentTime() time.Time {
	if s.now == nil {
		return time.Now()
	}
	return s.now()
}

func (s *Syncer) service(ctx context.Context) (*googlecalendar.Service, error) {
	if s.newService == nil {
		return newCalendarService(ctx, s.tokenSource)
	}
	return s.newService(ctx, s.tokenSource)
}

func (s *Syncer) Sync(ctx context.Context, calendarID, syncToken string) ([]Event, string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	srv, err := s.service(ctx)
	if err != nil {
		return nil, "", false, fmt.Errorf("creating calendar service: %w", err)
	}

	var allEvents []Event
	var nextSyncToken string
	pageToken := ""
	plan := buildSyncQueryPlan(s.currentTime(), syncToken)

	for {
		call := applySyncRequestSpec(srv.Events.List(calendarID), plan.requestSpec(pageToken))

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

	return allEvents, nextSyncToken, plan.fullSync, nil
}

func (s *Syncer) FetchNextEvent(ctx context.Context, calendarID string) (*Event, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	srv, err := s.service(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating calendar service: %w", err)
	}

	now := s.currentTime()
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

	srv, err := s.service(ctx)
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
	status := "confirmed"
	if item.Status != "" {
		status = item.Status
	}

	if status == "cancelled" && (item.Start == nil || (item.Start.DateTime == "" && item.Start.Date == "")) {
		return Event{
			ID:       item.Id,
			Title:    item.Summary,
			Calendar: calendarID,
			Status:   status,
		}, true
	}

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
