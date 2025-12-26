# Google Gemini CLI - MCP Integration Research

**Research Date:** 2025-12-26
**Repository:** https://github.com/google-gemini/gemini-cli
**Version Researched:** Latest (main branch)

---

## 1. MCP Configuration

### 1.1 Settings File Location

Gemini CLI uses a **layered configuration system** with settings files in multiple locations:

| **Location** | **Path** | **Scope** | **Priority** |
|------------|---------|-----------|-------------|
| **User Settings** | `~/.gemini/settings.json` | All Gemini sessions for current user | Medium |
| **Project Settings** | `.gemini/settings.json` (in project root) | Project-specific | High |
| **System Defaults** | `/etc/gemini-cli/system-defaults.json` (Linux)<br>`/Library/Application Support/GeminiCli/system-defaults.json` (macOS)<br>`C:\ProgramData\gemini-cli\system-defaults.json` (Windows) | System-wide defaults | Lowest |
| **System Settings** | `/etc/gemini-cli/settings.json` (Linux)<br>`/Library/Application Support/GeminiCli/settings.json` (macOS)<br>`C:\ProgramData\gemini-cli\settings.json` (Windows) | System-wide overrides | Highest |

**Configuration Precedence (lowest to highest):**
1. Default values (hardcoded)
2. System defaults file
3. User settings file
4. Project settings file
5. System settings file (overrides all)
6. Environment variables
7. Command-line arguments

**Key Constant:**
```typescript
// packages/core/src/utils/paths.ts
export const GEMINI_DIR = '.gemini';
```

### 1.2 MCP Configuration Structure

MCPs are defined in the `mcpServers` object in `settings.json`:

```json
{
  "mcpServers": {
    "serverName": {
      "command": "path/to/server",
      "args": ["--arg1", "value1"],
      "env": {
        "API_KEY": "$MY_API_TOKEN"
      },
      "cwd": "./server-directory",
      "timeout": 30000,
      "trust": false
    }
  }
}
```

**Transport Types Supported:**
- **Stdio Transport:** Local subprocess via stdin/stdout
- **SSE Transport:** Server-Sent Events (remote)
- **HTTP Transport:** Streamable HTTP (remote)

### 1.3 Configuration Properties

**Required (one of):**
- `command` (string) - Executable path for stdio transport
- `url` (string) - SSE endpoint URL (e.g., `http://localhost:8080/sse`)
- `httpUrl` (string) - HTTP streaming endpoint URL

**Optional:**
- `args` (string[]) - Command-line arguments
- `headers` (object) - Custom HTTP headers for remote transports
- `env` (object) - Environment variables (supports `$VAR_NAME` or `${VAR_NAME}` syntax)
- `cwd` (string) - Working directory for stdio transport
- `timeout` (number) - Request timeout in ms (default: 600,000ms = 10 minutes)
- `trust` (boolean) - Bypass all tool call confirmations (default: false)
- `includeTools` (string[]) - Allowlist of tool names to include
- `excludeTools` (string[]) - Denylist of tool names to exclude (takes precedence over includeTools)
- `targetAudience` (string) - OAuth Client ID for IAP-protected apps
- `targetServiceAccount` (string) - Service Account email for impersonation

**Global MCP Settings:**

The `mcp` object in settings.json controls global MCP behavior:

```json
{
  "mcp": {
    "serverCommand": "global-command-to-start-servers",
    "allowed": ["trusted-server-1", "trusted-server-2"],
    "excluded": ["experimental-server"]
  }
}
```

- `mcp.serverCommand` - Global command to start an MCP server
- `mcp.allowed` - Allowlist of MCP server names (if set, only these will be loaded)
- `mcp.excluded` - Denylist of MCP server names

---

## 2. MCP Discovery & Loading

### 2.1 Discovery Process Architecture

**Key Files:**
- `packages/core/src/tools/mcp-client-manager.ts` - Manages lifecycle of all MCP clients
- `packages/core/src/tools/mcp-client.ts` - Individual MCP server client
- `packages/cli/src/config/settings.ts` - Settings file loading

**Discovery Flow:**

```
Startup
  ↓
McpClientManager.startConfiguredMcpServers()
  ↓
For each server in settings.json mcpServers:
  ↓
McpClientManager.maybeDiscoverMcpServer(name, config)
  ↓
  1. Check allowed/excluded lists (mcp.allowed, mcp.excluded)
  2. Check folder trust
  3. Check extension active status
  ↓
McpClient.connect()
  ↓
  - Select transport (httpUrl → StreamableHTTPClientTransport, url → SSEClientTransport, command → StdioClientTransport)
  - Establish connection
  - Update status: CONNECTING → CONNECTED
  ↓
McpClient.discover()
  ↓
  - discoverTools() - List available tools via tools/list
  - discoverPrompts() - List available prompts via prompts/list
  - discoverResources() - List available resources via resources/list
  ↓
  - Validate tool schemas
  - Filter tools (includeTools, excludeTools)
  - Sanitize tool names (max 63 chars, alphanumeric + _ . -)
  - Handle name conflicts (prefix with serverName__)
  ↓
ToolRegistry.registerTool()
  ↓
Gemini API configured with new tools
```

### 2.2 Discovery State Tracking

**Server Status Enum:**
```typescript
export enum MCPServerStatus {
  DISCONNECTED = 'disconnected',
  DISCONNECTING = 'disconnecting',
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
}
```

**Discovery State Enum:**
```typescript
export enum MCPDiscoveryState {
  NOT_STARTED = 'not_started',
  IN_PROGRESS = 'in_progress',
  COMPLETED = 'completed',
}
```

### 2.3 Programmatic Detection of Loaded MCPs

**From within Gemini CLI code:**

```typescript
// Get all loaded MCP servers
const mcpClientManager = config.getMcpClientManager();
const mcpServers = mcpClientManager.getMcpServers();
// Returns: Record<string, MCPServerConfig>

// Check discovery state
const discoveryState = mcpClientManager.getDiscoveryState();
// Returns: MCPDiscoveryState

// Get specific client
const client = mcpClientManager.getClient('serverName');
if (client) {
  const status = client.getStatus(); // MCPServerStatus
  const config = client.getServerConfig();
}
```

**From settings.json:**
```typescript
// Load settings
import { loadSettings } from './config/settings.js';
const settings = await loadSettings(cwd);

// Check mcpServers configuration
const configuredServers = settings.mcpServers;
```

### 2.4 Environment Variables for MCP Control

**No direct environment variables** control MCP behavior. However:

1. **Settings file location override:**
   - `GEMINI_CLI_SYSTEM_SETTINGS_PATH` - Override system settings path
   - `GEMINI_CLI_SYSTEM_DEFAULTS_PATH` - Override system defaults path

2. **MCP server env variables:**
   - Defined in `mcpServers[name].env` object
   - Supports `$VAR_NAME` or `${VAR_NAME}` syntax

---

## 3. MCP Runtime

### 3.1 Tool Exposure to Conversation

**Tool Registration Flow:**

1. **Discovery Phase:**
   ```typescript
   // packages/core/src/tools/mcp-client.ts
   async function discoverTools(
     serverName: string,
     serverConfig: MCPServerConfig,
     client: Client,
     cliConfig: Config,
     messageBus?: MessageBus,
     options?: { timeout?: number; signal?: AbortSignal }
   ): Promise<DiscoveredMCPTool[]>
   ```

2. **Tool Wrapping:**
   Each tool is wrapped in a `DiscoveredMCPTool` instance:
   ```typescript
   // packages/core/src/tools/mcp-tool.ts
   export class DiscoveredMCPTool implements CallableTool {
     constructor(
       private readonly serverName: string,
       private readonly serverToolName: string,
       private readonly tool: Tool,
       private readonly trust: boolean,
       private readonly client: Client,
       private readonly messageBus?: MessageBus
     )
   }
   ```

3. **Registration:**
   ```typescript
   // packages/core/src/tools/tool-registry.ts
   toolRegistry.registerTool(discoveredMCPTool);
   ```

4. **Gemini API Update:**
   After tools are registered, Gemini client is notified:
   ```typescript
   const geminiClient = cliConfig.getGeminiClient();
   if (geminiClient.isInitialized()) {
     await geminiClient.setTools(); // Updates Gemini API with new tool definitions
   }
   ```

### 3.2 Tool Name Sanitization & Conflict Resolution

**Sanitization Rules:**
- Invalid characters (non-alphanumeric, underscore, dot, hyphen) → replaced with underscores
- Names > 63 characters → truncated with middle replacement (`___`)

**Conflict Resolution:**
- **First registration wins:** First server to register a tool gets the unprefixed name
- **Subsequent servers:** Get prefixed names (`serverName__toolName`)
- Example: If `server1` and `server2` both expose `search` tool:
  - `server1` gets `search`
  - `server2` gets `server2__search`

### 3.3 Hot Reload / Runtime Attach/Detach

**Dynamic Tool Updates:**

Gemini CLI **supports dynamic tool updates** via MCP protocol notifications:

```typescript
// packages/core/src/tools/mcp-client.ts (lines 278-316)
private registerNotificationHandlers(): void {
  const capabilities = this.client.getServerCapabilities();

  // Listen for tool list changes
  if (capabilities?.tools?.listChanged) {
    this.client.setNotificationHandler(
      ToolListChangedNotificationSchema,
      async () => {
        await this.refreshTools(); // Re-query tools/list endpoint
      }
    );
  }

  // Listen for resource list changes
  if (capabilities?.resources?.listChanged) {
    this.client.setNotificationHandler(
      ResourceListChangedNotificationSchema,
      async () => {
        await this.refreshResources(); // Re-query resources/list endpoint
      }
    );
  }
}
```

**Tool Refresh Implementation (Coalescing Pattern):**

The `refreshTools()` method implements a **coalescing pattern** to handle rapid notification bursts:

```typescript
// packages/core/src/tools/mcp-client.ts (lines 391-451)
private async refreshTools(): Promise<void> {
  if (this.isRefreshingTools) {
    // Already refreshing, mark pending for another refresh
    this.pendingToolRefresh = true;
    return;
  }

  this.isRefreshingTools = true;

  try {
    do {
      this.pendingToolRefresh = false;

      // Re-discover tools
      const newTools = await this.discoverTools(this.cliConfig, { signal });

      // Remove old tools from this server
      this.toolRegistry.removeMcpToolsByServer(this.serverName);

      // Register new tools
      for (const tool of newTools) {
        this.toolRegistry.registerTool(tool);
      }

      // Update Gemini API
      if (this.onToolsUpdated) {
        await this.onToolsUpdated(abortController.signal);
      }

    } while (this.pendingToolRefresh); // Process any pending refreshes
  } finally {
    this.isRefreshingTools = false;
  }
}
```

**Manual Server Management:**

```typescript
// Restart a specific server
await mcpClientManager.restartServer('serverName');

// Restart all servers
await mcpClientManager.restart();

// Stop extension and its MCP servers
await mcpClientManager.stopExtension(extension);

// Start extension and its MCP servers
await mcpClientManager.startExtension(extension);
```

**Limitation:** MCPs **cannot be attached/detached** without restarting the server connection. However:
- **Dynamic updates** are supported if the server implements `tools.listChanged` capability
- **Server restart** can be done programmatically without restarting Gemini CLI

### 3.4 Tracking Active MCPs

**At Runtime:**

```typescript
// Get all connected MCP clients
const mcpClientManager = config.getMcpClientManager();
const clients = mcpClientManager.getMcpServers();

// Check which servers are connected
for (const [name, config] of Object.entries(clients)) {
  const client = mcpClientManager.getClient(name);
  if (client) {
    const status = client.getStatus();
    console.log(`${name}: ${status}`); // 'connected', 'disconnected', etc.
  }
}
```

**Via `/mcp` Command:**

Gemini CLI provides a built-in `/mcp` slash command to display:
- All configured MCP servers
- Connection status (CONNECTED, CONNECTING, DISCONNECTED)
- Server details (command, timeout, etc.)
- Available tools from each server
- Available prompts
- Available resources

---

## 4. CLI Commands for MCP Management

### 4.1 `gemini mcp add`

Add a new MCP server to settings.json:

```bash
# Basic stdio server
gemini mcp add my-server python server.py

# With environment variables
gemini mcp add -e API_KEY=123 -e DEBUG=true my-server /path/to/server arg1 arg2

# HTTP server with auth
gemini mcp add --transport http --header "Authorization: Bearer token" http-server https://api.example.com/mcp

# SSE server
gemini mcp add --transport sse sse-server https://api.example.com/sse

# With tool filtering
gemini mcp add --include-tools "tool1,tool2" --exclude-tools "dangerous_tool" my-server python server.py
```

**Flags:**
- `-s, --scope` - Configuration scope (user or project) [default: "project"]
- `-t, --transport` - Transport type (stdio, sse, http) [default: "stdio"]
- `-e, --env` - Set environment variables (e.g., `-e KEY=value`)
- `-H, --header` - Set HTTP headers (e.g., `-H "X-Api-Key: abc123"`)
- `--timeout` - Connection timeout in milliseconds
- `--trust` - Trust the server (bypass confirmation prompts)
- `--description` - Server description
- `--include-tools` - Comma-separated list of tools to include
- `--exclude-tools` - Comma-separated list of tools to exclude

### 4.2 `gemini mcp list`

List all configured MCP servers:

```bash
gemini mcp list
```

**Example Output:**
```
✓ stdio-server: command: python3 server.py (stdio) - Connected
✓ http-server: https://api.example.com/mcp (http) - Connected
✗ sse-server: https://api.example.com/sse (sse) - Disconnected
```

### 4.3 `gemini mcp remove`

Remove an MCP server:

```bash
gemini mcp remove my-server

# Specify scope
gemini mcp remove -s user my-server
```

### 4.4 `/mcp` In-Chat Command

Interactive command within Gemini CLI:

```bash
/mcp               # Show all MCP servers, tools, prompts, resources
/mcp auth          # List servers requiring authentication
/mcp auth serverName  # Authenticate with specific server
```

---

## 5. OAuth Authentication for Remote MCPs

### 5.1 Automatic OAuth Discovery

Gemini CLI supports **automatic OAuth discovery** for remote MCP servers:

```json
{
  "mcpServers": {
    "remote-server": {
      "url": "https://api.example.com/sse"
    }
  }
}
```

**OAuth Flow:**
1. Initial connection fails with 401 Unauthorized
2. OAuth endpoints discovered from server metadata
3. Dynamic client registration (if supported)
4. Browser opens for user authentication (requires local browser access)
5. Authorization code exchanged for access tokens
6. Tokens stored in `~/.gemini/mcp-oauth-tokens.json`
7. Connection retry succeeds with valid tokens

**Browser Requirements:**
- Local machine must be able to open a web browser
- Must be able to receive redirects on `http://localhost:7777/oauth/callback`

**Not supported in:**
- Headless environments without browser access
- Remote SSH sessions without X11 forwarding
- Containerized environments without browser support

### 5.2 Authentication Provider Types

**Configured via `authProviderType` property:**

```typescript
export enum AuthProviderType {
  DYNAMIC_DISCOVERY = 'dynamic_discovery',        // Auto-discover OAuth config (default)
  GOOGLE_CREDENTIALS = 'google_credentials',      // Google Application Default Credentials
  SERVICE_ACCOUNT_IMPERSONATION = 'service_account_impersonation'  // GCP Service Account
}
```

**Google Credentials Example:**
```json
{
  "mcpServers": {
    "gcp-server": {
      "httpUrl": "https://my-service.run.app/mcp",
      "authProviderType": "google_credentials",
      "oauth": {
        "scopes": ["https://www.googleapis.com/auth/userinfo.email"]
      }
    }
  }
}
```

**Service Account Impersonation Example:**
```json
{
  "mcpServers": {
    "iap-server": {
      "url": "https://my-iap-service.run.app/sse",
      "authProviderType": "service_account_impersonation",
      "targetAudience": "YOUR_CLIENT_ID.apps.googleusercontent.com",
      "targetServiceAccount": "your-sa@your-project.iam.gserviceaccount.com"
    }
  }
}
```

### 5.3 OAuth Token Management

**Token Storage:**
- File: `~/.gemini/mcp-oauth-tokens.json`
- Managed by `MCPOAuthTokenStorage` class

**Automatic Token Management:**
- Stored securely after successful authentication
- Refreshed when expired (if refresh tokens available)
- Validated before each connection attempt
- Cleaned up when invalid or expired

---

## 6. MCP Tool Execution & Confirmation

### 6.1 Confirmation System

**Trust-Based Bypass:**
```typescript
if (this.trust) {  // serverConfig.trust = true
  return false;  // No confirmation needed
}
```

**Dynamic Allow-Listing:**

The system maintains internal allow-lists:
- **Server-level:** `serverName` → All tools from this server are trusted
- **Tool-level:** `serverName.toolName` → This specific tool is trusted

**User Choices During Confirmation:**
- **Proceed once:** Execute this time only
- **Always allow this tool:** Add to tool-level allow-list
- **Always allow this server:** Add to server-level allow-list
- **Cancel:** Abort execution

### 6.2 Tool Execution Flow

```
Model generates FunctionCall
  ↓
DiscoveredMCPTool.execute()
  ↓
Check confirmation (if not trusted):
  - Check server-level allow-list
  - Check tool-level allow-list
  - Prompt user if needed
  ↓
If confirmed:
  ↓
Validate parameters against schema
  ↓
CallableTool.invoke(serverToolName, params)
  ↓
MCP server executes tool
  ↓
Return response:
  - llmContent: Raw response for LLM context
  - returnDisplay: Formatted output for user
```

### 6.3 Rich Content Responses

MCP tools can return **multi-part content** (text, images, audio, binary data):

```json
{
  "content": [
    {
      "type": "text",
      "text": "Here is the logo you requested."
    },
    {
      "type": "image",
      "data": "BASE64_ENCODED_IMAGE_DATA",
      "mimeType": "image/png"
    },
    {
      "type": "text",
      "text": "The logo was created in 2025."
    }
  ]
}
```

**Supported Content Types:**
- `text`
- `image`
- `audio`
- `resource` (embedded content)
- `resource_link`

---

## 7. MCP Prompts as Slash Commands

### 7.1 Defining Prompts on Server

Example using `@modelcontextprotocol/sdk`:

```typescript
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { z } from 'zod';

const server = new McpServer({
  name: 'prompt-server',
  version: '1.0.0',
});

server.registerPrompt(
  'poem-writer',
  {
    title: 'Poem Writer',
    description: 'Write a nice haiku',
    argsSchema: { title: z.string(), mood: z.string().optional() },
  },
  ({ title, mood }) => ({
    messages: [
      {
        role: 'user',
        content: {
          type: 'text',
          text: `Write a haiku${mood ? ` with the mood ${mood}` : ''} called ${title}`,
        },
      },
    ],
  }),
);

const transport = new StdioServerTransport();
await server.connect(transport);
```

### 7.2 Invoking Prompts

**Named arguments:**
```bash
/poem-writer --title="Gemini CLI" --mood="reverent"
```

**Positional arguments:**
```bash
/poem-writer "Gemini CLI" reverent
```

**Execution Flow:**
1. User types `/poem-writer` command
2. CLI calls `prompts/get` method on MCP server
3. Server substitutes arguments into prompt template
4. CLI sends final prompt to Gemini model
5. Model responds

---

## 8. MCP Resources

### 8.1 Resource Discovery

MCP servers can expose **contextual resources** (files, API payloads, reports):

```typescript
// Discovery
const resources = await client.request(
  { method: 'resources/list' },
  ListResourcesResultSchema
);

// Registration
resourceRegistry.setResourcesForServer(serverName, resources);
```

### 8.2 Referencing Resources in Conversation

Use `@` syntax to reference resources:

```
@server://resource/path
```

**Auto-completion:**
- Resource URIs appear in completion menu alongside filesystem paths
- CLI calls `resources/read` when referenced
- Content is injected into conversation

### 8.3 Dynamic Resource Updates

Similar to tools, resources support dynamic updates:

```typescript
if (capabilities?.resources?.listChanged) {
  this.client.setNotificationHandler(
    ResourceListChangedNotificationSchema,
    async () => {
      await this.refreshResources();
    }
  );
}
```

---

## 9. Example MCP Configurations

### 9.1 Python MCP Server (stdio)

```json
{
  "mcpServers": {
    "pythonTools": {
      "command": "python",
      "args": ["-m", "my_mcp_server", "--port", "8080"],
      "cwd": "./mcp-servers/python",
      "env": {
        "DATABASE_URL": "$DB_CONNECTION_STRING",
        "API_KEY": "${EXTERNAL_API_KEY}"
      },
      "timeout": 15000
    }
  }
}
```

### 9.2 Node.js MCP Server (stdio)

```json
{
  "mcpServers": {
    "nodeServer": {
      "command": "node",
      "args": ["dist/server.js", "--verbose"],
      "cwd": "./mcp-servers/node",
      "trust": true
    }
  }
}
```

### 9.3 Docker-Based MCP Server

```json
{
  "mcpServers": {
    "dockerServer": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "API_KEY",
        "-v", "${PWD}:/workspace",
        "my-mcp-server:latest"
      ],
      "env": {
        "API_KEY": "$EXTERNAL_SERVICE_TOKEN"
      }
    }
  }
}
```

### 9.4 HTTP-Based MCP Server with Auth

```json
{
  "mcpServers": {
    "httpServerWithAuth": {
      "httpUrl": "http://localhost:3000/mcp",
      "headers": {
        "Authorization": "Bearer your-api-token",
        "X-Custom-Header": "custom-value"
      },
      "timeout": 5000
    }
  }
}
```

### 9.5 Tool Filtering Example

```json
{
  "mcpServers": {
    "filteredServer": {
      "command": "python",
      "args": ["-m", "my_mcp_server"],
      "includeTools": ["safe_tool", "file_reader", "data_processor"],
      "excludeTools": ["dangerous_tool", "file_deleter"],
      "timeout": 30000
    }
  }
}
```

---

## 10. Key Differences from Claude CLI

### 10.1 Settings File Format

**Gemini CLI:**
- Hierarchical structure with categories (`general`, `ui`, `model`, `context`, `tools`, `mcp`)
- MCPs under top-level `mcpServers` object
- Global MCP settings in `mcp` object

**Claude CLI (for comparison):**
- Flat structure with `mcpServers` at root level
- Simpler configuration format

### 10.2 MCP Global Controls

**Gemini CLI provides additional global MCP controls:**
- `mcp.serverCommand` - Global command for MCP servers
- `mcp.allowed` - Allowlist of MCP servers
- `mcp.excluded` - Denylist of MCP servers

### 10.3 Tool Confirmation System

**Gemini CLI:**
- Trust-based bypass (`serverConfig.trust`)
- Dynamic allow-listing (server-level and tool-level)
- Message bus integration for policy-based confirmation
- User choices: Proceed once, Always allow tool, Always allow server, Cancel

### 10.4 OAuth Support

**Gemini CLI has sophisticated OAuth integration:**
- Automatic OAuth discovery
- Multiple auth provider types (dynamic_discovery, google_credentials, service_account_impersonation)
- Token storage and automatic refresh
- Browser-based authentication flow

---

## 11. Source Code Key Files

| **File** | **Purpose** |
|---------|-----------|
| `packages/core/src/tools/mcp-client-manager.ts` | Manages lifecycle of all MCP clients |
| `packages/core/src/tools/mcp-client.ts` | Individual MCP server client implementation |
| `packages/core/src/tools/mcp-tool.ts` | Wrapper for discovered MCP tools |
| `packages/cli/src/config/settings.ts` | Settings file loading and management |
| `packages/cli/src/config/settingsSchema.ts` | Settings schema definition (2000+ lines) |
| `packages/cli/src/commands/mcp/add.ts` | `gemini mcp add` command |
| `packages/cli/src/commands/mcp/list.ts` | `gemini mcp list` command |
| `packages/cli/src/commands/mcp/remove.ts` | `gemini mcp remove` command |
| `packages/cli/src/ui/commands/mcpCommand.ts` | `/mcp` in-chat command |
| `packages/core/src/mcp/oauth-provider.ts` | OAuth authentication provider |
| `packages/core/src/mcp/google-auth-provider.ts` | Google Credentials provider |
| `packages/core/src/mcp/sa-impersonation-provider.ts` | Service Account Impersonation provider |
| `packages/core/src/mcp/oauth-token-storage.ts` | OAuth token storage manager |
| `docs/tools/mcp-server.md` | Comprehensive MCP documentation (1045 lines) |

---

## 12. Schema Validation

Gemini CLI uses **JSON Schema validation** for settings:

**Schema File:**
- `schemas/settings.schema.json` (generated from TypeScript types)
- Can be referenced for IDE autocomplete and validation:
  ```
  https://raw.githubusercontent.com/google-gemini/gemini-cli/main/schemas/settings.schema.json
  ```

**MCP Server Config Schema:**
```typescript
// packages/cli/src/config/settingsSchema.ts (lines 1681-1786)
MCPServerConfig: {
  type: 'object',
  properties: {
    command: { type: 'string' },
    args: { type: 'array', items: { type: 'string' } },
    env: { type: 'object', additionalProperties: { type: 'string' } },
    cwd: { type: 'string' },
    url: { type: 'string' },
    httpUrl: { type: 'string' },
    headers: { type: 'object', additionalProperties: { type: 'string' } },
    tcp: { type: 'string' },
    type: { type: 'string', enum: ['stdio', 'sse', 'http'] },
    timeout: { type: 'number' },
    trust: { type: 'boolean' },
    description: { type: 'string' },
    includeTools: { type: 'array', items: { type: 'string' } },
    excludeTools: { type: 'array', items: { type: 'string' } },
    oauth: { type: 'object' },
    authProviderType: { type: 'string', enum: ['dynamic_discovery', 'google_credentials', 'service_account_impersonation'] },
    targetAudience: { type: 'string' },
    targetServiceAccount: { type: 'string' }
  }
}
```

---

## 13. Summary & Recommendations for agent-deck

### 13.1 Key Insights

1. **Hierarchical Settings Structure:**
   - Gemini uses a category-based settings structure
   - MCPs are at top-level but have global controls in `mcp` object
   - Consider if agent-deck should adopt similar hierarchical config

2. **Layered Configuration System:**
   - System defaults → User settings → Project settings → System overrides
   - Allows enterprise deployment with system-wide controls
   - agent-deck could benefit from user vs project MCP configs

3. **Dynamic Tool Updates:**
   - Gemini supports hot-reload via MCP notifications
   - Implements coalescing pattern to handle rapid updates
   - agent-deck could implement similar for updating MCP tools without restart

4. **Sophisticated OAuth Support:**
   - Multiple auth provider types
   - Automatic token refresh
   - Browser-based flow with local callback server
   - agent-deck likely doesn't need this complexity for local MCPs

5. **Tool Confirmation System:**
   - Trust-based bypass
   - Dynamic allow-listing
   - Rich user choices (proceed once, always allow tool/server)
   - agent-deck should implement similar for security

6. **MCP CLI Commands:**
   - Gemini provides `gemini mcp add/list/remove` commands
   - agent-deck already has `mcp list/attached/attach/detach` - consider adding `mcp add/remove`

### 13.2 Recommendations for agent-deck

**1. MCP Configuration Location:**
- Keep `~/.agent-deck/config.toml` for global MCP definitions (similar to Gemini's user settings)
- Add support for **project-level MCPs** in `.mcp.json` (similar to Gemini's project settings)
- This is already implemented in agent-deck! Just document it clearly

**2. Settings Structure:**
- Current TOML structure is clean and appropriate for agent-deck
- Consider adding `[mcp.global]` section for global MCP controls:
  ```toml
  [mcp.global]
  allowed = ["exa", "firecrawl"]  # Allowlist
  excluded = ["experimental"]      # Denylist
  ```

**3. Dynamic MCP Updates:**
- Implement MCP notification handlers (similar to Gemini)
- Use coalescing pattern to handle rapid updates
- Add `session restart <id>` flag `--update-mcps` to refresh MCP list without full restart

**4. MCP Manager UI Enhancements:**
- Add indicators for MCP status (CONNECTING, CONNECTED, DISCONNECTED) - similar to Gemini's status tracking
- Show which MCPs are actively loaded vs just configured
- Add "Refresh" action to reload MCP list without restarting session

**5. CLI Commands:**
- Add `agent-deck mcp add <name> <command>` to add MCPs to config.toml
- Add `agent-deck mcp remove <name>` to remove MCPs from config.toml
- Keep existing `mcp list/attached/attach/detach` commands

**6. Documentation:**
- Clearly document MCP configuration in `CLAUDE.md` and agent-deck docs
- Provide examples for common MCP setups (stdio, Docker, npx packages)
- Document difference between global MCPs (config.toml) and project MCPs (.mcp.json)

**7. OAuth/Auth:**
- **NOT RECOMMENDED** for agent-deck (too complex for local use case)
- Focus on local stdio MCPs and Docker-based MCPs

**8. Tool Confirmation:**
- Gemini's trust-based bypass is good inspiration
- Consider adding `trust = true` flag to MCP definitions in config.toml
- Implement similar allow-listing for trusted tools

---

## Appendix: Quick Reference

### Environment Variables
```bash
GEMINI_CLI_SYSTEM_SETTINGS_PATH  # Override system settings path
GEMINI_CLI_SYSTEM_DEFAULTS_PATH  # Override system defaults path
```

### Key Constants
```typescript
GEMINI_DIR = '.gemini'
MCP_DEFAULT_TIMEOUT_MSEC = 600000  // 10 minutes
```

### Settings File Locations
```
~/.gemini/settings.json              # User settings
.gemini/settings.json                # Project settings (current directory)
/etc/gemini-cli/settings.json        # System settings (Linux)
/etc/gemini-cli/system-defaults.json # System defaults (Linux)
```

### MCP Configuration Template
```json
{
  "mcpServers": {
    "server-name": {
      "command": "executable",
      "args": ["arg1", "arg2"],
      "env": { "KEY": "$VALUE" },
      "cwd": "./path",
      "timeout": 30000,
      "trust": false,
      "includeTools": ["tool1"],
      "excludeTools": ["tool2"]
    }
  },
  "mcp": {
    "allowed": ["server-name"],
    "excluded": ["other-server"]
  }
}
```

---

**End of Research Document**
