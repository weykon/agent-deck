package tmux

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Debug flag - set via environment variable AGENTDECK_DEBUG=1
var debugStatusEnabled = os.Getenv("AGENTDECK_DEBUG") == "1"

func debugLog(format string, args ...interface{}) {
	if debugStatusEnabled {
		log.Printf("[STATUS] "+format, args...)
	}
}

const SessionPrefix = "agentdeck_"

// IsTmuxAvailable checks if tmux is installed and accessible
// Returns nil if tmux is available, otherwise returns an error with details
func IsTmuxAvailable() error {
	cmd := exec.Command("tmux", "-V")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux not found or not working: %w (output: %s)", err, string(output))
	}
	return nil
}

// TerminalInfo contains detected terminal information
type TerminalInfo struct {
	Name           string // Terminal name (warp, iterm2, kitty, alacritty, etc.)
	SupportsOSC8   bool   // Supports OSC 8 hyperlinks
	SupportsOSC52  bool   // Supports OSC 52 clipboard
	SupportsTrueColor bool // Supports 24-bit color
}

// DetectTerminal identifies the current terminal emulator from environment variables
// Returns terminal name: "warp", "iterm2", "kitty", "alacritty", "vscode", "windows-terminal", or "unknown"
func DetectTerminal() string {
	// Check terminal-specific environment variables (most reliable)

	// Warp Terminal
	if os.Getenv("TERM_PROGRAM") == "WarpTerminal" || os.Getenv("WARP_IS_LOCAL_SHELL_SESSION") != "" {
		return "warp"
	}

	// iTerm2
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" || os.Getenv("ITERM_SESSION_ID") != "" {
		return "iterm2"
	}

	// kitty
	if os.Getenv("TERM") == "xterm-kitty" || os.Getenv("KITTY_WINDOW_ID") != "" {
		return "kitty"
	}

	// Alacritty
	if os.Getenv("ALACRITTY_SOCKET") != "" || os.Getenv("ALACRITTY_LOG") != "" {
		return "alacritty"
	}

	// VS Code integrated terminal
	if os.Getenv("TERM_PROGRAM") == "vscode" || os.Getenv("VSCODE_INJECTION") != "" {
		return "vscode"
	}

	// Windows Terminal
	if os.Getenv("WT_SESSION") != "" {
		return "windows-terminal"
	}

	// WezTerm
	if os.Getenv("TERM_PROGRAM") == "WezTerm" || os.Getenv("WEZTERM_PANE") != "" {
		return "wezterm"
	}

	// Apple Terminal.app
	if os.Getenv("TERM_PROGRAM") == "Apple_Terminal" {
		return "apple-terminal"
	}

	// Hyper
	if os.Getenv("TERM_PROGRAM") == "Hyper" {
		return "hyper"
	}

	// Check TERM_PROGRAM as fallback
	if termProgram := os.Getenv("TERM_PROGRAM"); termProgram != "" {
		return strings.ToLower(termProgram)
	}

	return "unknown"
}

// GetTerminalInfo returns detailed terminal capabilities
func GetTerminalInfo() TerminalInfo {
	terminal := DetectTerminal()

	info := TerminalInfo{
		Name:           terminal,
		SupportsOSC8:   false,
		SupportsOSC52:  false,
		SupportsTrueColor: false,
	}

	// Check COLORTERM for true color support
	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		info.SupportsTrueColor = true
	}

	// Set capabilities based on terminal
	// Reference: https://github.com/Alhadis/OSC8-Adoption
	switch terminal {
	case "warp":
		// Warp: Full modern terminal support
		info.SupportsOSC8 = true   // Native clickable paths
		info.SupportsOSC52 = true  // Clipboard integration
		info.SupportsTrueColor = true

	case "iterm2":
		// iTerm2: Excellent escape sequence support
		info.SupportsOSC8 = true
		info.SupportsOSC52 = true
		info.SupportsTrueColor = true

	case "kitty":
		// kitty: Full modern terminal support
		info.SupportsOSC8 = true
		info.SupportsOSC52 = true
		info.SupportsTrueColor = true

	case "alacritty":
		// Alacritty: OSC 8 since v0.11, OSC 52 supported
		info.SupportsOSC8 = true
		info.SupportsOSC52 = true
		info.SupportsTrueColor = true

	case "wezterm":
		// WezTerm: Full support
		info.SupportsOSC8 = true
		info.SupportsOSC52 = true
		info.SupportsTrueColor = true

	case "windows-terminal":
		// Windows Terminal: OSC 8 since v1.4
		info.SupportsOSC8 = true
		info.SupportsOSC52 = true
		info.SupportsTrueColor = true

	case "vscode":
		// VS Code: OSC 8 supported in integrated terminal
		info.SupportsOSC8 = true
		info.SupportsOSC52 = true
		info.SupportsTrueColor = true

	case "hyper":
		// Hyper: Limited OSC support
		info.SupportsOSC8 = false
		info.SupportsOSC52 = true
		info.SupportsTrueColor = true

	case "apple-terminal":
		// Apple Terminal.app: No OSC 8 support
		info.SupportsOSC8 = false
		info.SupportsOSC52 = false
		info.SupportsTrueColor = false

	default:
		// Unknown terminal - assume basic support
		// Most modern terminals support these features
		info.SupportsOSC8 = true  // Optimistic default
		info.SupportsOSC52 = true
	}

	return info
}

// SupportsHyperlinks returns true if the current terminal supports OSC 8 hyperlinks
func SupportsHyperlinks() bool {
	return GetTerminalInfo().SupportsOSC8
}

// Tool detection patterns (used by DetectTool for initial tool identification)
var toolDetectionPatterns = map[string][]*regexp.Regexp{
	"claude": {
		regexp.MustCompile(`(?i)claude`),
		regexp.MustCompile(`(?i)anthropic`),
	},
	"gemini": {
		regexp.MustCompile(`(?i)gemini`),
		regexp.MustCompile(`(?i)google ai`),
	},
	"aider": {
		regexp.MustCompile(`(?i)aider`),
	},
	"codex": {
		regexp.MustCompile(`(?i)codex`),
		regexp.MustCompile(`(?i)openai`),
	},
}

// StateTracker tracks content changes for notification-style status detection
//
// StateTracker implements a simple 3-state model:
//
//	GREEN (active)   = Content changed within 2 seconds
//	YELLOW (waiting) = Content stable, user hasn't seen it
//	GRAY (idle)      = Content stable, user has seen it
type StateTracker struct {
	lastHash              string    // SHA256 of normalized content (for fallback)
	lastChangeTime        time.Time // When sustained activity was last confirmed
	acknowledged          bool      // User has seen this state (yellow vs gray)
	lastActivityTimestamp int64     // tmux window_activity timestamp for spike detection

	// Non-blocking spike detection: track changes across tick cycles
	activityCheckStart time.Time // When we started tracking for sustained activity
	activityChangeCount int      // How many timestamp changes seen in current window
}

// activityCooldown is how long to show GREEN after content stops changing.
// This prevents flickering during natural micro-pauses in AI output.
// - 2 seconds: Covers most pauses between output bursts
// - 3 seconds: More conservative, fewer false yellows
const activityCooldown = 2 * time.Second

// Session represents a tmux session
// NOTE: All mutable fields are protected by mu. The Bubble Tea event loop is single-threaded,
// but we use mutex protection for defensive programming and future-proofing.
type Session struct {
	Name        string
	DisplayName string
	WorkDir     string
	Command     string
	Created     time.Time

	// mu protects all mutable fields below from concurrent access
	mu sync.Mutex

	// Content tracking for HasUpdated (separate from StateTracker)
	lastHash    string
	lastContent string

	// Cached tool detection (avoids re-detecting every status check)
	detectedTool     string
	toolDetectedAt   time.Time
	toolDetectExpiry time.Duration // How long before re-detecting (default 30s)

	// Simple state tracking (hash-based)
	stateTracker *StateTracker

	// Last status returned (for debugging)
	lastStableStatus string
}

// ensureStateTrackerLocked lazily allocates the tracker so callers can safely
// acknowledge even before the first GetStatus call.
// MUST be called with mu held.
func (s *Session) ensureStateTrackerLocked() {
	if s.stateTracker == nil {
		s.stateTracker = &StateTracker{
			lastHash:       "",
			lastChangeTime: time.Now().Add(-activityCooldown),
			acknowledged:   false,
		}
	}
}

// LogFile returns the path to this session's pipe-pane log file
// Logs are stored in ~/.agent-deck/logs/<session-name>.log
func (s *Session) LogFile() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	logDir := filepath.Join(homeDir, ".agent-deck", "logs")
	return filepath.Join(logDir, s.Name+".log")
}

// LogDir returns the directory containing all session logs
func LogDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	return filepath.Join(homeDir, ".agent-deck", "logs")
}

// NewSession creates a new Session instance with a unique name
func NewSession(name, workDir string) *Session {
	sanitized := sanitizeName(name)
	// Add unique suffix to prevent name collisions
	uniqueSuffix := generateShortID()
	return &Session{
		Name:             SessionPrefix + sanitized + "_" + uniqueSuffix,
		DisplayName:      name,
		WorkDir:          workDir,
		Created:          time.Now(),
		lastStableStatus: "waiting",
		toolDetectExpiry: 30 * time.Second, // Re-detect tool every 30 seconds
		// stateTracker and promptDetector will be created lazily on first status check
	}
}

// ReconnectSession creates a Session object for an existing tmux session
// This is used when loading sessions from storage - it properly initializes
// all fields needed for status detection to work correctly
func ReconnectSession(tmuxName, displayName, workDir, command string) *Session {
	sess := &Session{
		Name:             tmuxName,
		DisplayName:      displayName,
		WorkDir:          workDir,
		Command:          command,
		Created:          time.Now(), // Approximate - we don't persist this
		lastStableStatus: "waiting",
		toolDetectExpiry: 30 * time.Second,
		// stateTracker and promptDetector will be created lazily on first status check
	}

	// Enable pipe-pane for event-driven status detection
	if sess.Exists() {
		if err := sess.EnablePipePane(); err != nil {
			debugLog("Warning: failed to enable pipe-pane for %s: %v", tmuxName, err)
		}
		// Configure status bar for existing sessions
		sess.ConfigureStatusBar()
	}

	return sess
}

// ReconnectSessionWithStatus creates a Session with pre-initialized state based on previous status
// This restores the exact status state across app restarts:
//   - "idle" (gray): acknowledged=true, cooldown expired
//   - "waiting" (yellow): acknowledged=false, cooldown expired
//   - "active" (green): will be recalculated based on actual content changes
func ReconnectSessionWithStatus(tmuxName, displayName, workDir, command string, previousStatus string) *Session {
	sess := ReconnectSession(tmuxName, displayName, workDir, command)

	switch previousStatus {
	case "idle":
		// Session was acknowledged (user saw it) - restore as GRAY
		sess.stateTracker = &StateTracker{
			lastHash:       "",                                // Will be set on first GetStatus
			lastChangeTime: time.Now().Add(-10 * time.Second), // Cooldown expired
			acknowledged:   true,
		}
		sess.lastStableStatus = "idle"

	case "waiting", "active":
		// Session needs attention - restore as YELLOW
		// Active sessions will show green when content changes
		sess.stateTracker = &StateTracker{
			lastHash:       "",                                // Will be set on first GetStatus
			lastChangeTime: time.Now().Add(-10 * time.Second), // Cooldown expired
			acknowledged:   false,
		}
		sess.lastStableStatus = "waiting"

	default:
		// Unknown status - default to waiting
		sess.lastStableStatus = "waiting"
	}

	// Enable pipe-pane for event-driven status detection
	// (Note: Also called in ReconnectSession, but we ensure it's enabled after state restoration)
	if sess.Exists() {
		if err := sess.EnablePipePane(); err != nil {
			debugLog("Warning: failed to enable pipe-pane for %s: %v", tmuxName, err)
		}
	}

	return sess
}

// generateShortID generates a short random ID for uniqueness
func generateShortID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp
		return fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	}
	return hex.EncodeToString(b)
}

// SetEnvironment sets an environment variable for this tmux session
func (s *Session) SetEnvironment(key, value string) error {
	cmd := exec.Command("tmux", "set-environment", "-t", s.Name, key, value)
	return cmd.Run()
}

// GetEnvironment gets an environment variable from this tmux session
// Returns the value or error if not found
func (s *Session) GetEnvironment(key string) (string, error) {
	cmd := exec.Command("tmux", "show-environment", "-t", s.Name, key)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("variable not found or session doesn't exist: %s", key)
	}
	// Output format: "KEY=value\n"
	line := strings.TrimSpace(string(output))
	prefix := key + "="
	if strings.HasPrefix(line, prefix) {
		return strings.TrimPrefix(line, prefix), nil
	}
	return "", fmt.Errorf("variable not found: %s", key)
}

// sanitizeName converts a display name to a valid tmux session name
func sanitizeName(name string) string {
	// Replace spaces and special characters with hyphens
	re := regexp.MustCompile(`[^a-zA-Z0-9-]+`)
	return re.ReplaceAllString(name, "-")
}

// Start creates and starts a tmux session
func (s *Session) Start(command string) error {
	s.Command = command

	// Check if session already exists (shouldn't happen with unique IDs, but handle gracefully)
	if s.Exists() {
		// Session with this exact name exists - regenerate with new unique suffix
		sanitized := sanitizeName(s.DisplayName)
		s.Name = SessionPrefix + sanitized + "_" + generateShortID()
	}

	// Ensure working directory exists
	workDir := s.WorkDir
	if workDir == "" {
		workDir = os.Getenv("HOME")
	}

	// Create new tmux session in detached mode
	cmd := exec.Command("tmux", "new-session", "-d", "-s", s.Name, "-c", workDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create tmux session: %w (output: %s)", err, string(output))
	}

	// Set default window/pane styles to prevent color issues in some terminals (Warp, etc.)
	// This ensures no unexpected background colors are applied
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "window-style", "default").Run()
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "window-active-style", "default").Run()

	// Enable mouse mode for proper scrolling (per-session, doesn't affect user's other sessions)
	// This allows:
	// - Mouse wheel scrolling through terminal history
	// - Text selection with mouse
	// - Pane resizing with mouse
	// Non-fatal: session still works, just without mouse support
	// This can fail on very old tmux versions
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "mouse", "on").Run()

	// Enable escape sequence passthrough for modern terminal features (tmux 3.2+)
	// This allows:
	// - OSC 8: Clickable hyperlinks/file paths (Warp, iTerm2, kitty, Alacritty, etc.)
	// - OSC 52: Clipboard integration (copy/paste from remote sessions)
	// - Image protocols: Inline images in terminals that support it
	// Uses -q flag to silently ignore on older tmux versions (< 3.2)
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "-q", "allow-passthrough", "on").Run()

	// Enable hyperlink support in terminal features (tmux 3.4+, server-wide option)
	// This tells tmux to track hyperlinks like it tracks colors/attributes
	// Required for OSC 8 hyperlinks to work - passthrough alone isn't enough
	// Uses -as to append to existing terminal-features, -q to ignore if unsupported
	_ = exec.Command("tmux", "set", "-asq", "terminal-features", ",*:hyperlinks").Run()

	// Enable OSC 52 clipboard integration for seamless copy/paste
	// Works with: Warp, iTerm2, kitty, Alacritty, WezTerm, Windows Terminal, VS Code
	// The 'on' value (tmux 2.6+) allows apps inside tmux to set the clipboard
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "set-clipboard", "on").Run()

	// Set large history buffer for AI agent sessions (default is 2000)
	// AI agents produce extensive output, 50000 lines covers ~1000 screens
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "history-limit", "50000").Run()

	// Reduce escape-time for responsive Vim/editor usage (default 500ms is too slow)
	// 10ms is a good balance between responsiveness and SSH reliability
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "escape-time", "10").Run()

	// Configure status bar with session info for easy identification
	// Shows: session title on left, project folder on right
	s.ConfigureStatusBar()

	// Send the command to the session
	if command != "" {
		if err := s.SendKeys(command); err != nil {
			return fmt.Errorf("failed to send command: %w", err)
		}
		if err := s.SendEnter(); err != nil {
			return fmt.Errorf("failed to send enter: %w", err)
		}
	}

	// Enable pipe-pane to log output for event-driven status detection
	if err := s.EnablePipePane(); err != nil {
		// Non-fatal: status detection will fall back to polling
		debugLog("Warning: failed to enable pipe-pane for %s: %v", s.Name, err)
	}

	return nil
}

// Exists checks if the tmux session exists
func (s *Session) Exists() bool {
	cmd := exec.Command("tmux", "has-session", "-t", s.Name)
	return cmd.Run() == nil
}

// ConfigureStatusBar sets up the tmux status bar with session info
// Shows: session title on left, project folder on right
// Uses a compact, informative layout that helps developers know where they are
func (s *Session) ConfigureStatusBar() {
	// Get short folder name from WorkDir
	folderName := filepath.Base(s.WorkDir)
	if folderName == "" || folderName == "." {
		folderName = "~"
	}

	// Enable status bar
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "status", "on").Run()

	// Style: dark background with accent colors (Tokyo Night inspired)
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "status-style", "bg=#1a1b26,fg=#a9b1d6").Run()

	// Left side: session title with icon
	leftStatus := fmt.Sprintf(" 📁 %s ", s.DisplayName)
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "status-left", leftStatus).Run()
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "status-left-length", "40").Run()

	// Right side: project folder path
	rightStatus := fmt.Sprintf(" %s ", folderName)
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "status-right", rightStatus).Run()
	_ = exec.Command("tmux", "set-option", "-t", s.Name, "status-right-length", "30").Run()
}

// EnablePipePane enables tmux pipe-pane to stream output to a log file
// This is used for event-driven status detection via fsnotify
func (s *Session) EnablePipePane() error {
	logFile := s.LogFile()

	// Ensure log directory exists
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log dir: %w", err)
	}

	// Enable pipe-pane: stream pane output to log file
	cmd := exec.Command("tmux", "pipe-pane", "-t", s.Name, "-o", fmt.Sprintf("cat >> '%s'", logFile))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable pipe-pane: %w", err)
	}

	return nil
}

// DisablePipePane disables pipe-pane logging
func (s *Session) DisablePipePane() error {
	cmd := exec.Command("tmux", "pipe-pane", "-t", s.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable pipe-pane for %s: %w", s.Name, err)
	}
	return nil
}

// EnableMouseMode enables mouse scrolling, clipboard integration, and optimal settings
// Safe to call multiple times - just sets the options again
//
// Enables:
// - mouse on: Mouse wheel scrolling, text selection, pane resizing
// - set-clipboard on: OSC 52 clipboard integration (works with modern terminals)
// - allow-passthrough on: OSC 8 hyperlinks, advanced escape sequences (tmux 3.2+)
// - history-limit 50000: Large scrollback buffer for AI agent output
// - escape-time 10: Fast Vim/editor responsiveness (default 500ms is too slow)
//
// Terminal compatibility:
// - Warp, iTerm2, kitty, Alacritty, WezTerm: Full support (hyperlinks, clipboard, true color)
// - Windows Terminal, VS Code: Full support
// - Apple Terminal.app: Limited (no hyperlinks or clipboard)
//
// Note: With mouse mode on, hold Shift while selecting to use native terminal selection
// instead of tmux's selection (useful for copying to system clipboard in some terminals)
func (s *Session) EnableMouseMode() error {
	// Enable mouse support
	mouseCmd := exec.Command("tmux", "set-option", "-t", s.Name, "mouse", "on")
	if err := mouseCmd.Run(); err != nil {
		return err
	}

	// Enable OSC 52 clipboard integration
	// This allows tmux to copy directly to system clipboard in supported terminals
	// (Warp, iTerm2, Alacritty, kitty, WezTerm, Windows Terminal, VS Code, etc.)
	clipboardCmd := exec.Command("tmux", "set-option", "-t", s.Name, "set-clipboard", "on")
	if err := clipboardCmd.Run(); err != nil {
		// Non-fatal: older tmux versions may not support this
		debugLog("%s: failed to enable clipboard: %v", s.DisplayName, err)
	}

	// Enable escape sequence passthrough for modern terminal features (tmux 3.2+)
	// This allows:
	// - OSC 8: Clickable hyperlinks/file paths in Warp, iTerm2, kitty, etc.
	// - OSC 52: Clipboard integration (apps inside tmux can set clipboard)
	// - Image protocols: Inline images in supported terminals
	// Uses -q flag to silently ignore on older tmux versions
	passthroughCmd := exec.Command("tmux", "set-option", "-t", s.Name, "-q", "allow-passthrough", "on")
	if err := passthroughCmd.Run(); err != nil {
		// Non-fatal: tmux < 3.2 doesn't support this option
		debugLog("%s: failed to enable passthrough (tmux < 3.2?): %v", s.DisplayName, err)
	}

	// Enable hyperlink support in terminal features (tmux 3.4+, server-wide option)
	// This tells tmux to track hyperlinks like it tracks colors/attributes
	// Required for OSC 8 hyperlinks to work - passthrough alone isn't enough
	hyperlinkCmd := exec.Command("tmux", "set", "-asq", "terminal-features", ",*:hyperlinks")
	if err := hyperlinkCmd.Run(); err != nil {
		// Non-fatal: tmux < 3.4 doesn't support hyperlinks in terminal-features
		debugLog("%s: failed to enable hyperlinks (tmux < 3.4?): %v", s.DisplayName, err)
	}

	// Set large history limit for AI agent sessions (default is 2000)
	// AI agents produce a lot of output, so we need more scrollback
	historyCmd := exec.Command("tmux", "set-option", "-t", s.Name, "history-limit", "50000")
	if err := historyCmd.Run(); err != nil {
		// Non-fatal: history limit is a nice-to-have
		debugLog("%s: failed to set history-limit: %v", s.DisplayName, err)
	}

	// Reduce escape-time for responsive Vim/editor usage (default 500ms is too slow)
	// 10ms is a good balance between responsiveness and SSH reliability
	escapeCmd := exec.Command("tmux", "set-option", "-t", s.Name, "escape-time", "10")
	if err := escapeCmd.Run(); err != nil {
		// Non-fatal: escape-time is a nice-to-have
		debugLog("%s: failed to set escape-time: %v", s.DisplayName, err)
	}

	return nil
}

// Kill terminates the tmux session
func (s *Session) Kill() error {
	// Disable pipe-pane first
	_ = s.DisablePipePane()

	// Remove log file
	logFile := s.LogFile()
	os.Remove(logFile) // Ignore errors

	// Kill the tmux session
	cmd := exec.Command("tmux", "kill-session", "-t", s.Name)
	return cmd.Run()
}

// GetWindowActivity returns Unix timestamp of last tmux window activity
// This is a fast operation (~4ms) that checks when the window last had output
func (s *Session) GetWindowActivity() (int64, error) {
	cmd := exec.Command("tmux", "display-message", "-t", s.Name, "-p", "#{window_activity}")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get window activity: %w", err)
	}
	var ts int64
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &ts)
	if err != nil {
		return 0, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	return ts, nil
}

// CapturePane captures the visible pane content
func (s *Session) CapturePane() (string, error) {
	// -J joins wrapped lines and trims trailing spaces so hashes don't change on resize
	cmd := exec.Command("tmux", "capture-pane", "-t", s.Name, "-p", "-J")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane: %w", err)
	}
	return string(output), nil
}

// CaptureFullHistory captures the scrollback history (limited to last 2000 lines for performance)
func (s *Session) CaptureFullHistory() (string, error) {
	// Limit to last 2000 lines to balance content availability with memory usage
	// AI agent conversations can be long - 2000 lines captures ~40-80 screens of content
	// -J joins wrapped lines and trims trailing spaces so hashes don't change on resize
	cmd := exec.Command("tmux", "capture-pane", "-t", s.Name, "-p", "-J", "-S", "-2000")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture history: %w", err)
	}
	return string(output), nil
}

// HasUpdated checks if the pane content has changed since last check
func (s *Session) HasUpdated() (bool, error) {
	content, err := s.CapturePane()
	if err != nil {
		return false, err
	}

	// Calculate SHA256 hash of content
	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])

	// Protect access to lastHash and lastContent
	s.mu.Lock()
	defer s.mu.Unlock()

	// First time check
	if s.lastHash == "" {
		s.lastHash = hashStr
		s.lastContent = content
		return true, nil
	}

	// Compare with previous hash
	if hashStr != s.lastHash {
		s.lastHash = hashStr
		s.lastContent = content
		return true, nil
	}

	return false, nil
}

// DetectTool detects which AI coding tool is running in the session
// Uses caching to avoid re-detection on every call
func (s *Session) DetectTool() string {
	// Check cache first (read lock pattern for better concurrency)
	s.mu.Lock()
	if s.detectedTool != "" && time.Since(s.toolDetectedAt) < s.toolDetectExpiry {
		result := s.detectedTool
		s.mu.Unlock()
		return result
	}
	s.mu.Unlock()

	// Detect tool from command first (most reliable)
	if s.Command != "" {
		cmdLower := strings.ToLower(s.Command)
		var tool string
		if strings.Contains(cmdLower, "claude") {
			tool = "claude"
		} else if strings.Contains(cmdLower, "gemini") {
			tool = "gemini"
		} else if strings.Contains(cmdLower, "aider") {
			tool = "aider"
		} else if strings.Contains(cmdLower, "codex") {
			tool = "codex"
		}
		if tool != "" {
			s.mu.Lock()
			s.detectedTool = tool
			s.toolDetectedAt = time.Now()
			s.mu.Unlock()
			return tool
		}
	}

	// Fallback to content detection
	content, err := s.CapturePane()
	if err != nil {
		s.mu.Lock()
		s.detectedTool = "shell"
		s.toolDetectedAt = time.Now()
		s.mu.Unlock()
		return "shell"
	}

	// Strip ANSI codes for accurate matching
	cleanContent := StripANSI(content)

	// Check using pre-compiled patterns
	detectedTool := "shell"
	for tool, patterns := range toolDetectionPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(cleanContent) {
				detectedTool = tool
				break
			}
		}
		if detectedTool != "shell" {
			break
		}
	}

	s.mu.Lock()
	s.detectedTool = detectedTool
	s.toolDetectedAt = time.Now()
	s.mu.Unlock()
	return detectedTool
}

// ForceDetectTool forces a re-detection of the tool, ignoring cache
func (s *Session) ForceDetectTool() string {
	s.mu.Lock()
	s.detectedTool = ""
	s.toolDetectedAt = time.Time{}
	s.mu.Unlock()
	return s.DetectTool()
}

// AcknowledgeWithSnapshot marks the session as seen and baselines the current
// content hash. Called when user detaches from session.
func (s *Session) AcknowledgeWithSnapshot() {
	shortName := s.DisplayName
	if len(shortName) > 12 {
		shortName = shortName[:12]
	}

	// Capture content before acquiring lock (CapturePane is slow)
	var content string
	var captureErr error
	exists := s.Exists()
	if exists {
		content, captureErr = s.CapturePane()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureStateTrackerLocked()

	if !exists {
		s.stateTracker.acknowledged = true
		s.lastStableStatus = "inactive"
		debugLog("%s: AckSnapshot session gone → inactive", shortName)
		return
	}

	if captureErr != nil {
		s.stateTracker.acknowledged = true
		s.lastStableStatus = "idle"
		debugLog("%s: AckSnapshot capture error → idle", shortName)
		return
	}

	// Snapshot current content so next poll doesn't trigger "active"
	cleanContent := s.normalizeContent(content)
	newHash := s.hashContent(cleanContent)
	prevHash := s.stateTracker.lastHash
	s.stateTracker.lastHash = newHash
	s.stateTracker.acknowledged = true
	s.lastStableStatus = "idle"
	prevHashShort := "(empty)"
	if len(prevHash) >= 16 {
		prevHashShort = prevHash[:16]
	}
	debugLog("%s: AckSnapshot hash %s → %s, ack=true → idle", shortName, prevHashShort, newHash[:16])
}

// GetStatus returns the current status of the session
//
// Activity-based 3-state model with spike filtering:
//
//	GREEN (active)   = Sustained activity (2+ changes in 1s) within cooldown
//	YELLOW (waiting) = Cooldown expired, NOT acknowledged (needs attention)
//	GRAY (idle)      = Cooldown expired, acknowledged (user has seen it)
//
// Key insight: Status bar updates cause single timestamp changes (spikes).
// Real AI work causes multiple timestamp changes over 1 second (sustained).
// This filters spikes to prevent false GREEN flashes.
//
// Logic:
// 1. Check busy indicator (immediate GREEN if present)
// 2. Get activity timestamp (fast ~4ms)
// 3. If timestamp changed → check if sustained or spike
//   - Sustained (1+ more changes in 1s) → GREEN
//   - Spike (no more changes) → filtered (no state change)
// 4. Check cooldown → GREEN if within
// 5. Cooldown expired → YELLOW or GRAY based on acknowledged
func (s *Session) GetStatus() (string, error) {
	shortName := s.DisplayName
	if len(shortName) > 12 {
		shortName = shortName[:12]
	}

	if !s.Exists() {
		s.mu.Lock()
		s.lastStableStatus = "inactive"
		s.mu.Unlock()
		return "inactive", nil
	}

	// Get current activity timestamp (fast: ~4ms)
	currentTS, err := s.GetWindowActivity()
	if err != nil {
		// Fallback to content-hash based detection
		return s.getStatusFallback()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Skip expensive busy indicator check if no activity change and not in active state
	// This is the key optimization: only call CapturePane() when activity detected
	needsBusyCheck := false
	if s.stateTracker != nil {
		// Check busy indicator if:
		// 1. timestamp changed (new activity)
		// 2. we're in cooldown (might still be working)
		// 3. we're in spike detection window (activity recently detected, waiting to confirm)
		inSpikeWindow := !s.stateTracker.activityCheckStart.IsZero() &&
			time.Since(s.stateTracker.activityCheckStart) < 1*time.Second
		if s.stateTracker.lastActivityTimestamp != currentTS ||
			time.Since(s.stateTracker.lastChangeTime) < activityCooldown ||
			inSpikeWindow {
			needsBusyCheck = true
		}
	} else {
		// First call - check for busy indicator
		needsBusyCheck = true
	}

	if needsBusyCheck {
		// Release lock for slow CapturePane operation
		s.mu.Unlock()
		content, err := s.CapturePane()
		s.mu.Lock()

		if err == nil && s.hasBusyIndicator(content) {
			s.ensureStateTrackerLocked()
			s.stateTracker.lastChangeTime = time.Now()
			s.stateTracker.acknowledged = false
			s.stateTracker.lastActivityTimestamp = currentTS
			s.lastStableStatus = "active"
			debugLog("%s: BUSY INDICATOR → active", shortName)
			return "active", nil
		}
	}

	// Initialize on first call
	if s.stateTracker == nil {
		s.stateTracker = &StateTracker{
			lastChangeTime:        time.Now().Add(-activityCooldown),
			acknowledged:          false, // Start unacknowledged so stopped sessions show YELLOW
			lastActivityTimestamp: currentTS,
		}
		s.lastStableStatus = "waiting"
		debugLog("%s: INIT → waiting", shortName)
		return "waiting", nil
	}

	// Restored session (lastActivityTimestamp == 0)
	if s.stateTracker.lastActivityTimestamp == 0 {
		s.stateTracker.lastActivityTimestamp = currentTS
		if s.stateTracker.acknowledged {
			s.lastStableStatus = "idle"
			return "idle", nil
		}
		s.lastStableStatus = "waiting"
		return "waiting", nil
	}

	// Activity timestamp changed → non-blocking spike detection across tick cycles
	if s.stateTracker.lastActivityTimestamp != currentTS {
		oldTS := s.stateTracker.lastActivityTimestamp
		s.stateTracker.lastActivityTimestamp = currentTS

		// Check if we're in a detection window
		const spikeWindow = 1 * time.Second
		now := time.Now()

		if s.stateTracker.activityCheckStart.IsZero() || now.Sub(s.stateTracker.activityCheckStart) > spikeWindow {
			// Start new detection window
			s.stateTracker.activityCheckStart = now
			s.stateTracker.activityChangeCount = 1
			debugLog("%s: ACTIVITY_START ts=%d→%d count=1", shortName, oldTS, currentTS)
		} else {
			// Within detection window - count this change
			s.stateTracker.activityChangeCount++
			debugLog("%s: ACTIVITY_COUNT ts=%d→%d count=%d", shortName, oldTS, currentTS, s.stateTracker.activityChangeCount)

			// 2+ changes within 1 second = sustained activity
			if s.stateTracker.activityChangeCount >= 2 {
				s.stateTracker.lastChangeTime = now
				s.stateTracker.acknowledged = false
				s.stateTracker.activityCheckStart = time.Time{} // Reset window
				s.stateTracker.activityChangeCount = 0
				s.lastStableStatus = "active"
				debugLog("%s: SUSTAINED count=%d → active", shortName, s.stateTracker.activityChangeCount)
				return "active", nil
			}
		}
		// Not enough changes yet - continue with current status (don't block)
	} else {
		// No timestamp change - check if spike window expired with only 1 change
		if s.stateTracker.activityChangeCount == 1 && !s.stateTracker.activityCheckStart.IsZero() {
			if time.Since(s.stateTracker.activityCheckStart) > 1*time.Second {
				// Only 1 change in 1 second = spike, reset tracking
				debugLog("%s: SPIKE_EXPIRED count=1 (filtered)", shortName)
				s.stateTracker.activityCheckStart = time.Time{}
				s.stateTracker.activityChangeCount = 0
			}
		}
	}

	// During spike detection window (waiting to confirm sustained activity),
	// keep the PREVIOUS stable status instead of flashing GREEN
	// Only confirmed sustained activity (2+ changes in 1s) triggers GREEN
	if !s.stateTracker.activityCheckStart.IsZero() &&
		time.Since(s.stateTracker.activityCheckStart) < 1*time.Second {
		// Return previous status - don't flash GREEN on unconfirmed single spike
		debugLog("%s: SPIKE_WINDOW_PENDING → keeping %s (not flashing green)", shortName, s.lastStableStatus)
		if s.lastStableStatus != "" {
			return s.lastStableStatus, nil
		}
		// Fallback if no previous status
		return "waiting", nil
	}

	// Check cooldown
	if time.Since(s.stateTracker.lastChangeTime) < activityCooldown {
		s.lastStableStatus = "active"
		return "active", nil
	}

	// Cooldown expired
	if s.stateTracker.acknowledged {
		s.lastStableStatus = "idle"
		return "idle", nil
	}
	s.lastStableStatus = "waiting"
	return "waiting", nil
}

// getStatusFallback uses content-hash based detection as fallback
// when activity timestamp detection fails
func (s *Session) getStatusFallback() (string, error) {
	shortName := s.DisplayName
	if len(shortName) > 12 {
		shortName = shortName[:12]
	}

	content, err := s.CapturePane()
	if err != nil {
		s.mu.Lock()
		s.lastStableStatus = "inactive"
		s.mu.Unlock()
		return "inactive", nil
	}

	if s.hasBusyIndicator(content) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ensureStateTrackerLocked()
		s.stateTracker.lastChangeTime = time.Now()
		s.stateTracker.acknowledged = false
		s.lastStableStatus = "active"
		return "active", nil
	}

	cleanContent := s.normalizeContent(content)
	currentHash := s.hashContent(cleanContent)
	if currentHash == "" {
		currentHash = "__empty__"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stateTracker == nil {
		s.stateTracker = &StateTracker{
			lastHash:       currentHash,
			lastChangeTime: time.Now().Add(-activityCooldown),
			acknowledged:   false, // Start unacknowledged so stopped sessions show YELLOW
		}
		s.lastStableStatus = "waiting"
		return "waiting", nil
	}

	if s.stateTracker.lastHash == "" {
		s.stateTracker.lastHash = currentHash
		if s.stateTracker.acknowledged {
			s.lastStableStatus = "idle"
			return "idle", nil
		}
		s.lastStableStatus = "waiting"
		return "waiting", nil
	}

	if s.stateTracker.lastHash != currentHash {
		s.stateTracker.lastHash = currentHash
		s.stateTracker.lastChangeTime = time.Now()
		s.stateTracker.acknowledged = false
		s.lastStableStatus = "active"
		debugLog("%s: FALLBACK CHANGED → active", shortName)
		return "active", nil
	}

	if time.Since(s.stateTracker.lastChangeTime) < activityCooldown {
		s.lastStableStatus = "active"
		return "active", nil
	}

	if s.stateTracker.acknowledged {
		s.lastStableStatus = "idle"
		return "idle", nil
	}
	s.lastStableStatus = "waiting"
	return "waiting", nil
}

// Acknowledge marks the session as "seen" by the user
// Call this when user attaches to the session
func (s *Session) Acknowledge() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureStateTrackerLocked()
	s.stateTracker.acknowledged = true
	s.lastStableStatus = "idle"
}

// ResetAcknowledged marks the session as needing attention
// Call this when a hook event indicates the agent finished (Stop, AfterAgent)
// This ensures the session shows yellow (waiting) instead of gray (idle)
func (s *Session) ResetAcknowledged() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureStateTrackerLocked()
	s.stateTracker.acknowledged = false
	s.lastStableStatus = "waiting"
}

// SignalFileActivity signals that file output was detected (from LogWatcher)
// This directly triggers GREEN status by updating the cooldown timer
// Call this when pipe-pane log file is written to
func (s *Session) SignalFileActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureStateTrackerLocked()
	s.stateTracker.lastChangeTime = time.Now()
	s.stateTracker.acknowledged = false
	s.lastStableStatus = "active"
}

// hasBusyIndicator checks if the terminal shows explicit busy indicators
// This is a quick check used in GetStatus() to detect active processing
//
// Busy indicators for different tools:
// - Claude Code: "esc to interrupt", spinner chars, "Thinking...", "Connecting..."
// - Gemini: Similar spinner patterns
// - Aider: Processing indicators
// - Shell: Running commands (no prompt visible)
func (s *Session) hasBusyIndicator(content string) bool {
	shortName := s.DisplayName
	if len(shortName) > 12 {
		shortName = shortName[:12]
	}

	// Get last 10 lines for analysis
	lines := strings.Split(content, "\n")
	start := len(lines) - 10
	if start < 0 {
		start = 0
	}
	recentContent := strings.ToLower(strings.Join(lines[start:], "\n"))

	// ═══════════════════════════════════════════════════════════════════════
	// Text-based busy indicators
	// ═══════════════════════════════════════════════════════════════════════
	busyIndicators := []string{
		"esc to interrupt",   // Claude Code main indicator
		"(esc to interrupt)", // Claude Code in parentheses
		"· esc to interrupt", // With separator
	}

	for _, indicator := range busyIndicators {
		if strings.Contains(recentContent, indicator) {
			debugLog("%s: BUSY_REASON=text_indicator matched=%q", shortName, indicator)
			return true
		}
	}

	// Check for "Thinking... (Xs · Y tokens)" pattern
	if strings.Contains(recentContent, "thinking") && strings.Contains(recentContent, "tokens") {
		debugLog("%s: BUSY_REASON=thinking+tokens pattern", shortName)
		return true
	}

	// Check for "Connecting..." pattern
	if strings.Contains(recentContent, "connecting") && strings.Contains(recentContent, "tokens") {
		debugLog("%s: BUSY_REASON=connecting+tokens pattern", shortName)
		return true
	}

	// ═══════════════════════════════════════════════════════════════════════
	// Spinner characters (from cli-spinners "dots" - used by Claude Code)
	// These braille characters animate to show processing
	// ═══════════════════════════════════════════════════════════════════════
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	// Only check last 5 lines for spinners (they appear near the bottom)
	last5 := lines
	if len(last5) > 5 {
		last5 = last5[len(last5)-5:]
	}

	for lineIdx, line := range last5 {
		for _, spinner := range spinnerChars {
			if strings.Contains(line, spinner) {
				debugLog("%s: BUSY_REASON=spinner char=%q line=%d content=%q", shortName, spinner, lineIdx, truncateForLog(line, 50))
				return true
			}
		}
	}

	// ═══════════════════════════════════════════════════════════════════════
	// Additional busy indicators (for other tools)
	// ═══════════════════════════════════════════════════════════════════════

	// Generic "working" indicators that appear in various tools
	workingIndicators := []string{
		"processing",
		"loading",
		"please wait",
		"working",
	}

	// Only match these if they're standalone (not part of other text)
	for _, indicator := range workingIndicators {
		// Check if indicator appears at start of a line (more reliable)
		for lineIdx, line := range last5 {
			lineLower := strings.ToLower(strings.TrimSpace(line))
			if strings.HasPrefix(lineLower, indicator) {
				debugLog("%s: BUSY_REASON=working_indicator matched=%q line=%d content=%q", shortName, indicator, lineIdx, truncateForLog(line, 50))
				return true
			}
		}
	}

	return false
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// isSustainedActivity checks if activity is sustained (real work) or a spike.
// Checks 5 times over 1 second, counts timestamp changes.
// Returns true if 1+ changes detected AFTER initial check (sustained activity).
// Returns false if no additional changes (spike - status bar update, etc).
//
// This filters out false positives from:
// - Status bar time updates (e.g., Claude Code's auto-compact %)
// - Single cursor movements
// - Terminal refresh events
func (s *Session) isSustainedActivity() bool {
	const (
		checkCount    = 5
		checkInterval = 200 * time.Millisecond
	)

	prevTS, err := s.GetWindowActivity()
	if err != nil {
		return false
	}

	changes := 0
	for i := 0; i < checkCount; i++ {
		time.Sleep(checkInterval)
		currentTS, err := s.GetWindowActivity()
		if err != nil {
			continue
		}
		if currentTS != prevTS {
			changes++
			prevTS = currentTS
		}
	}

	isSustained := changes >= 1 // At least 1 MORE change after initial detection
	debugLog("%s: isSustainedActivity changes=%d sustained=%v", s.DisplayName, changes, isSustained)
	return isSustained
}

// Precompiled regex patterns for dynamic content stripping
// These are compiled once at package init for performance
var (
	// Matches Claude Code status line: "(45s · 1234 tokens · esc to interrupt)"
	dynamicStatusPattern = regexp.MustCompile(`\([^)]*\d+s\s*·[^)]*tokens[^)]*\)`)

	// Matches "Thinking..." or "Connecting..." with timing info
	thinkingPattern = regexp.MustCompile(`(Thinking|Connecting)[^(]*\([^)]*\)`)
)

// normalizeContent strips ANSI codes, spinner characters, and normalizes whitespace
// This is critical for stable hashing - prevents flickering from:
// 1. Color/style changes in terminal output
// 2. Animated spinner characters (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏)
// 3. Other non-printing control characters
// 4. Terminal resize (which can add trailing spaces with tmux -J flag)
// 5. Multiple consecutive blank lines
// 6. Dynamic time/token counters (e.g., "45s · 1234 tokens")
func (s *Session) normalizeContent(content string) string {
	// Strip ANSI escape codes first (handles CSI, OSC, and C1 codes)
	result := StripANSI(content)

	// Strip other non-printing control characters
	result = stripControlChars(result)

	// Strip braille spinner characters (used by Claude Code and others)
	// These animate while processing and cause hash changes
	spinners := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	for _, r := range spinners {
		result = strings.ReplaceAll(result, string(r), "")
	}

	// Strip dynamic time/token counters that change every second
	// This prevents flickering when Claude Code shows "(45s · 1234 tokens · esc to interrupt)"
	// which updates to "(46s · 1234 tokens · esc to interrupt)" one second later
	result = dynamicStatusPattern.ReplaceAllString(result, "(STATUS)")
	result = thinkingPattern.ReplaceAllString(result, "$1...")

	// Normalize trailing whitespace per line (fixes resize false positives)
	// tmux capture-pane -J can add trailing spaces when terminal is resized
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	result = strings.Join(lines, "\n")

	// Normalize multiple consecutive blank lines to a single blank line
	// This prevents hash changes from cursor position variations
	result = normalizeBlankLines(result)

	return result
}

// normalizeBlankLines collapses runs of 3+ newlines to 2 newlines (one blank line)
func normalizeBlankLines(content string) string {
	// Match 3 or more consecutive newlines and replace with 2
	re := regexp.MustCompile(`\n{3,}`)
	return re.ReplaceAllString(content, "\n\n")
}

// stripControlChars removes all ASCII control characters except for tab, newline,
// and carriage return. This helps stabilize content for hashing.
func stripControlChars(content string) string {
	var result strings.Builder
	result.Grow(len(content))
	for _, r := range content {
		// Keep printable characters (space and above), and essential whitespace.
		// DEL (127) is excluded.
		if (r >= 32 && r != 127) || r == '\t' || r == '\n' || r == '\r' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// hashContent generates SHA256 hash of content (same as Claude Squad)
func (s *Session) hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// SendKeys sends keys to the tmux session
// Uses -l flag to treat keys as literal text, preventing tmux special key interpretation
func (s *Session) SendKeys(keys string) error {
	// The -l flag makes tmux treat the string as literal text, not key names
	// This prevents issues like "Enter" being interpreted as the Enter key
	// and provides a layer of safety against tmux special sequences
	cmd := exec.Command("tmux", "send-keys", "-l", "-t", s.Name, keys)
	return cmd.Run()
}

// SendEnter sends an Enter key to the tmux session
func (s *Session) SendEnter() error {
	cmd := exec.Command("tmux", "send-keys", "-t", s.Name, "Enter")
	return cmd.Run()
}

// GetWorkDir returns the current working directory of the tmux pane
// This is the live directory from the pane, not the initial WorkDir
func (s *Session) GetWorkDir() string {
	if !s.Exists() {
		return ""
	}

	cmd := exec.Command("tmux", "display-message", "-t", s.Name, "-p", "#{pane_current_path}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// ListAllSessions returns all Agent Deck tmux sessions
func ListAllSessions() ([]*Session, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// No sessions exist
		if strings.Contains(err.Error(), "no server running") ||
			strings.Contains(err.Error(), "no sessions") {
			return []*Session{}, nil
		}
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var sessions []*Session

	for _, line := range lines {
		if strings.HasPrefix(line, SessionPrefix) {
			displayName := strings.TrimPrefix(line, SessionPrefix)
			// Get session info
			sess := &Session{
				Name:        line,
				DisplayName: displayName,
			}
			// Try to get working directory
			workDirCmd := exec.Command("tmux", "display-message", "-t", line, "-p", "#{pane_current_path}")
			if workDirOutput, err := workDirCmd.Output(); err == nil {
				sess.WorkDir = strings.TrimSpace(string(workDirOutput))
			}
			sessions = append(sessions, sess)
		}
	}

	return sessions, nil
}

// DiscoverAllTmuxSessions returns all tmux sessions (including non-Agent Deck ones)
func DiscoverAllTmuxSessions() ([]*Session, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}:#{pane_current_path}")
	output, err := cmd.Output()
	if err != nil {
		// No sessions exist
		if strings.Contains(err.Error(), "no server running") ||
			strings.Contains(err.Error(), "no sessions") {
			return []*Session{}, nil
		}
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var sessions []*Session

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		sessionName := parts[0]
		workDir := ""
		if len(parts) == 2 {
			workDir = parts[1]
		}

		// Create session object
		sess := &Session{
			Name:        sessionName,
			DisplayName: sessionName,
			WorkDir:     workDir,
		}

		// If it's an agent-deck session, clean up the display name
		if strings.HasPrefix(sessionName, SessionPrefix) {
			sess.DisplayName = strings.TrimPrefix(sessionName, SessionPrefix)
		}

		sessions = append(sessions, sess)
	}

	return sessions, nil
}
