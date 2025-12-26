# MCP Socket Pool - Quick Guide

## What It Does

**Shares MCP processes across Claude sessions via Unix sockets.**

Instead of each session spawning separate MCP processes, all sessions connect to shared processes.

```
Without pool:              With pool:
Session 1 → memory         Session 1 ─┐
Session 2 → memory         Session 2 ─┼─→ memory (shared)
Session 3 → memory         Session 3 ─┘
= 3 processes              = 1 process
```

**Memory savings: 85-90%**

---

## Enable Pool

Add to `~/.agent-deck/config.toml`:

```toml
[mcp_pool]
enabled = true     # Enable socket pooling
pool_all = true    # Pool ALL available MCPs

# Optional: exclude specific MCPs
exclude_mcps = ["chrome-devtools"]
```

---

## How It Works

### On Startup
- Pool starts socket proxies for ALL MCPs in `[mcps.*]`
- Sockets created at `/tmp/agentdeck-mcp-{name}.sock`

### When You Attach an MCP
- MCP Manager shows 🔌 for pooled MCPs
- Attachment writes socket config (not stdio)

### When You Restart a Session
- `.mcp.json` auto-regenerates with socket configs
- Claude connects via socket to shared MCP

---

## Verification Commands

```bash
# Check sockets exist
ls /tmp/agentdeck-mcp-*.sock

# Check MCP processes (should be one per MCP type)
ps aux | grep "node.*mcp" | grep -v grep

# Check .mcp.json uses socket
cat <project>/.mcp.json | jq '.mcpServers'
# Should show: {"command": "nc", "args": ["-U", "/tmp/..."]}

# Check active connections
lsof /tmp/agentdeck-mcp-*.sock
```

---

## Visual Indicators

In MCP Manager (`M` key):
- 🔌 = MCP is pooled and running via socket
- No icon = MCP uses stdio (per-session)

---

## Rollback

Disable pool:
```toml
[mcp_pool]
enabled = false
```

Restart agent-deck. Sessions revert to stdio on next restart.

---

## Troubleshooting

| Problem | Solution |
|---------|----------|
| No sockets created | Check `[mcp_pool] enabled = true` in config |
| No 🔌 in MCP Manager | Pool may have failed - check logs |
| .mcp.json shows npx | Restart session (`R` key) to regenerate |
| MCP tools not working | Check `lsof /tmp/agentdeck-mcp-*.sock` |

**Pool logs:** `~/.agent-deck/logs/mcppool/*.log`
