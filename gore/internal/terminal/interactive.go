// Package terminal provides cross-platform terminal output utilities.
package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MenuItem represents a menu option
type MenuItem struct {
	Key      string
	Label    string
	Action   func() error
	SubMenu  *Menu
}

// Menu represents an interactive menu
type Menu struct {
	Title   string
	Items   []MenuItem
	Parent  *Menu
	Styler  *Styler
}

// NewMenu creates a new menu
func NewMenu(title string) *Menu {
	return &Menu{
		Title:  title,
		Items:  make([]MenuItem, 0),
		Styler: NewStyler(),
	}
}

// AddItem adds a menu item
func (m *Menu) AddItem(key, label string, action func() error) *Menu {
	m.Items = append(m.Items, MenuItem{
		Key:    key,
		Label:  label,
		Action: action,
	})
	return m
}

// AddSubMenu adds a submenu item
func (m *Menu) AddSubMenu(key, label string, subMenu *Menu) *Menu {
	subMenu.Parent = m
	m.Items = append(m.Items, MenuItem{
		Key:     key,
		Label:   label,
		SubMenu: subMenu,
	})
	return m
}

// Display shows the menu and handles user input
func (m *Menu) Display() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		m.print()

		fmt.Print(m.Styler.Dim("Select an option: "))
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}
		input = strings.TrimSpace(input)

		// Handle back command for submenus
		if input == "b" && m.Parent != nil {
			return nil
		}

		// Handle quit command
		if input == "q" || input == "quit" || input == "exit" {
			fmt.Println(m.Styler.Success("Goodbye!"))
			os.Exit(0)
		}

		// Find and execute the selected item
		for _, item := range m.Items {
			if item.Key == input {
				if item.SubMenu != nil {
					if err := item.SubMenu.Display(); err != nil {
						return err
					}
				} else if item.Action != nil {
					if err := item.Action(); err != nil {
						fmt.Fprintf(os.Stderr, "%s Error: %v\n", m.Styler.Error("✗"), err)
					} else {
						fmt.Println(m.Styler.Success("✓ Done!"))
					}
					fmt.Println()
				}
				break
			}
		}
	}
}

func (m *Menu) print() {
	fmt.Println()
	m.Styler.PrintHeader(m.Title)

	for _, item := range m.Items {
		if item.SubMenu != nil {
			fmt.Printf("  %s %s\n", m.Styler.Info(item.Key), m.Styler.Bold(item.Label)+" →")
		} else {
			fmt.Printf("  %s %s\n", m.Styler.Info(item.Key), item.Label)
		}
	}

	if m.Parent != nil {
		fmt.Printf("  %s %s\n", m.Styler.Dim("b"), m.Styler.Dim("← Back"))
	}
	fmt.Printf("  %s %s\n", m.Styler.Dim("q"), m.Styler.Dim("Quit"))
	fmt.Println()
}

// Confirm prompts user for yes/no confirmation
func Confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	styler := NewStyler()

	for {
		fmt.Printf("%s %s [y/N]: ", styler.Info("?"), prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "y" || input == "yes" {
			return true
		}
		if input == "n" || input == "no" || input == "" {
			return false
		}
		fmt.Println(styler.Warning("Please enter y or n"))
	}
}

// SelectFromList presents a list and lets user select one item
func SelectFromList[T any](title string, items []T, displayFunc func(T) string) (int, error) {
	reader := bufio.NewReader(os.Stdin)
	styler := NewStyler()

	if len(items) == 0 {
		return -1, fmt.Errorf("no items to select from")
	}

	fmt.Println()
	styler.PrintHeader(title)
	for i, item := range items {
		fmt.Printf("  %s %s\n", styler.Info(strconv.Itoa(i+1)), displayFunc(item))
	}
	fmt.Println()

	for {
		fmt.Printf("%s Enter number (1-%d): ", styler.Dim("?"), len(items))
		input, err := reader.ReadString('\n')
		if err != nil {
			return -1, err
		}
		input = strings.TrimSpace(input)

		idx, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println(styler.Warning("Invalid number"))
			continue
		}

		if idx < 1 || idx > len(items) {
			fmt.Printf(styler.Warning("Please enter a number between 1 and %d\n"), len(items))
			continue
		}

		return idx - 1, nil
	}
}

// ProgressBar displays a progress bar
type ProgressBar struct {
	Width   int
	Styler  *Styler
	current int
	total   int
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int) *ProgressBar {
	return &ProgressBar{
		Width:  40,
		Styler: NewStyler(),
		total:  total,
	}
}

// Increment increases the progress by 1
func (p *ProgressBar) Increment() {
	p.current++
	p.Draw()
}

// Set sets the current progress value
func (p *ProgressBar) Set(current int) {
	p.current = current
	p.Draw()
}

// Draw renders the progress bar
func (p *ProgressBar) Draw() {
	if p.total == 0 {
		return
	}

	percent := float64(p.current) / float64(p.total)
	filled := int(float64(p.Width) * percent)

	fmt.Printf("\r[%s%s] %3d%%",
		strings.Repeat("█", filled),
		strings.Repeat("░", p.Width-filled),
		int(percent*100))

	if p.current >= p.total {
		fmt.Println()
	}
}

// Spinner displays an animated spinner
type Spinner struct {
	frames []string
	index  int
	stop   chan bool
}

// NewSpinner creates a new spinner
func NewSpinner() *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stop:   make(chan bool),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start(message string) {
	go func() {
		for {
			select {
			case <-s.stop:
				return
			default:
				fmt.Printf("\r%s %s", s.frames[s.index], message)
				s.index = (s.index + 1) % len(s.frames)
			}
		}
	}()
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	s.stop <- true
	fmt.Printf("\r%s\n", strings.Repeat(" ", 50))
}
