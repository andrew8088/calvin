package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/andrew8088/calvin/internal/hooks"
	"github.com/andrew8088/calvin/internal/logging"
)

type EventTimers struct {
	PreEvent   *time.Timer
	EventStart *time.Timer
	EventEnd   *time.Timer
	Event      calendar.Event
}

type Scheduler struct {
	cfg      *config.Config
	database *db.DB
	executor *hooks.Executor
	mu       sync.Mutex
	timers   map[string]*EventTimers
	cancel   context.CancelFunc
}

func New(cfg *config.Config, database *db.DB, executor *hooks.Executor) *Scheduler {
	return &Scheduler{
		cfg:      cfg,
		database: database,
		executor: executor,
		timers:   make(map[string]*EventTimers),
	}
}

func (s *Scheduler) ScheduleFromDB(ctx context.Context) error {
	now := time.Now()
	window := 2*time.Hour + time.Duration(s.cfg.PreEventMinutes)*time.Minute
	windowEnd := now.Add(window)

	events, err := s.database.ListUpcomingEvents(now.Add(-1*time.Hour), 100)
	if err != nil {
		return fmt.Errorf("listing events for scheduling: %w", err)
	}

	scheduled := make(map[string]bool)
	for _, event := range events {
		if event.Start.After(windowEnd) {
			continue
		}
		s.scheduleEvent(ctx, event, now)
		scheduled[event.ID] = true
	}

	s.mu.Lock()
	for id, et := range s.timers {
		if !scheduled[id] {
			s.cancelTimers(et)
			delete(s.timers, id)
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *Scheduler) ProcessDiff(ctx context.Context, diffs []calendar.DiffResult) {
	log := logging.Get()

	for _, diff := range diffs {
		switch diff.Type {
		case calendar.DiffAdded:
			log.Info("scheduler", fmt.Sprintf("New event: %s (%s)", diff.Event.Title, diff.Event.ID))
		case calendar.DiffModified:
			log.Info("scheduler", fmt.Sprintf("Modified event: %s (%s)", diff.Event.Title, diff.Event.ID))
			s.mu.Lock()
			if et, ok := s.timers[diff.Event.ID]; ok {
				s.cancelTimers(et)
				delete(s.timers, diff.Event.ID)
			}
			s.mu.Unlock()
		case calendar.DiffDeleted:
			log.Info("scheduler", fmt.Sprintf("Deleted event: %s (%s)", diff.Event.Title, diff.Event.ID))
			s.mu.Lock()
			if et, ok := s.timers[diff.Event.ID]; ok {
				s.cancelTimers(et)
				delete(s.timers, diff.Event.ID)
			}
			s.mu.Unlock()
		}
	}
}

func (s *Scheduler) scheduleEvent(ctx context.Context, event calendar.Event, now time.Time) {
	s.mu.Lock()
	if _, exists := s.timers[event.ID]; exists {
		s.mu.Unlock()
		return
	}

	et := &EventTimers{Event: event}
	s.timers[event.ID] = et
	s.mu.Unlock()

	preEventTime := event.Start.Add(-time.Duration(s.cfg.PreEventMinutes) * time.Minute)

	if preEventTime.After(now) {
		delay := preEventTime.Sub(now)
		et.PreEvent = time.AfterFunc(delay, func() {
			s.fireHook(ctx, event, "pre_event")
		})
	}

	if event.Start.After(now) {
		delay := event.Start.Sub(now)
		et.EventStart = time.AfterFunc(delay, func() {
			s.fireHook(ctx, event, "event_start")
		})
	}

	if event.End.After(now) {
		delay := event.End.Sub(now)
		et.EventEnd = time.AfterFunc(delay, func() {
			s.fireHook(ctx, event, "event_end")
		})
	}
}

func (s *Scheduler) fireHook(ctx context.Context, event calendar.Event, hookType string) {
	log := logging.Get()

	fireTime := time.Now()
	var scheduledTime time.Time
	switch hookType {
	case "pre_event":
		scheduledTime = event.Start.Add(-time.Duration(s.cfg.PreEventMinutes) * time.Minute)
	case "event_start":
		scheduledTime = event.Start
	case "event_end":
		scheduledTime = event.End
	}

	drift := fireTime.Sub(scheduledTime)
	if drift > 60*time.Second {
		log.Warn("scheduler", fmt.Sprintf("Stale timer for %s/%s (drift: %s), discarding", event.Title, hookType, drift))
		return
	}

	allHooks, err := hooks.Discover()
	if err != nil {
		log.Error("scheduler", fmt.Sprintf("Hook discovery failed: %v", err))
		return
	}

	hookList, ok := allHooks[hookType]
	if !ok || len(hookList) == 0 {
		return
	}

	hookList = s.filterHooksByCalendar(hookList, event.Calendar)

	if len(hookList) == 0 {
		return
	}

	log.Info("scheduler", fmt.Sprintf("Firing %d %s hooks for: %s", len(hookList), hookType, event.Title))

	prev, next, _ := s.database.GetAdjacentEvents(event.ID, event.Start, event.End)
	results := s.executor.FireHooks(ctx, event, hookType, hookList, prev, next)

	for _, r := range results {
		if r.Status == "failed" || r.Status == "timeout" {
			log.Error("hooks", fmt.Sprintf("%s %s: %s (%s)", hookType, r.HookName, r.Status, r.EventID))
		}
	}
}

func (s *Scheduler) filterHooksByCalendar(hookList []hooks.Hook, calendarID string) []hooks.Hook {
	for _, cal := range s.cfg.ResolvedCalendars() {
		if cal.ID != calendarID {
			continue
		}
		if len(cal.HookDirs) == 0 {
			return hookList
		}
		allowed := make(map[string]bool)
		for _, name := range cal.HookDirs {
			allowed[name] = true
		}
		var filtered []hooks.Hook
		for _, h := range hookList {
			if allowed[h.Name] {
				filtered = append(filtered, h)
			}
		}
		return filtered
	}
	return hookList
}

func (s *Scheduler) cancelTimers(et *EventTimers) {
	if et.PreEvent != nil {
		et.PreEvent.Stop()
	}
	if et.EventStart != nil {
		et.EventStart.Stop()
	}
	if et.EventEnd != nil {
		et.EventEnd.Stop()
	}
}

func (s *Scheduler) CancelAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, et := range s.timers {
		s.cancelTimers(et)
		delete(s.timers, id)
	}
}

func (s *Scheduler) TimerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.timers)
}
