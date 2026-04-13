package calendar

import (
	"crypto/sha256"
	"fmt"
)

func Diff(dbEvents []Event, apiEvents []Event, syncGen int64) []DiffResult {
	dbMap := make(map[string]Event, len(dbEvents))
	for _, e := range dbEvents {
		dbMap[e.ID] = e
	}

	var results []DiffResult
	seen := make(map[string]bool, len(apiEvents))

	for _, apiEvent := range apiEvents {
		seen[apiEvent.ID] = true

		if apiEvent.Status == "cancelled" {
			if _, exists := dbMap[apiEvent.ID]; exists {
				results = append(results, DiffResult{Type: DiffDeleted, Event: apiEvent})
			}
			continue
		}

		dbEvent, exists := dbMap[apiEvent.ID]
		if !exists {
			results = append(results, DiffResult{Type: DiffAdded, Event: apiEvent})
			continue
		}

		if eventModified(dbEvent, apiEvent) {
			results = append(results, DiffResult{Type: DiffModified, Event: apiEvent})
		}
	}

	return results
}

func eventModified(old, new Event) bool {
	if old.Title != new.Title {
		return true
	}
	if !old.Start.Equal(new.Start) {
		return true
	}
	if !old.End.Equal(new.End) {
		return true
	}
	if old.Status != new.Status {
		return true
	}
	if hashAttendees(old.Attendees) != hashAttendees(new.Attendees) {
		return true
	}
	return false
}

func hashAttendees(attendees []Attendee) string {
	h := sha256.New()
	for _, a := range attendees {
		fmt.Fprintf(h, "%s:%s:%s,", a.Email, a.Name, a.Response)
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
