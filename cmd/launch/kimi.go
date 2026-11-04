package launch

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/ollama/ollama/cmd/internal/fileutil"
	"github.com/ollama/ollama/envconfig"
	modelpkg "github.com/ollama/ollama/types/model"
)

// Kimi implements Runner for Kimi Code CLI integration.
type Kimi struct{}

const (
	kimiDefaultModelAlias     = "ollama"
	kimiDefaultMaxContextSize = 32768
	kimiCodeProviderName      = "ollama"
	kimiLegacyVersionPrefix   = "kimi, version "
)

var kimiGOOS = runtime.GOOS

func (k *Kimi) String() string { return "Kimi Code CLI" }

func (k *Kimi) args(config string, extra []string) []string {
	args := []string{"--config", config}
	args = append(args, extra...)
	return args
}

func (k *Kimi) Run(model string, models []LaunchModel, args []string) error {
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required")
	}
	if err := validateKimiPassthroughArgs(args); err != nil {
		return err
	}

	bin, err := ensureKimiInstalled()
	if err != nil {
		return err
	}

	var cmdArgs []string
	if _, legacy := kimiLegacyCLIVersion(bin); legacy {
		config, err := buildKimiInlineConfig(model, resolveKimiMaxContextSize(model))
		if err != nil {
			return fmt.Errorf("failed to build kimi config: %w", err)
		}
		cmdArgs = k.args(config, args)
	} else {
		if err := writeKimiCodeConfig(model, models); err != nil {
			return fmt.Errorf("failed to configure kimi: %w", err)
		}
		cmdArgs = append([]string{"-m", kimiCodeModelAlias(model)}, args...)
	}

	cmd := exec.Command(bin, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func kimiLegacyCLIVersion(bin string) (string, bool) {
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return "", false
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	version, ok := strings.CutPrefix(strings.TrimSpace(line), kimiLegacyVersionPrefix)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(version), true
}

func kimiCodeModelAlias(model string) string {
	return kimiCodeProviderName + "/" + model
}

func kimiCodeConfigPath() (string, error) {
	if home := strings.TrimSpace(os.Getenv("KIMI_CODE_HOME")); home != "" {
		return filepath.Join(home, "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kimi-code", "config.toml"), nil
}

func writeKimiCodeConfig(primary string, models []LaunchModel) error {
	path, err := kimiCodeConfigPath()
	if err != nil {
		return err
	}

	cfg := map[string]any{}
	exists := false
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		exists = true
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	case !os.IsNotExist(err):
		return err
	}

	managedBefore := kimiCodeManagedConfig(cfg)

	providers := kimiCodeTable(cfg, "providers")
	providers[kimiCodeProviderName] = map[string]any{
		"type":     "openai",
		"base_url": envconfig.ConnectableHost().String() + "/v1",
		"api_key":  "ollama",
	}

	aliases := kimiCodeTable(cfg, "models")
	for alias, entry := range aliases {
		if table, ok := entry.(map[string]any); ok && table["provider"] == kimiCodeProviderName {
			delete(aliases, alias)
		}
	}
	if current, _ := cfg["default_model"].(string); current == "" || aliases[current] == nil {
		cfg["default_model"] = kimiCodeModelAlias(primary)
	}
	if !slices.ContainsFunc(models, func(m LaunchModel) bool { return m.Name == primary }) {
		models = append([]LaunchModel{{Name: primary}}, models...)
	}
	for _, m := range models {
		aliases[kimiCodeModelAlias(m.Name)] = kimiCodeModelEntry(m)
	}

	if exists && reflect.DeepEqual(managedBefore, kimiCodeManagedConfig(cfg)) {
		return nil
	}
	if exists && kimiCodeHasUserContent(cfg) {
		ok, err := ConfirmPrompt(fmt.Sprintf("Kimi Code config %s already has settings. Update it for Ollama now? (your settings are kept and a backup is saved)", path))
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("kimi config %s was not updated; re-run with --yes to update it", path)
		}
	}

	out, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fileutil.WriteWithBackup(path, out, "kimi")
}

func kimiCodeTable(cfg map[string]any, key string) map[string]any {
	if table, ok := cfg[key].(map[string]any); ok {
		return table
	}
	table := map[string]any{}
	cfg[key] = table
	return table
}

func kimiCodeOwnedModel(entry any) bool {
	table, ok := entry.(map[string]any)
	return ok && table["provider"] == kimiCodeProviderName
}

func kimiCodeManagedConfig(cfg map[string]any) map[string]any {
	out := map[string]any{}
	if table, ok := cfg["providers"].(map[string]any); ok {
		if entry, ok := table[kimiCodeProviderName]; ok {
			out["providers"] = map[string]any{kimiCodeProviderName: entry}
		}
	}
	owned := map[string]any{}
	if table, ok := cfg["models"].(map[string]any); ok {
		for alias, entry := range table {
			if kimiCodeOwnedModel(entry) {
				owned[alias] = entry
			}
		}
	}
	if len(owned) > 0 {
		out["models"] = owned
	}
	if value, ok := cfg["default_model"]; ok {
		out["default_model"] = value
	}
	return out
}

func kimiCodeHasUserContent(cfg map[string]any) bool {
	for key, value := range cfg {
		switch key {
		case "providers":
			table, ok := value.(map[string]any)
			if !ok {
				return true
			}
			for name := range table {
				if name != kimiCodeProviderName {
					return true
				}
			}
		case "models":
			table, ok := value.(map[string]any)
			if !ok {
				return true
			}
			for _, entry := range table {
				if !kimiCodeOwnedModel(entry) {
					return true
				}
			}
		case "default_model":
		default:
			return true
		}
	}
	return false
}

func kimiCodeModelEntry(m LaunchModel) map[string]any {
	if m.Thinking == nil && m.HasCapability(modelpkg.CapabilityThinking) {
		if meta, ok := lookupServerModelMetadata(m.Name); ok {
			m.Thinking = meta.Thinking.Clone()
		}
	}
	maxContextSize := m.ContextLength
	if maxContextSize <= 0 {
		maxContextSize = resolveKimiMaxContextSize(m.Name)
	}
	entry := map[string]any{
		"provider":         kimiCodeProviderName,
		"model":            m.Name,
		"max_context_size": maxContextSize,
	}

	capabilities := []string{}
	if m.ToolCapable || m.HasCapability(modelpkg.CapabilityTools) {
		capabilities = append(capabilities, "tool_use")
	}
	if m.HasCapability(modelpkg.CapabilityVision) {
		capabilities = append(capabilities, "image_in")
	}

	levels, defaultLevel := launchReasoningEffortLevels(m.Thinking)
	efforts := slices.DeleteFunc(slices.Clone(levels), func(level string) bool { return level == "none" })
	if len(efforts) > 0 || m.HasCapability(modelpkg.CapabilityThinking) {
		capabilities = append(capabilities, "thinking")
		if slices.Contains(levels, "none") {
			entry["off_effort"] = "none"
		} else if len(efforts) > 0 {
			capabilities = append(capabilities, "always_thinking")
		}
		if len(efforts) > 0 {
			entry["support_efforts"] = efforts
			if slices.Contains(efforts, defaultLevel) {
				entry["default_effort"] = defaultLevel
			}
		}
	}
	entry["capabilities"] = capabilities
	return entry
}

func findKimiBinary() (string, error) {
	home, _ := os.UserHomeDir()

	kimiCodeBin := filepath.Join(home, ".kimi-code", "bin", "kimi")
	if kimiGOOS == "windows" {
		kimiCodeBin += ".exe"
	}
	if info, err := os.Stat(kimiCodeBin); err == nil && !info.IsDir() {
		return kimiCodeBin, nil
	}

	if path, err := exec.LookPath("kimi"); err == nil {
		return path, nil
	}

	var candidates []string
	switch kimiGOOS {
	case "windows":
		candidates = appendWindowsKimiCandidates(candidates, filepath.Join(home, ".local", "bin"))
		candidates = appendWindowsKimiCandidates(candidates, filepath.Join(home, "bin"))

		if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
			candidates = appendWindowsKimiCandidates(candidates, filepath.Join(appData, "uv", "bin"))
		}
		if localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); localAppData != "" {
			candidates = appendWindowsKimiCandidates(candidates, filepath.Join(localAppData, "uv", "bin"))
		}
	default:
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "kimi"),
			filepath.Join(home, "bin", "kimi"),
			filepath.Join(home, ".local", "share", "uv", "tools", "kimi-cli", "bin", "kimi"),
			filepath.Join(home, ".local", "share", "uv", "tools", "kimi", "bin", "kimi"),
		)

		if xdgDataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdgDataHome != "" {
			candidates = append(candidates,
				filepath.Join(xdgDataHome, "uv", "tools", "kimi-cli", "bin", "kimi"),
				filepath.Join(xdgDataHome, "uv", "tools", "kimi", "bin", "kimi"),
			)
		}

		// WSL users can inherit Windows env vars while launching from Linux shells.
		if profile := windowsPathToWSL(os.Getenv("USERPROFILE")); profile != "" {
			candidates = appendWindowsKimiCandidates(candidates, filepath.Join(profile, ".local", "bin"))
		}
		if appData := windowsPathToWSL(os.Getenv("APPDATA")); appData != "" {
			candidates = appendWindowsKimiCandidates(candidates, filepath.Join(appData, "uv", "bin"))
		}
		if localAppData := windowsPathToWSL(os.Getenv("LOCALAPPDATA")); localAppData != "" {
			candidates = appendWindowsKimiCandidates(candidates, filepath.Join(localAppData, "uv", "bin"))
		}
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("kimi binary not found")
}

func appendWindowsKimiCandidates(candidates []string, dir string) []string {
	if strings.TrimSpace(dir) == "" {
		return candidates
	}

	return append(candidates,
		filepath.Join(dir, "kimi.exe"),
		filepath.Join(dir, "kimi.cmd"),
		filepath.Join(dir, "kimi.bat"),
	)
}

func windowsPathToWSL(path string) string {
	trimmed := strings.TrimSpace(path)
	if len(trimmed) < 3 || trimmed[1] != ':' {
		return ""
	}

	drive := strings.ToLower(string(trimmed[0]))
	rest := strings.ReplaceAll(trimmed[2:], "\\", "/")
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		return filepath.Join("/mnt", drive)
	}

	return filepath.Join("/mnt", drive, rest)
}

func validateKimiPassthroughArgs(args []string) error {
	for _, arg := range args {
		switch {
		case arg == "--config", strings.HasPrefix(arg, "--config="):
			return fmt.Errorf("conflicting extra argument %q: ollama launch kimi manages --config", arg)
		case arg == "--config-file", strings.HasPrefix(arg, "--config-file="):
			return fmt.Errorf("conflicting extra argument %q: ollama launch kimi manages --config-file", arg)
		case arg == "--model", strings.HasPrefix(arg, "--model="):
			return fmt.Errorf("conflicting extra argument %q: ollama launch kimi manages --model", arg)
		case arg == "-m", strings.HasPrefix(arg, "-m="):
			return fmt.Errorf("conflicting extra argument %q: ollama launch kimi manages -m/--model", arg)
		}
	}
	return nil
}

func buildKimiInlineConfig(model string, maxContextSize int) (string, error) {
	cfg := map[string]any{
		"default_model": kimiDefaultModelAlias,
		"providers": map[string]any{
			kimiDefaultModelAlias: map[string]any{
				"type":     "openai_legacy",
				"base_url": envconfig.ConnectableHost().String() + "/v1",
				"api_key":  "ollama",
			},
		},
		"models": map[string]any{
			kimiDefaultModelAlias: map[string]any{
				"provider":         kimiDefaultModelAlias,
				"model":            model,
				"max_context_size": maxContextSize,
			},
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func resolveKimiMaxContextSize(model string) int {
	if l, ok := lookupCloudModelLimit(model); ok {
		return l.Context
	}

	if n := showContextLength(model); n > 0 {
		return n
	}

	return kimiDefaultMaxContextSize
}

func ensureKimiInstalled() (string, error) {
	existing, findErr := findKimiBinary()
	version := ""
	if findErr == nil {
		var legacy bool
		version, legacy = kimiLegacyCLIVersion(existing)
		if !legacy {
			return existing, nil
		}
	}

	prompt := "Kimi is not installed. Install now?"
	if existing != "" {
		prompt = fmt.Sprintf("kimi-cli %s is no longer maintained and has been replaced by Kimi Code. Install Kimi Code now?", version)
	}

	if err := checkKimiInstallerDependencies(); err != nil {
		return "", err
	}

	ok, err := ConfirmPrompt(prompt)
	if err != nil {
		return "", err
	}
	if !ok {
		if existing != "" {
			fmt.Fprintf(os.Stderr, "%sContinuing with kimi-cli %s; re-run with --yes to install Kimi Code%s\n\n", ansiYellow, version, ansiReset)
			return existing, nil
		}
		return "", fmt.Errorf("kimi installation cancelled")
	}

	bin, args, err := kimiInstallerCommand(kimiGOOS)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(os.Stderr, "\nInstalling Kimi...\n")
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to install kimi: %w", err)
	}

	path, err := findKimiBinary()
	if err != nil {
		return "", fmt.Errorf("kimi was installed but the binary was not found on PATH\n\nYou may need to restart your shell")
	}

	fmt.Fprintf(os.Stderr, "%sKimi installed successfully%s\n\n", ansiGreen, ansiReset)
	return path, nil
}

func checkKimiInstallerDependencies() error {
	switch kimiGOOS {
	case "windows":
		if _, err := exec.LookPath("powershell"); err != nil {
			return fmt.Errorf("kimi is not installed and required dependencies are missing\n\nInstall the following first:\n  PowerShell: https://learn.microsoft.com/powershell/\n\nThen re-run:\n  ollama launch kimi")
		}
	default:
		var missing []string
		if _, err := exec.LookPath("curl"); err != nil {
			missing = append(missing, "curl: https://curl.se/")
		}
		if _, err := exec.LookPath("bash"); err != nil {
			missing = append(missing, "bash: https://www.gnu.org/software/bash/")
		}
		if len(missing) > 0 {
			return fmt.Errorf("kimi is not installed and required dependencies are missing\n\nInstall the following first:\n  %s\n\nThen re-run:\n  ollama launch kimi", strings.Join(missing, "\n  "))
		}
	}
	return nil
}

func kimiInstallerCommand(goos string) (string, []string, error) {
	switch goos {
	case "windows":
		return "powershell", []string{
			"-NoProfile",
			"-ExecutionPolicy",
			"Bypass",
			"-Command",
			"Invoke-RestMethod https://code.kimi.com/kimi-code/install.ps1 | Invoke-Expression",
		}, nil
	case "darwin", "linux":
		return "bash", []string{
			"-c",
			"curl -LsSf https://code.kimi.com/kimi-code/install.sh | bash",
		}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform for kimi install: %s", goos)
	}
}
