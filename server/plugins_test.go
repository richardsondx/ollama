package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPluginRoutesViaGenerateRoutes proves the plugin routes
// are actually registered by GenerateRoutes, through the gin
// router, not only by the handler function itself. It lists a
// plugin, enables it through the mounted route, and then sees
// it on the mounted tools route. It does not boot the full
// Ollama server, and it does not wire plugins into chat.
func TestPluginRoutesViaGenerateRoutes(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "browser")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name": "browser", "description": "Test browser plugin.", "version": "0.1.0", "tools": [{"name": "do_thing", "description": "Do a thing.", "parameters": {"type": "object", "properties": {}}}]}`
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: browser\ndescription: Test browser plugin.\n---\n\nBody.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLLAMA_PLUGINS_DIR", root)
	t.Setenv("OLLAMA_PLUGIN_STATE", filepath.Join(t.TempDir(), "state.json"))
	t.Setenv("OLLAMA_MODELS", t.TempDir())

	s := &Server{}
	h, err := s.GenerateRoutes()
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	listed := call(http.MethodGet, "/api/plugins")
	if listed.Code != http.StatusOK {
		t.Fatalf("GET /api/plugins expected 200, got %d: %s", listed.Code, listed.Body.String())
	}
	var pluginsResp struct {
		Plugins []struct {
			Name string `json:"name"`
		} `json:"plugins"`
	}
	if err := json.NewDecoder(listed.Body).Decode(&pluginsResp); err != nil {
		t.Fatal(err)
	}
	if len(pluginsResp.Plugins) != 1 || pluginsResp.Plugins[0].Name != "browser" {
		t.Fatalf("unexpected plugin list via routes: %+v", pluginsResp.Plugins)
	}
	enabled := call(http.MethodPost, "/api/plugins/browser/enable")
	if enabled.Code != http.StatusOK {
		t.Fatalf("POST enable expected 200, got %d: %s", enabled.Code, enabled.Body.String())
	}
	tools := call(http.MethodGet, "/api/tools")
	if tools.Code != http.StatusOK {
		t.Fatalf("GET /api/tools expected 200, got %d: %s", tools.Code, tools.Body.String())
	}
	if !strings.Contains(tools.Body.String(), "browser.do_thing") {
		t.Fatalf("expected browser.do_thing on /api/tools, got %s", tools.Body.String())
	}
}

// TestPluginHandlerMount proves the server-side plugin handler,
// the same handler GenerateRoutes mounts at /api/plugins, lists
// a plugin folder and returns it as JSON. It does not boot the
// full Ollama server, and it does not wire plugins into chat.
func TestPluginHandlerMount(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "browser")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name": "browser", "description": "Test browser plugin.", "version": "0.1.0", "tools": [{"name": "do_thing", "description": "Do a thing.", "parameters": {"type": "object", "properties": {}}}]}`
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: browser\ndescription: Test browser plugin.\n---\n\nBody.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLLAMA_PLUGINS_DIR", root)
	t.Setenv("OLLAMA_PLUGIN_STATE", filepath.Join(t.TempDir(), "state.json"))

	req := httptest.NewRequest(http.MethodGet, "/api/plugins", nil)
	rec := httptest.NewRecorder()
	pluginHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var listed struct {
		Plugins []struct {
			Name string `json:"name"`
		} `json:"plugins"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Plugins) != 1 || listed.Plugins[0].Name != "browser" {
		t.Fatalf("unexpected plugin list: %+v", listed.Plugins)
	}
}
