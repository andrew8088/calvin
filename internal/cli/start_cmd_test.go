package cli

import (
	"testing"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/db"
)

func openStartTestDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.Open(":memory:", false)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}

	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("database.Close: %v", err)
		}
	})

	return database
}

func TestDiffEventsBeforeUpsert(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		existing     *calendar.Event
		apiEvent     calendar.Event
		expectedDiff calendar.DiffType
		expectDiff   bool
	}{
		{
			name: "unchanged ongoing all-day event",
			existing: &calendar.Event{
				ID:       "home-1",
				Title:    "Home",
				Start:    time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC),
				End:      time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC),
				AllDay:   true,
				Calendar: "primary",
				Status:   "confirmed",
			},
			apiEvent: calendar.Event{
				ID:       "home-1",
				Title:    "Home",
				Start:    time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC),
				End:      time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC),
				AllDay:   true,
				Calendar: "primary",
				Status:   "confirmed",
			},
		},
		{
			name: "modified event",
			existing: &calendar.Event{
				ID:       "evt-1",
				Title:    "Standup",
				Start:    now,
				End:      now.Add(time.Hour),
				Calendar: "primary",
				Status:   "confirmed",
			},
			apiEvent: calendar.Event{
				ID:       "evt-1",
				Title:    "Standup v2",
				Start:    now,
				End:      now.Add(time.Hour),
				Calendar: "primary",
				Status:   "confirmed",
			},
			expectDiff:   true,
			expectedDiff: calendar.DiffModified,
		},
		{
			name: "new event",
			apiEvent: calendar.Event{
				ID:       "evt-2",
				Title:    "Planning",
				Start:    now,
				End:      now.Add(time.Hour),
				Calendar: "primary",
				Status:   "confirmed",
			},
			expectDiff:   true,
			expectedDiff: calendar.DiffAdded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database := openStartTestDB(t)
			if tt.existing != nil {
				if err := database.UpsertEvent(*tt.existing, 1); err != nil {
					t.Fatalf("database.UpsertEvent: %v", err)
				}
			}

			diffs, err := diffEventsBeforeUpsert(database, []calendar.Event{tt.apiEvent})
			if err != nil {
				t.Fatalf("diffEventsBeforeUpsert: %v", err)
			}

			if !tt.expectDiff {
				if len(diffs) != 0 {
					t.Fatalf("expected no diffs, got %v", diffs)
				}
				return
			}

			if len(diffs) != 1 {
				t.Fatalf("expected 1 diff, got %d", len(diffs))
			}
			if diffs[0].Type != tt.expectedDiff {
				t.Fatalf("expected diff type %d, got %d", tt.expectedDiff, diffs[0].Type)
			}
		})
	}
}
