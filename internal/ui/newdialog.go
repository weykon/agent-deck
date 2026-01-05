package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NewDialog represents the new session creation dialog
type NewDialog struct {
	nameInput            textinput.Model
	pathInput            textinput.Model
	commandInput         textinput.Model
	focusIndex           int
	width                int
	height               int
	visible              bool
	presetCommands       []string
	commandCursor        int
	parentGroupPath      string
	parentGroupName      string
	pathSuggestions      []string // stores all available path suggestions
	pathSuggestionCursor int      // tracks selected suggestion in dropdown
	pathSuggestionSource  string   // "recent" or "autocomplete"
	pathSuggestionOffset int      // scroll offset for displaying suggestions
}

// NewNewDialog creates a new NewDialog instance
func NewNewDialog() *NewDialog {
	// Create name input
	nameInput := textinput.New()
	nameInput.Placeholder = "session-name"
	nameInput.Focus()
	nameInput.CharLimit = 100
	nameInput.Width = 40

	// Create path input
	pathInput := textinput.New()
	pathInput.Placeholder = "~/project/path"
	pathInput.CharLimit = 256
	pathInput.Width = 40
	pathInput.ShowSuggestions = true // enable built-in suggestions

	// Get current working directory for default path
	cwd, err := os.Getwd()
	if err == nil {
		pathInput.SetValue(cwd)
	}

	// Create command input
	commandInput := textinput.New()
	commandInput.Placeholder = "custom command"
	commandInput.CharLimit = 100
	commandInput.Width = 40

	return &NewDialog{
		nameInput:       nameInput,
		pathInput:       pathInput,
		commandInput:    commandInput,
		focusIndex:      0,
		visible:         false,
		presetCommands:  []string{"", "claude", "gemini", "opencode", "codex"},
		commandCursor:   0,
		parentGroupPath: "default",
		parentGroupName: "default",
	}
}

// ShowInGroup shows the dialog with a pre-selected parent group
func (d *NewDialog) ShowInGroup(groupPath, groupName string) {
	if groupPath == "" {
		groupPath = "default"
		groupName = "default"
	}
	d.parentGroupPath = groupPath
	d.parentGroupName = groupName
	d.visible = true
	d.focusIndex = 0
	d.nameInput.SetValue("")
	d.nameInput.Focus()
	// Keep commandCursor at previously set default (don't reset to 0)

	// Clear suggestion state when showing dialog
	d.pathSuggestions = []string{}
	d.pathSuggestionCursor = 0
	d.pathSuggestionOffset = 0
	d.pathSuggestionSource = ""
}

// SetDefaultTool sets the pre-selected command based on tool name
// Call this before Show/ShowInGroup to apply user's preferred default
func (d *NewDialog) SetDefaultTool(tool string) {
	if tool == "" {
		d.commandCursor = 0 // Default to shell
		return
	}

	// Find the tool in preset commands
	for i, cmd := range d.presetCommands {
		if cmd == tool {
			d.commandCursor = i
			return
		}
	}

	// Tool not found in presets, default to shell
	d.commandCursor = 0
}

// GetSelectedGroup returns the parent group path
func (d *NewDialog) GetSelectedGroup() string {
	return d.parentGroupPath
}

// SetSize sets the dialog dimensions
func (d *NewDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetPathSuggestions sets the available path suggestions for autocomplete
func (d *NewDialog) SetPathSuggestions(paths []string) {
	d.pathSuggestions = paths
	d.pathSuggestionCursor = 0
	d.pathSuggestionOffset = 0
	d.pathSuggestionSource = "recent"
	d.pathInput.SetSuggestions(paths)
}

// Show makes the dialog visible (uses default group)
func (d *NewDialog) Show() {
	d.ShowInGroup("default", "default")
}

// Hide hides the dialog
func (d *NewDialog) Hide() {
	d.visible = false
}

// IsVisible returns whether the dialog is visible
func (d *NewDialog) IsVisible() bool {
	return d.visible
}

// GetValues returns the current dialog values with expanded paths
func (d *NewDialog) GetValues() (name, path, command string) {
	name = strings.TrimSpace(d.nameInput.Value())
	path = strings.TrimSpace(d.pathInput.Value())

	// Fix malformed paths that have ~ in the middle (e.g., "/some/path~/actual/path")
	// This can happen when textinput suggestion appends instead of replaces
	if idx := strings.Index(path, "~/"); idx > 0 {
		// Extract the part after the malformed prefix (the actual tilde-prefixed path)
		path = path[idx:]
	}

	// Expand tilde in path (handles both "~/" prefix and just "~")
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	} else if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			path = home
		}
	}

	// Get command - either from preset or custom input
	if d.commandCursor < len(d.presetCommands) {
		command = d.presetCommands[d.commandCursor]
	}
	if command == "" && d.commandInput.Value() != "" {
		command = strings.TrimSpace(d.commandInput.Value())
	}

	return name, path, command
}

// Validate checks if the dialog values are valid and returns an error message if not
func (d *NewDialog) Validate() string {
	name := strings.TrimSpace(d.nameInput.Value())
	path := strings.TrimSpace(d.pathInput.Value())

	// Check for empty name
	if name == "" {
		return "Session name cannot be empty"
	}

	// Check name length
	if len(name) > 50 {
		return "Session name too long (max 50 characters)"
	}

	// Check for empty path
	if path == "" {
		return "Project path cannot be empty"
	}

	return "" // Valid
}

// tryCompletePath attempts to autocomplete current path input
// Returns true if path was completed, false otherwise
func (d *NewDialog) tryCompletePath(currentPath string) bool {
	// Expand tilde for path matching
	searchPath := currentPath
	if strings.HasPrefix(searchPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		searchPath = filepath.Join(home, searchPath[2:])
	} else if searchPath == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		searchPath = home
	}

	// Get directory and prefix
	dir, prefix := filepath.Split(searchPath)
	if dir == "" {
		dir = "."
	}

	// Read directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	// Find matches
	var matches []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}

	// No matches
	if len(matches) == 0 {
		return false
	}

	// Single match: auto-complete
	if len(matches) == 1 {
		completedPath := filepath.Join(dir, matches[0])
		// Add trailing slash if it's a directory
		if info, err := os.Stat(completedPath); err == nil && info.IsDir() {
			completedPath += "/"
		}
		// Convert back to ~ format if in home directory
		home, _ := os.UserHomeDir()
		if home != "" && strings.HasPrefix(completedPath, home) {
			completedPath = "~" + completedPath[len(home):]
		}
		d.pathInput.SetValue(completedPath)
		d.pathSuggestionSource = "" // Clear suggestions after completion
		return true
	}

	// Multiple matches: always show suggestions list (more intuitive)
	// Don't complete to common prefix first - show all matches directly
	d.pathSuggestions = matches
	d.pathSuggestionCursor = 0
	d.pathSuggestionOffset = 0
	d.pathSuggestionSource = "autocomplete" // Mark as autocomplete source
	return true
}

// updatePathSuggestions updates file suggestions based on current path input
func (d *NewDialog) updatePathSuggestions(currentPath string) {
	// Expand tilde for path matching
	searchPath := currentPath
	if strings.HasPrefix(searchPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		searchPath = filepath.Join(home, searchPath[2:])
	} else if searchPath == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		searchPath = home + "/"
	}

	// Get directory and prefix
	dir, prefix := filepath.Split(searchPath)
	if dir == "" {
		dir = "."
	}

	// Read directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		d.pathSuggestions = nil
		d.pathInput.SetSuggestions(nil)
		return
	}

	// Find matches (only directories for path completion)
	var matches []string
	home, _ := os.UserHomeDir()
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			fullPath := filepath.Join(dir, name)
			// Convert to ~ format if in home directory
			if home != "" && strings.HasPrefix(fullPath, home) {
				fullPath = "~" + fullPath[len(home):]
			}
			if entry.IsDir() {
				fullPath += "/"
			}
			matches = append(matches, fullPath)
		}
	}

	d.pathSuggestions = matches
	d.pathSuggestionCursor = 0
	d.pathSuggestionOffset = 0
	d.pathSuggestionSource = "autocomplete"
	d.pathInput.SetSuggestions(matches)
}

// findCommonPrefix finds the common prefix among strings
func findCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	minLen := len(strs[0])
	for _, s := range strs[1:] {
		if len(s) < minLen {
			minLen = len(s)
		}
	}

	prefix := ""
	for i := 0; i < minLen; i++ {
		c := strs[0][i]
		allMatch := true
		for _, s := range strs[1:] {
			if s[i] != c {
				allMatch = false
				break
			}
		}
		if allMatch {
			prefix += string(c)
		} else {
			break
		}
	}

	return prefix
}

// scrollSuggestionsUp scrolls up through the suggestion list
func (d *NewDialog) scrollSuggestionsUp() {
	if len(d.pathSuggestions) == 0 {
		return
	}
	if d.pathSuggestionCursor > 0 {
		d.pathSuggestionCursor--
		// If cursor moves above current view, scroll up
		if d.pathSuggestionCursor < d.pathSuggestionOffset {
			d.pathSuggestionOffset = d.pathSuggestionCursor
		}
	}
}

// scrollSuggestionsDown scrolls down through the suggestion list
func (d *NewDialog) scrollSuggestionsDown() {
	if len(d.pathSuggestions) == 0 {
		return
	}
	if d.pathSuggestionCursor < len(d.pathSuggestions)-1 {
		d.pathSuggestionCursor++
		// If cursor moves below current view, scroll down
		const maxVisible = 10
		maxOffset := max(0, len(d.pathSuggestions)-maxVisible)
		if d.pathSuggestionCursor >= d.pathSuggestionOffset+maxVisible {
			d.pathSuggestionOffset = min(d.pathSuggestionCursor-9, maxOffset)
		}
	}
}

// updateFocus updates which input has focus
func (d *NewDialog) updateFocus() {
	d.nameInput.Blur()
	d.pathInput.Blur()
	d.commandInput.Blur()

	switch d.focusIndex {
	case 0:
		d.nameInput.Focus()
	case 1:
		d.pathInput.Focus()
	case 2:
		// Command selection (no text input focus needed for presets)
	}
}

// Update handles key messages
func (d *NewDialog) Update(msg tea.Msg) (*NewDialog, tea.Cmd) {
	if !d.visible {
		return d, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// On path field: try path completion first
			if d.focusIndex == 1 {
				currentPath := strings.TrimSpace(d.pathInput.Value())
				completed := d.tryCompletePath(currentPath)
				if completed {
					return d, nil
				}
				// If no completion, try to use preset suggestions
				if len(d.pathSuggestions) > 0 {
					if d.pathSuggestionCursor < len(d.pathSuggestions) {
						d.pathInput.SetValue(d.pathSuggestions[d.pathSuggestionCursor])
					}
				}
			}
			// Move to next field
			d.focusIndex = (d.focusIndex + 1) % 3
			d.updateFocus()
			return d, cmd

		case "ctrl+n":
			// Next suggestion (when on path field)
			if d.focusIndex == 1 && len(d.pathSuggestions) > 0 {
				d.scrollSuggestionsDown()
				return d, nil
			}

		case "ctrl+p":
			// Previous suggestion (when on path field)
			if d.focusIndex == 1 && len(d.pathSuggestions) > 0 {
				d.scrollSuggestionsUp()
				return d, nil
			}

		case "down":
			// On path field with suggestions: scroll down
			if d.focusIndex == 1 && len(d.pathSuggestions) > 0 {
				d.scrollSuggestionsDown()
				return d, nil
			}
			// Otherwise navigate fields
			d.focusIndex = (d.focusIndex + 1) % 3
			d.updateFocus()
			return d, nil

		case "up":
			// On path field with suggestions: scroll up
			if d.focusIndex == 1 && len(d.pathSuggestions) > 0 {
				d.scrollSuggestionsUp()
				return d, nil
			}
			// Otherwise navigate fields (shift+tab behavior)
			d.focusIndex--
			if d.focusIndex < 0 {
				d.focusIndex = 2
			}
			d.updateFocus()
			return d, nil

		case "shift+tab":
			d.focusIndex--
			if d.focusIndex < 0 {
				d.focusIndex = 2
			}
			d.updateFocus()
			return d, nil

		case "esc":
			d.Hide()
			return d, nil

		case "enter":
			// Let parent handle enter (create session)
			return d, nil

		case "left":
			// Command selection
			if d.focusIndex == 2 {
				d.commandCursor--
				if d.commandCursor < 0 {
					d.commandCursor = len(d.presetCommands) - 1
				}
				return d, nil
			}

		case "right":
			// Command selection
			if d.focusIndex == 2 {
				d.commandCursor = (d.commandCursor + 1) % len(d.presetCommands)
				return d, nil
			}
		}
	}

	// Update focused input
	switch d.focusIndex {
	case 0:
		d.nameInput, cmd = d.nameInput.Update(msg)
	case 1:
		oldPath := d.pathInput.Value()
		d.pathInput, cmd = d.pathInput.Update(msg)
		newPath := d.pathInput.Value()
		// When path input changes, update file suggestions
		if oldPath != newPath && newPath != "" {
			d.updatePathSuggestions(newPath)
		}
	}

	return d, cmd
}

// View renders the dialog
func (d *NewDialog) View() string {
	if !d.visible {
		return ""
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorCyan).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorText)

	// Responsive dialog width
	dialogWidth := 60
	if d.width > 0 && d.width < dialogWidth+10 {
		dialogWidth = d.width - 10
		if dialogWidth < 40 {
			dialogWidth = 40
		}
	}

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorCyan).
		Background(ColorSurface).
		Padding(2, 4).
		Width(dialogWidth)

	// Active field indicator style
	activeLabelStyle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true)

	// Build content
	var content strings.Builder

	// Title with parent group info
	content.WriteString(titleStyle.Render("New Session"))
	content.WriteString("\n")
	groupInfoStyle := lipgloss.NewStyle().Foreground(ColorPurple) // Purple for group context
	content.WriteString(groupInfoStyle.Render("  in group: " + d.parentGroupName))
	content.WriteString("\n\n")

	// Name input
	if d.focusIndex == 0 {
		content.WriteString(activeLabelStyle.Render("▶ Name:"))
	} else {
		content.WriteString(labelStyle.Render("  Name:"))
	}
	content.WriteString("\n")
	content.WriteString("  ")
	content.WriteString(d.nameInput.View())
	content.WriteString("\n\n")

	// Path input
	if d.focusIndex == 1 {
		label := "▶ Path:"
		if len(d.pathSuggestions) > 0 {
			label += " (Tab: 自动补全)"
		}
		content.WriteString(activeLabelStyle.Render(label))
	} else {
		content.WriteString(labelStyle.Render("  Path:"))
	}
	content.WriteString("\n")
	content.WriteString("  ")
	content.WriteString(d.pathInput.View())
	content.WriteString("\n")

	// Show path suggestions dropdown when path field is focused
	if d.focusIndex == 1 && len(d.pathSuggestions) > 0 {
		suggestionStyle := lipgloss.NewStyle().
			Foreground(ColorComment)
		selectedStyle := lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true)

		// Show up to 10 suggestions (increased from 5)
		maxShow := 10
		if len(d.pathSuggestions) < maxShow {
			maxShow = len(d.pathSuggestions)
		}

		// Display different titles based on source
		var title string
		if d.pathSuggestionSource == "autocomplete" {
			title = fmt.Sprintf("─ Tab补全 (%d 个匹配) ─", len(d.pathSuggestions))
		} else {
			title = fmt.Sprintf("─ 最近路径 (%d 个) ─", len(d.pathSuggestions))
		}

		content.WriteString("  ")
		content.WriteString(lipgloss.NewStyle().Foreground(ColorComment).Render(title))
		content.WriteString("\n")

		// Calculate display range based on scroll offset
		startIdx := d.pathSuggestionOffset
		endIdx := min(startIdx+maxShow, len(d.pathSuggestions))

		for i := startIdx; i < endIdx; i++ {
			style := suggestionStyle
			prefix := "    "
			if i == d.pathSuggestionCursor {
				style = selectedStyle
				prefix = "  ▶ "
			}
			content.WriteString(style.Render(prefix + d.pathSuggestions[i]))
			content.WriteString("\n")
		}

		// Show scroll hints if there are more items
		if len(d.pathSuggestions) > maxShow {
			if d.pathSuggestionOffset > 0 {
				content.WriteString(suggestionStyle.Render("    ↑ 向上滚动显示更多"))
				content.WriteString("\n")
			}
			if d.pathSuggestionOffset+maxShow < len(d.pathSuggestions) {
				remaining := len(d.pathSuggestions) - d.pathSuggestionOffset - maxShow
				content.WriteString(suggestionStyle.Render(fmt.Sprintf("    ↓ 向下滚动 (还有 %d 个)", remaining)))
				content.WriteString("\n")
			}
		}

		// Display operation hints based on source
		if d.pathSuggestionSource == "autocomplete" {
			content.WriteString(suggestionStyle.Render("    Ctrl+N/P 或 ↑↓: 切换  Tab: 选择"))
		} else {
			content.WriteString(suggestionStyle.Render("    Ctrl+N/P 或 ↑↓: 切换  Tab: 选择"))
		}
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Command selection
	if d.focusIndex == 2 {
		content.WriteString(activeLabelStyle.Render("▶ Command:"))
	} else {
		content.WriteString(labelStyle.Render("  Command:"))
	}
	content.WriteString("\n  ")

	// Render command options as consistent pill buttons
	var cmdButtons []string
	for i, cmd := range d.presetCommands {
		displayName := cmd
		if displayName == "" {
			displayName = "shell"
		}

		var btnStyle lipgloss.Style
		if i == d.commandCursor {
			// Selected: bright background, bold (active pill)
			btnStyle = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorAccent).
				Bold(true).
				Padding(0, 2)
		} else {
			// Unselected: subtle background pill (consistent style)
			btnStyle = lipgloss.NewStyle().
				Foreground(ColorTextDim).
				Background(ColorSurface).
				Padding(0, 2)
		}

		cmdButtons = append(cmdButtons, btnStyle.Render(displayName))
	}
	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, cmdButtons...))
	content.WriteString("\n\n")

	// Custom command input (only if shell is selected)
	if d.commandCursor == 0 {
		content.WriteString(labelStyle.Render("  Custom command:"))
		content.WriteString("\n  ")
		content.WriteString(d.commandInput.View())
		content.WriteString("\n\n")
	}

	// Help text with better contrast
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorComment). // Use consistent theme color
		MarginTop(1)
	content.WriteString(helpStyle.Render("Tab next/accept │ ↑↓ navigate │ Enter create │ Esc cancel"))

	// Wrap in dialog box
	dialog := dialogStyle.Render(content.String())

	// Center the dialog
	return lipgloss.Place(
		d.width,
		d.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}
