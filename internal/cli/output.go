package cli

import (
	"fmt"
	"os"
	"strings"
)

var noColor bool

func init() {
	if os.Getenv("NO_COLOR") != "" {
		noColor = true
	}
	if !isTerminal() {
		noColor = true
	}
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
)

func green(s string) string {
	if noColor {
		return s
	}
	return colorGreen + s + colorReset
}

func red(s string) string {
	if noColor {
		return s
	}
	return colorRed + s + colorReset
}

func yellow(s string) string {
	if noColor {
		return s
	}
	return colorYellow + s + colorReset
}

func blue(s string) string {
	if noColor {
		return s
	}
	return colorBlue + s + colorReset
}

func cyan(s string) string {
	if noColor {
		return s
	}
	return colorCyan + s + colorReset
}

func dim(s string) string {
	if noColor {
		return s
	}
	return colorDim + s + colorReset
}

func bold(s string) string {
	if noColor {
		return s
	}
	return colorBold + s + colorReset
}

func symPass() string  { return green("✓") }
func symFail() string  { return red("✗") }
func symWarn() string  { return yellow("△") }
func symRun() string   { return green("●") }
func symStop() string  { return dim("○") }

func errMsg(what, why, fix string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", symFail(), what)
	fmt.Fprintf(os.Stderr, "           %s\n", why)
	fmt.Fprintf(os.Stderr, "           Fix: %s\n", cyan(fix))
}

func warnMsg(what, why, fix string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", symWarn(), what)
	fmt.Fprintf(os.Stderr, "           %s\n", why)
	if fix != "" {
		fmt.Fprintf(os.Stderr, "           Fix: %s\n", cyan(fix))
	}
}

func formatDuration(d fmt.Stringer) string {
	return d.String()
}

func formatCountdown(until fmt.Stringer) string {
	return until.String()
}

func humanDuration(secs int64) string {
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%dm %ds", secs/60, secs%60)
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	return fmt.Sprintf("%dh %dm", h, m)
}

func humanCountdown(t int64) string {
	if t <= 0 {
		return "now"
	}
	return "in " + humanDuration(t)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
