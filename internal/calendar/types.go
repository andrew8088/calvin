package calendar

import "time"

type Event struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Start           time.Time  `json:"start"`
	End             time.Time  `json:"end"`
	Location        string     `json:"location"`
	Description     string     `json:"description"`
	MeetingLink     string     `json:"meeting_link"`
	MeetingProvider string     `json:"meeting_provider"`
	Attendees       []Attendee `json:"attendees"`
	Organizer       string     `json:"organizer"`
	Calendar        string     `json:"calendar"`
	Status          string     `json:"status"`
	AttendeesHash   string     `json:"-"`
}

type Attendee struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Response string `json:"response"`
}

type DiffType int

const (
	DiffAdded DiffType = iota
	DiffModified
	DiffDeleted
)

type DiffResult struct {
	Type  DiffType
	Event Event
}

type HookPayload struct {
	SchemaVersion   int        `json:"schema_version"`
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Start           string     `json:"start"`
	End             string     `json:"end"`
	Location        string     `json:"location"`
	Description     string     `json:"description"`
	MeetingLink     *string    `json:"meeting_link"`
	MeetingProvider string     `json:"meeting_provider"`
	Attendees       []Attendee `json:"attendees"`
	Organizer       string     `json:"organizer"`
	Calendar        string     `json:"calendar"`
	Status          string     `json:"status"`
	HookType        string     `json:"hook_type"`
	PreviousEvent   *Adjacent  `json:"previous_event"`
	NextEvent       *Adjacent  `json:"next_event"`
}

type Adjacent struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Start       string `json:"start"`
	End         string `json:"end"`
	MeetingLink string `json:"meeting_link"`
}

func EventToPayload(e Event, hookType string, prev, next *Event) HookPayload {
	p := HookPayload{
		SchemaVersion:   1,
		ID:              e.ID,
		Title:           e.Title,
		Start:           e.Start.Format(time.RFC3339),
		End:             e.End.Format(time.RFC3339),
		Location:        e.Location,
		Description:     e.Description,
		MeetingProvider: e.MeetingProvider,
		Attendees:       e.Attendees,
		Organizer:       e.Organizer,
		Calendar:        e.Calendar,
		Status:          e.Status,
		HookType:        hookType,
	}
	if e.MeetingLink != "" {
		link := e.MeetingLink
		p.MeetingLink = &link
	}
	if prev != nil {
		p.PreviousEvent = &Adjacent{
			ID:          prev.ID,
			Title:       prev.Title,
			Start:       prev.Start.Format(time.RFC3339),
			End:         prev.End.Format(time.RFC3339),
			MeetingLink: prev.MeetingLink,
		}
	}
	if next != nil {
		p.NextEvent = &Adjacent{
			ID:          next.ID,
			Title:       next.Title,
			Start:       next.Start.Format(time.RFC3339),
			End:         next.End.Format(time.RFC3339),
			MeetingLink: next.MeetingLink,
		}
	}
	return p
}
