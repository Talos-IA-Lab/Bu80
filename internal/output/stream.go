package output

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func ToolName(line string) string {
	return detectToolName(line)
}

func SimplifyDisplayLine(agentName string, line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	if agentName != "claude-code" || !strings.HasPrefix(trimmed, "{") {
		return line
	}
	text := extractClaudeText(trimmed)
	if strings.TrimSpace(text) == "" {
		return line
	}
	return text
}

func FormatToolSummary(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s=%d", name, counts[name]))
	}
	return "[loop] tools: " + strings.Join(parts, ", ")
}

func extractClaudeText(line string) string {
	var payload any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		return ""
	}
	texts := collectDisplayText(payload)
	joined := strings.TrimSpace(strings.Join(texts, "\n"))
	if joined == "" || joined == line {
		return ""
	}
	return joined
}

func collectDisplayText(value any) []string {
	switch v := value.(type) {
	case map[string]any:
		keys := []string{"text", "content", "message", "delta"}
		out := []string{}
		for _, key := range keys {
			if nested, ok := v[key]; ok {
				out = append(out, collectDisplayText(nested)...)
			}
		}
		return uniqueStrings(out)
	case []any:
		out := []string{}
		for _, item := range v {
			out = append(out, collectDisplayText(item)...)
		}
		return uniqueStrings(out)
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" || strings.HasPrefix(trimmed, "{") {
			return nil
		}
		return []string{trimmed}
	default:
		return nil
	}
}

func uniqueStrings(input []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(input))
	for _, item := range input {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
