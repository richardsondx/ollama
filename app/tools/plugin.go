//go:build windows || darwin

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Plugin tool wiring for the desktop chat loop.
//
// A PluginTool is one tool declared by an enabled plugin. Its name is
// the dotted pair "plugin.tool", for example "browser.webmcp.list_tools".
// That dotted name is what the model calls and what Registry.Execute
// looks up, so each tool needs its own registry entry. The task text
// phrase "plugin.tool" is that dotted form, not one literal name shared
// by every tool: a single shared name would let only one plugin tool
// exist in the registry at a time.
//
// Execution goes to the local seed server only. The seed server is the
// only executor in this wiring, because the Go fork has no execution
// engine of its own. Computer actions stay dry-run in the seed, and a
// reply that says confirmation is required is returned here as an
// error, so the existing chat loop records a tool error and the action
// does not run and is never counted as success.

const (
	// DefaultPluginToolsURL is where the fork server lists enabled
	// plugin tools. The desktop app and the fork server normally share
	// this port.
	DefaultPluginToolsURL = "http://127.0.0.1:11434/api/tools"
	// DefaultPluginExecutorURL is the local seed server.
	DefaultPluginExecutorURL = "http://127.0.0.1:11435"
	// pluginExecuteTimeout caps one tool execution. The seed server is
	// local, so a reply that takes longer than this is an error, not a
	// reason to keep chat waiting.
	pluginExecuteTimeout = 10 * time.Second
	// pluginBrowserExecuteTimeout caps one browser tool execution.
	// The seed server waits up to 25 seconds for a Chrome extension
	// result, so a 10 second cap in chat would cut a real Chrome
	// execution short. Browser tools get 30 seconds, other plugin
	// tools keep the 10 second cap.
	pluginBrowserExecuteTimeout = 30 * time.Second
)

// pluginTimeoutFor returns the execution timeout for one plugin.
func pluginTimeoutFor(plugin string) time.Duration {
	if plugin == "browser" {
		return pluginBrowserExecuteTimeout
	}
	return pluginExecuteTimeout
}

// PluginToolsURLFromEnv returns the tools list URL, overridable only
// to a local URL through OLLAMA_PLUGIN_TOOLS_URL. An invalid or
// non-local override falls back to the default, so a bad environment
// value cannot redirect tool definitions to a remote host.
func PluginToolsURLFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("OLLAMA_PLUGIN_TOOLS_URL")); v != "" {
		if err := validateLocalURL(v); err == nil {
			return v
		}
	}
	return DefaultPluginToolsURL
}

// PluginExecutorURLFromEnv returns the executor base URL, overridable
// only to a local URL through OLLAMA_PLUGIN_EXECUTOR_URL, with the
// same fallback rule as PluginToolsURLFromEnv.
func PluginExecutorURLFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("OLLAMA_PLUGIN_EXECUTOR_URL")); v != "" {
		if err := validateLocalURL(v); err == nil {
			return v
		}
	}
	return DefaultPluginExecutorURL
}

// validateLocalURL accepts only http or https URLs whose host is
// 127.0.0.1 or localhost. Anything else is rejected, including
// hostnames that merely contain those strings.
func validateLocalURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL %q must use http or https", raw)
	}
	host := u.Hostname()
	if host != "127.0.0.1" && host != "localhost" {
		return fmt.Errorf("URL %q is not a local host (only 127.0.0.1 or localhost are allowed)", raw)
	}
	return nil
}

func validatePluginToolName(plugin, tool string) error {
	if plugin == "" || tool == "" {
		return fmt.Errorf("plugin and tool names must both be non-empty")
	}
	if strings.ContainsAny(plugin, "/ \t\n") || strings.ContainsAny(tool, "/ \t\n") {
		return fmt.Errorf("plugin %q and tool %q must not contain slashes or whitespace", plugin, tool)
	}
	return nil
}

// PluginTool implements the Tool interface for one plugin tool.
type PluginTool struct {
	Plugin      string
	Tool        string
	description string
	schema      map[string]any
	executorURL string
	client      *http.Client
}

// NewPluginTool builds one PluginTool. executorURL may be empty, in
// which case the default seed server URL is used. A non-local
// executor URL is rejected here, before any request is made.
func NewPluginTool(plugin, tool, description string, schema map[string]any, executorURL string) (*PluginTool, error) {
	if err := validatePluginToolName(plugin, tool); err != nil {
		return nil, err
	}
	if executorURL == "" {
		executorURL = DefaultPluginExecutorURL
	}
	if err := validateLocalURL(executorURL); err != nil {
		return nil, fmt.Errorf("plugin tool %s.%s: %w", plugin, tool, err)
	}
	if schema == nil {
		schema = map[string]any{"type": "object", "properties": map[string]any{}}
	}
	return &PluginTool{
		Plugin:      plugin,
		Tool:        tool,
		description: description,
		schema:      schema,
		executorURL: strings.TrimRight(executorURL, "/"),
		client:      &http.Client{Timeout: pluginTimeoutFor(plugin)},
	}, nil
}

// Name returns the dotted pair the model calls, for example
// "browser.webmcp.list_tools".
func (p *PluginTool) Name() string {
	return p.Plugin + "." + p.Tool
}

// Description returns the description from the plugin manifest.
func (p *PluginTool) Description() string {
	return p.description
}

// Schema returns the parameter schema from the plugin manifest.
func (p *PluginTool) Schema() map[string]any {
	return p.schema
}

// Prompt returns no extra prompt. The tool description and schema are
// what the model sees.
func (p *PluginTool) Prompt() string {
	return ""
}

// Execute posts the arguments to the seed server and returns the seed
// reply as the stored result and a readable JSON form as the model
// content. A confirmation_required reply, a 403, and any other error
// status are all returned as errors, never as success.
func (p *PluginTool) Execute(ctx context.Context, args map[string]any) (any, string, error) {
	if err := validateLocalURL(p.executorURL); err != nil {
		return nil, "", fmt.Errorf("plugin tool %s: %w", p.Name(), err)
	}
	if args == nil {
		args = map[string]any{}
	}
	body, err := json.Marshal(args)
	if err != nil {
		return nil, "", fmt.Errorf("plugin tool %s: cannot encode arguments: %w", p.Name(), err)
	}
	endpoint := fmt.Sprintf("%s/api/tools/%s/%s/execute", p.executorURL, url.PathEscape(p.Plugin), url.PathEscape(p.Tool))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("plugin tool %s: cannot build request: %w", p.Name(), err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := p.client
	if client == nil {
		client = &http.Client{Timeout: pluginTimeoutFor(p.Plugin)}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("plugin tool %s: executor request failed: %w", p.Name(), err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, "", fmt.Errorf("plugin tool %s: cannot read executor reply: %w", p.Name(), err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("plugin tool %s: executor returned status %d: %s", p.Name(), resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var reply map[string]any
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, "", fmt.Errorf("plugin tool %s: executor reply is not JSON: %w", p.Name(), err)
	}
	if confirmed, _ := reply["confirmation_required"].(bool); confirmed {
		return nil, "", fmt.Errorf("plugin tool %s: confirmation required, action not run", p.Name())
	}
	if ok, _ := reply["ok"].(bool); !ok {
		// The seed marks successful dry-run replies with ok true. A
		// reply without ok true and without confirmation_required is
		// not a success we can report to the model as one.
		if _, hasError := reply["error"]; hasError {
			return nil, "", fmt.Errorf("plugin tool %s: executor error: %v", p.Name(), reply["error"])
		}
		return nil, "", fmt.Errorf("plugin tool %s: executor reply did not confirm success: %s", p.Name(), strings.TrimSpace(string(raw)))
	}
	content, err := json.Marshal(reply)
	if err != nil {
		return nil, "", fmt.Errorf("plugin tool %s: cannot encode reply: %w", p.Name(), err)
	}
	return reply, string(content), nil
}

// MentionedPlugins returns the enabled plugins named with @ in a
// prompt, in first mention order, without duplicates. A mention is
// @ followed by a plugin name, for example @browser or @computer,
// at the start of the prompt or after whitespace or an opening
// bracket, and ending at the first character that is not a letter,
// digit, or hyphen. Matching is case insensitive, but the returned
// names use the spelling from the enabled list, so @Browser still
// selects browser. An @ token that is part of an email address,
// for example user@browser, is not a mention, because the @ is
// preceded by a letter or digit. A mention of a plugin that is not
// in the enabled list is not returned, so it gets no tools: the
// Plugins page enable stays the master switch, and a typed mention
// can never turn a plugin on by itself.
//
// Callers use this rule: if the prompt contains any @ mention at
// all, only the plugins returned here are registered for that turn.
// If it contains no @ mention, all enabled plugins are registered.
// That means a prompt that mentions only a non-enabled plugin
// registers no plugin tools for that turn, instead of silently
// falling back to every enabled plugin.
func MentionedPlugins(prompt string, enabled []string) []string {
	byLower := make(map[string]string, len(enabled))
	for _, name := range enabled {
		if _, ok := byLower[strings.ToLower(name)]; !ok {
			byLower[strings.ToLower(name)] = name
		}
	}
	seen := make(map[string]bool)
	out := []string{}
	for _, token := range mentionTokens(prompt) {
		if canonical, ok := byLower[strings.ToLower(token)]; ok && !seen[canonical] {
			seen[canonical] = true
			out = append(out, canonical)
		}
	}
	return out
}

// HasMention reports whether a prompt contains any @ mention token,
// whether or not that token names an enabled plugin. Callers use
// this to decide between the mentioned-only rule and the
// all-enabled rule in MentionedPlugins.
func HasMention(prompt string) bool {
	return len(mentionTokens(prompt)) > 0
}

// mentionTokens extracts @ tokens from a prompt, in order, without
// duplicates removed, so callers can decide on duplicates. A token
// is the run of letters, digits, and hyphens after an @ that is at
// the start of the prompt or preceded by whitespace, an opening
// bracket, a quote, or a colon, and not by a letter or digit.
func mentionTokens(prompt string) []string {
	tokens := []string{}
	runes := []rune(prompt)
	for i := 0; i < len(runes); i++ {
		if runes[i] != '@' {
			continue
		}
		if i > 0 {
			prev := runes[i-1]
			if prev >= 'a' && prev <= 'z' || prev >= 'A' && prev <= 'Z' || prev >= '0' && prev <= '9' {
				continue
			}
		}
		j := i + 1
		for j < len(runes) {
			r := runes[j]
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
				j++
			} else {
				break
			}
		}
		if j > i+1 {
			tokens = append(tokens, string(runes[i+1:j]))
		}
		i = j - 1
	}
	return tokens
}

// pluginToolsResponse is the shape of GET /api/tools from both the
// fork server and the seed server.
type pluginToolsResponse struct {
	Tools []struct {
		Plugin   string `json:"plugin"`
		Type     string `json:"type"`
		Function struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			Parameters  map[string]any `json:"parameters"`
		} `json:"function"`
	} `json:"tools"`
}

// LoadPluginTools GETs tool definitions from toolsURL and builds one
// PluginTool per definition, skipping malformed entries instead of
// failing the whole list. A malformed entry is one with no usable
// dotted function name. toolsURL may be empty for the default, and
// executorURL may be empty for the default seed server URL. Both must
// be local URLs.
func LoadPluginTools(ctx context.Context, toolsURL, executorURL string) ([]*PluginTool, error) {
	if toolsURL == "" {
		toolsURL = DefaultPluginToolsURL
	}
	if executorURL == "" {
		executorURL = DefaultPluginExecutorURL
	}
	if err := validateLocalURL(toolsURL); err != nil {
		return nil, fmt.Errorf("plugin tools list: %w", err)
	}
	if err := validateLocalURL(executorURL); err != nil {
		return nil, fmt.Errorf("plugin tool executor: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, toolsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("plugin tools list: cannot build request: %w", err)
	}
	client := &http.Client{Timeout: pluginExecuteTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plugin tools list: request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("plugin tools list: status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed pluginToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("plugin tools list: invalid JSON: %w", err)
	}
	out := make([]*PluginTool, 0, len(parsed.Tools))
	for _, entry := range parsed.Tools {
		name := entry.Function.Name
		dot := strings.Index(name, ".")
		if dot <= 0 || dot == len(name)-1 {
			continue
		}
		plugin := name[:dot]
		tool := name[dot+1:]
		// If the entry carries a plugin field, it must agree with the
		// name prefix. A disagreement means the entry is malformed.
		if entry.Plugin != "" && entry.Plugin != plugin {
			continue
		}
		schema := entry.Function.Parameters
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		built, err := NewPluginTool(plugin, tool, entry.Function.Description, schema, executorURL)
		if err != nil {
			continue
		}
		out = append(out, built)
	}
	return out, nil
}
