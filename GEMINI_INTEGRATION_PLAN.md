# Gemini CLI Integration Implementation Plan

**Date:** 2025-12-26
**Project:** agent-deck
**Goal:** Add full Gemini CLI support matching Claude Code integration quality

---

## Executive Summary

This plan provides a complete implementation roadmap for integrating Google's Gemini CLI into agent-deck with feature parity to our existing Claude Code integration. All necessary technical details have been verified through research and examination of actual Gemini session files.

**Status:** ✅ All critical information gathered and verified
**Complexity:** Medium (similar patterns to existing Claude integration)
**Estimated Implementation Phases:** 4 phases

---

## I. VERIFIED TECHNICAL SPECIFICATIONS

### 1.1 Session Storage Architecture

**Directory Structure:**
```
~/.gemini/
├── settings.json              # User-level configuration
├── tmp/
│   └── <project_hash>/        # SHA256 hash of absolute project path
│       └── chats/
│           └── session-<ISO_TIMESTAMP>-<UUID_PREFIX>.json
```

**Project Hash Algorithm:** ✅ **VERIFIED**
```go
// SHA256 of absolute project path
hash := sha256.Sum256([]byte("/Users/ashesh/claude-deck"))
projectHash := hex.EncodeToString(hash[:])
// Result: "791e1ce1b3651ae5c05fc40e2ff27287a9a59008bcd7a449daf0cfb365d43bac"
```

**Filename Format:** ✅ **VERIFIED**
```
session-2025-12-23T00-24-299ea882.json
         ^^^^^^^^^^^^^^^^ ^^^^^^^^
         ISO timestamp    first 8 chars of UUID
```

### 1.2 Session File Format

**Verified JSON Structure** (from actual session file):
```json
{
  "sessionId": "299ea882-31f1-4fe7-9657-97fa127371e3",  // Full UUID v4
  "projectHash": "791e1ce1b3651ae5c05fc40e2ff27287a9a59008bcd7a449daf0cfb365d43bac",
  "startTime": "2025-12-23T00:25:32.773Z",
  "lastUpdated": "2025-12-23T00:25:45.473Z",
  "messages": [
    {
      "id": "13dd6bda-6a91-4e85-9e80-6010b7260fc0",
      "timestamp": "2025-12-23T00:25:32.773Z",
      "type": "user",  // or "gemini", "info", "error", "warning"
      "content": "Design authentication schema with OAuth2"
    },
    {
      "id": "2d1f6bc8-c959-451a-a7cc-b4ee36716cb1",
      "timestamp": "2025-12-23T00:25:45.473Z",
      "type": "gemini",
      "content": "",  // Can be empty if only tool calls
      "toolCalls": [
        {
          "id": "read_file-1766449545452-12565a2cc0326",
          "name": "read_file",
          "args": {"file_path": "current.code-workspace"},
          "result": [...],
          "status": "success",  // or "cancelled", "failed"
          "timestamp": "2025-12-23T00:25:45.473Z",
          "displayName": "ReadFile",
          "description": "..."
        }
      ],
      "thoughts": [  // Extended thinking (Gemini Deep Research)
        {
          "subject": "Planning Authentication Schema Design",
          "description": "...",
          "timestamp": "2025-12-23T00:25:36.410Z"
        }
      ],
      "model": "gemini-3-pro-preview",
      "tokens": {
        "input": 7641,
        "output": 15,
        "cached": 3107,
        "thoughts": 917,
        "tool": 0,
        "total": 8573
      }
    }
  ]
}
```

### 1.3 CLI Command-Line Interface

**Session Management:**
```bash
# Resume commands
gemini --resume                # Resume latest session
gemini --resume 2              # Resume by index (1-based)
gemini --resume 299ea882-...   # Resume by UUID

# List sessions
gemini --list-sessions

# Delete sessions
gemini --delete-session 2
gemini --delete-session 299ea882-...

# Interactive browser (within Gemini)
/resume   # Opens session browser with search
```

**MCP Management:**
```bash
gemini mcp add <name> <command> [args...]
  -s, --scope <user|project>
  -t, --transport <stdio|sse|http>
  -e, --env KEY=value
  --timeout <ms>
  --trust

gemini mcp list
gemini mcp remove <name>

# In-chat command
/mcp      # Show all MCPs, tools, status
```

### 1.4 MCP Configuration

**settings.json Location:** `~/.gemini/settings.json` (user) or `.gemini/settings.json` (project)

**Verified Structure** (simpler than research docs suggested):
```json
{
  "mcpServers": {
    "exa": {
      "command": "npx",
      "args": ["-y", "exa-mcp-server"],
      "env": {"EXA_API_KEY": "$EXA_API_KEY"},
      "timeout": 30000,
      "trust": false
    }
  },
  "mcp": {
    "allowed": ["exa", "firecrawl"],
    "excluded": []
  },
  "security": {
    "auth": {
      "selectedType": "oauth-personal"
    }
  }
}
```

**Transport Types:**
- `stdio`: Local subprocess (command + args)
- `sse`: Server-Sent Events (url)
- `http`: HTTP streaming (httpUrl)

---

## II. FEATURE PARITY MATRIX

| Feature | Claude Code | Gemini CLI | Implementation Status |
|---------|-------------|------------|----------------------|
| **Session ID Detection** | UUID from `.claude/projects/<path>/<uuid>.jsonl` | UUID from `~/.gemini/tmp/<hash>/chats/session-*.json` | ✅ Algorithm verified |
| **Project Path Hashing** | Direct path in config | SHA256 of path | ✅ Algorithm verified |
| **Session Resume** | `claude --resume <id>` | `gemini --resume <id>` | ⏳ Need to implement |
| **Session File Format** | JSONL (streaming) | JSON (complete) | ⏳ Parser needed |
| **MCP Config Location** | `~/.claude/.claude.json` | `~/.gemini/settings.json` | ⏳ Reader needed |
| **MCP Config Format** | Flat `mcpServers` object | Hierarchical with `mcp` section | ⏳ Parser needed |
| **MCP Scope** | Global + Project + Local | Global + Project | ⏳ Need to implement |
| **MCP Management** | Manual JSON editing | `gemini mcp add/remove` CLI | ⏳ CLI commands needed |
| **Session List** | Via config file | `--list-sessions` flag | ⏳ Need to implement |
| **Fork/Branch** | `--fork-session` flag | ❌ Not available | ✅ Use sub-sessions |
| **Config Env Var** | `CLAUDE_CONFIG_DIR` | ❌ Fixed `~/.gemini` | ✅ Hardcode path |
| **Output Format** | `--output-format json` | `--output-format stream-json` | ⏳ Parse for session ID |

---

## III. IMPLEMENTATION PHASES

### Phase 1: Core Session Detection & Tracking

**Goal:** Detect Gemini sessions and track them in agent-deck

**New Files:**
```
internal/session/
├── gemini.go           # Session detection, path hashing, session discovery
└── gemini_test.go      # Unit tests
```

**Functions to Implement:**

```go
// internal/session/gemini.go

// GetGeminiConfigDir returns ~/.gemini (no env var override)
func GetGeminiConfigDir() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".gemini")
}

// HashProjectPath generates SHA256 hash of absolute project path
func HashProjectPath(projectPath string) string {
    absPath, err := filepath.Abs(projectPath)
    if err != nil {
        return ""
    }
    hash := sha256.Sum256([]byte(absPath))
    return hex.EncodeToString(hash[:])
}

// GetGeminiSessionsDir returns the chats directory for a project
func GetGeminiSessionsDir(projectPath string) string {
    configDir := GetGeminiConfigDir()
    projectHash := HashProjectPath(projectPath)
    return filepath.Join(configDir, "tmp", projectHash, "chats")
}

// GeminiSessionInfo holds parsed session metadata
type GeminiSessionInfo struct {
    SessionID    string    // Full UUID
    Filename     string    // session-2025-12-23T00-24-299ea882.json
    StartTime    time.Time
    LastUpdated  time.Time
    MessageCount int
}

// ListGeminiSessions returns all sessions for a project path
// Scans ~/.gemini/tmp/<hash>/chats/ and parses session files
func ListGeminiSessions(projectPath string) ([]GeminiSessionInfo, error) {
    sessionsDir := GetGeminiSessionsDir(projectPath)
    files, err := filepath.Glob(filepath.Join(sessionsDir, "session-*.json"))
    if err != nil {
        return nil, err
    }

    var sessions []GeminiSessionInfo
    for _, file := range files {
        info, err := parseGeminiSessionFile(file)
        if err != nil {
            continue // Skip malformed files
        }
        sessions = append(sessions, info)
    }

    // Sort by LastUpdated (most recent first)
    sort.Slice(sessions, func(i, j int) bool {
        return sessions[i].LastUpdated.After(sessions[j].LastUpdated)
    })

    return sessions, nil
}

// FindGeminiSessionForInstance finds session created after instance start
// Parameters:
//   - projectPath: project directory
//   - createdAfter: only consider sessions created after this time
//   - excludeIDs: session IDs already claimed by other instances
func FindGeminiSessionForInstance(projectPath string, createdAfter time.Time, excludeIDs map[string]bool) string {
    sessions, err := ListGeminiSessions(projectPath)
    if err != nil {
        return ""
    }

    for _, session := range sessions {
        // Skip if already claimed
        if excludeIDs[session.SessionID] {
            continue
        }

        // Only consider sessions created after instance
        if !session.StartTime.Before(createdAfter) {
            return session.SessionID
        }
    }

    return ""
}

// parseGeminiSessionFile reads a session file and extracts metadata
func parseGeminiSessionFile(filePath string) (GeminiSessionInfo, error) {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return GeminiSessionInfo{}, err
    }

    var session struct {
        SessionID   string    `json:"sessionId"`
        StartTime   string    `json:"startTime"`
        LastUpdated string    `json:"lastUpdated"`
        Messages    []json.RawMessage `json:"messages"`
    }

    if err := json.Unmarshal(data, &session); err != nil {
        return GeminiSessionInfo{}, err
    }

    startTime, _ := time.Parse(time.RFC3339, session.StartTime)
    lastUpdated, _ := time.Parse(time.RFC3339, session.LastUpdated)

    return GeminiSessionInfo{
        SessionID:    session.SessionID,
        Filename:     filepath.Base(filePath),
        StartTime:    startTime,
        LastUpdated:  lastUpdated,
        MessageCount: len(session.Messages),
    }, nil
}
```

**Modifications to Existing Files:**

```go
// internal/session/instance.go

type Instance struct {
    // ... existing fields ...

    // Gemini CLI integration
    GeminiSessionID  string    `json:"gemini_session_id,omitempty"`
    GeminiDetectedAt time.Time `json:"gemini_detected_at,omitempty"`
}

// UpdateGeminiSession updates the Gemini session ID using detection
func (i *Instance) UpdateGeminiSession(excludeIDs map[string]bool) {
    if i.Tool != "gemini" {
        return
    }

    // If we already have a recent session ID, skip
    if i.GeminiSessionID != "" && time.Since(i.GeminiDetectedAt) < 5*time.Minute {
        return
    }

    // Detect session from files
    sessionID := FindGeminiSessionForInstance(i.ProjectPath, i.CreatedAt, excludeIDs)
    if sessionID != "" {
        i.GeminiSessionID = sessionID
        i.GeminiDetectedAt = time.Now()
    }
}

// In UpdateStatus(), add after Claude session update:
if i.Tool == "gemini" {
    i.UpdateGeminiSession(nil)
}
```

**Testing:**
```go
// internal/session/gemini_test.go

func TestHashProjectPath(t *testing.T) {
    // Test known hash
    hash := HashProjectPath("/Users/ashesh")
    expected := "791e1ce1b3651ae5c05fc40e2ff27287a9a59008bcd7a449daf0cfb365d43bac"
    if hash != expected {
        t.Errorf("Hash mismatch: got %s, want %s", hash, expected)
    }
}

func TestParseGeminiSessionFile(t *testing.T) {
    // Create temp session file with test data
    // ...
}

func TestFindGeminiSessionForInstance(t *testing.T) {
    // Test timestamp filtering
    // Test exclusion list
    // ...
}
```

---

### Phase 2: MCP Integration

**Goal:** Read and write Gemini MCP configurations

**New Files:**
```
internal/session/
└── gemini_mcp.go      # MCP configuration management
```

**Functions to Implement:**

```go
// internal/session/gemini_mcp.go

// GeminiMCPConfig represents settings.json structure
type GeminiMCPConfig struct {
    MCPServers map[string]MCPServerConfig `json:"mcpServers"`
    MCP        struct {
        Allowed  []string `json:"allowed"`
        Excluded []string `json:"excluded"`
    } `json:"mcp"`
}

// GetGeminiMCPInfo reads MCP configuration from settings.json
// Returns MCPInfo with Global MCPs (no project-level distinction for Gemini)
func GetGeminiMCPInfo(projectPath string) *MCPInfo {
    configFile := filepath.Join(GetGeminiConfigDir(), "settings.json")

    data, err := os.ReadFile(configFile)
    if err != nil {
        return &MCPInfo{}
    }

    var config GeminiMCPConfig
    if err := json.Unmarshal(data, &config); err != nil {
        return &MCPInfo{}
    }

    info := &MCPInfo{}

    // All MCPs are global in Gemini
    for name := range config.MCPServers {
        // Check if allowed/excluded
        if len(config.MCP.Allowed) > 0 {
            if !contains(config.MCP.Allowed, name) {
                continue // Not in allowlist
            }
        }
        if contains(config.MCP.Excluded, name) {
            continue // In denylist
        }

        info.Global = append(info.Global, name)
    }

    sort.Strings(info.Global)
    return info
}

// WriteGeminiMCPSettings writes MCPs to ~/.gemini/settings.json
func WriteGeminiMCPSettings(enabledNames []string) error {
    configFile := filepath.Join(GetGeminiConfigDir(), "settings.json")

    // Read existing config (preserve other fields)
    var rawConfig map[string]interface{}
    if data, err := os.ReadFile(configFile); err == nil {
        if err := json.Unmarshal(data, &rawConfig); err != nil {
            rawConfig = make(map[string]interface{})
        }
    } else {
        rawConfig = make(map[string]interface{})
    }

    // Get available MCPs from agent-deck config
    availableMCPs := GetAvailableMCPs()
    pool := GetGlobalPool()

    mcpServers := make(map[string]MCPServerConfig)
    for _, name := range enabledNames {
        if def, ok := availableMCPs[name]; ok {
            // Check if should use socket pool mode
            if pool != nil && pool.ShouldPool(name) && pool.IsRunning(name) {
                // Use Unix socket
                socketPath := pool.GetSocketPath(name)
                mcpServers[name] = MCPServerConfig{
                    Command: "nc",
                    Args:    []string{"-U", socketPath},
                }
            } else {
                // Use stdio mode
                mcpServers[name] = MCPServerConfig{
                    Command: def.Command,
                    Args:    def.Args,
                    Env:     def.Env,
                }
            }
        }
    }

    rawConfig["mcpServers"] = mcpServers

    // Write atomically
    newData, err := json.MarshalIndent(rawConfig, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    tmpPath := configFile + ".tmp"
    if err := os.WriteFile(tmpPath, newData, 0600); err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }

    if err := os.Rename(tmpPath, configFile); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("failed to save config: %w", err)
    }

    return nil
}

// GetGeminiMCPNames returns names of configured MCPs
func GetGeminiMCPNames() []string {
    info := GetGeminiMCPInfo("")
    return info.Global
}
```

**Modifications to Instance:**

```go
// internal/session/instance.go

// GetMCPInfo returns MCP server information for this session
func (i *Instance) GetMCPInfo() *MCPInfo {
    switch i.Tool {
    case "claude":
        return GetMCPInfo(i.ProjectPath)
    case "gemini":
        return GetGeminiMCPInfo(i.ProjectPath)
    default:
        return nil
    }
}
```

**MCP Manager UI Integration:**

```go
// internal/ui/mcp_manager.go

// In loadAvailableMCPs()
func (m *MCPManager) loadAvailableMCPs() {
    if m.instance.Tool == "claude" {
        // ... existing Claude logic ...
    } else if m.instance.Tool == "gemini" {
        availableMCPs := session.GetAvailableMCPs()
        attachedMCPs := session.GetGeminiMCPNames()

        // Build available list
        for name, def := range availableMCPs {
            item := MCPItem{
                Name:        name,
                Description: def.Description,
                Attached:    contains(attachedMCPs, name),
            }
            m.available = append(m.available, item)
        }
    }
}

// In applyChanges()
func (m *MCPManager) applyChanges() error {
    if m.instance.Tool == "claude" {
        // ... existing Claude logic ...
    } else if m.instance.Tool == "gemini" {
        // Gemini only has global scope (no project-level MCPs)
        return session.WriteGeminiMCPSettings(m.getSelectedMCPs())
    }
    return nil
}
```

---

### Phase 3: Command Building & Restart

**Goal:** Build Gemini commands with session resume support

**Modifications to Instance:**

```go
// internal/session/instance.go

// buildGeminiCommand builds gemini command with optional resume
func (i *Instance) buildGeminiCommand(baseCommand string) string {
    if i.Tool != "gemini" {
        return baseCommand
    }

    // If baseCommand is just "gemini", check for resume
    if baseCommand == "gemini" {
        // If we have a detected session ID, use resume
        if i.GeminiSessionID != "" {
            return fmt.Sprintf("gemini --resume %s", i.GeminiSessionID)
        }
    }

    // Otherwise, return as-is (new session or custom command)
    return baseCommand
}

// buildGeminiCommandWithMessage builds command with initial message
func (i *Instance) buildGeminiCommandWithMessage(baseCommand, message string) string {
    cmd := i.buildGeminiCommand(baseCommand)
    if message != "" {
        // Use existing wrapCommandWithMessage() - it's tool-agnostic
        // Waits for prompt and sends message
        return i.wrapCommandWithMessage(cmd, message)
    }
    return cmd
}

// In Start(), modify to use buildGeminiCommand for Gemini sessions
func (i *Instance) Start() error {
    if i.tmuxSession == nil {
        return fmt.Errorf("tmux session not initialized")
    }

    var command string
    switch i.Tool {
    case "claude":
        command = i.buildClaudeCommand(i.Command)
    case "gemini":
        command = i.buildGeminiCommand(i.Command)
    default:
        command = i.Command
    }

    // ... rest of Start() logic ...
}

// Restart logic for Gemini
func (i *Instance) Restart() error {
    // ... existing Claude logic ...

    // Add Gemini restart logic
    if i.Tool == "gemini" && i.GeminiSessionID != "" && i.tmuxSession != nil && i.tmuxSession.Exists() {
        // Use respawn-pane with gemini --resume
        resumeCmd := fmt.Sprintf("gemini --resume %s", i.GeminiSessionID)

        if err := i.tmuxSession.RespawnPane(resumeCmd); err != nil {
            return fmt.Errorf("failed to restart Gemini session: %w", err)
        }

        // Re-capture MCPs after restart
        i.CaptureLoadedMCPs()
        i.Status = StatusWaiting
        return nil
    }

    // ... fallback logic for dead sessions ...
}

// CanRestart returns true if the session can be restarted
func (i *Instance) CanRestart() bool {
    // Gemini sessions with known session ID can always be restarted
    if i.Tool == "gemini" && i.GeminiSessionID != "" {
        return true
    }

    // Claude logic...
    if i.Tool == "claude" && i.ClaudeSessionID != "" {
        return true
    }

    // Other sessions: only if dead or error
    return i.Status == StatusError || i.tmuxSession == nil || !i.tmuxSession.Exists()
}
```

---

### Phase 4: Session Output & Response Extraction

**Goal:** Extract last assistant response from Gemini session files

**Implementation:**

```go
// internal/session/instance.go

// GetLastResponse returns the last assistant response from the session
func (i *Instance) GetLastResponse() (*ResponseOutput, error) {
    if i.Tool == "claude" {
        return i.getClaudeLastResponse()
    }
    if i.Tool == "gemini" {
        return i.getGeminiLastResponse()
    }
    return i.getTerminalLastResponse()
}

// getGeminiLastResponse extracts the last assistant message from Gemini's JSON file
func (i *Instance) getGeminiLastResponse() (*ResponseOutput, error) {
    sessionsDir := GetGeminiSessionsDir(i.ProjectPath)

    // Find the session file
    var sessionFile string
    if i.GeminiSessionID != "" {
        // Try to find file by session ID
        files, _ := filepath.Glob(filepath.Join(sessionsDir, "session-*-" + i.GeminiSessionID[:8] + ".json"))
        if len(files) > 0 {
            sessionFile = files[0]
        }
    }

    if sessionFile == "" {
        // Detect session
        sessionID := FindGeminiSessionForInstance(i.ProjectPath, i.CreatedAt.Add(-time.Hour), nil)
        if sessionID == "" {
            return nil, fmt.Errorf("no Gemini session found for this instance")
        }

        files, _ := filepath.Glob(filepath.Join(sessionsDir, "session-*-" + sessionID[:8] + ".json"))
        if len(files) == 0 {
            return nil, fmt.Errorf("session file not found")
        }
        sessionFile = files[0]
    }

    // Read and parse the JSON file
    data, err := os.ReadFile(sessionFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read session file: %w", err)
    }

    return parseGeminiLastAssistantMessage(data)
}

// parseGeminiLastAssistantMessage parses a Gemini JSON file to extract the last assistant message
func parseGeminiLastAssistantMessage(data []byte) (*ResponseOutput, error) {
    var session struct {
        SessionID string `json:"sessionId"`
        Messages  []struct {
            ID        string          `json:"id"`
            Timestamp string          `json:"timestamp"`
            Type      string          `json:"type"`
            Content   string          `json:"content"`
            ToolCalls []json.RawMessage `json:"toolCalls"`
            Thoughts  []json.RawMessage `json:"thoughts"`
            Model     string          `json:"model"`
            Tokens    json.RawMessage `json:"tokens"`
        } `json:"messages"`
    }

    if err := json.Unmarshal(data, &session); err != nil {
        return nil, fmt.Errorf("failed to parse session file: %w", err)
    }

    // Find last "gemini" type message
    for i := len(session.Messages) - 1; i >= 0; i-- {
        msg := session.Messages[i]
        if msg.Type == "gemini" {
            return &ResponseOutput{
                Tool:      "gemini",
                Role:      "assistant",
                Content:   msg.Content,
                Timestamp: msg.Timestamp,
                SessionID: session.SessionID,
            }, nil
        }
    }

    return nil, fmt.Errorf("no assistant response found in session")
}
```

---

## IV. CLI COMMANDS INTEGRATION

### 4.1 Existing Commands to Extend

**`session show [id]`** - Already generic, will work for Gemini

**`session restart <id>`** - Add Gemini logic (Phase 3)

**`session output [id]`** - Add Gemini response extraction (Phase 4)

**`mcp attached [id]`** - Extend to read from Gemini settings.json

**`mcp attach <id> <mcp>`** - Extend to write to Gemini settings.json

**`mcp detach <id> <mcp>`** - Extend to write to Gemini settings.json

### 4.2 New Commands to Add (Optional)

**`session list-gemini <project>`** - List all Gemini sessions for a project
```go
// cmd/agent-deck/session_cmd.go

func listGeminiSessionsCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list-gemini <project-path>",
        Short: "List all Gemini sessions for a project",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            sessions, err := session.ListGeminiSessions(args[0])
            if err != nil {
                return err
            }

            for _, s := range sessions {
                fmt.Printf("%s  %s  (%d messages)\n",
                    s.SessionID[:8],
                    s.LastUpdated.Format("2006-01-02 15:04"),
                    s.MessageCount)
            }
            return nil
        },
    }
}
```

---

## V. TESTING STRATEGY

### 5.1 Unit Tests

**`internal/session/gemini_test.go`:**
- ✅ `TestHashProjectPath` - Verify SHA256 algorithm
- ✅ `TestParseGeminiSessionFile` - Parse session JSON
- ✅ `TestFindGeminiSessionForInstance` - Timestamp filtering
- ✅ `TestGetGeminiMCPInfo` - Parse settings.json
- ✅ `TestWriteGeminiMCPSettings` - Atomic write

### 5.2 Integration Tests

**Manual Testing Workflow:**
1. ✅ Start Gemini CLI session normally: `gemini`
2. ✅ Import into agent-deck with `i` (discover existing tmux sessions)
3. ✅ Verify session ID detected in agent-deck UI
4. ✅ Restart session from agent-deck - should use `gemini --resume <id>`
5. ✅ Add MCP via MCP Manager (press `M`)
6. ✅ Restart session - verify MCP loads in Gemini
7. ✅ Test `session output` command - extract last response
8. ✅ Test session discovery on startup

### 5.3 Edge Cases

- [ ] Multiple Gemini sessions in same project
- [ ] Session file corruption (malformed JSON)
- [ ] Missing settings.json file
- [ ] Gemini not installed
- [ ] Session files with no messages
- [ ] Concurrent writes to settings.json

---

## VI. KNOWN LIMITATIONS & WORKAROUNDS

| Limitation | Impact | Workaround |
|------------|--------|------------|
| **No fork functionality** | Can't branch conversations | Use sub-sessions (set ParentSessionID) |
| **No config dir env var** | Can't use custom profiles | Hardcode `~/.gemini/` |
| **No tmux env capture** | Can't capture session ID from tmux env | File scanning only |
| **Global MCPs only** | No per-project MCP override | Document limitation |
| **Session format is JSON** | Different parser than Claude | Write separate parser |

---

## VII. IMPLEMENTATION CHECKLIST

### Phase 1: Core Session Detection (Week 1)
- [ ] Create `internal/session/gemini.go`
  - [ ] `GetGeminiConfigDir()`
  - [ ] `HashProjectPath()`
  - [ ] `GetGeminiSessionsDir()`
  - [ ] `ListGeminiSessions()`
  - [ ] `FindGeminiSessionForInstance()`
  - [ ] `parseGeminiSessionFile()`

- [ ] Modify `internal/session/instance.go`
  - [ ] Add `GeminiSessionID`, `GeminiDetectedAt` fields
  - [ ] Add `UpdateGeminiSession()`
  - [ ] Update `UpdateStatus()` to call `UpdateGeminiSession()`

- [ ] Create `internal/session/gemini_test.go`
  - [ ] Test hash algorithm
  - [ ] Test session file parsing
  - [ ] Test session discovery

- [ ] Test manually with real Gemini session

### Phase 2: MCP Integration (Week 2)
- [ ] Create `internal/session/gemini_mcp.go`
  - [ ] `GeminiMCPConfig` type
  - [ ] `GetGeminiMCPInfo()`
  - [ ] `WriteGeminiMCPSettings()`
  - [ ] `GetGeminiMCPNames()`

- [ ] Modify `internal/session/instance.go`
  - [ ] Update `GetMCPInfo()` to handle Gemini

- [ ] Modify `internal/ui/mcp_manager.go`
  - [ ] Add Gemini detection in `loadAvailableMCPs()`
  - [ ] Add Gemini writing in `applyChanges()`

- [ ] Test MCP attach/detach with Gemini session

### Phase 3: Command Building & Restart (Week 3)
- [ ] Modify `internal/session/instance.go`
  - [ ] Add `buildGeminiCommand()`
  - [ ] Add `buildGeminiCommandWithMessage()`
  - [ ] Update `Start()` to use `buildGeminiCommand()`
  - [ ] Update `Restart()` for Gemini
  - [ ] Update `CanRestart()` for Gemini

- [ ] Test session restart with resume

### Phase 4: Session Output (Week 3-4)
- [ ] Modify `internal/session/instance.go`
  - [ ] Add `getGeminiLastResponse()`
  - [ ] Add `parseGeminiLastAssistantMessage()`
  - [ ] Update `GetLastResponse()` to handle Gemini

- [ ] Test `session output` command with Gemini

### Documentation & Polish (Week 4)
- [ ] Update README with Gemini support
- [ ] Update CLAUDE.md with Gemini MCP management
- [ ] Add examples for Gemini sessions
- [ ] Update CHANGELOG
- [ ] Create demo video/tape for Gemini workflow

---

## VIII. SUCCESS CRITERIA

**Must Have (MVP):**
- ✅ Detect Gemini sessions from filesystem
- ✅ Display Gemini sessions in agent-deck UI with correct status
- ✅ Restart Gemini sessions with `gemini --resume <id>`
- ✅ Manage MCPs via MCP Manager UI
- ✅ Extract last response with `session output`

**Nice to Have (v2):**
- ⭐ CLI commands for Gemini session management
- ⭐ Sub-session workflow as fork alternative
- ⭐ Integration with Gemini's checkpointing system
- ⭐ Support for Gemini's "thoughts" (extended thinking)

**Performance:**
- Session detection should complete in <100ms
- MCP updates should not block UI
- No regressions to existing Claude integration

---

## IX. RISK MITIGATION

| Risk | Mitigation |
|------|------------|
| **Breaking existing Claude integration** | Comprehensive tests before merge |
| **Session ID detection failures** | Fallback to manual session ID input |
| **MCP config corruption** | Atomic writes with .tmp files |
| **Gemini CLI updates breaking detection** | Version detection and warnings |
| **Performance degradation** | Cache session list, use 30s recheck for ghosts |

---

## X. DEPENDENCIES

**External:**
- Gemini CLI must be installed (`gemini --version`)
- Go 1.21+ (for SHA256 crypto package)

**Internal:**
- Existing session management infrastructure
- MCP pool system (optional, for socket pooling)
- tmux integration layer

---

## XI. NEXT STEPS

1. **Review this plan** with stakeholders
2. **Create feature branch** `feature/gemini-integration`
3. **Implement Phase 1** (session detection)
4. **Test manually** with real Gemini sessions
5. **Iterate** based on findings
6. **Document** as you go

---

**Questions or Blockers?**
- Open an issue: `github.com/asheshgoplani/agent-deck/issues`
- Discussion: `github.com/asheshgoplani/agent-deck/discussions`

---

**End of Implementation Plan**

*Last Updated: 2025-12-26*
