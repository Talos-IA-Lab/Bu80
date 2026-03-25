package agent

import (
	"fmt"
	"runtime"
	"strings"
)

type Definition struct {
	Name        string
	DisplayName string
	Command     string
	EnvOverride string
}

type InvocationOptions struct {
	Prompt    string
	Model     string
	ExtraArgs []string
	AllowAll  bool
}

func Builtins() map[string]Definition {
	return map[string]Definition{
		"opencode": {
			Name:        "opencode",
			DisplayName: "OpenCode",
			Command:     "opencode",
			EnvOverride: "RALPH_OPENCODE_BINARY",
		},
		"claude-code": {
			Name:        "claude-code",
			DisplayName: "Claude Code",
			Command:     "claude",
			EnvOverride: "RALPH_CLAUDE_BINARY",
		},
		"codex": {
			Name:        "codex",
			DisplayName: "Codex",
			Command:     "codex",
			EnvOverride: "RALPH_CODEX_BINARY",
		},
		"copilot": {
			Name:        "copilot",
			DisplayName: "Copilot CLI",
			Command:     "copilot",
			EnvOverride: "RALPH_COPILOT_BINARY",
		},
	}
}

type RotationEntry struct {
	Agent string
	Model string
}

func ParseRotation(input string, known map[string]Definition) ([]RotationEntry, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}

	parts := strings.Split(input, ",")
	entries := make([]RotationEntry, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		pieces := strings.Split(part, ":")
		if len(pieces) != 2 {
			return nil, fmt.Errorf("invalid rotation entry %q: expected agent:model", part)
		}

		agentName := strings.TrimSpace(pieces[0])
		model := strings.TrimSpace(pieces[1])
		if agentName == "" || model == "" {
			return nil, fmt.Errorf("invalid rotation entry %q: both agent and model are required", part)
		}
		if _, ok := known[agentName]; !ok {
			return nil, fmt.Errorf("unknown agent in rotation: %s", agentName)
		}

		entries = append(entries, RotationEntry{Agent: agentName, Model: model})
	}

	return entries, nil
}

func FormatRotation(entries []RotationEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Agent+":"+entry.Model)
	}
	return out
}

func CurrentRotationEntry(entries []string, index int, fallbackAgent string, fallbackModel string) RotationEntry {
	if len(entries) == 0 {
		return RotationEntry{Agent: fallbackAgent, Model: fallbackModel}
	}
	if index < 0 {
		index = 0
	}
	entry := strings.TrimSpace(entries[index%len(entries)])
	parts := strings.SplitN(entry, ":", 2)
	if len(parts) != 2 {
		return RotationEntry{Agent: fallbackAgent, Model: fallbackModel}
	}
	agentName := strings.TrimSpace(parts[0])
	model := strings.TrimSpace(parts[1])
	if agentName == "" {
		agentName = fallbackAgent
	}
	if model == "" {
		model = fallbackModel
	}
	return RotationEntry{Agent: agentName, Model: model}
}

func ResolveCommand(def Definition, env map[string]string) string {
	if def.EnvOverride != "" {
		if override := strings.TrimSpace(env[def.EnvOverride]); override != "" {
			return normalizeWindowsCommand(override)
		}
	}
	return normalizeWindowsCommand(def.Command)
}

func BuildArgs(def Definition, opts InvocationOptions) ([]string, error) {
	prompt := strings.TrimSpace(opts.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for agent %s", def.Name)
	}

	switch def.Name {
	case "opencode":
		args := []string{"run"}
		if strings.TrimSpace(opts.Model) != "" {
			args = append(args, "--model", strings.TrimSpace(opts.Model))
		}
		args = append(args, opts.ExtraArgs...)
		args = append(args, prompt)
		return args, nil
	case "claude-code":
		args := []string{"-p", prompt}
		if strings.TrimSpace(opts.Model) != "" {
			args = append(args, "--model", strings.TrimSpace(opts.Model))
		}
		args = append(args, opts.ExtraArgs...)
		return args, nil
	case "codex":
		args := []string{"exec"}
		if strings.TrimSpace(opts.Model) != "" {
			args = append(args, "--model", strings.TrimSpace(opts.Model))
		}
		if opts.AllowAll {
			args = append(args, "--full-auto")
		}
		args = append(args, opts.ExtraArgs...)
		args = append(args, prompt)
		return args, nil
	case "copilot":
		args := []string{"-p", prompt}
		if strings.TrimSpace(opts.Model) != "" {
			args = append(args, "--model", strings.TrimSpace(opts.Model))
		}
		args = append(args, opts.ExtraArgs...)
		return args, nil
	default:
		return nil, fmt.Errorf("unsupported agent: %s", def.Name)
	}
}

func normalizeWindowsCommand(command string) string {
	if runtime.GOOS == "windows" && !strings.Contains(command, ".") {
		return command + ".cmd"
	}
	return command
}
