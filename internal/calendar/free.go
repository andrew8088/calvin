package calendar

import (
	"sort"
	"time"
)

type FreeSlot struct {
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	DurationSeconds int64     `json:"duration_seconds"`
}

func FreeSlotsForWindow(start, end time.Time, events []Event) []FreeSlot {
	if !start.Before(end) {
		return nil
	}

	loc := start.Location()

	type interval struct {
		start time.Time
		end   time.Time
	}

	busy := make([]interval, 0, len(events))
	for _, event := range events {
		busyStart := event.Start.In(loc)
		if busyStart.Before(start) {
			busyStart = start
		}
		busyEnd := event.End.In(loc)
		if busyEnd.After(end) {
			busyEnd = end
		}
		if busyStart.Before(busyEnd) {
			busy = append(busy, interval{start: busyStart, end: busyEnd})
		}
	}

	if len(busy) == 0 {
		return []FreeSlot{{
			Start:           start,
			End:             end,
			DurationSeconds: int64(end.Sub(start).Seconds()),
		}}
	}

	sort.Slice(busy, func(i, j int) bool {
		if busy[i].start.Equal(busy[j].start) {
			return busy[i].end.Before(busy[j].end)
		}
		return busy[i].start.Before(busy[j].start)
	})

	merged := make([]interval, 0, len(busy))
	for _, current := range busy {
		if len(merged) == 0 {
			merged = append(merged, current)
			continue
		}
		last := &merged[len(merged)-1]
		if current.start.After(last.end) {
			merged = append(merged, current)
			continue
		}
		if current.end.After(last.end) {
			last.end = current.end
		}
	}

	free := make([]FreeSlot, 0, len(merged)+1)
	cursor := start
	for _, block := range merged {
		if cursor.Before(block.start) {
			free = append(free, FreeSlot{
				Start:           cursor,
				End:             block.start,
				DurationSeconds: int64(block.start.Sub(cursor).Seconds()),
			})
		}
		if block.end.After(cursor) {
			cursor = block.end
		}
	}
	if cursor.Before(end) {
		free = append(free, FreeSlot{
			Start:           cursor,
			End:             end,
			DurationSeconds: int64(end.Sub(cursor).Seconds()),
		})
	}

	return free
}
