---
name: agent-deck
description: Terminal session manager for AI coding agents. Use when user says "launch sub-agent", "create sub-agent", "start session", "check session", or needs to manage Claude/AI sessions via CLI. Handles session lifecycle (create/start/stop/restart/fork), MCP attachment, session output retrieval, and parent-child session hierarchies.
---

# Agent Deck CLI

## Sub-Agent Launch

**Trigger:** User says "launch sub-agent", "create sub-agent", or similar.

```bash
scripts/launch-subagent.sh "Title" "Prompt" [--mcp name] [--wait]
```

The script auto-detects current session/profile and creates a child session.

### Retrieval Modes

| Mode | Command | Use When |
|------|---------|----------|
| **Fire & forget** | (no --wait) | Default. Tell user: "Ask me to check when ready" |
| **On-demand** | `agent-deck session output "Title"` | User asks to check |
| **Blocking** | `--wait` flag | Need immediate result |

### Recommended MCPs

| Task Type | MCPs |
|-----------|------|
| Web research | `exa`, `firecrawl` |
| Code docs | `context7` |
| Complex reasoning | `sequential-thinking` |

---

## Quick Reference

```bash
# Session lifecycle
agent-deck add -t "Name" -c claude /path          # Create
agent-deck add -t "Name" --parent "Parent" /path  # Create as child
agent-deck session start|stop|restart "Name"      # Control
agent-deck session send "Name" "message"          # Send
agent-deck session output "Name"                  # Get response
agent-deck session current [-q|--json]            # Detect current

# MCPs
agent-deck mcp list                               # Available
agent-deck mcp attach "Name" <mcp>                # Attach (restart after)

# Status
agent-deck status                                 # Summary
agent-deck session show "Name"                    # Details
```

**Status:** `●` running | `◐` waiting | `○` idle | `✕` error

---

## Critical Rules

1. **Flags before arguments:** `session show -json name` ✓
2. **Restart after MCP attach:** `mcp attach` then `session restart`
3. **Avoid polling from other agents** - can interfere with target session

---

## References

- [cli-reference.md](references/cli-reference.md) - Full command reference
- [mcp-management.md](references/mcp-management.md) - MCP guide
- [automation-patterns.md](references/automation-patterns.md) - Scripting
