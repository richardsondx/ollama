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

func TestRegistryBareNameResolution(t *testing.T) {
	mustTool := func(plugin, tool string) *PluginTool {
		t.Helper()
		built, err := NewPluginTool(plugin, tool, "Test.", nil, "")
		if err != nil {
			t.Fatal(err)
		}
		return built
	}

	// Only cua tools registered, as in an @cua turn.
	cuaOnly := NewRegistry()
	cuaOnly.Register(mustTool("cua", "create_tab"))
	cuaOnly.Register(mustTool("cua", "get_state"))
	if tool, ok := cuaOnly.Get("create_tab"); !ok || tool.Name() != "cua.create_tab" {
		t.Fatalf("bare create_tab should resolve to cua.create_tab, got ok=%v tool=%v", ok, tool)
	}
	if tool, ok := cuaOnly.Get("cua.create_tab"); !ok || tool.Name() != "cua.create_tab" {
		t.Fatalf("exact cua.create_tab must still work, got ok=%v tool=%v", ok, tool)
	}

	// Only computer tools registered, as in an @computer turn.
	computerOnly := NewRegistry()
	computerOnly.Register(mustTool("computer", "list_apps"))
	if tool, ok := computerOnly.Get("list_apps"); !ok || tool.Name() != "computer.list_apps" {
		t.Fatalf("bare list_apps should resolve to computer.list_apps, got ok=%v tool=%v", ok, tool)
	}

	// Browser tools keep their dotted middle part.
	browserOnly := NewRegistry()
	browserOnly.Register(mustTool("browser", "webmcp.list_tools"))
	if tool, ok := browserOnly.Get("webmcp.list_tools"); !ok || tool.Name() != "browser.webmcp.list_tools" {
		t.Fatalf("webmcp.list_tools should resolve to browser.webmcp.list_tools, got ok=%v tool=%v", ok, tool)
	}
	if tool, ok := browserOnly.Get("list_tools"); !ok || tool.Name() != "browser.webmcp.list_tools" {
		t.Fatalf("bare list_tools should resolve to browser.webmcp.list_tools, got ok=%v tool=%v", ok, tool)
	}

	// Both click tools registered, as when no @ mention is used:
	// bare click must error naming both and execute neither.
	both := NewRegistry()
	both.Register(mustTool("cua", "click"))
	both.Register(mustTool("computer", "click"))
	if tool, ok := both.Get("click"); ok || tool != nil {
		t.Fatalf("ambiguous bare click must not resolve, got ok=%v tool=%v", ok, tool)
	}
	if _, _, err := both.Execute(t.Context(), "click", map[string]any{}); err == nil {
		t.Fatal("ambiguous bare click must error, got success")
	} else if !strings.Contains(err.Error(), "cua.click") || !strings.Contains(err.Error(), "computer.click") {
		t.Fatalf("ambiguous error must name both candidates, got: %v", err)
	}

	// Unknown names keep the old error.
	if _, _, err := cuaOnly.Execute(t.Context(), "no_such_tool", map[string]any{}); err == nil || !strings.Contains(err.Error(), "unknown tool: no_such_tool") {
		t.Fatalf("unknown tool must keep its error, got: %v", err)
	}
}

func TestParseTextJSONToolCall(t *testing.T) {
	call, ok := ParseTextJSONToolCall(`{"name": "cua.create_tab", "arguments": {"url": "google.com", "browser": "chrome"}}`)
	if !ok || call.Name != "cua.create_tab" || call.Args["url"] != "google.com" {
		t.Fatalf("exact JSON object must parse, got ok=%v call=%v", ok, call)
	}
	call, ok = ParseTextJSONToolCall("  {\"name\": \"browser.webmcp.list_tools\"}  ")
	if !ok || call.Name != "browser.webmcp.list_tools" || len(call.Args) != 0 {
		t.Fatalf("missing arguments must parse as empty, got ok=%v call=%v", ok, call)
	}
	rejected := []string{
		`Here is the call: {"name": "cua.create_tab", "arguments": {}}`,
		`{"name": "cua.create_tab", "arguments": {}} and more`,
		"```json\n{\"name\": \"cua.create_tab\", \"arguments\": {}}\n```",
		`[{"name": "cua.create_tab", "arguments": {}}]`,
		`{"name": 42, "arguments": {}}`,
		`{"arguments": {}}`,
		`{"name": "cua.create_tab", "arguments": "google.com"}`,
		`{"name": "cua.create_tab", "arguments": null}`,
		`not json at all`,
		``,
	}
	for _, content := range rejected {
		if _, ok := ParseTextJSONToolCall(content); ok {
			t.Fatalf("ParseTextJSONToolCall(%q) must return false", content)
		}
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
