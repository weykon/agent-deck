package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGetMCPInfo_ParentDirectoryLookup tests that GetMCPInfo walks up
// parent directories to find .mcp.json files, matching Claude Code's behavior
func TestGetMCPInfo_ParentDirectoryLookup(t *testing.T) {
	// Create temp directory structure:
	// /tmp/test-XXXX/
	//   .mcp.json (airbnb)
	//   project/
	//     subdir/
	tmpRoot, err := os.MkdirTemp("", "mcp-parent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpRoot)

	// Create nested directories
	projectDir := filepath.Join(tmpRoot, "project")
	subdirPath := filepath.Join(projectDir, "subdir")
	if err := os.MkdirAll(subdirPath, 0755); err != nil {
		t.Fatalf("Failed to create subdirs: %v", err)
	}

	// Write .mcp.json in root with "airbnb" MCP
	rootMCPConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"airbnb": map[string]interface{}{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"-y", "@openbnb/mcp-server-airbnb"},
			},
		},
	}
	rootMCPData, _ := json.MarshalIndent(rootMCPConfig, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpRoot, ".mcp.json"), rootMCPData, 0644); err != nil {
		t.Fatalf("Failed to write root .mcp.json: %v", err)
	}

	// Set CLAUDE_CONFIG_DIR to prevent reading real config
	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	tmpClaudeConfig, _ := os.MkdirTemp("", "claude-config-*")
	defer os.RemoveAll(tmpClaudeConfig)
	os.Setenv("CLAUDE_CONFIG_DIR", tmpClaudeConfig)
	defer func() {
		if oldConfigDir != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()

	// Create minimal Claude config
	claudeConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	}
	claudeConfigData, _ := json.MarshalIndent(claudeConfig, "", "  ")
	_ = os.WriteFile(filepath.Join(tmpClaudeConfig, ".claude.json"), claudeConfigData, 0600)

	// TEST 1: Project in subdir should find parent's .mcp.json
	t.Run("finds MCP in parent directory", func(t *testing.T) {
		info := GetMCPInfo(subdirPath)

		if len(info.LocalMCPs) != 1 {
			t.Errorf("Expected 1 local MCP, got %d", len(info.LocalMCPs))
		}

		if len(info.LocalMCPs) > 0 {
			if info.LocalMCPs[0].Name != "airbnb" {
				t.Errorf("Expected MCP name 'airbnb', got %q", info.LocalMCPs[0].Name)
			}
			if info.LocalMCPs[0].SourcePath != tmpRoot {
				t.Errorf("Expected source path %q, got %q", tmpRoot, info.LocalMCPs[0].SourcePath)
			}
		}
	})

	// TEST 2: Backward compatibility - Local() method should return names
	t.Run("Local() method returns names for backward compatibility", func(t *testing.T) {
		info := GetMCPInfo(subdirPath)

		localNames := info.Local()
		if len(localNames) != 1 {
			t.Errorf("Expected 1 local MCP name, got %d", len(localNames))
		}
		if len(localNames) > 0 && localNames[0] != "airbnb" {
			t.Errorf("Expected 'airbnb', got %q", localNames[0])
		}
	})

	// TEST 3: Direct project dir should also find it
	t.Run("finds MCP in same directory", func(t *testing.T) {
		info := GetMCPInfo(tmpRoot)

		if len(info.LocalMCPs) != 1 {
			t.Errorf("Expected 1 local MCP, got %d", len(info.LocalMCPs))
		}
		if len(info.LocalMCPs) > 0 && info.LocalMCPs[0].SourcePath != tmpRoot {
			t.Errorf("Expected source path %q, got %q", tmpRoot, info.LocalMCPs[0].SourcePath)
		}
	})
}

// TestGetMCPInfo_StopsAtFirstMCPJson tests that lookup stops at the first
// .mcp.json found when walking up the directory tree
func TestGetMCPInfo_StopsAtFirstMCPJson(t *testing.T) {
	// Create temp directory structure:
	// /tmp/test-XXXX/
	//   .mcp.json (airbnb)
	//   project/
	//     .mcp.json (exa, firecrawl)
	//     subdir/
	tmpRoot, err := os.MkdirTemp("", "mcp-stop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpRoot)

	projectDir := filepath.Join(tmpRoot, "project")
	subdirPath := filepath.Join(projectDir, "subdir")
	if err := os.MkdirAll(subdirPath, 0755); err != nil {
		t.Fatalf("Failed to create subdirs: %v", err)
	}

	// Root .mcp.json (airbnb)
	rootMCP := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"airbnb": map[string]interface{}{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"-y", "@openbnb/mcp-server-airbnb"},
			},
		},
	}
	rootData, _ := json.MarshalIndent(rootMCP, "", "  ")
	_ = os.WriteFile(filepath.Join(tmpRoot, ".mcp.json"), rootData, 0644)

	// Project .mcp.json (exa, firecrawl)
	projectMCP := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"exa": map[string]interface{}{
				"command": "npx",
				"args":    []string{"-y", "exa-mcp-server"},
			},
			"firecrawl": map[string]interface{}{
				"command": "npx",
				"args":    []string{"-y", "firecrawl-mcp"},
			},
		},
	}
	projectData, _ := json.MarshalIndent(projectMCP, "", "  ")
	_ = os.WriteFile(filepath.Join(projectDir, ".mcp.json"), projectData, 0644)

	// Setup Claude config
	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	tmpClaudeConfig, _ := os.MkdirTemp("", "claude-config-*")
	defer os.RemoveAll(tmpClaudeConfig)
	os.Setenv("CLAUDE_CONFIG_DIR", tmpClaudeConfig)
	defer func() {
		if oldConfigDir != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()
	claudeConfig := map[string]interface{}{"mcpServers": map[string]interface{}{}}
	claudeData, _ := json.MarshalIndent(claudeConfig, "", "  ")
	_ = os.WriteFile(filepath.Join(tmpClaudeConfig, ".claude.json"), claudeData, 0600)

	// TEST: Subdir should find project/.mcp.json (closer), NOT root/.mcp.json
	info := GetMCPInfo(subdirPath)

	if len(info.LocalMCPs) != 2 {
		t.Errorf("Expected 2 local MCPs from project/.mcp.json, got %d", len(info.LocalMCPs))
	}

	// Verify we got exa and firecrawl, NOT airbnb
	names := info.Local()
	hasExa := false
	hasFirecrawl := false
	hasAirbnb := false
	for _, name := range names {
		if name == "exa" {
			hasExa = true
		}
		if name == "firecrawl" {
			hasFirecrawl = true
		}
		if name == "airbnb" {
			hasAirbnb = true
		}
	}

	if !hasExa || !hasFirecrawl {
		t.Errorf("Expected exa and firecrawl from project/.mcp.json, got %v", names)
	}
	if hasAirbnb {
		t.Errorf("Should NOT have airbnb from root (should stop at project/.mcp.json), got %v", names)
	}

	// Verify source path is project dir, not root
	if len(info.LocalMCPs) > 0 && info.LocalMCPs[0].SourcePath != projectDir {
		t.Errorf("Expected source path %q, got %q", projectDir, info.LocalMCPs[0].SourcePath)
	}
}

// TestGetMCPInfo_NoMCPJson tests behavior when no .mcp.json exists anywhere
func TestGetMCPInfo_NoMCPJson(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-none-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup Claude config
	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	tmpClaudeConfig, _ := os.MkdirTemp("", "claude-config-*")
	defer os.RemoveAll(tmpClaudeConfig)
	os.Setenv("CLAUDE_CONFIG_DIR", tmpClaudeConfig)
	defer func() {
		if oldConfigDir != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()
	claudeConfig := map[string]interface{}{"mcpServers": map[string]interface{}{}}
	claudeData, _ := json.MarshalIndent(claudeConfig, "", "  ")
	_ = os.WriteFile(filepath.Join(tmpClaudeConfig, ".claude.json"), claudeData, 0600)

	info := GetMCPInfo(tmpDir)

	if len(info.LocalMCPs) != 0 {
		t.Errorf("Expected 0 local MCPs when no .mcp.json exists, got %d", len(info.LocalMCPs))
	}

	// Backward compat check
	if len(info.Local()) != 0 {
		t.Errorf("Expected Local() to return empty slice, got %d items", len(info.Local()))
	}
}

// TestGetMCPInfo_RootBoundary tests that lookup stops at filesystem root
func TestGetMCPInfo_RootBoundary(t *testing.T) {
	// Use a directory deep in /tmp without any .mcp.json
	tmpDir, err := os.MkdirTemp("", "mcp-root-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	deepDir := filepath.Join(tmpDir, "a", "b", "c", "d")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("Failed to create deep dirs: %v", err)
	}

	// Setup Claude config
	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	tmpClaudeConfig, _ := os.MkdirTemp("", "claude-config-*")
	defer os.RemoveAll(tmpClaudeConfig)
	os.Setenv("CLAUDE_CONFIG_DIR", tmpClaudeConfig)
	defer func() {
		if oldConfigDir != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()
	claudeConfig := map[string]interface{}{"mcpServers": map[string]interface{}{}}
	claudeData, _ := json.MarshalIndent(claudeConfig, "", "  ")
	_ = os.WriteFile(filepath.Join(tmpClaudeConfig, ".claude.json"), claudeData, 0600)

	// Should not panic or infinite loop when reaching root
	info := GetMCPInfo(deepDir)

	if len(info.LocalMCPs) != 0 {
		t.Errorf("Expected 0 local MCPs, got %d", len(info.LocalMCPs))
	}
}
