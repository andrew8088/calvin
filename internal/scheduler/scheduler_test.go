package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/andrew8088/calvin/internal/hooks"
	"github.com/andrew8088/calvin/internal/logging"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:", false)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func newTestScheduler(t *testing.T) (*Scheduler, *db.DB) {
	t.Helper()
	logging.InitStdout()
	d := openTestDB(t)
	cfg := config.Default()
	executor := hooks.NewExecutor(cfg, d)
	s := New(cfg, d, executor)
	return s, d
}

func futureEvent(id string, minutesFromNow int) calendar.Event {
	now := time.Now()
	return calendar.Event{
		ID:     id,
		Title:  "Event " + id,
		Start:  now.Add(time.Duration(minutesFromNow) * time.Minute),
		End:    now.Add(time.Duration(minutesFromNow+30) * time.Minute),
		Status: "confirmed",
	}
}

func TestNew(t *testing.T) {
	s, _ := newTestScheduler(t)
	if s.TimerCount() != 0 {
		t.Errorf("new scheduler should have 0 timers, got %d", s.TimerCount())
	}
}

func TestScheduleFromDB_SchedulesUpcomingEvents(t *testing.T) {
	s, d := newTestScheduler(t)

	e := futureEvent("evt-1", 30)
	d.UpsertEvent(e, 1)

	if err := s.ScheduleFromDB(context.Background()); err != nil {
		t.Fatalf("ScheduleFromDB: %v", err)
	}

	if s.TimerCount() != 1 {
		t.Errorf("expected 1 timer, got %d", s.TimerCount())
	}

	s.CancelAll()
}

func TestScheduleFromDB_IgnoresFarFutureEvents(t *testing.T) {
	s, d := newTestScheduler(t)

	e := futureEvent("evt-far", 300)
	d.UpsertEvent(e, 1)

	s.ScheduleFromDB(context.Background())

	if s.TimerCount() != 0 {
		t.Errorf("far future event should not be scheduled, got %d timers", s.TimerCount())
	}
}

func TestScheduleFromDB_NoDuplicateTimers(t *testing.T) {
	s, d := newTestScheduler(t)

	e := futureEvent("evt-1", 30)
	d.UpsertEvent(e, 1)

	s.ScheduleFromDB(context.Background())
	s.ScheduleFromDB(context.Background())

	if s.TimerCount() != 1 {
		t.Errorf("expected 1 timer (no duplicates), got %d", s.TimerCount())
	}

	s.CancelAll()
}

func TestCancelAll(t *testing.T) {
	s, d := newTestScheduler(t)

	d.UpsertEvent(futureEvent("evt-1", 30), 1)
	d.UpsertEvent(futureEvent("evt-2", 60), 1)

	s.ScheduleFromDB(context.Background())
	if s.TimerCount() != 2 {
		t.Fatalf("expected 2 timers, got %d", s.TimerCount())
	}

	s.CancelAll()
	if s.TimerCount() != 0 {
		t.Errorf("expected 0 timers after CancelAll, got %d", s.TimerCount())
	}
}

func TestProcessDiff_DeletedCancelsTimers(t *testing.T) {
	s, d := newTestScheduler(t)

	e := futureEvent("evt-1", 30)
	d.UpsertEvent(e, 1)
	s.ScheduleFromDB(context.Background())

	if s.TimerCount() != 1 {
		t.Fatalf("expected 1 timer, got %d", s.TimerCount())
	}

	s.ProcessDiff(context.Background(), []calendar.DiffResult{
		{Type: calendar.DiffDeleted, Event: e},
	})

	if s.TimerCount() != 0 {
		t.Errorf("expected 0 timers after delete, got %d", s.TimerCount())
	}
}

func TestProcessDiff_ModifiedCancelsTimers(t *testing.T) {
	s, d := newTestScheduler(t)

	e := futureEvent("evt-1", 30)
	d.UpsertEvent(e, 1)
	s.ScheduleFromDB(context.Background())

	s.ProcessDiff(context.Background(), []calendar.DiffResult{
		{Type: calendar.DiffModified, Event: e},
	})

	if s.TimerCount() != 0 {
		t.Errorf("expected 0 timers after modify, got %d", s.TimerCount())
	}
}

func TestProcessDiff_AddedDoesNotCrash(t *testing.T) {
	s, _ := newTestScheduler(t)

	s.ProcessDiff(context.Background(), []calendar.DiffResult{
		{Type: calendar.DiffAdded, Event: futureEvent("new", 30)},
	})

	if s.TimerCount() != 0 {
		t.Errorf("added event should not create timers via ProcessDiff, got %d", s.TimerCount())
	}
}

func TestScheduleFromDB_CleansRemovedEvents(t *testing.T) {
	s, d := newTestScheduler(t)

	d.UpsertEvent(futureEvent("evt-1", 30), 1)
	s.ScheduleFromDB(context.Background())
	if s.TimerCount() != 1 {
		t.Fatalf("expected 1 timer")
	}

	d.DeleteStaleSyncGeneration(2)
	s.ScheduleFromDB(context.Background())

	if s.TimerCount() != 0 {
		t.Errorf("expected 0 timers after event removed from DB, got %d", s.TimerCount())
	}
}
