# Google Gemini CLI - Configuration and Settings Structure Research

**Research Date**: 2025-12-26
**Purpose**: Understanding Gemini CLI configuration for agent-deck Gemini integration

---

## 1. Configuration Files

### 1.1 Main Configuration File

**Primary Config File**: `settings.json` (JSON format, NOT TOML)

**Location Hierarchy** (in precedence order, lowest to highest):
1. **Default values** - Hardcoded application defaults
2. **System defaults** - Platform-specific:
   - Linux: `/etc/gemini-cli/system-defaults.json`
   - Windows: `C:\ProgramData\gemini-cli\system-defaults.json`
   - macOS: `/Library/Application Support/GeminiCli/system-defaults.json`
   - Override with: `GEMINI_CLI_SYSTEM_DEFAULTS_PATH` env var
3. **User settings** - `~/.gemini/settings.json` (applies to ALL sessions for current user)
4. **Project settings** - `.gemini/settings.json` (in project root, project-specific)
5. **System settings** - Platform-specific:
   - Linux: `/etc/gemini-cli/settings.json`
   - Windows: `C:\ProgramData\gemini-cli\settings.json`
   - macOS: `/Library/Application Support/GeminiCli/settings.json`
   - Override with: `GEMINI_CLI_SYSTEM_SETTINGS_PATH` env var
6. **Environment variables** - Including `.env` files
7. **Command-line arguments** - Highest priority

### 1.2 JSON Schema

**Schema Location**: `https://raw.githubusercontent.com/google-gemini/gemini-cli/main/schemas/settings.schema.json`

**Usage**: JSON-aware editors can use autocomplete and validation by adding:
```json
{
  "$schema": "https://raw.githubusercontent.com/google-gemini/gemini-cli/main/schemas/settings.schema.json"
}
```

**Known Issue**: Schema has `additionalProperties: false` at root and doesn't explicitly include `$schema` property, causing some editors to show validation errors.

---

## 2. Directory Structure

### 2.1 Global User Directory: `~/.gemini/`

```
~/.gemini/
├── settings.json                    # User-level configuration
├── trusted-folders.json             # Workspace trust database
├── oauth_creds.json                 # OAuth tokens (auto-refreshed)
├── user_id                          # Unique user identifier
├── google_account_id                # Google account reference
├── mcp-oauth-tokens.json            # MCP server OAuth authentication
├── .env                             # User-wide environment variables
├── .oauth/
│   └── tokens.json                  # Encrypted access/refresh tokens
├── history/
│   └── <project-hash>/              # Git shadow repository for checkpoints
│       └── .git/                    # Git snapshot storage
├── tmp/
│   └── <project-hash>/              # Project-specific temporary files
│       ├── shell_history            # Shell command history
│       ├── chats/                   # Session storage
│       │   └── <session-uuid>.json  # Individual session data
│       ├── checkpoints/             # Conversation history JSON
│       └── otel/                    # OpenTelemetry logs
│           ├── collector.log        # Local telemetry logs
│           └── collector-gcp.log    # GCP telemetry logs
└── extensions/                      # Extension installation directory
    └── <extension-name>/
        ├── gemini-extension.json    # Extension manifest
        ├── GEMINI.md                # Extension context file
        └── mcpServers.json          # Extension MCP definitions
```

**Notes**:
- `<project-hash>`: Unique identifier generated from project's root path
- `<session-uuid>`: UUID for individual sessions

### 2.2 Project-Specific Directory: `.gemini/` (in project root)

```
<project-root>/
└── .gemini/
    ├── settings.json                # Project-specific settings (overrides user settings)
    ├── .env                         # Project environment variables
    ├── sandbox-macos-custom.sb      # Custom macOS sandbox profile
    ├── sandbox.Dockerfile           # Custom Docker sandbox
    └── telemetry.log                # Local telemetry output (if enabled)
```

### 2.3 Alternative Chat Storage (Manual Saves)

**Manual chat checkpoints** (using `/chat save` command):
- **Linux/macOS**: `~/.config/google-generative-ai/checkpoints/`
- **Windows**: `C:\Users\<YourUsername>\AppData\Roaming\google-generative-ai\checkpoints\`

**Note**: Chats are saved into project-specific directories and only accessible from the same project.

---

## 3. Configuration Settings Structure

### 3.1 Settings Schema Categories (20 Primary Categories)

| Category | Type | Purpose |
|----------|------|---------|
| `mcpServers` | Object | MCP server configurations |
| `general` | Object | Application-wide behaviors |
| `output` | Object | CLI output formatting |
| `ui` | Object | User interface settings |
| `ide` | Object | IDE integration controls |
| `privacy` | Object | Data collection preferences |
| `telemetry` | Object | Observability configuration |
| `model` | Object | Generative model parameters |
| `modelConfigs` | Object | Named model presets |
| `context` | Object | Context management |
| `tools` | Object | Built-in/custom tool settings |
| `mcp` | Object | MCP server configuration |
| `security` | Object | Authentication/safety |
| `advanced` | Object | Power-user configs |
| `experimental` | Object | Preview feature toggles |
| `extensions` | Object | Extension management |
| `hooks` | Object | Event lifecycle interceptors |
| `checkpointing` | Object | Checkpoint settings |
| `fileFiltering` | Object | Git-aware file filtering |
| `sandbox` | Boolean/String | Sandboxing configuration |

### 3.2 Key Configurable Settings

#### General Settings
```json
{
  "general": {
    "theme": "Default",                        // Visual theme
    "vimMode": false,                          // Vim-style editing
    "hideTips": false,                         // Suppress tips
    "hideBanner": false,                       // Hide ASCII logo
    "preferredEditor": "vscode",               // Editor for diffs
    "usageStatisticsEnabled": true,            // Usage metrics
    "maxSessionTurns": -1,                     // Session turn limit (-1 = unlimited)
    "autoAccept": false,                       // Auto-approve safe operations
    "contextFileName": "GEMINI.md",            // Context file name (string or array)
    "loadMemoryFromIncludeDirectories": false  // Load context from included dirs
  }
}
```

#### UI Configuration
```json
{
  "ui": {
    "theme": "Default",
    "hideBanner": false,
    "hideTips": false,
    "showLineNumbers": true,
    "showCitations": true,
    "windowTitleDisplay": true,
    "footerVisibility": true,
    "accessibilityFeatures": true,
    "screenReaderSupport": false
  }
}
```

#### Tool Management
```json
{
  "tools": {
    "coreTools": ["tool1", "tool2"],           // Restrict available tools
    "excludeTools": ["dangerous-tool"],        // Blacklist tools
    "allowMCPServers": ["server1", "server2"], // Whitelist MCP servers
    "excludeMCPServers": ["untrusted"],        // Blacklist MCP servers
    "toolDiscoveryCommand": "my-tool-discovery", // Custom tool discovery
    "toolCallCommand": "my-tool-executor"      // Custom tool executor
  }
}
```

#### File & Context
```json
{
  "context": {
    "contextFileName": ["GEMINI.md", "README.md"],
    "includeDirectories": ["/path/to/extra/context"]
  },
  "fileFiltering": {
    "respectGitIgnore": true,
    "enableRecursiveFileSearch": true
  }
}
```

#### MCP Servers
```json
{
  "mcpServers": {
    "server-name": {
      "command": "npx",
      "args": ["-y", "@package/name"],
      "env": {
        "API_KEY": "value"
      },
      "cwd": "/path/to/working/directory",
      "timeout": 30000,
      "trust": "required"
    }
  }
}
```

#### Checkpointing
```json
{
  "checkpointing": {
    "enabled": false  // Disabled by default, enable via settings.json only
  }
}
```

#### Telemetry
```json
{
  "telemetry": {
    "enabled": true,
    "target": "local",              // "local" or "gcp"
    "endpoint": ".gemini/telemetry.log",
    "logPrompts": false
  }
}
```

#### Sandbox
```json
{
  "sandbox": true,           // boolean: true/false
  // OR
  "sandbox": "docker",       // string: "docker" or "podman"
  "sandboxProfile": ".gemini/sandbox-macos-custom.sb"
}
```

#### Model Configuration
```json
{
  "model": {
    "maxSessionTurns": -1,
    "summarizeToolOutput": {
      "enabled": true,
      "tokenBudget": 1000
    }
  },
  "modelConfigs": {
    "my-alias": {
      "inheritsFrom": "base-config",
      "modelName": "gemini-2.5-pro",
      "thinkingBudget": 1000,
      "thinkingLevel": "high"
    }
  }
}
```

#### Hooks System
```json
{
  "hooks": {
    "BeforeTool": ["command", "arg1"],
    "AfterTool": ["command", "arg1"],
    "BeforeAgent": ["command"],
    "AfterAgent": ["command"],
    "BeforeModel": ["command"],
    "AfterModel": ["command"],
    "SessionStart": ["command"],
    "SessionEnd": ["command"]
  }
}
```

### 3.3 Example Complete Settings File

```json
{
  "$schema": "https://raw.githubusercontent.com/google-gemini/gemini-cli/main/schemas/settings.schema.json",
  "general": {
    "theme": "Default",
    "vimMode": false,
    "autoAccept": false,
    "preferredEditor": "vscode",
    "contextFileName": "GEMINI.md"
  },
  "ui": {
    "hideBanner": false,
    "hideTips": false
  },
  "fileFiltering": {
    "respectGitIgnore": true,
    "enableRecursiveFileSearch": true
  },
  "checkpointing": {
    "enabled": false
  },
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/Users/username"],
      "trust": "required"
    }
  },
  "telemetry": {
    "enabled": true,
    "target": "local",
    "endpoint": ".gemini/telemetry.log"
  }
}
```

---

## 4. Authentication

### 4.1 Authentication Methods

1. **Login with Google (Recommended)**
   - Interactive browser-based authentication
   - Credentials cached at `~/.gemini/oauth_creds.json`
   - Tokens stored at `~/.gemini/.oauth/tokens.json` (encrypted)
   - Command: `gemini` → select "Login with Google"

2. **Gemini API Key**
   - Non-interactive method
   - Set via `GEMINI_API_KEY` environment variable
   - Can be obtained from Google AI Studio

3. **Vertex AI (Multiple Options)**

   **Option A: Application Default Credentials (ADC)**
   - Command: `gcloud auth application-default login`
   - Requires Google Cloud CLI
   - Must unset `GOOGLE_API_KEY` and `GEMINI_API_KEY` if previously set

   **Option B: Service Account JSON Key**
   - Set `GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json`
   - Recommended for CI/CD pipelines
   - Requires "Vertex AI User" role

   **Option C: Google Cloud API Key**
   - Set `GOOGLE_API_KEY` environment variable
   - May be restricted by organizational policies

### 4.2 Credential Storage Locations

| Type | Location | Format |
|------|----------|--------|
| OAuth tokens | `~/.gemini/oauth_creds.json` | JSON (auto-refreshed) |
| OAuth tokens (encrypted) | `~/.gemini/.oauth/tokens.json` | JSON (encrypted) |
| MCP OAuth tokens | `~/.gemini/mcp-oauth-tokens.json` | JSON |
| User ID | `~/.gemini/user_id` | Plain text |
| Google Account ID | `~/.gemini/google_account_id` | Plain text |

### 4.3 Environment Variables for Authentication

| Variable | Purpose |
|----------|---------|
| `GEMINI_API_KEY` | Gemini API key |
| `GOOGLE_API_KEY` | Vertex AI API key |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to service account JSON |
| `GOOGLE_CLOUD_PROJECT` | Google Cloud project ID |
| `GOOGLE_CLOUD_LOCATION` | Vertex AI resource location (e.g., us-central1) |

---

## 5. Runtime Paths and Session Data

### 5.1 Session Storage

**Location**: `~/.gemini/tmp/<project-hash>/chats/`

**Session Data Structure** (stored in `<session-uuid>.json`):
```json
{
  "sessionId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "createdAt": "2025-12-26T10:00:00Z",
  "messages": [
    {
      "role": "user",
      "content": "Hello"
    },
    {
      "role": "assistant",
      "content": "Hi there!",
      "toolCalls": [],
      "tokenUsage": {
        "inputTokens": 10,
        "outputTokens": 5,
        "cachedTokens": 0
      }
    }
  ],
  "toolExecutions": [],
  "reasoning": []
}
```

**Session Commands**:
- `gemini --resume` - Resume most recent session
- `gemini --resume [index]` - Resume by position
- `gemini --resume [UUID]` - Resume by full UUID
- `gemini --list-sessions` - List all sessions
- `/resume` - Interactive session browser

### 5.2 Checkpointing

**Feature**: Automatic snapshots before file modifications

**Storage Locations**:
1. **Git snapshot**: `~/.gemini/history/<project-hash>/.git/`
2. **Conversation JSON**: `~/.gemini/tmp/<project-hash>/checkpoints/`

**Configuration** (in `settings.json`):
```json
{
  "checkpointing": {
    "enabled": false  // Disabled by default
  }
}
```

**Note**: Command-line flag `--checkpointing` removed in v0.11.0. Must be enabled via settings.json.

### 5.3 Shell History

**Location**: `~/.gemini/tmp/<project-hash>/shell_history`

**Scope**: Project-specific, isolated per project hash

### 5.4 Logs

| Type | Location | Purpose |
|------|----------|---------|
| Local telemetry | `.gemini/telemetry.log` | Development/debugging logs |
| OTEL collector (local) | `~/.gemini/tmp/<project-hash>/otel/collector.log` | OpenTelemetry local logs |
| OTEL collector (GCP) | `~/.gemini/tmp/<project-hash>/otel/collector-gcp.log` | GCP telemetry logs |

### 5.5 Temporary Files

**Base Directory**: `~/.gemini/tmp/<project-hash>/`

**Contents**:
- Shell history
- Chat sessions
- Checkpoints
- Telemetry logs

---

## 6. Project-Specific Configuration

### 6.1 Project Detection

**Project Hash Calculation**:
- Unique identifier generated from project's root path
- Used to isolate session data, history, and temporary files
- Switching directories switches to different project's session history

**Workspace Detection**:
- CLI operates within a `rootDirectory` (usually CWD where CLI launched)
- All file system tools resolve paths relative to root directory
- IDE integration uses `GEMINI_CLI_IDE_WORKSPACE_PATH` env var

### 6.2 Project Settings Scope

**Global Settings** (`~/.gemini/settings.json`):
- Apply to ALL Gemini CLI sessions for current user
- User-wide defaults

**Project Settings** (`.gemini/settings.json` in project root):
- Apply ONLY to sessions launched from that project
- Override user settings
- Project-specific MCP servers, tools, context files

### 6.3 Context File Loading

**Default Context File**: `GEMINI.md`

**Search Locations** (in order):
1. `.gemini/GEMINI.md` (project-specific)
2. `GEMINI.md` (project root)
3. `~/.gemini/GEMINI.md` (global, loaded for ALL sessions)

**Configuration**:
```json
{
  "general": {
    "contextFileName": ["GEMINI.md", "README.md"]  // Multiple filenames supported
  }
}
```

---

## 7. Environment Variables

### 7.1 Environment File Loading

**Search Order** (loads from FIRST file found, NOT merged):
1. `.env` in current working directory
2. `.env` in parent directories (up to project root with `.git` folder)
3. `~/.gemini/.env` (recommended for user-wide settings)
4. `~/.env`

**Project-Specific**:
- `.gemini/.env` variables are NEVER excluded
- Other `.env` files exclude certain variables (e.g., `DEBUG`, `DEBUG_MODE`) to prevent interference

### 7.2 Key Environment Variables

#### Essential
| Variable | Purpose |
|----------|---------|
| `GEMINI_API_KEY` | Gemini API key (required) |

#### Model & API
| Variable | Purpose |
|----------|---------|
| `GEMINI_MODEL` | Override default model |
| `GEMINI_API_KEY_AUTH_MECHANISM` | Authentication method ("x-goog-api-key" or "bearer") |
| `GEMINI_CLI_CUSTOM_HEADERS` | Extra HTTP headers (comma-separated) |

#### Google Cloud
| Variable | Purpose |
|----------|---------|
| `GOOGLE_CLOUD_PROJECT` | Project ID |
| `GOOGLE_CLOUD_LOCATION` | Region (e.g., us-central1) |
| `GOOGLE_API_KEY` | Vertex AI API key |
| `GOOGLE_APPLICATION_CREDENTIALS` | Service account JSON path |

#### IDE Integration
| Variable | Purpose |
|----------|---------|
| `GEMINI_CLI_IDE_WORKSPACE_PATH` | IDE workspace path |
| `GEMINI_CLI_IDE_SERVER_PORT` | IDE server port |

#### System Paths
| Variable | Purpose |
|----------|---------|
| `GEMINI_CLI_SYSTEM_DEFAULTS_PATH` | Custom system defaults location |
| `GEMINI_CLI_SYSTEM_SETTINGS_PATH` | Custom system settings location |

#### Telemetry
| Variable | Purpose |
|----------|---------|
| `OTLP_GOOGLE_CLOUD_PROJECT` | Telemetry project |

#### Network
| Variable | Purpose |
|----------|---------|
| `HTTP_PROXY` | HTTP proxy |
| `HTTPS_PROXY` | HTTPS proxy |
| `GEMINI_SANDBOX` | Override sandbox setting |

---

## 8. Comparison with Claude CLI

### 8.1 Similarities

| Feature | Claude CLI | Gemini CLI |
|---------|------------|------------|
| Config format | JSON | JSON |
| User config | `~/.claude/.claude.json` | `~/.gemini/settings.json` |
| Project config | `.claude.json` | `.gemini/settings.json` |
| Session storage | `~/.claude/projects/` | `~/.gemini/tmp/<project-hash>/chats/` |
| Context files | `CLAUDE.md` | `GEMINI.md` |
| MCP support | Yes | Yes |
| Environment files | Yes | Yes |

### 8.2 Key Differences

| Aspect | Claude CLI | Gemini CLI |
|--------|------------|------------|
| **Session ID tracking** | `lastSessionId` in config | UUID-based, stored in `~/.gemini/tmp/<project-hash>/chats/` |
| **Project tracking** | Explicit "projects" map in config | Implicit via project hash calculation from CWD |
| **Session resume** | `--resume <session-id>` | `--resume [index|UUID]` or interactive `/resume` |
| **Checkpointing** | Not documented | Git shadow repository at `~/.gemini/history/<project-hash>/` |
| **Authentication** | API key only | OAuth, API key, Vertex AI (ADC, Service Account) |
| **Config hierarchy** | User → Project → CLI args | System defaults → User → Project → System settings → Env → CLI args |
| **Telemetry** | Not documented | OpenTelemetry with local/GCP targets |

### 8.3 Critical Differences for agent-deck

1. **No explicit project tracking**: Gemini doesn't maintain a "projects" map like Claude. Projects are identified by hash of their root path.

2. **Session ID storage**: Claude stores `lastSessionId` in config. Gemini stores full session data in `~/.gemini/tmp/<project-hash>/chats/<session-uuid>.json`.

3. **Resume mechanism**:
   - Claude: `claude --resume <session-id>`
   - Gemini: `gemini --resume [index|UUID]` or interactive browser

4. **Checkpointing**: Gemini has automatic checkpointing with Git snapshots. Claude doesn't document this feature.

---

## 9. TypeScript Types (Inferred from Schema)

### 9.1 Settings Interface (Partial)

```typescript
interface Settings {
  $schema?: string;
  mcpServers?: Record<string, MCPServerConfig>;
  general?: GeneralSettings;
  output?: OutputSettings;
  ui?: UISettings;
  ide?: IDESettings;
  privacy?: PrivacySettings;
  telemetry?: TelemetrySettings;
  model?: ModelSettings;
  modelConfigs?: Record<string, ModelConfig>;
  context?: ContextSettings;
  tools?: ToolsSettings;
  mcp?: MCPSettings;
  security?: SecuritySettings;
  advanced?: AdvancedSettings;
  experimental?: ExperimentalSettings;
  extensions?: ExtensionsSettings;
  hooks?: HooksSettings;
  checkpointing?: CheckpointingSettings;
  fileFiltering?: FileFilteringSettings;
  sandbox?: boolean | "docker" | "podman";
}

interface MCPServerConfig {
  command: string;
  args?: string[];
  env?: Record<string, string>;
  cwd?: string;
  timeout?: number;
  trust?: "required" | "optional";
}

interface GeneralSettings {
  theme?: string;
  vimMode?: boolean;
  hideTips?: boolean;
  hideBanner?: boolean;
  preferredEditor?: string;
  usageStatisticsEnabled?: boolean;
  maxSessionTurns?: number;
  autoAccept?: boolean;
  contextFileName?: string | string[];
  loadMemoryFromIncludeDirectories?: boolean;
}

interface UISettings {
  theme?: string;
  hideBanner?: boolean;
  hideTips?: boolean;
  showLineNumbers?: boolean;
  showCitations?: boolean;
  windowTitleDisplay?: boolean;
  footerVisibility?: boolean;
  accessibilityFeatures?: boolean;
  screenReaderSupport?: boolean;
}

interface ToolsSettings {
  coreTools?: string[];
  excludeTools?: string[];
  allowMCPServers?: string[];
  excludeMCPServers?: string[];
  toolDiscoveryCommand?: string;
  toolCallCommand?: string;
}

interface CheckpointingSettings {
  enabled: boolean;
}

interface FileFilteringSettings {
  respectGitIgnore?: boolean;
  enableRecursiveFileSearch?: boolean;
}

interface TelemetrySettings {
  enabled: boolean;
  target: "local" | "gcp";
  endpoint?: string;
  logPrompts?: boolean;
}

interface HooksSettings {
  BeforeTool?: string[];
  AfterTool?: string[];
  BeforeAgent?: string[];
  AfterAgent?: string[];
  BeforeModel?: string[];
  AfterModel?: string[];
  SessionStart?: string[];
  SessionEnd?: string[];
}
```

### 9.2 Session Data Interface (Inferred)

```typescript
interface Session {
  sessionId: string;  // UUID
  createdAt: string;  // ISO 8601
  messages: Message[];
  toolExecutions: ToolExecution[];
  reasoning?: string[];
}

interface Message {
  role: "user" | "assistant" | "system";
  content: string;
  toolCalls?: ToolCall[];
  tokenUsage?: TokenUsage;
}

interface TokenUsage {
  inputTokens: number;
  outputTokens: number;
  cachedTokens: number;
}

interface ToolCall {
  id: string;
  name: string;
  arguments: Record<string, unknown>;
}

interface ToolExecution {
  toolCallId: string;
  result: unknown;
  status: "success" | "error";
}
```

---

## 10. Key Findings for agent-deck Integration

### 10.1 Critical Insights

1. **No explicit project map**: Unlike Claude's `.claude.json` with a "projects" object, Gemini doesn't maintain an explicit map of projects. Projects are identified by hashing their root path.

2. **Session tracking**: Gemini stores full session data in individual JSON files at `~/.gemini/tmp/<project-hash>/chats/<session-uuid>.json`. There's no single "lastSessionId" in a config file.

3. **Config file format**: JSON only (NOT TOML like Claude's optional format).

4. **Authentication complexity**: Multiple auth methods (OAuth, API key, Vertex AI with 3 sub-options) stored in various files (`oauth_creds.json`, `~/.oauth/tokens.json`, env vars).

5. **Checkpointing system**: Unique feature with Git shadow repository for snapshots.

### 10.2 Implications for agent-deck

#### For Session Tracking:
- **Challenge**: No single source of truth for "current session" like Claude's `lastSessionId`
- **Solution**: Need to parse `~/.gemini/tmp/<project-hash>/chats/` directory to find sessions, or track via tmux environment variables

#### For Session Resume:
- **Challenge**: Gemini uses `--resume [index|UUID]` instead of explicit session ID
- **Solution**: Store session UUID in tmux environment variable (like `GEMINI_SESSION_ID`) and use `gemini --resume <uuid>`

#### For Project Detection:
- **Challenge**: No explicit project-to-path mapping
- **Solution**: Derive project hash from path, or use CWD detection

#### For Configuration:
- **Location**: Check both `~/.gemini/settings.json` and `.gemini/settings.json`
- **Format**: JSON parsing (simpler than TOML)
- **MCP servers**: Similar structure to Claude, stored in `mcpServers` object

### 10.3 Recommended Implementation Strategy

1. **Session ID Capture**:
   ```bash
   # After starting Gemini, capture session UUID from output or list-sessions
   session_id=$(gemini --list-sessions --format json | jq -r '.[0].sessionId')
   tmux set-environment GEMINI_SESSION_ID "$session_id"
   ```

2. **Session Resume**:
   ```bash
   gemini --resume "$GEMINI_SESSION_ID"
   ```

3. **Config Reading**:
   ```go
   // Read settings.json from both user and project locations
   userConfig := readJSON("~/.gemini/settings.json")
   projectConfig := readJSON(".gemini/settings.json")
   mergedConfig := mergeConfigs(userConfig, projectConfig)
   ```

4. **MCP Management**:
   ```go
   // Similar to Claude, but modify settings.json instead of .claude.json
   func AttachMCP(sessionID, mcpName string, scope string) {
       configPath := getConfigPath(scope) // ~/.gemini/settings.json or .gemini/settings.json
       config := readJSON(configPath)
       config["mcpServers"][mcpName] = mcpConfig
       writeJSON(configPath, config)
       // Restart session to apply changes
   }
   ```

---

## 11. References

### 11.1 Official Documentation

- [Gemini CLI GitHub Repository](https://github.com/google-gemini/gemini-cli)
- [Gemini CLI Official Website](https://geminicli.com/)
- [Gemini CLI Configuration Documentation](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/configuration.md)
- [Gemini CLI Get Started - Configuration](https://github.com/google-gemini/gemini-cli/blob/main/docs/get-started/configuration.md)
- [Gemini CLI Authentication](https://github.com/google-gemini/gemini-cli/blob/main/docs/get-started/authentication.md)
- [Gemini CLI Session Management](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/session-management.md)
- [Gemini CLI Checkpointing](https://geminicli.com/docs/cli/checkpointing/)
- [Gemini CLI Settings Schema](https://raw.githubusercontent.com/google-gemini/gemini-cli/main/schemas/settings.schema.json)

### 11.2 NPM Package

- [NPM: @google/gemini-cli](https://www.npmjs.com/package/@google/gemini-cli)
- [NPM: @google/gemini-cli-core](https://www.npmjs.com/package/@google/gemini-cli-core)

### 11.3 Community Resources

- [Google Gemini CLI Cheatsheet](https://www.philschmid.de/gemini-cli-cheatsheet)
- [Gemini CLI Tutorial - DEV Community](https://dev.to/auden/google-gemini-cli-tutorial-how-to-install-and-use-it-with-images-4phb)
- [Hands-on with Gemini CLI - Google Codelabs](https://codelabs.developers.google.com/gemini-cli-hands-on)
- [5 Things to Try with Gemini 3 Pro - Google Developers Blog](https://developers.googleblog.com/en/5-things-to-try-with-gemini-3-pro-in-gemini-cli/)
- [Session Management in Gemini CLI - Google Developers Blog](https://developers.googleblog.com/en/pick-up-exactly-where-you-left-off-with-session-management-in-gemini-cli/)

### 11.4 GitHub Issues (Relevant)

- [Issue #6709: Redesign Settings Schema with Logical Groups](https://github.com/google-gemini/gemini-cli/issues/6709)
- [Issue #12695: settings.schema.json reports "$schema is not allowed" error](https://github.com/google-gemini/gemini-cli/issues/12695)
- [Issue #7328: Checkpointing causes massive disk usage](https://github.com/google-gemini/gemini-cli/issues/7328)
- [Issue #4115: Checkpointing failing when not a Git repository](https://github.com/google-gemini/gemini-cli/issues/4115)
- [Issue #3383: Display MCP Server Working Directory (cwd)](https://github.com/google-gemini/gemini-cli/issues/3383)

---

## 12. Summary

**Gemini CLI** uses a JSON-based configuration system with a clear hierarchy (System defaults → User → Project → System settings → Env → CLI args). The primary user config is `~/.gemini/settings.json` with project overrides in `.gemini/settings.json`.

**Key differences from Claude**:
- No explicit project-to-path mapping (uses project hash)
- Session data stored in individual JSON files, not tracked via config
- More complex authentication (OAuth, API keys, Vertex AI)
- Built-in checkpointing with Git shadow repository

**For agent-deck integration**:
- Need to track session UUIDs via tmux environment variables
- Parse `~/.gemini/tmp/<project-hash>/chats/` for session discovery
- Modify `settings.json` for MCP management (similar to Claude)
- Handle project hash calculation or rely on CWD detection

**Configuration is straightforward**: JSON format, well-documented schema, similar MCP structure to Claude, making integration feasible with proper session tracking mechanism.
