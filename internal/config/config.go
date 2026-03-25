package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"bu80/internal/state"
)

func ResolvePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = state.DefaultConfigPath
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func InitDefaultConfig(path string) (string, error) {
	resolved := ResolvePath(path)
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(defaultAgentsConfig(), "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(resolved, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return resolved, nil
}

func BuildOpenCodeEnv(baseEnv map[string]string, configPath string, disablePlugins bool, allowAll bool) (map[string]string, error) {
	if !disablePlugins && !allowAll {
		return cloneEnv(baseEnv), nil
	}
	resolved := ResolvePath(configPath)
	generated, err := buildGeneratedOpenCodeConfig(resolved, disablePlugins, allowAll)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(state.OpenCodeConfigFile), 0o755); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(generated, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(state.OpenCodeConfigFile, append(data, '\n'), 0o644); err != nil {
		return nil, err
	}
	env := cloneEnv(baseEnv)
	env["OPENCODE_CONFIG"] = state.OpenCodeConfigFile
	return env, nil
}

func defaultAgentsConfig() map[string]any {
	return map[string]any{
		"agents": []map[string]any{
			{"name": "codex", "type": "codex"},
			{"name": "claude-code", "type": "claude-code"},
			{"name": "opencode", "type": "opencode"},
			{"name": "copilot", "type": "copilot"},
		},
	}
}

func buildGeneratedOpenCodeConfig(path string, disablePlugins bool, allowAll bool) (map[string]any, error) {
	config := map[string]any{}
	if strings.TrimSpace(path) != "" {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			_ = json.Unmarshal(data, &config)
		}
	}
	if disablePlugins {
		config["plugins"] = filteredPlugins(config["plugins"])
	}
	if allowAll {
		config["permissions"] = map[string]any{
			"bash":  "allow",
			"edit":  "allow",
			"read":  "allow",
			"write": "allow",
			"web":   "allow",
			"mcp":   "allow",
		}
	}
	return config, nil
}

func filteredPlugins(value any) []any {
	plugins, ok := value.([]any)
	if !ok {
		return []any{}
	}
	out := make([]any, 0, len(plugins))
	for _, plugin := range plugins {
		switch v := plugin.(type) {
		case string:
			if strings.Contains(strings.ToLower(v), "auth") {
				out = append(out, v)
			}
		case map[string]any:
			name := strings.ToLower(strings.TrimSpace(stringValue(v["name"])))
			if strings.Contains(name, "auth") {
				out = append(out, v)
			}
		}
	}
	return out
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func cloneEnv(base map[string]string) map[string]string {
	out := make(map[string]string, len(base))
	for k, v := range base {
		out[k] = v
	}
	return out
}
