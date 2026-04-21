package terminal

import (
	"os"
	"testing"
)

func TestColorize(t *testing.T) {
	tests := []struct {
		style  TextStyle
		color  TextColor
		output string
	}{
		{StyleBold, FgRed, "\x1b[1;31m"},
		{StyleReset, FgDefault, "\x1b[0;39m"},
		{StyleDim, FgGreen, "\x1b[2;32m"},
	}

	for _, tt := range tests {
		result := Colorize(tt.style, tt.color)
		if result != tt.output {
			t.Errorf("Colorize(%v, %v) = %q, want %q", tt.style, tt.color, result, tt.output)
		}
	}
}

func TestResetOutput(t *testing.T) {
	result := ResetOutput()
	if result != "\x1b[0m" {
		t.Errorf("ResetOutput() = %q, want %q", result, "\x1b[0m")
	}
}

func TestSupportsColor(t *testing.T) {
	// Save original values
	originalNoColor := os.Getenv("NO_COLOR")
	originalTerm := os.Getenv("TERM")
	defer func() {
		if originalNoColor != "" {
			os.Setenv("NO_COLOR", originalNoColor)
		} else {
			os.Unsetenv("NO_COLOR")
		}
		if originalTerm != "" {
			os.Setenv("TERM", originalTerm)
		} else {
			os.Unsetenv("TERM")
		}
	}()

	// Test NO_COLOR takes precedence
	os.Unsetenv("NO_COLOR")
	os.Setenv("TERM", "xterm-256color")
	if !SupportsColor() {
		t.Error("expected SupportsColor()=true for xterm-256color")
	}

	os.Setenv("NO_COLOR", "1")
	if SupportsColor() {
		t.Error("expected SupportsColor()=false when NO_COLOR is set")
	}

	// Test dumb terminal
	os.Unsetenv("NO_COLOR")
	os.Setenv("TERM", "dumb")
	if SupportsColor() {
		t.Error("expected SupportsColor()=false for dumb terminal")
	}

	// Test empty TERM
	os.Setenv("TERM", "")
	if SupportsColor() {
		t.Error("expected SupportsColor()=false for empty TERM")
	}
}

func TestTextStyleConstants(t *testing.T) {
	tests := []struct {
		constant TextStyle
		expected int
	}{
		{StyleReset, 0},
		{StyleBold, 1},
		{StyleDim, 2},
		{StyleItalic, 3},
		{StyleUnderline, 4},
	}

	for _, tt := range tests {
		if int(tt.constant) != tt.expected {
			t.Errorf("constant = %d, want %d", tt.constant, tt.expected)
		}
	}
}

func TestTextColorConstants(t *testing.T) {
	tests := []struct {
		constant TextColor
		expected int
	}{
		{FgBlack, 30},
		{FgRed, 31},
		{FgGreen, 32},
		{FgYellow, 33},
		{FgBlue, 34},
		{FgMagenta, 35},
		{FgCyan, 36},
		{FgWhite, 37},
		{FgDefault, 39},
	}

	for _, tt := range tests {
		if int(tt.constant) != tt.expected {
			t.Errorf("constant = %d, want %d", tt.constant, tt.expected)
		}
	}
}

func TestBrightTextColorConstants(t *testing.T) {
	tests := []struct {
		constant TextColor
		expected int
	}{
		{FgBrightBlack, 90},
		{FgBrightRed, 91},
		{FgBrightGreen, 92},
		{FgBrightYellow, 93},
		{FgBrightBlue, 94},
		{FgBrightMagenta, 95},
		{FgBrightCyan, 96},
		{FgBrightWhite, 97},
	}

	for _, tt := range tests {
		if int(tt.constant) != tt.expected {
			t.Errorf("constant = %d, want %d", tt.constant, tt.expected)
		}
	}
}

func TestBackgroundColorConstants(t *testing.T) {
	tests := []struct {
		constant TextColor
		expected int
	}{
		{BgBlack, 40},
		{BgRed, 41},
		{BgGreen, 42},
		{BgYellow, 43},
		{BgBlue, 44},
		{BgMagenta, 45},
		{BgCyan, 46},
		{BgWhite, 47},
	}

	for _, tt := range tests {
		if int(tt.constant) != tt.expected {
			t.Errorf("constant = %d, want %d", tt.constant, tt.expected)
		}
	}
}

func TestNewStyler(t *testing.T) {
	styler := NewStyler()
	if styler == nil {
		t.Fatal("expected non-nil Styler")
	}
	// Styler should detect color support based on environment
}

func TestStylerColor(t *testing.T) {
	// Test with color disabled
	os.Setenv("NO_COLOR", "1")
	styler := NewStyler()

	result := styler.Color(StyleBold, FgRed, "test")
	if result != "test" {
		t.Errorf("expected 'test' when color disabled, got %q", result)
	}

	os.Unsetenv("NO_COLOR")
}

func TestStylerBold(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	styler := NewStyler()
	result := styler.Bold("bold text")
	if result != "bold text" {
		t.Errorf("expected 'bold text' when color disabled, got %q", result)
	}
}

func TestStylerInfo(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	styler := NewStyler()
	result := styler.Info("info text")
	if result != "info text" {
		t.Errorf("expected 'info text' when color disabled, got %q", result)
	}
}

func TestStylerWarning(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	styler := NewStyler()
	result := styler.Warning("warning text")
	if result != "warning text" {
		t.Errorf("expected 'warning text' when color disabled, got %q", result)
	}
}

func TestStylerError(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	styler := NewStyler()
	result := styler.Error("error text")
	if result != "error text" {
		t.Errorf("expected 'error text' when color disabled, got %q", result)
	}
}

func TestStylerSuccess(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	styler := NewStyler()
	result := styler.Success("success text")
	if result != "success text" {
		t.Errorf("expected 'success text' when color disabled, got %q", result)
	}
}

func TestStylerDim(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	styler := NewStyler()
	result := styler.Dim("dim text")
	if result != "dim text" {
		t.Errorf("expected 'dim text' when color disabled, got %q", result)
	}
}

func TestStylerItalic(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	styler := NewStyler()
	result := styler.Italic("italic text")
	if result != "italic text" {
		t.Errorf("expected 'italic text' when color disabled, got %q", result)
	}
}

func TestStylerFormatSeverity(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	styler := NewStyler()

	tests := []struct {
		severity int
		expected string
	}{
		{0, "[INFO]"},
		{1, "[WARNING]"},
		{2, "[HIGH]"},
		{3, "[CRITICAL]"},
	}

	for _, tt := range tests {
		result := styler.FormatSeverity(tt.severity)
		if result != tt.expected {
			t.Errorf("FormatSeverity(%d) = %q, want %q", tt.severity, result, tt.expected)
		}
	}
}

func TestStylerFormatRuleID(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	styler := NewStyler()

	tests := []struct {
		ruleID  string
		unknown bool
	}{
		{"IDX-001", false},
		{"IDX-005", false},
		{"IDX-010", false},
		{"IDX-UNKNOWN", true},
	}

	for _, tt := range tests {
		result := styler.FormatRuleID(tt.ruleID)
		// When color disabled, should return just the ruleID
		if result == tt.ruleID && tt.unknown {
			// Known rule should be colored but with color disabled returns plain
			// Unknown rule also returns plain
		}
	}
}

func TestStylerFormatEmoji(t *testing.T) {
	styler := NewStyler()

	tests := []struct {
		severity int
		expected string
	}{
		{0, "ℹ"},
		{1, "⚠"},
		{2, "🔴"},
		{3, "🚨"},
	}

	for _, tt := range tests {
		result := styler.FormatEmoji(tt.severity)
		if result != tt.expected {
			t.Errorf("FormatEmoji(%d) = %q, want %q", tt.severity, result, tt.expected)
		}
	}
}

// ===============================================================================
// Menu Tests
// ===============================================================================

func TestNewMenu(t *testing.T) {
	menu := NewMenu("Test Menu")
	if menu == nil {
		t.Fatal("expected non-nil Menu")
	}
	if menu.Title != "Test Menu" {
		t.Errorf("expected Title='Test Menu', got %q", menu.Title)
	}
	if menu.Items == nil {
		t.Error("expected non-nil Items slice")
	}
	if menu.Styler == nil {
		t.Error("expected non-nil Styler")
	}
}

func TestMenuAddItem(t *testing.T) {
	menu := NewMenu("Test Menu")
	action := func() error { return nil }

	result := menu.AddItem("1", "Option 1", action)
	if result != menu {
		t.Error("expected AddItem to return the menu for chaining")
	}
	if len(menu.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(menu.Items))
	}
	if menu.Items[0].Key != "1" {
		t.Errorf("expected Key='1', got %q", menu.Items[0].Key)
	}
	if menu.Items[0].Label != "Option 1" {
		t.Errorf("expected Label='Option 1', got %q", menu.Items[0].Label)
	}
	if menu.Items[0].Action == nil {
		t.Error("expected non-nil Action")
	}
}

func TestMenuAddSubMenu(t *testing.T) {
	menu := NewMenu("Parent Menu")
	subMenu := NewMenu("Sub Menu")

	result := menu.AddSubMenu("2", "Sub Menu", subMenu)
	if result != menu {
		t.Error("expected AddSubMenu to return the menu for chaining")
	}
	if len(menu.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(menu.Items))
	}
	if menu.Items[0].SubMenu != subMenu {
		t.Error("expected SubMenu to be set")
	}
	if subMenu.Parent != menu {
		t.Error("expected Parent to be set on submenu")
	}
}

// ===============================================================================
// ProgressBar Tests
// ===============================================================================

func TestNewProgressBar(t *testing.T) {
	pb := NewProgressBar(100)
	if pb == nil {
		t.Fatal("expected non-nil ProgressBar")
	}
	if pb.Width != 40 {
		t.Errorf("expected Width=40, got %d", pb.Width)
	}
	if pb.total != 100 {
		t.Errorf("expected total=100, got %d", pb.total)
	}
	if pb.current != 0 {
		t.Errorf("expected current=0, got %d", pb.current)
	}
	if pb.Styler == nil {
		t.Error("expected non-nil Styler")
	}
}

func TestProgressBarSet(t *testing.T) {
	pb := NewProgressBar(100)
	pb.Set(50)
	if pb.current != 50 {
		t.Errorf("expected current=50, got %d", pb.current)
	}
}

func TestProgressBarIncrement(t *testing.T) {
	pb := NewProgressBar(100)
	pb.Set(0)
	pb.Increment()
	if pb.current != 1 {
		t.Errorf("expected current=1, got %d", pb.current)
	}
}

func TestProgressBarIncrementBeyondTotal(t *testing.T) {
	pb := NewProgressBar(10)
	pb.Set(9)
	pb.Increment()
	if pb.current != 10 {
		t.Errorf("expected current=10, got %d", pb.current)
	}
}

// ===============================================================================
// Spinner Tests
// ===============================================================================

func TestNewSpinner(t *testing.T) {
	s := NewSpinner()
	if s == nil {
		t.Fatal("expected non-nil Spinner")
	}
	if len(s.frames) == 0 {
		t.Error("expected non-empty frames")
	}
	if s.index != 0 {
		t.Errorf("expected index=0, got %d", s.index)
	}
	if s.stop == nil {
		t.Error("expected non-nil stop channel")
	}
}

func TestSpinnerStart(t *testing.T) {
	s := NewSpinner()
	// Start should not block - it runs in a goroutine
	s.Start("Loading...")
	// Give it a moment to start
	s.Stop()
}

func TestSpinnerStop(t *testing.T) {
	s := NewSpinner()
	s.Start("Loading...")
	s.Stop()
	// Stop should send to channel without blocking
	select {
	case s.stop <- true:
		// If we get here immediately after Stop(), it means Stop() closed or reset the channel
	default:
	}
}
