package hooks

import (
	"fmt"
	"path"
	"strings"

	"github.com/andrew8088/calvin/internal/calendar"
)

type MatchCriteria struct {
	TitlePatterns     []string
	CalendarPatterns  []string
	OrganizerPatterns []string
	StatusPatterns    []string
	HookTypePatterns  []string
	MinAttendees      *int
	MaxAttendees      *int
}

type MatchResult struct {
	Matched bool
	Reasons []string
}

func MatchHookPayload(payload calendar.HookPayload, criteria MatchCriteria) (MatchResult, error) {
	if criteria.MinAttendees != nil && criteria.MaxAttendees != nil && *criteria.MinAttendees > *criteria.MaxAttendees {
		return MatchResult{}, fmt.Errorf("min_attendees cannot be greater than max_attendees")
	}

	if !hasCriteria(criteria) {
		return MatchResult{Matched: true, Reasons: []string{"no filters provided; matched by default"}}, nil
	}

	matched := true
	reasons := make([]string, 0, 8)

	ok, reason, err := matchStringField("title", payload.Title, criteria.TitlePatterns)
	if err != nil {
		return MatchResult{}, err
	}
	if reason != "" {
		reasons = append(reasons, reason)
	}
	if !ok {
		matched = false
	}

	ok, reason, err = matchStringField("calendar", payload.Calendar, criteria.CalendarPatterns)
	if err != nil {
		return MatchResult{}, err
	}
	if reason != "" {
		reasons = append(reasons, reason)
	}
	if !ok {
		matched = false
	}

	ok, reason, err = matchStringField("organizer", payload.Organizer, criteria.OrganizerPatterns)
	if err != nil {
		return MatchResult{}, err
	}
	if reason != "" {
		reasons = append(reasons, reason)
	}
	if !ok {
		matched = false
	}

	ok, reason, err = matchStringField("status", payload.Status, criteria.StatusPatterns)
	if err != nil {
		return MatchResult{}, err
	}
	if reason != "" {
		reasons = append(reasons, reason)
	}
	if !ok {
		matched = false
	}

	ok, reason, err = matchStringField("hook_type", payload.HookType, criteria.HookTypePatterns)
	if err != nil {
		return MatchResult{}, err
	}
	if reason != "" {
		reasons = append(reasons, reason)
	}
	if !ok {
		matched = false
	}

	attendeeCount := len(payload.Attendees)
	if criteria.MinAttendees != nil {
		if attendeeCount < *criteria.MinAttendees {
			reasons = append(reasons, fmt.Sprintf("attendees %d is less than min_attendees %d", attendeeCount, *criteria.MinAttendees))
			matched = false
		} else {
			reasons = append(reasons, fmt.Sprintf("attendees %d is >= min_attendees %d", attendeeCount, *criteria.MinAttendees))
		}
	}

	if criteria.MaxAttendees != nil {
		if attendeeCount > *criteria.MaxAttendees {
			reasons = append(reasons, fmt.Sprintf("attendees %d is greater than max_attendees %d", attendeeCount, *criteria.MaxAttendees))
			matched = false
		} else {
			reasons = append(reasons, fmt.Sprintf("attendees %d is <= max_attendees %d", attendeeCount, *criteria.MaxAttendees))
		}
	}

	return MatchResult{Matched: matched, Reasons: reasons}, nil
}

func hasCriteria(criteria MatchCriteria) bool {
	return len(criteria.TitlePatterns) > 0 ||
		len(criteria.CalendarPatterns) > 0 ||
		len(criteria.OrganizerPatterns) > 0 ||
		len(criteria.StatusPatterns) > 0 ||
		len(criteria.HookTypePatterns) > 0 ||
		criteria.MinAttendees != nil ||
		criteria.MaxAttendees != nil
}

func matchStringField(field, value string, patterns []string) (bool, string, error) {
	if len(patterns) == 0 {
		return true, "", nil
	}

	valueLower := strings.ToLower(value)
	for _, pattern := range patterns {
		ok, err := path.Match(strings.ToLower(pattern), valueLower)
		if err != nil {
			return false, "", fmt.Errorf("invalid %s pattern %q: %w", field, pattern, err)
		}
		if ok {
			return true, fmt.Sprintf("%s matched %q", field, pattern), nil
		}
	}

	return false, fmt.Sprintf("%s %q did not match any pattern", field, value), nil
}
