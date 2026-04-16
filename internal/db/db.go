package db

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"strings"

	"time"

	"github.com/andrew8088/calvin/internal/calendar"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	conn *sqlite.Conn
}

func Open(path string, readOnly bool) (*DB, error) {
	flags := sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL
	if readOnly {
		flags = sqlite.OpenReadOnly | sqlite.OpenWAL
	}

	conn, err := sqlite.OpenConn(path, flags)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := sqlitex.ExecuteTransient(conn, "PRAGMA busy_timeout = 5000", nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("setting busy_timeout: %w", err)
	}
	if err := sqlitex.ExecuteTransient(conn, "PRAGMA journal_mode = WAL", nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	if !readOnly {
		if err := migrate(conn); err != nil {
			conn.Close()
			return nil, fmt.Errorf("running migrations: %w", err)
		}
	}

	return &DB{conn: conn}, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) Checkpoint() error {
	return sqlitex.ExecuteTransient(d.conn, "PRAGMA wal_checkpoint(TRUNCATE)", nil)
}

func migrate(conn *sqlite.Conn) error {
	if err := sqlitex.ExecuteTransient(conn, `CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`, nil); err != nil {
		return err
	}

	var current int
	if err := sqlitex.ExecuteTransient(conn, `SELECT COALESCE(MAX(version), 0) FROM schema_version`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			current = stmt.ColumnInt(0)
			return nil
		},
	}); err != nil {
		return err
	}

	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}

	for i, entry := range entries {
		version := i + 1
		if version <= current {
			continue
		}
		data, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", entry.Name(), err)
		}
		if err := sqlitex.ExecuteScript(conn, string(data), nil); err != nil {
			return fmt.Errorf("executing migration %s: %w", entry.Name(), err)
		}
		if err := sqlitex.ExecuteTransient(conn, `INSERT INTO schema_version (version) VALUES (?)`, &sqlitex.ExecOptions{
			Args: []any{version},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) UpsertEvent(e calendar.Event, syncGen int64) error {
	hash := attendeesHash(e.Attendees)
	allDay := 0
	if e.AllDay {
		allDay = 1
	}
	return sqlitex.ExecuteTransient(d.conn, `
		INSERT INTO events (id, title, start_time, end_time, all_day, location, description,
			meeting_link, meeting_provider, attendees_json, organizer, calendar_id,
			status, attendees_hash, sync_generation)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title=excluded.title, start_time=excluded.start_time, end_time=excluded.end_time,
			all_day=excluded.all_day, location=excluded.location, description=excluded.description,
			meeting_link=excluded.meeting_link, meeting_provider=excluded.meeting_provider,
			attendees_json=excluded.attendees_json, organizer=excluded.organizer,
			calendar_id=excluded.calendar_id, status=excluded.status,
			attendees_hash=excluded.attendees_hash, sync_generation=excluded.sync_generation,
			updated_at=datetime('now')
	`, &sqlitex.ExecOptions{
		Args: []any{
			e.ID, e.Title, e.Start.Format(time.RFC3339), e.End.Format(time.RFC3339),
			allDay, e.Location, e.Description, e.MeetingLink, e.MeetingProvider,
			attendeesJSON(e.Attendees), e.Organizer, e.Calendar, e.Status, hash, syncGen,
		},
	})
}

func (d *DB) GetEvent(id string) (*calendar.Event, error) {
	var event *calendar.Event
	err := sqlitex.ExecuteTransient(d.conn, `
		SELECT id, title, start_time, end_time, all_day, location, description,
			meeting_link, meeting_provider, attendees_json, organizer, calendar_id,
			status, attendees_hash
		FROM events WHERE id = ?
	`, &sqlitex.ExecOptions{
		Args: []any{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			e, err := scanEvent(stmt)
			if err != nil {
				return err
			}
			event = &e
			return nil
		},
	})
	return event, err
}

func (d *DB) ListEventsForDay(day time.Time) ([]calendar.Event, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.Add(24 * time.Hour)
	return d.ListEventsBetween(start, end)
}

func (d *DB) ListUpcomingEvents(from time.Time, limit int) ([]calendar.Event, error) {
	var events []calendar.Event
	err := sqlitex.ExecuteTransient(d.conn, `
		SELECT id, title, start_time, end_time, all_day, location, description,
			meeting_link, meeting_provider, attendees_json, organizer, calendar_id,
			status, attendees_hash
		FROM events WHERE start_time >= ? AND status != 'cancelled'
		ORDER BY start_time ASC LIMIT ?
	`, &sqlitex.ExecOptions{
		Args: []any{from.Format(time.RFC3339), limit},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			e, err := scanEvent(stmt)
			if err != nil {
				return err
			}
			events = append(events, e)
			return nil
		},
	})
	return events, err
}

func (d *DB) ListEventsBetween(start, end time.Time) ([]calendar.Event, error) {
	var events []calendar.Event
	err := sqlitex.ExecuteTransient(d.conn, `
		SELECT id, title, start_time, end_time, all_day, location, description,
			meeting_link, meeting_provider, attendees_json, organizer, calendar_id,
			status, attendees_hash
		FROM events
		WHERE start_time >= ? AND start_time < ? AND status != 'cancelled'
		ORDER BY start_time ASC
	`, &sqlitex.ExecOptions{
		Args: []any{start.Format(time.RFC3339), end.Format(time.RFC3339)},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			e, err := scanEvent(stmt)
			if err != nil {
				return err
			}
			events = append(events, e)
			return nil
		},
	})
	return events, err
}

func (d *DB) DeleteStaleSyncGeneration(currentGen int64) ([]string, error) {
	var deleted []string
	err := sqlitex.ExecuteTransient(d.conn, `
		SELECT id FROM events WHERE sync_generation < ?
	`, &sqlitex.ExecOptions{
		Args: []any{currentGen},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			deleted = append(deleted, stmt.ColumnText(0))
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	if len(deleted) > 0 {
		err = sqlitex.ExecuteTransient(d.conn, `DELETE FROM events WHERE sync_generation < ?`, &sqlitex.ExecOptions{
			Args: []any{currentGen},
		})
	}
	return deleted, err
}

func (d *DB) GetSyncToken(calendarID string) (string, error) {
	var token string
	err := sqlitex.ExecuteTransient(d.conn, `SELECT token FROM sync_state WHERE calendar_id = ?`, &sqlitex.ExecOptions{
		Args: []any{calendarID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			token = stmt.ColumnText(0)
			return nil
		},
	})
	return token, err
}

func (d *DB) SetSyncToken(calendarID, token string) error {
	return sqlitex.ExecuteTransient(d.conn, `
		INSERT INTO sync_state (id, calendar_id, token, updated_at) VALUES (NULL, ?, ?, datetime('now'))
		ON CONFLICT(calendar_id) DO UPDATE SET token=excluded.token, updated_at=excluded.updated_at
	`, &sqlitex.ExecOptions{
		Args: []any{calendarID, token},
	})
}

func (d *DB) GetSyncGeneration() (int64, error) {
	var gen int64
	err := sqlitex.ExecuteTransient(d.conn, `SELECT COALESCE(MAX(sync_generation), 0) FROM events`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			gen = stmt.ColumnInt64(0)
			return nil
		},
	})
	return gen, err
}

func (d *DB) RecordHookExecution(eventID, hookName, hookType, status, stdout, stderr string, durationMs int64) error {
	return sqlitex.ExecuteTransient(d.conn, `
		INSERT INTO hook_executions (event_id, hook_name, hook_type, status, stdout, stderr, duration_ms, executed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, &sqlitex.ExecOptions{
		Args: []any{eventID, hookName, hookType, status, stdout, stderr, durationMs},
	})
}

func (d *DB) HasHookExecuted(eventID, hookName, hookType string) (bool, error) {
	var exists bool
	err := sqlitex.ExecuteTransient(d.conn, `
		SELECT 1 FROM hook_executions
		WHERE event_id = ? AND hook_name = ? AND hook_type = ? AND status = 'success'
		LIMIT 1
	`, &sqlitex.ExecOptions{
		Args: []any{eventID, hookName, hookType},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			exists = true
			return nil
		},
	})
	return exists, err
}

func (d *DB) GetHookExecutions(eventID string) ([]HookExecution, error) {
	var execs []HookExecution
	err := sqlitex.ExecuteTransient(d.conn, `
		SELECT event_id, hook_name, hook_type, status, stdout, stderr, duration_ms, executed_at
		FROM hook_executions WHERE event_id = ?
		ORDER BY executed_at DESC
	`, &sqlitex.ExecOptions{
		Args: []any{eventID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			execs = append(execs, HookExecution{
				EventID:    stmt.ColumnText(0),
				HookName:   stmt.ColumnText(1),
				HookType:   stmt.ColumnText(2),
				Status:     stmt.ColumnText(3),
				Stdout:     stmt.ColumnText(4),
				Stderr:     stmt.ColumnText(5),
				DurationMs: stmt.ColumnInt64(6),
				ExecutedAt: stmt.ColumnText(7),
			})
			return nil
		},
	})
	return execs, err
}

func (d *DB) GetHookStats() (success, failed, timeout int, err error) {
	today := time.Now().Format("2006-01-02")
	err = sqlitex.ExecuteTransient(d.conn, `
		SELECT status, COUNT(*) FROM hook_executions
		WHERE executed_at >= ? GROUP BY status
	`, &sqlitex.ExecOptions{
		Args: []any{today},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			s := stmt.ColumnText(0)
			c := stmt.ColumnInt(1)
			switch s {
			case "success":
				success = c
			case "failed":
				failed = c
			case "timeout":
				timeout = c
			}
			return nil
		},
	})
	return
}

func (d *DB) PruneOldExecutions(retentionDays int) error {
	return sqlitex.ExecuteTransient(d.conn, `
		DELETE FROM hook_executions WHERE executed_at < datetime('now', ?)
	`, &sqlitex.ExecOptions{
		Args: []any{fmt.Sprintf("-%d days", retentionDays)},
	})
}

func (d *DB) GetAdjacentEvents(eventID string, eventStart, eventEnd time.Time) (prev, next *calendar.Event, err error) {
	err = sqlitex.ExecuteTransient(d.conn, `
		SELECT id, title, start_time, end_time, all_day, location, description,
			meeting_link, meeting_provider, attendees_json, organizer, calendar_id,
			status, attendees_hash
		FROM events
		WHERE end_time <= ? AND id != ? AND status != 'cancelled'
		ORDER BY end_time DESC LIMIT 1
	`, &sqlitex.ExecOptions{
		Args: []any{eventStart.Format(time.RFC3339), eventID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			e, err := scanEvent(stmt)
			if err != nil {
				return err
			}
			prev = &e
			return nil
		},
	})
	if err != nil {
		return nil, nil, err
	}

	err = sqlitex.ExecuteTransient(d.conn, `
		SELECT id, title, start_time, end_time, all_day, location, description,
			meeting_link, meeting_provider, attendees_json, organizer, calendar_id,
			status, attendees_hash
		FROM events
		WHERE start_time >= ? AND id != ? AND status != 'cancelled'
		ORDER BY start_time ASC LIMIT 1
	`, &sqlitex.ExecOptions{
		Args: []any{eventEnd.Format(time.RFC3339), eventID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			e, err := scanEvent(stmt)
			if err != nil {
				return err
			}
			next = &e
			return nil
		},
	})
	if err != nil {
		return prev, nil, err
	}

	return prev, next, nil
}

func (d *DB) EventCount() (int, error) {
	var count int
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	err := sqlitex.ExecuteTransient(d.conn, `
		SELECT COUNT(*) FROM events WHERE start_time >= ? AND start_time < ? AND status != 'cancelled'
	`, &sqlitex.ExecOptions{
		Args: []any{today, tomorrow},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			count = stmt.ColumnInt(0)
			return nil
		},
	})
	return count, err
}

func (d *DB) IntegrityCheck() error {
	var result string
	err := sqlitex.ExecuteTransient(d.conn, `PRAGMA integrity_check`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			result = stmt.ColumnText(0)
			return nil
		},
	})
	if err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("integrity check failed: %s", result)
	}
	return nil
}

type HookExecution struct {
	EventID    string
	HookName   string
	HookType   string
	Status     string
	Stdout     string
	Stderr     string
	DurationMs int64
	ExecutedAt string
}

func scanEvent(stmt *sqlite.Stmt) (calendar.Event, error) {
	startStr := stmt.ColumnText(2)
	endStr := stmt.ColumnText(3)
	allDay := stmt.ColumnInt(4) == 1
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return calendar.Event{}, fmt.Errorf("parsing start time: %w", err)
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return calendar.Event{}, fmt.Errorf("parsing end time: %w", err)
	}

	return calendar.Event{
		ID:              stmt.ColumnText(0),
		Title:           stmt.ColumnText(1),
		Start:           start,
		End:             end,
		AllDay:          allDay,
		Location:        stmt.ColumnText(5),
		Description:     stmt.ColumnText(6),
		MeetingLink:     stmt.ColumnText(7),
		MeetingProvider: stmt.ColumnText(8),
		Attendees:       parseAttendees(stmt.ColumnText(9)),
		Organizer:       stmt.ColumnText(10),
		Calendar:        stmt.ColumnText(11),
		Status:          stmt.ColumnText(12),
		AttendeesHash:   stmt.ColumnText(13),
	}, nil
}

func attendeesHash(attendees []calendar.Attendee) string {
	h := sha256.New()
	for _, a := range attendees {
		fmt.Fprintf(h, "%s:%s:%s,", a.Email, a.Name, a.Response)
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func attendeesJSON(attendees []calendar.Attendee) string {
	if len(attendees) == 0 {
		return "[]"
	}
	var parts []string
	for _, a := range attendees {
		parts = append(parts, fmt.Sprintf(`{"email":%q,"name":%q,"response":%q}`, a.Email, a.Name, a.Response))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func parseAttendees(jsonStr string) []calendar.Attendee {
	if jsonStr == "" || jsonStr == "[]" {
		return nil
	}
	var attendees []calendar.Attendee
	i := 0
	for i < len(jsonStr) {
		emailStart := strings.Index(jsonStr[i:], `"email":"`)
		if emailStart == -1 {
			break
		}
		emailStart += i + 9
		emailEnd := strings.Index(jsonStr[emailStart:], `"`)
		email := jsonStr[emailStart : emailStart+emailEnd]

		nameStart := strings.Index(jsonStr[emailStart:], `"name":"`)
		name := ""
		if nameStart != -1 {
			nameStart += emailStart + 8
			nameEnd := strings.Index(jsonStr[nameStart:], `"`)
			name = jsonStr[nameStart : nameStart+nameEnd]
		}

		respStart := strings.Index(jsonStr[emailStart:], `"response":"`)
		resp := ""
		if respStart != -1 {
			respStart += emailStart + 12
			respEnd := strings.Index(jsonStr[respStart:], `"`)
			resp = jsonStr[respStart : respStart+respEnd]
		}

		attendees = append(attendees, calendar.Attendee{Email: email, Name: name, Response: resp})
		nextObj := strings.Index(jsonStr[emailStart:], "}")
		if nextObj == -1 {
			break
		}
		i = emailStart + nextObj + 1
	}
	return attendees
}

func (d *DB) BeginTransaction() error {
	return sqlitex.ExecuteTransient(d.conn, "BEGIN IMMEDIATE", nil)
}

func (d *DB) Commit() error {
	return sqlitex.ExecuteTransient(d.conn, "COMMIT", nil)
}

func (d *DB) Rollback() error {
	return sqlitex.ExecuteTransient(d.conn, "ROLLBACK", nil)
}

func (d *DB) WithTransaction(ctx context.Context, fn func() error) error {
	if err := d.BeginTransaction(); err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(); err != nil {
		d.Rollback()
		return err
	}
	return d.Commit()
}
