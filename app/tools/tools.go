//go:build windows || darwin

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Tool defines the interface that all tools must implement
type Tool interface {
	// Name returns the unique identifier for the tool
	Name() string

	// Description returns a human-readable description of what the tool does
	Description() string

	// Schema returns the JSON schema for the tool's parameters
	Schema() map[string]any

	// Execute runs the tool with the given arguments and returns result to store in db, and a string result for the model
	Execute(ctx context.Context, args map[string]any) (any, string, error)

	// Prompt returns a prompt for the tool
	Prompt() string
}

// Registry manages the available tools and their execution
type Registry struct {
	tools      map[string]Tool
	workingDir string // Working directory for all tool operations
}

// NewRegistry creates a new tool registry with no tools
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// resolve finds a tool by name. An exact name always wins. A
// non-exact name is also resolved against the registered names:
// it matches a registered name when the part after the first dot
// equals it (so "webmcp.list_tools" matches
// "browser.webmcp.list_tools") or when the last segment after the
// last dot equals it (so "list_tools" matches it, "create_tab"
// matches "cua.create_tab", and "list_apps" matches
// "computer.list_apps"). Some models send only that last part, so
// without this step a call they meant for a registered tool fails
// as unknown. If more than one registered name matches, resolve
// returns an error naming every candidate and no tool, so the
// caller never guesses between, for example, "cua.click" and
// "computer.click" when both are registered.
func (r *Registry) resolve(name string) (Tool, error) {
	if tool, exists := r.tools[name]; exists {
		return tool, nil
	}
	var candidates []string
	for registered := range r.tools {
		afterFirst := registered[strings.Index(registered, ".")+1:]
		last := registered[strings.LastIndex(registered, ".")+1:]
		if afterFirst == name || last == name {
			candidates = append(candidates, registered)
		}
	}
	sort.Strings(candidates)
	switch len(candidates) {
	case 0:
		return nil, fmt.Errorf("unknown tool: %s", name)
	case 1:
		return r.tools[candidates[0]], nil
	default:
		return nil, fmt.Errorf("ambiguous tool name %q, matches %s, use a full tool name", name, strings.Join(candidates, ", "))
	}
}

// Get retrieves a tool by name, using the same resolution as
// Execute: exact first, then a unique bare name, never a guess
// between several matches.
func (r *Registry) Get(name string) (Tool, bool) {
	tool, err := r.resolve(name)
	return tool, err == nil
}

// ResolveName returns the registered full name for a requested
// name, or the resolution error: unknown, or ambiguous with every
// candidate named. It executes nothing, so callers can tell an
// unknown name (leave it alone) from an ambiguous one (report
// the candidates) before deciding whether to run a tool.
func (r *Registry) ResolveName(name string) (string, error) {
	tool, err := r.resolve(name)
	if err != nil {
		return "", err
	}
	return tool.Name(), nil
}

// TextJSONToolCall is one tool call written as plain assistant
// text instead of a structured tool call.
type TextJSONToolCall struct {
	Name string
	Args map[string]any
}

// ParseTextJSONToolCall parses an assistant message that is,
// after trimming whitespace, exactly one JSON object with a
// string "name" and an "arguments" object. It returns false for
// anything else: text plus JSON, a code block, a JSON array, a
// missing or non-string name, or non-object arguments. A missing
// "arguments" field is treated as an empty object. Parsing alone
// does not mean the named tool exists; callers must still resolve
// the name through the registry, and must leave an unknown name
// as chat text rather than inventing a tool.
func ParseTextJSONToolCall(content string) (TextJSONToolCall, bool) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return TextJSONToolCall{}, false
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	var raw map[string]json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return TextJSONToolCall{}, false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return TextJSONToolCall{}, false
	}
	var name string
	if err := json.Unmarshal(raw["name"], &name); err != nil || name == "" {
		return TextJSONToolCall{}, false
	}
	args := map[string]any{}
	if rawArgs, ok := raw["arguments"]; ok {
		if err := json.Unmarshal(rawArgs, &args); err != nil || args == nil {
			return TextJSONToolCall{}, false
		}
		// Reject a JSON null for arguments as well as a non-object.
		if string(bytes.TrimSpace(rawArgs)) == "null" {
			return TextJSONToolCall{}, false
		}
	}
	return TextJSONToolCall{Name: name, Args: args}, true
}

// List returns all available tools
func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// SetWorkingDir sets the working directory for all tool operations
func (r *Registry) SetWorkingDir(dir string) {
	r.workingDir = dir
}

// Execute runs a tool with the given name and arguments
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (any, string, error) {
	tool, err := r.resolve(name)
	if err != nil {
		return nil, "", err
	}

	result, text, err := tool.Execute(ctx, args)
	if err != nil {
		return nil, "", err
	}
	return result, text, nil
}

// ToolCall represents a request to execute a tool
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction represents the function call details
type ToolFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    any    `json:"content"`
	Error      string `json:"error,omitempty"`
}

// ToolSchemas returns all tools as schema maps suitable for API calls
func (r *Registry) AvailableTools() []map[string]any {
	schemas := make([]map[string]any, 0, len(r.tools))
	for _, tool := range r.tools {
		schema := map[string]any{
			"name":        tool.Name(),
			"description": tool.Description(),
			"schema":      tool.Schema(),
		}
		schemas = append(schemas, schema)
	}
	return schemas
}

// ToolNames returns a list of all tool names
func (r *Registry) ToolNames() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
