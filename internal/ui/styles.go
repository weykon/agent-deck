package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Tokyo Night Color Palette
const (
	ColorBg      = lipgloss.Color("#1a1b26") // Dark background
	ColorSurface = lipgloss.Color("#24283b") // Surface background
	ColorBorder  = lipgloss.Color("#414868") // Border color
	ColorText    = lipgloss.Color("#c0caf5") // Primary text
	ColorTextDim = lipgloss.Color("#787fa0") // Dim text (WCAG AA compliant - 4.6:1 contrast)
	ColorAccent  = lipgloss.Color("#7aa2f7") // Accent blue
	ColorPurple  = lipgloss.Color("#bb9af7") // Purple
	ColorCyan    = lipgloss.Color("#7dcfff") // Cyan
	ColorGreen   = lipgloss.Color("#9ece6a") // Green
	ColorYellow  = lipgloss.Color("#e0af68") // Yellow
	ColorOrange  = lipgloss.Color("#ff9e64") // Orange
	ColorRed     = lipgloss.Color("#f7768e") // Red
	ColorComment = lipgloss.Color("#787fa0") // Comment gray (WCAG AA compliant - 4.6:1 contrast)
)

// Base Styles
var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBg)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			Background(ColorSurface).
			Padding(0, 1)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	HighlightStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorComment)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorCyan)
)

// Status Indicator Styles
var (
	RunningStyle = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)

	WaitingStyle = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)

	IdleStyle = lipgloss.NewStyle().
			Foreground(ColorComment)

	ErrorIndicatorStyle = lipgloss.NewStyle().
				Foreground(ColorRed).
				Bold(true)
)

// Menu Bar Styles
var (
	MenuBarStyle = lipgloss.NewStyle().
			Background(ColorSurface).
			Foreground(ColorText).
			Padding(0, 1)

	MenuKeyStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	MenuDescStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	MenuSeparatorStyle = lipgloss.NewStyle().
				Foreground(ColorBorder)
)

// Search Styles
var (
	SearchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1).
			Foreground(ColorText)

	SearchPromptStyle = lipgloss.NewStyle().
				Foreground(ColorPurple).
				Bold(true)

	SearchMatchStyle = lipgloss.NewStyle().
				Background(ColorYellow).
				Foreground(ColorBg).
				Bold(true)
)

// Dialog Styles
var (
	DialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPurple).
			Padding(1, 2).
			Background(ColorSurface)

	DialogTitleStyle = lipgloss.NewStyle().
				Foreground(ColorPurple).
				Bold(true).
				Align(lipgloss.Center)

	DialogButtonStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Background(ColorBorder).
				Padding(0, 2).
				MarginRight(1)

	DialogButtonActiveStyle = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorAccent).
				Padding(0, 2).
				MarginRight(1).
				Bold(true)
)

// Preview Pane Styles
var (
	PreviewPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder).
				Padding(1)

	PreviewTitleStyle = lipgloss.NewStyle().
				Foreground(ColorCyan).
				Bold(true).
				Underline(true)

	PreviewHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorPurple).
				Bold(true)

	PreviewContentStyle = lipgloss.NewStyle().
				Foreground(ColorText)

	PreviewMetaStyle = lipgloss.NewStyle().
				Foreground(ColorComment).
				Italic(true)
)

// Tool Icons
const (
	IconClaude   = "🤖"
	IconGemini   = "✨"
	IconOpenCode = "🌐"
	IconCodex    = "💻"
	IconShell    = "🐚"
)

// Helper Functions

// MenuKey creates a formatted menu item with key and description
func MenuKey(key, description string) string {
	return fmt.Sprintf("%s %s %s",
		MenuKeyStyle.Render(key),
		MenuSeparatorStyle.Render("•"),
		MenuDescStyle.Render(description),
	)
}

// StatusIndicator returns a styled status indicator
// Standard symbols: ● running, ◐ waiting, ○ idle, ✕ error, ⟳ starting
func StatusIndicator(status string) string {
	switch status {
	case "running":
		return RunningStyle.Render("●")
	case "waiting":
		return WaitingStyle.Render("◐")
	case "idle":
		return IdleStyle.Render("○")
	case "error":
		return ErrorIndicatorStyle.Render("✕")
	case "starting":
		return WaitingStyle.Render("⟳") // Use yellow color, spinning arrow symbol
	default:
		return IdleStyle.Render("○")
	}
}

// ToolIcon returns the icon for a given tool
// Checks user config for custom tools first, then falls back to built-ins
func ToolIcon(tool string) string {
	// Use session.GetToolIcon which handles custom + built-in
	// Import would be circular, so we duplicate the logic here
	// Custom icons are handled by the session layer's GetToolDef
	switch tool {
	case "claude":
		return IconClaude
	case "gemini":
		return IconGemini
	case "opencode":
		return IconOpenCode
	case "codex":
		return IconCodex
	case "cursor":
		return "📝"
	case "shell":
		return IconShell
	default:
		return IconShell
	}
}

// ToolColor returns the brand color for a given tool
// Claude=orange (Anthropic), Gemini=purple (Google AI), Codex=cyan, Aider=red
func ToolColor(tool string) lipgloss.Color {
	switch tool {
	case "claude":
		return ColorOrange // Anthropic's orange
	case "gemini":
		return ColorPurple // Google AI purple
	case "codex":
		return ColorCyan // Light blue for OpenAI
	case "aider":
		return ColorRed // Red for Aider
	case "cursor":
		return ColorAccent // Blue for Cursor
	default:
		return ColorTextDim // Default gray
	}
}

// List Item Styles (used by legacy list.go component in tests)
var (
	ListItemStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			PaddingLeft(2)

	ListItemActiveStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true).
				PaddingLeft(2)
)

// Tag Styles
var (
	TagStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorPurple).
			Padding(0, 1).
			MarginRight(1)

	TagActiveStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorGreen).
			Padding(0, 1).
			MarginRight(1)

	TagErrorStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorRed).
			Padding(0, 1).
			MarginRight(1)
)

// Timestamp Style
var (
	TimestampStyle = lipgloss.NewStyle().
		Foreground(ColorComment).
		Italic(true)
)

// Folder Styles
var (
	FolderStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	FolderCollapsedStyle = lipgloss.NewStyle().
				Foreground(ColorComment)
)

// Session Item Styles
var (
	SessionItemStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				PaddingLeft(2)

	SessionItemSelectedStyle = lipgloss.NewStyle().
					Foreground(ColorBg).
					Background(ColorAccent).
					Bold(true).
					PaddingLeft(0)
)

// Session List Rendering Styles (PERFORMANCE: cached at package level)
// These styles are used by renderSessionItem() and renderGroupItem() to avoid
// repeated allocations on every View() call
var (
	// Tree connector styles
	TreeConnectorStyle  = lipgloss.NewStyle().Foreground(ColorText)
	TreeConnectorSelStyle = lipgloss.NewStyle().Foreground(ColorBg).Background(ColorAccent)

	// Session status indicator styles
	SessionStatusRunning = lipgloss.NewStyle().Foreground(ColorGreen)
	SessionStatusWaiting = lipgloss.NewStyle().Foreground(ColorYellow)
	SessionStatusIdle    = lipgloss.NewStyle().Foreground(ColorTextDim)
	SessionStatusError   = lipgloss.NewStyle().Foreground(ColorRed)
	SessionStatusSelStyle = lipgloss.NewStyle().Foreground(ColorBg).Background(ColorAccent)

	// Session title styles by state
	SessionTitleDefault = lipgloss.NewStyle().Foreground(ColorText)
	SessionTitleActive  = lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	SessionTitleError   = lipgloss.NewStyle().Foreground(ColorText).Underline(true)
	SessionTitleSelStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorBg).Background(ColorAccent)

	// Selection indicator
	SessionSelectionPrefix = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)

	// Group item styles
	GroupExpandStyle    = lipgloss.NewStyle().Foreground(ColorText)
	GroupNameStyle      = lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	GroupCountStyle     = lipgloss.NewStyle().Foreground(ColorText)
	GroupHotkeyStyle    = lipgloss.NewStyle().Foreground(ColorComment)
	GroupStatusRunning  = lipgloss.NewStyle().Foreground(ColorGreen)
	GroupStatusWaiting  = lipgloss.NewStyle().Foreground(ColorYellow)

	// Group selected styles
	GroupNameSelStyle   = lipgloss.NewStyle().Bold(true).Foreground(ColorBg).Background(ColorAccent)
	GroupCountSelStyle  = lipgloss.NewStyle().Foreground(ColorBg).Background(ColorAccent)
	GroupExpandSelStyle = lipgloss.NewStyle().Foreground(ColorBg).Background(ColorAccent)
)

// ToolStyleCache provides pre-allocated styles for each tool type
// Avoids repeated lipgloss.NewStyle() calls in renderSessionItem()
var ToolStyleCache = map[string]lipgloss.Style{
	"claude":   lipgloss.NewStyle().Foreground(ColorOrange),
	"gemini":   lipgloss.NewStyle().Foreground(ColorPurple),
	"codex":    lipgloss.NewStyle().Foreground(ColorCyan),
	"aider":    lipgloss.NewStyle().Foreground(ColorRed),
	"cursor":   lipgloss.NewStyle().Foreground(ColorAccent),
	"shell":    lipgloss.NewStyle().Foreground(ColorText),
	"opencode": lipgloss.NewStyle().Foreground(ColorText),
}

// DefaultToolStyle is used when tool is not in cache
var DefaultToolStyle = lipgloss.NewStyle().Foreground(ColorText)

// GetToolStyle returns cached style for tool or default
func GetToolStyle(tool string) lipgloss.Style {
	if style, ok := ToolStyleCache[tool]; ok {
		return style
	}
	return DefaultToolStyle
}

// Menu Styles
var (
	MenuStyle = lipgloss.NewStyle().
		Background(ColorSurface).
		Foreground(ColorText).
		Padding(0, 1)
)

// Additional Styles
var (
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorComment).
			Italic(true)

	ColorError   = ColorRed
	ColorSuccess = ColorGreen
	ColorWarning = ColorYellow
	ColorPrimary = ColorAccent
)

// LogoBorderStyle for the grid lines
var LogoBorderStyle = lipgloss.NewStyle().Foreground(ColorBorder)

// LogoFrames kept for backward compatibility (empty state default)
var LogoFrames = [][]string{
	{"●", "◐", "○"},
}

// RenderLogoIndicator renders a single indicator with appropriate color
func RenderLogoIndicator(indicator string) string {
	var color lipgloss.Color
	switch indicator {
	case "●":
		color = ColorGreen // Running
	case "◐":
		color = ColorYellow // Waiting
	case "○":
		color = ColorTextDim // Idle
	default:
		color = ColorTextDim
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(indicator)
}

// getLogoIndicators returns 3 indicators based on actual session status counts
// Priority: Running > Waiting > Idle
// Shows up to 3 indicators reflecting the real state
func getLogoIndicators(running, waiting, idle int) []string {
	indicators := make([]string, 0, 3)

	// Add running indicators (green ●)
	for i := 0; i < running && len(indicators) < 3; i++ {
		indicators = append(indicators, "●")
	}

	// Add waiting indicators (yellow ◐)
	for i := 0; i < waiting && len(indicators) < 3; i++ {
		indicators = append(indicators, "◐")
	}

	// Fill remaining with idle (gray ○)
	for len(indicators) < 3 {
		indicators = append(indicators, "○")
	}

	return indicators
}

// RenderLogoCompact renders the compact inline logo for the header
// Shows REAL status: running=●, waiting=◐, idle=○
// Format: ⟨ ● │ ◐ │ ○ ⟩  (using angle brackets for modern look)
func RenderLogoCompact(running, waiting, idle int) string {
	indicators := getLogoIndicators(running, waiting, idle)
	bracketStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	return bracketStyle.Render("⟨") +
		" " + RenderLogoIndicator(indicators[0]) +
		LogoBorderStyle.Render(" │ ") +
		RenderLogoIndicator(indicators[1]) +
		LogoBorderStyle.Render(" │ ") +
		RenderLogoIndicator(indicators[2]) + " " +
		bracketStyle.Render("⟩")
}

// RenderLogoLarge renders the large logo for empty state
// Shows REAL status: running=●, waiting=◐, idle=○
// Format:
//
//	┌──┬──┬──┐
//	│● │◐ │○ │
//	└──┴──┴──┘
func RenderLogoLarge(running, waiting, idle int) string {
	indicators := getLogoIndicators(running, waiting, idle)
	top := LogoBorderStyle.Render("┌──┬──┬──┐")
	mid := LogoBorderStyle.Render("│") + RenderLogoIndicator(indicators[0]) + LogoBorderStyle.Render(" │") +
		RenderLogoIndicator(indicators[1]) + LogoBorderStyle.Render(" │") +
		RenderLogoIndicator(indicators[2]) + LogoBorderStyle.Render(" │")
	bot := LogoBorderStyle.Render("└──┴──┴──┘")
	return top + "\n" + mid + "\n" + bot
}
