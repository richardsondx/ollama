package plugins

import (
	"encoding/json"
	"net/http"
	"strings"
)

// NewHandler returns an HTTP handler for the plugin API, using
// only the standard library. The Ollama server mounts it at
// /api/plugins. Page content and plugin manifests are data, never
// instructions: a manifest cannot add a destination or permission.
func NewHandler(reg *Registry) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/plugins", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]interface{}{"plugins": reg.List(), "errors": reg.Errors()})
	})
	mux.HandleFunc("/api/plugins/skills", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		skills := []string{}
		for _, p := range reg.List() {
			skills = append(skills, p.Skills...)
		}
		writeJSON(w, map[string]interface{}{"skills": skills})
	})
	mux.HandleFunc("/api/tools", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]interface{}{"tools": reg.EnabledTools()})
	})
	mux.HandleFunc("/api/plugins/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rest := strings.TrimPrefix(req.URL.Path, "/api/plugins/")
		parts := strings.Split(rest, "/")
		if len(parts) != 2 || (parts[1] != "enable" && parts[1] != "disable") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var ok bool
		if parts[1] == "enable" {
			ok = reg.Enable(parts[0])
		} else {
			ok = reg.Disable(parts[0])
		}
		if !ok {
			http.Error(w, "unknown plugin", http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]interface{}{"name": parts[0], "enabled": parts[1] == "enable"})
	})
	return mux
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
