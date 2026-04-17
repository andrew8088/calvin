package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/andrew8088/calvin/internal/hooks"
	"github.com/spf13/cobra"
)

type hookFilterOptions struct {
	titlePatterns     []string
	calendarPatterns  []string
	organizerPatterns []string
	statusPatterns    []string
	hookTypePatterns  []string
	minAttendees      int
	maxAttendees      int
	eventFile         string
	why               bool
}

func (o *hookFilterOptions) criteria() (hooks.MatchCriteria, error) {
	criteria := hooks.MatchCriteria{
		TitlePatterns:     o.titlePatterns,
		CalendarPatterns:  o.calendarPatterns,
		OrganizerPatterns: o.organizerPatterns,
		StatusPatterns:    o.statusPatterns,
		HookTypePatterns:  o.hookTypePatterns,
	}

	if o.minAttendees >= 0 {
		criteria.MinAttendees = &o.minAttendees
	}
	if o.maxAttendees >= 0 {
		criteria.MaxAttendees = &o.maxAttendees
	}

	if criteria.MinAttendees != nil && *criteria.MinAttendees < 0 {
		return hooks.MatchCriteria{}, fmt.Errorf("--min-attendees must be >= 0")
	}
	if criteria.MaxAttendees != nil && *criteria.MaxAttendees < 0 {
		return hooks.MatchCriteria{}, fmt.Errorf("--max-attendees must be >= 0")
	}
	if criteria.MinAttendees != nil && criteria.MaxAttendees != nil && *criteria.MinAttendees > *criteria.MaxAttendees {
		return hooks.MatchCriteria{}, fmt.Errorf("--min-attendees cannot be greater than --max-attendees")
	}

	return criteria, nil
}

var matchOpts = hookFilterOptions{minAttendees: -1, maxAttendees: -1}
var ignoreOpts = hookFilterOptions{minAttendees: -1, maxAttendees: -1}

var matchCmd = &cobra.Command{
	Use:   "match",
	Short: "Assert that the current hook event matches filters",
	Example: "  calvin match --calendar 'primary' --title '*standup*'\n" +
		"  calvin match --min-attendees 2 --max-attendees 8",
	Run: func(cmd *cobra.Command, args []string) {
		criteria, err := matchOpts.criteria()
		if err != nil {
			if wantsJSON() {
				_ = writeJSONError(os.Stderr, commandErrorResult{
					OK:      false,
					Command: "match",
					Error:   commandError{Code: "invalid_filters", Message: err.Error()},
				})
				os.Exit(2)
			}
			fmt.Fprintf(os.Stderr, "calvin match: %v\n", err)
			os.Exit(2)
		}
		if wantsJSON() {
			outcome, err := evaluateHookFilter(criteria, matchOpts.eventFile)
			if err != nil {
				_ = writeJSONError(os.Stderr, commandErrorResult{
					OK:      false,
					Command: "match",
					Error:   commandError{Code: "filter_error", Message: err.Error()},
				})
				os.Exit(2)
			}
			_ = writeCommandJSON("match", map[string]any{
				"matched":   outcome.Matched,
				"exit_code": outcome.ExitCode,
				"reasons":   outcome.Reasons,
			})
			if outcome.ExitCode != 0 {
				os.Exit(outcome.ExitCode)
			}
			return
		}
		code := runHookFilter(criteria, matchOpts.eventFile, matchOpts.why, os.Stderr)
		if code != 0 {
			os.Exit(code)
		}
	},
}

var ignoreCmd = &cobra.Command{
	Use:   "ignore",
	Short: "Assert that the current hook event matches ignore filters",
	Example: "  calvin ignore --title '*OOO*' && exit 0\n" +
		"  calvin ignore --calendar 'personal*' && exit 0",
	Run: func(cmd *cobra.Command, args []string) {
		criteria, err := ignoreOpts.criteria()
		if err != nil {
			if wantsJSON() {
				_ = writeJSONError(os.Stderr, commandErrorResult{
					OK:      false,
					Command: "ignore",
					Error:   commandError{Code: "invalid_filters", Message: err.Error()},
				})
				os.Exit(2)
			}
			fmt.Fprintf(os.Stderr, "calvin ignore: %v\n", err)
			os.Exit(2)
		}
		if wantsJSON() {
			outcome, err := evaluateHookFilter(criteria, ignoreOpts.eventFile)
			if err != nil {
				_ = writeJSONError(os.Stderr, commandErrorResult{
					OK:      false,
					Command: "ignore",
					Error:   commandError{Code: "filter_error", Message: err.Error()},
				})
				os.Exit(2)
			}
			_ = writeCommandJSON("ignore", map[string]any{
				"matched":   outcome.Matched,
				"exit_code": outcome.ExitCode,
				"reasons":   outcome.Reasons,
			})
			if outcome.ExitCode != 0 {
				os.Exit(outcome.ExitCode)
			}
			return
		}
		code := runHookFilter(criteria, ignoreOpts.eventFile, ignoreOpts.why, os.Stderr)
		if code != 0 {
			os.Exit(code)
		}
	},
}

func init() {
	addHookFilterFlags(matchCmd, &matchOpts)
	addHookFilterFlags(ignoreCmd, &ignoreOpts)
}

func addHookFilterFlags(cmd *cobra.Command, opts *hookFilterOptions) {
	cmd.Flags().StringArrayVar(&opts.titlePatterns, "title", nil, "Title glob pattern (repeatable)")
	cmd.Flags().StringArrayVar(&opts.calendarPatterns, "calendar", nil, "Calendar glob pattern (repeatable)")
	cmd.Flags().StringArrayVar(&opts.organizerPatterns, "organizer", nil, "Organizer glob pattern (repeatable)")
	cmd.Flags().StringArrayVar(&opts.statusPatterns, "status", nil, "Status glob pattern (repeatable)")
	cmd.Flags().StringArrayVar(&opts.hookTypePatterns, "hook-type", nil, "Hook type glob pattern (repeatable)")
	cmd.Flags().IntVar(&opts.minAttendees, "min-attendees", -1, "Minimum attendee count")
	cmd.Flags().IntVar(&opts.maxAttendees, "max-attendees", -1, "Maximum attendee count")
	cmd.Flags().StringVar(&opts.eventFile, "event-file", "", "Event JSON file path (defaults to CALVIN_EVENT_FILE)")
	cmd.Flags().BoolVar(&opts.why, "why", false, "Print match reasoning to stderr")
}

func runHookFilter(criteria hooks.MatchCriteria, eventFile string, why bool, stderr io.Writer) int {
	result, err := evaluateHookFilter(criteria, eventFile)
	if err != nil {
		fmt.Fprintf(stderr, "calvin: %v\n", err)
		return 2
	}

	if why {
		for _, reason := range result.Reasons {
			fmt.Fprintln(stderr, reason)
		}
	}

	return result.ExitCode
}

type hookFilterOutcome struct {
	Matched  bool
	ExitCode int
	Reasons  []string
}

func evaluateHookFilter(criteria hooks.MatchCriteria, eventFile string) (hookFilterOutcome, error) {
	if !hasAnyCriteria(criteria) {
		return hookFilterOutcome{}, fmt.Errorf("at least one filter is required")
	}

	if eventFile == "" {
		eventFile = os.Getenv("CALVIN_EVENT_FILE")
	}
	if eventFile == "" {
		return hookFilterOutcome{}, fmt.Errorf("no event context found (set CALVIN_EVENT_FILE or pass --event-file)")
	}

	payload, err := hooks.LoadEventContextFile(eventFile)
	if err != nil {
		return hookFilterOutcome{}, err
	}

	result, err := hooks.MatchHookPayload(payload, criteria)
	if err != nil {
		return hookFilterOutcome{}, err
	}

	code := 1
	if result.Matched {
		code = 0
	}
	return hookFilterOutcome{Matched: result.Matched, ExitCode: code, Reasons: result.Reasons}, nil
}

func hasAnyCriteria(criteria hooks.MatchCriteria) bool {
	return len(criteria.TitlePatterns) > 0 ||
		len(criteria.CalendarPatterns) > 0 ||
		len(criteria.OrganizerPatterns) > 0 ||
		len(criteria.StatusPatterns) > 0 ||
		len(criteria.HookTypePatterns) > 0 ||
		criteria.MinAttendees != nil ||
		criteria.MaxAttendees != nil
}
