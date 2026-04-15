// Package terminal provides cross-platform terminal output utilities.
package terminal

import (
	"fmt"
	"os"
)

// TextStyle represents ANSI text style codes
type TextStyle int

// TextColor represents foreground text color
type TextColor int

// Style constants
const (
	StyleReset    TextStyle = 0
	StyleBold     TextStyle = 1
	StyleDim      TextStyle = 2
	StyleItalic   TextStyle = 3
	StyleUnderline TextStyle = 4
)

// Foreground color constants
const (
	FgBlack   TextColor = 30
	FgRed     TextColor = 31
	FgGreen   TextColor = 32
	FgYellow  TextColor = 33
	FgBlue    TextColor = 34
	FgMagenta TextColor = 35
	FgCyan    TextColor = 36
	FgWhite   TextColor = 37
	FgDefault TextColor = 39
)

// Bright foreground color constants
const (
	FgBrightBlack   TextColor = 90
	FgBrightRed     TextColor = 91
	FgBrightGreen   TextColor = 92
	FgBrightYellow  TextColor = 93
	FgBrightBlue    TextColor = 94
	FgBrightMagenta TextColor = 95
	FgBrightCyan    TextColor = 96
	FgBrightWhite   TextColor = 97
)

// Background color constants
const (
	BgBlack   TextColor = 40
	BgRed     TextColor = 41
	BgGreen   TextColor = 42
	BgYellow  TextColor = 43
	BgBlue    TextColor = 44
	BgMagenta TextColor = 45
	BgCyan    TextColor = 46
	BgWhite   TextColor = 47
)

// Colorize returns ANSI escape sequence for styling
func Colorize(style TextStyle, color TextColor) string {
	return fmt.Sprintf("\x1b[%d;%dm", style, color)
}

// ResetOutput resets all terminal styling
func ResetOutput() string {
	return "\x1b[0m"
}

// SupportsColor checks if the terminal supports color output
func SupportsColor() bool {
	// Check NO_COLOR env var
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Check TERM env var
	term := os.Getenv("TERM")
	if term == "dumb" || term == "" {
		return false
	}
	return true
}

// Severity text colors
const (
	SeverityInfoColor    TextColor = FgCyan
	SeverityWarnColor    TextColor = FgYellow
	SeverityHighColor    TextColor = FgRed
	SeverityCriticalColor TextColor = FgBrightRed
)

// Rule ID colors
var ruleColors = map[string]TextColor{
	"IDX-001": FgCyan,
	"IDX-002": FgYellow,
	"IDX-003": FgMagenta,
	"IDX-004": FgYellow,
	"IDX-005": FgYellow,
	"IDX-006": FgRed,
	"IDX-007": FgGreen,
	"IDX-008": FgCyan,
	"IDX-009": FgYellow,
	"IDX-010": FgCyan,
}

// Severity labels
var severityLabels = map[int]string{
	0: "INFO",
	1: "WARNING",
	2: "HIGH",
	3: "CRITICAL",
}

// Severity emojis
var severityEmojis = map[int]string{
	0: "ℹ",
	1: "⚠",
	2: "🔴",
	3: "🚨",
}

// Styler provides fluent styling interface
type Styler struct {
	color bool
}

// NewStyler creates a new Styler instance
func NewStyler() *Styler {
	return &Styler{
		color: SupportsColor(),
	}
}

// Color returns a colored string if color is enabled
func (s *Styler) Color(style TextStyle, color TextColor, text string) string {
	if !s.color {
		return text
	}
	return fmt.Sprintf("%s%s%s", Colorize(style, color), text, ResetOutput())
}

// Bold prints bold text
func (s *Styler) Bold(text string) string {
	return s.Color(StyleBold, FgWhite, text)
}

// Info prints info level text
func (s *Styler) Info(text string) string {
	return s.Color(StyleBold, FgCyan, text)
}

// Warning prints warning level text
func (s *Styler) Warning(text string) string {
	return s.Color(StyleBold, FgYellow, text)
}

// Error prints error level text
func (s *Styler) Error(text string) string {
	return s.Color(StyleBold, FgRed, text)
}

// Success prints success text
func (s *Styler) Success(text string) string {
	return s.Color(StyleBold, FgGreen, text)
}

// Dim prints dimmed text
func (s *Styler) Dim(text string) string {
	return s.Color(StyleDim, FgWhite, text)
}

// Italic prints italic text
func (s *Styler) Italic(text string) string {
	return s.Color(StyleItalic, FgWhite, text)
}

// FormatSeverity formats a severity level with color and label
func (s *Styler) FormatSeverity(severity int) string {
	label := severityLabels[severity]
	var color TextColor
	switch severity {
	case 0:
		color = SeverityInfoColor
	case 1:
		color = SeverityWarnColor
	case 2:
		color = SeverityHighColor
	case 3:
		color = SeverityCriticalColor
	default:
		color = FgWhite
	}
	return s.Color(StyleBold, color, "["+label+"]")
}

// FormatRuleID formats a rule ID with its associated color
func (s *Styler) FormatRuleID(ruleID string) string {
	if color, ok := ruleColors[ruleID]; ok {
		return s.Color(StyleBold, color, ruleID)
	}
	return s.Color(StyleBold, FgWhite, ruleID)
}

// FormatEmoji returns emoji for severity
func (s *Styler) FormatEmoji(severity int) string {
	return severityEmojis[severity]
}

// PrintHeader prints a styled header
func (s *Styler) PrintHeader(title string) {
	line := s.Color(StyleBold, FgCyan, "═══════════════════════════════════════")
	fmt.Println(line)
	fmt.Println(s.Color(StyleBold, FgWhite, "  "+title))
	fmt.Println(line)
}

// PrintStats prints formatted statistics
func (s *Styler) PrintStats(total, info, warn, high, critical int) {
	stats := fmt.Sprintf("Found %d issue(s): %s %d, %s %d, %s %d, %s %d",
		total,
		s.Color(StyleBold, FgCyan, "info"),
		info,
		s.Color(StyleBold, FgYellow, "warning"),
		warn,
		s.Color(StyleBold, FgRed, "high"),
		high,
		s.Color(StyleBold, FgBrightRed, "critical"),
		critical,
	)
	fmt.Println(stats)
}

// PrintSuccess prints a success message
func (s *Styler) PrintSuccess(msg string) {
	fmt.Printf("%s %s\n", s.Color(StyleBold, FgGreen, "✓"), msg)
}

// PrintError prints an error message
func (s *Styler) PrintError(msg string) {
	fmt.Printf("%s %s\n", s.Color(StyleBold, FgRed, "✗"), msg)
}

// PrintWarning prints a warning message
func (s *Styler) PrintWarning(msg string) {
	fmt.Printf("%s %s\n", s.Color(StyleBold, FgYellow, "⚠"), msg)
}

// PrintInfo prints an info message
func (s *Styler) PrintInfo(msg string) {
	fmt.Printf("%s %s\n", s.Color(StyleBold, FgCyan, "ℹ"), msg)
}
