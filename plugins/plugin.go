// Package plugins loads Codex-style plugins for Ollama.
//
// A plugin is a folder with a plugin.json manifest and a SKILL.md skill.
// This package is standard library only, so it can be tested without
// building the full Ollama server. It is a port of the Python seed
// loader, not a copy of any OpenAI code.
package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Tool is one tool declared in a plugin manifest.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Plugin is a validated plugin folder.
type Plugin struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Tools       []Tool   `json:"tools"`
	Skills      []string `json:"skills"`
	Enabled     bool     `json:"enabled"`
}

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// parseFrontmatter reads the simple key: value frontmatter at the
// top of a SKILL.md file. It is not a full YAML parser, by design:
// skill frontmatter in this seed uses only flat keys.
func parseFrontmatter(text string) (map[string]string, string, error) {
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return nil, "", fmt.Errorf("missing frontmatter")
	}
	rest := text[4:]
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, "", fmt.Errorf("unterminated frontmatter")
	}
	out := map[string]string{}
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	body := rest[end+4:]
	return out, body, nil
}

func loadPlugin(dir string) (Plugin, error) {
	var p Plugin
	manifestPath := filepath.Join(dir, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return p, fmt.Errorf("missing plugin.json in %s", dir)
	}
	var raw struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     string `json:"version"`
		Tools       []Tool `json:"tools"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return p, fmt.Errorf("invalid plugin.json in %s: %w", dir, err)
	}
	if !namePattern.MatchString(raw.Name) {
		return p, fmt.Errorf("invalid plugin name %q", raw.Name)
	}
	if filepath.Base(dir) != raw.Name {
		return p, fmt.Errorf("folder name %q does not match plugin name %q", filepath.Base(dir), raw.Name)
	}
	if raw.Description == "" {
		return p, fmt.Errorf("missing description for plugin %q", raw.Name)
	}
	if raw.Tools == nil {
		return p, fmt.Errorf("missing tools for plugin %q", raw.Name)
	}
	for _, t := range raw.Tools {
		if t.Name == "" || t.Description == "" {
			return p, fmt.Errorf("tool in plugin %q is missing a name or description", raw.Name)
		}
		if t.Parameters == nil {
			return p, fmt.Errorf("tool %q in plugin %q is missing parameters", t.Name, raw.Name)
		}
		if typ, _ := t.Parameters["type"].(string); typ != "object" {
			return p, fmt.Errorf("tool %q in plugin %q must have object parameters", t.Name, raw.Name)
		}
	}
	skillData, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return p, fmt.Errorf("missing SKILL.md in %s", dir)
	}
	front, _, err := parseFrontmatter(string(skillData))
	if err != nil {
		return p, fmt.Errorf("invalid SKILL.md in %s: %w", dir, err)
	}
	if front["name"] != raw.Name {
		return p, fmt.Errorf("SKILL.md name %q does not match plugin name %q", front["name"], raw.Name)
	}
	if front["description"] == "" {
		return p, fmt.Errorf("SKILL.md for plugin %q is missing a description", raw.Name)
	}
	p = Plugin{Name: raw.Name, Description: raw.Description, Version: raw.Version, Tools: raw.Tools, Skills: []string{raw.Name}}
	return p, nil
}

// Discover loads every plugin folder under root. A bad plugin is
// reported in errors and does not hide the good plugins.
func Discover(root string) (map[string]Plugin, map[string]string) {
	plugins := map[string]Plugin{}
	errors := map[string]string{}
	entries, err := os.ReadDir(root)
	if err != nil {
		return plugins, errors
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || e.Name() == "__pycache__" {
			continue
		}
		p, err := loadPlugin(filepath.Join(root, e.Name()))
		if err != nil {
			errors[e.Name()] = err.Error()
			continue
		}
		plugins[p.Name] = p
	}
	return plugins, errors
}

// SortedNames returns plugin names in alphabetical order.
func SortedNames(plugins map[string]Plugin) []string {
	names := make([]string, 0, len(plugins))
	for name := range plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
