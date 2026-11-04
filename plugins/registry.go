package plugins

import (
	"encoding/json"
	"os"
	"sort"
	"sync"
)

// Registry holds discovered plugins and which ones are enabled.
// A disabled plugin exposes no tools.
type Registry struct {
	mu      sync.Mutex
	Root    string
	plugins map[string]Plugin
	errors  map[string]string
	enabled map[string]bool
}

// NewRegistry discovers plugins under root. It does not fail on a
// bad plugin; Discover reports those in Errors.
func NewRegistry(root string) *Registry {
	plugins, errs := Discover(root)
	return &Registry{Root: root, plugins: plugins, errors: errs, enabled: map[string]bool{}}
}

// List returns all discovered plugins, with Enabled set from state.
func (r *Registry) List() []Plugin {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Plugin, 0, len(r.plugins))
	for _, name := range SortedNames(r.plugins) {
		p := r.plugins[name]
		p.Enabled = r.enabled[name]
		out = append(out, p)
	}
	return out
}

// Errors returns loader errors, keyed by folder name.
func (r *Registry) Errors() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]string, len(r.errors))
	for k, v := range r.errors {
		out[k] = v
	}
	return out
}

// Enable turns a plugin on. It returns false for an unknown plugin.
func (r *Registry) Enable(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.plugins[name]; !ok {
		return false
	}
	r.enabled[name] = true
	return true
}

// Disable turns a plugin off. It returns false for an unknown plugin.
func (r *Registry) Disable(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.plugins[name]; !ok {
		return false
	}
	delete(r.enabled, name)
	return true
}

// IsEnabled reports whether a plugin is enabled.
func (r *Registry) IsEnabled(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.enabled[name]
}

// ToolDefinition is the Ollama chat tool shape for one plugin tool.
type ToolDefinition struct {
	Plugin   string `json:"plugin"`
	Type     string `json:"type"`
	Function struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Parameters  map[string]interface{} `json:"parameters"`
	} `json:"function"`
}

// EnabledTools returns tool definitions for enabled plugins only,
// sorted by plugin name, then tool name.
func (r *Registry) EnabledTools() []ToolDefinition {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []ToolDefinition
	for _, name := range SortedNames(r.plugins) {
		if !r.enabled[name] {
			continue
		}
		p := r.plugins[name]
		tools := append([]Tool(nil), p.Tools...)
		sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
		for _, t := range tools {
			var d ToolDefinition
			d.Plugin = name
			d.Type = "function"
			d.Function.Name = name + "." + t.Name
			d.Function.Description = t.Description
			d.Function.Parameters = t.Parameters
			out = append(out, d)
		}
	}
	if out == nil {
		out = []ToolDefinition{}
	}
	return out
}

// FindTool returns a tool only when its plugin is enabled and the
// tool is in the manifest. Anything else returns false.
func (r *Registry) FindTool(pluginName, toolName string) (Plugin, Tool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.plugins[pluginName]
	if !ok || !r.enabled[pluginName] {
		return Plugin{}, Tool{}, false
	}
	for _, t := range p.Tools {
		if t.Name == toolName {
			return p, t, true
		}
	}
	return Plugin{}, Tool{}, false
}

// SaveState writes enabled plugin names to a JSON file. It is
// best effort for the proof slice; the server does not depend on it.
func (r *Registry) SaveState(path string) error {
	r.mu.Lock()
	names := make([]string, 0, len(r.enabled))
	for name := range r.enabled {
		names = append(names, name)
	}
	r.mu.Unlock()
	sort.Strings(names)
	data, err := json.Marshal(map[string]interface{}{"enabled": names})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
