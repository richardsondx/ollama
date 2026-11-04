//go:build windows || darwin

package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testExecutor is a fake seed server. It answers by tool name so each
// case below can prove one rule without a live seed server.
func testExecutor(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/tools/skill-creator/validate_skill/execute":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "dry_run": true, "plugin": "skill-creator", "tool": "validate_skill"})
		case "/api/tools/computer/click/execute":
			_ = json.NewEncoder(w).Encode(map[string]any{"confirmation_required": true, "policy": "always_confirm", "action": "click", "message": "This computer action needs confirmation before it runs."})
		default:
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "tool not enabled or not in manifest"})
		}
	}))
}

func TestPluginToolNameIsDottedPair(t *testing.T) {
	tool, err := NewPluginTool("browser", "webmcp.list_tools", "List WebMCP tools.", map[string]any{"type": "object", "properties": map[string]any{}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := tool.Name(); got != "browser.webmcp.list_tools" {
		t.Fatalf("Name() = %q, want %q", got, "browser.webmcp.list_tools")
	}
}

func TestPluginToolBrowserTimeoutIsLonger(t *testing.T) {
	browser, err := NewPluginTool("browser", "webmcp.execute_tool", "Execute a WebMCP tool.", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if browser.client.Timeout != 30*time.Second {
		t.Fatalf("browser timeout = %v, want 30s", browser.client.Timeout)
	}
	other, err := NewPluginTool("computer", "click", "Click.", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if other.client.Timeout != 10*time.Second {
		t.Fatalf("computer timeout = %v, want 10s", other.client.Timeout)
	}
}

func TestPluginToolUnknownToolIsRejected(t *testing.T) {
	srv := testExecutor(t)
	defer srv.Close()
	tool, err := NewPluginTool("testecho", "do_thing", "A test tool.", nil, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := tool.Execute(t.Context(), map[string]any{}); err == nil {
		t.Fatal("expected an error for an unknown or not-enabled tool, got success")
	} else if !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected the 403 to be reported, got error: %v", err)
	}
}

func TestPluginToolConfirmationRequiredIsErrorNotSuccess(t *testing.T) {
	srv := testExecutor(t)
	defer srv.Close()
	tool, err := NewPluginTool("computer", "click", "Click something.", nil, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	result, content, err := tool.Execute(t.Context(), map[string]any{"action": "click"})
	if err == nil {
		t.Fatalf("expected confirmation_required to be an error, got success result=%v content=%q", result, content)
	}
	if result != nil || content != "" {
		t.Fatalf("confirmation case must return no result and no content, got result=%v content=%q", result, content)
	}
	if !strings.Contains(err.Error(), "confirmation required") {
		t.Fatalf("expected a confirmation error, got: %v", err)
	}
}

func TestPluginToolNonLocalhostExecutorIsRejected(t *testing.T) {
	if _, err := NewPluginTool("browser", "webmcp.list_tools", "List.", nil, "https://example.com"); err == nil {
		t.Fatal("expected a non-localhost executor URL to be rejected")
	}
	if _, err := NewPluginTool("browser", "webmcp.list_tools", "List.", nil, "http://ollama.com"); err == nil {
		t.Fatal("expected a non-localhost executor URL to be rejected")
	}
	if err := validateLocalURL("http://127.0.0.1.evil.example.com"); err == nil {
		t.Fatal("expected a hostname that merely ends in a local-looking suffix to be rejected")
	}
	ctx := context.Background()
	if _, err := LoadPluginTools(ctx, "https://example.com/api/tools", ""); err == nil {
		t.Fatal("expected a non-localhost tools URL to be rejected by the loader")
	}
}

func TestPluginToolSuccessfulDryRunReturnsOkContent(t *testing.T) {
	srv := testExecutor(t)
	defer srv.Close()
	tool, err := NewPluginTool("skill-creator", "validate_skill", "Validate a skill folder.", nil, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	result, content, err := tool.Execute(t.Context(), map[string]any{"path": "/tmp/none"})
	if err != nil {
		t.Fatalf("expected dry-run success, got error: %v", err)
	}
	reply, ok := result.(map[string]any)
	if !ok || reply["ok"] != true || reply["dry_run"] != true {
		t.Fatalf("expected ok dry-run result, got: %#v", result)
	}
	if !strings.Contains(content, "validate_skill") {
		t.Fatalf("expected readable content naming the tool, got: %q", content)
	}
}

func TestLoadPluginToolsSkipsMalformedEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tools": []any{
			map[string]any{"plugin": "browser", "type": "function", "function": map[string]any{"name": "browser.webmcp.list_tools", "description": "List.", "parameters": map[string]any{"type": "object", "properties": map[string]any{}}}},
			map[string]any{"plugin": "broken", "type": "function", "function": map[string]any{"name": "", "description": "No name.", "parameters": map[string]any{"type": "object"}}},
			map[string]any{"plugin": "broken", "type": "function", "function": map[string]any{"name": "nodot", "description": "No dot.", "parameters": map[string]any{"type": "object"}}},
			map[string]any{"plugin": "other", "type": "function", "function": map[string]any{"name": "browser.webmcp.list_tools", "description": "Plugin field disagrees.", "parameters": map[string]any{"type": "object"}}},
		}})
	}))
	defer srv.Close()
	loaded, err := LoadPluginTools(t.Context(), srv.URL+"/api/tools", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Name() != "browser.webmcp.list_tools" {
		t.Fatalf("expected only the one well-formed tool, got %d tools", len(loaded))
	}
}

func TestMentionedPlugins(t *testing.T) {
	enabled := []string{"browser", "computer"}
	cases := []struct {
		prompt string
		want   []string
	}{
		{"@browser list the tools", []string{"browser"}},
		{"@Browser list the tools", []string{"browser"}},
		{"use @computer and @browser", []string{"computer", "browser"}},
		{"@browser, then @browser again", []string{"browser"}},
		{"no mention here", nil},
		{"email user@browser is not a mention", nil},
		{"@skill-creator is not enabled here", nil},
	}
	for _, c := range cases {
		got := MentionedPlugins(c.prompt, enabled)
		if len(got) != len(c.want) {
			t.Fatalf("MentionedPlugins(%q) = %v, want %v", c.prompt, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("MentionedPlugins(%q) = %v, want %v", c.prompt, got, c.want)
			}
		}
	}
	if got := MentionedPlugins("@computer use it", []string{"browser"}); len(got) != 0 {
		t.Fatalf("non-enabled mention must return no plugins, got %v", got)
	}
	if !HasMention("@browser hello") {
		t.Fatal("HasMention should be true for @browser")
	}
	if HasMention("no mention") {
		t.Fatal("HasMention should be false with no @ token")
	}
	if HasMention("email user@browser") {
		t.Fatal("HasMention should be false for an email address")
	}
}
