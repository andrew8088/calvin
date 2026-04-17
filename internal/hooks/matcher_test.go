package hooks

import (
	"strings"
	"testing"

	"github.com/andrew8088/calvin/internal/calendar"
)

func TestMatchHookPayload_TitleGlobCaseInsensitive(t *testing.T) {
	payload := calendar.HookPayload{
		Title:     "Weekly Standup",
		Calendar:  "primary",
		Organizer: "alice@example.com",
		Attendees: []calendar.Attendee{
			{Email: "a@example.com"},
			{Email: "b@example.com"},
		},
	}

	result, err := MatchHookPayload(payload, MatchCriteria{TitlePatterns: []string{"*standup"}})
	if err != nil {
		t.Fatalf("MatchHookPayload: %v", err)
	}
	if !result.Matched {
		t.Fatalf("expected match, reasons=%v", result.Reasons)
	}
}

func TestMatchHookPayload_OrWithinField_AndAcrossFields(t *testing.T) {
	payload := calendar.HookPayload{
		Title:     "Weekly Standup",
		Calendar:  "primary",
		Organizer: "alice@example.com",
	}

	result, err := MatchHookPayload(payload, MatchCriteria{
		TitlePatterns:     []string{"*retro*", "*standup*"},
		CalendarPatterns:  []string{"primary"},
		OrganizerPatterns: []string{"alice@*"},
	})
	if err != nil {
		t.Fatalf("MatchHookPayload: %v", err)
	}
	if !result.Matched {
		t.Fatalf("expected match, reasons=%v", result.Reasons)
	}
}

func TestMatchHookPayload_AttendeeBounds(t *testing.T) {
	payload := calendar.HookPayload{
		Title: "Weekly Standup",
		Attendees: []calendar.Attendee{
			{Email: "a@example.com"},
			{Email: "b@example.com"},
		},
	}

	minTwo := 2
	maxThree := 3
	result, err := MatchHookPayload(payload, MatchCriteria{MinAttendees: &minTwo, MaxAttendees: &maxThree})
	if err != nil {
		t.Fatalf("MatchHookPayload: %v", err)
	}
	if !result.Matched {
		t.Fatalf("expected attendee bounds match, reasons=%v", result.Reasons)
	}

	minThree := 3
	result, err = MatchHookPayload(payload, MatchCriteria{MinAttendees: &minThree})
	if err != nil {
		t.Fatalf("MatchHookPayload: %v", err)
	}
	if result.Matched {
		t.Fatalf("expected attendee min mismatch")
	}
	if len(result.Reasons) == 0 {
		t.Fatalf("expected mismatch reason")
	}
}

func TestMatchHookPayload_InvalidPattern(t *testing.T) {
	payload := calendar.HookPayload{Title: "Weekly Standup"}

	_, err := MatchHookPayload(payload, MatchCriteria{TitlePatterns: []string{"["}})
	if err == nil {
		t.Fatal("expected invalid glob pattern error")
	}
}

func TestMatchHookPayload_NoCriteriaMatches(t *testing.T) {
	payload := calendar.HookPayload{Title: "Anything"}

	result, err := MatchHookPayload(payload, MatchCriteria{})
	if err != nil {
		t.Fatalf("MatchHookPayload: %v", err)
	}
	if !result.Matched {
		t.Fatalf("expected match with empty criteria")
	}
	if len(result.Reasons) == 0 {
		t.Fatalf("expected reason explaining default match")
	}
	if !strings.Contains(result.Reasons[0], "no filters") {
		t.Fatalf("unexpected reason: %q", result.Reasons[0])
	}
}
