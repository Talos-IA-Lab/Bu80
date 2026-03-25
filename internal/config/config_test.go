package config

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"bu80/internal/state"
)

func TestInitDefaultConfigWritesAgentsFile(t *testing.T) {
	path := t.TempDir() + "/agents.json"
	resolved, err := InitDefaultConfig(path)
	if err != nil {
		t.Fatalf("init config: %v", err)
	}
	if resolved != path {
		t.Fatalf("unexpected resolved path: %q", resolved)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if !strings.Contains(string(data), "\"codex\"") || !strings.Contains(string(data), "\"opencode\"") {
		t.Fatalf("unexpected config content: %s", string(data))
	}
}

func TestBuildOpenCodeEnvGeneratesFilteredConfig(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	baseConfig := dir + "/agents.json"
	if err := os.WriteFile(baseConfig, []byte(`{"plugins":["auth-provider","misc-plugin"],"permissions":{"bash":"deny"}}`), 0o644); err != nil {
		t.Fatalf("write base config: %v", err)
	}
	env, err := BuildOpenCodeEnv(map[string]string{"PATH": os.Getenv("PATH")}, baseConfig, true, true)
	if err != nil {
		t.Fatalf("build opencode env: %v", err)
	}
	if env["OPENCODE_CONFIG"] != state.OpenCodeConfigFile {
		t.Fatalf("expected OPENCODE_CONFIG to point at generated file, got %q", env["OPENCODE_CONFIG"])
	}
	data, readErr := os.ReadFile(state.OpenCodeConfigFile)
	if readErr != nil {
		t.Fatalf("read generated config: %v", readErr)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal generated config: %v", err)
	}
	plugins, ok := cfg["plugins"].([]any)
	if !ok || len(plugins) != 1 {
		t.Fatalf("expected filtered auth plugins, got %+v", cfg["plugins"])
	}
	permissions, ok := cfg["permissions"].(map[string]any)
	if !ok || permissions["bash"] != "allow" {
		t.Fatalf("expected permissive permissions, got %+v", cfg["permissions"])
	}
}
