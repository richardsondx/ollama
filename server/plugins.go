package server

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/ollama/ollama/plugins"
)

// pluginDir returns the plugin folder to load. It is configurable
// with OLLAMA_PLUGINS_DIR so the proof slice can point at a test
// folder without changing server code. The default, plugins, is a
// folder next to where the server runs, and an empty or missing
// folder simply lists no plugins.
func pluginDir() string {
	if v := os.Getenv("OLLAMA_PLUGINS_DIR"); v != "" {
		return v
	}
	return "plugins"
}

// pluginHandler builds the plugin API handler once per server.
// One registry is shared across all plugin routes, so an enable
// on one route is visible on the tools route. Enable state is
// in memory only in this slice, and is lost on restart.
func pluginHandler() http.Handler {
	dir := pluginDir()
	if _, err := os.Stat(dir); err != nil {
		slog.Warn("plugin folder not found, listing no plugins", "dir", dir)
	}
	return plugins.NewHandler(plugins.NewRegistry(dir))
}
