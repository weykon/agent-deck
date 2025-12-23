<div align="center">

<!-- Status Grid Logo -->
<img src="site/logo.svg" alt="Agent Deck Logo" width="120">

# Agent Deck

**Terminal session manager for AI agents**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey)](https://github.com/asheshgoplani/agent-deck)

[Features](#features) • [Installation](#installation) • [Usage](#usage) • [Documentation](#documentation)

</div>

---

![Agent Deck Demo](demos/agent-deck-overview.gif)

## Why Agent Deck?

Managing multiple AI coding sessions across projects can get overwhelming. Agent Deck provides a single dashboard to monitor and switch between all your sessions—Claude Code, Gemini CLI, Aider, Codex, or any terminal tool.

**What it does:**
- Organize sessions by project with collapsible groups
- See at a glance which agents are running, waiting, or idle
- Switch between sessions instantly with keyboard shortcuts
- Search and filter to find what you need
- Built on tmux for reliability

## Features

### 🚀 Session Forking (Claude Code)

Fork Claude conversations to explore multiple approaches in parallel. Each fork inherits full conversation context.

![Fork Session Demo](demos/fork-session.gif)

- Press `f` to quick-fork, `F` for custom name/group
- Forks inherit context and can be forked again
- Auto-detects Claude session ID across restarts

### 🔌 MCP Manager

Attach and detach MCP servers on the fly—no config editing required.

![MCP Manager Demo](demos/mcp-manager.gif)

- Press `M` to open, `Space` to toggle MCPs
- **LOCAL** scope (project) or **GLOBAL** (all projects)
- Session auto-restarts with new MCPs loaded

### 🔍 Search

Press `/` to search across sessions with fuzzy matching. Filter by status with `!` (running), `@` (waiting), `#` (idle), `$` (error).

### 🎯 Smart Status Detection

Automatically detects what your AI agent is doing:

| Status | Symbol | Meaning |
|--------|--------|---------|
| **Running** | `●` green | Agent is working |
| **Waiting** | `◐` yellow | Needs input |
| **Idle** | `○` gray | Ready |
| **Error** | `✕` red | Error |

Works with Claude Code, Gemini CLI, Aider, Codex, and any shell.

## Installation

**Works on:** macOS • Linux • Windows (WSL)

```bash
curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash
```

The installer downloads the binary, installs tmux if needed, and configures tmux for mouse/clipboard support.

Then run: `agent-deck`

> **Windows:** [Install WSL](https://learn.microsoft.com/en-us/windows/wsl/install) first.

<details>
<summary>Other install methods</summary>

**Homebrew**
```bash
brew install asheshgoplani/tap/agent-deck
```

**Go**
```bash
go install github.com/asheshgoplani/agent-deck/cmd/agent-deck@latest
```

**From Source**
```bash
git clone https://github.com/asheshgoplani/agent-deck.git && cd agent-deck && make install
```

</details>

## Usage

```bash
agent-deck                    # Launch TUI
agent-deck add .              # Add current directory as session
agent-deck add . -c claude    # Add with Claude Code
agent-deck list               # List all sessions
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate |
| `Enter` | Attach to session |
| `n` | New session |
| `g` | New group |
| `r` | Rename |
| `d` | Delete |
| `f` | Fork Claude session |
| `M` | MCP Manager |
| `/` | Search |
| `Ctrl+Q` | Detach from session |
| `?` | Help |

## Documentation

### Project Organization

```
▼ Projects (3)
  ├─ frontend     ●
  ├─ backend      ◐
  └─ api          ○
▼ Personal
  └─ blog         ○
```

Sessions are organized in collapsible groups. Create nested groups, reorder items, and import existing tmux sessions with `i`.

### Configuration

Data stored in `~/.agent-deck/`:

```
~/.agent-deck/
├── sessions.json     # Sessions and groups
└── config.toml       # User config (optional)
```

For custom Claude profile directory:

```toml
[claude]
config_dir = "~/.claude-work"
```

### tmux Configuration

The installer configures tmux automatically. For manual setup, see the [tmux configuration guide](https://github.com/asheshgoplani/agent-deck/wiki/tmux-Configuration).

## Development

```bash
make build    # Build
make test     # Test
make lint     # Lint
```

## Contributing

Contributions welcome! Fork, create a branch, and open a PR.

## License

MIT License - see [LICENSE](LICENSE)

---

<div align="center">

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [tmux](https://github.com/tmux/tmux)

</div>
