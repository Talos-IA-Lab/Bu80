package output

import (
	"regexp"
	"sort"
	"strings"
)

var (
	jsonToolPattern     = regexp.MustCompile(`(?i)"tool"\s*:\s*"([a-z0-9._-]+)"`)
	nameToolPattern     = regexp.MustCompile(`(?i)\bname\s*=\s*([a-z0-9._-]+)\b`)
	inlineToolPattern   = regexp.MustCompile(`(?i)\btool\s*[=:]\s*([a-z0-9._-]+)\b`)
	prefixedToolPattern = regexp.MustCompile(`(?i)^tool\s+([a-z0-9._-]+)\s*:`)
)

func ParseToolCounts(outputText string) map[string]int {
	counts := map[string]int{}
	for _, line := range strings.Split(NormalizeLineEndings(outputText), "\n") {
		tool := detectToolName(line)
		if tool == "" {
			continue
		}
		counts[tool]++
	}
	return counts
}

func ParseTools(outputText string) []string {
	counts := ParseToolCounts(outputText)
	if len(counts) == 0 {
		return nil
	}
	tools := make([]string, 0, len(counts))
	for name := range counts {
		tools = append(tools, name)
	}
	sort.Strings(tools)
	return tools
}

func detectToolName(line string) string {
	trimmed := strings.TrimSpace(StripANSI(line))
	if trimmed == "" {
		return ""
	}
	if match := jsonToolPattern.FindStringSubmatch(trimmed); len(match) == 2 {
		return strings.ToLower(strings.TrimSpace(match[1]))
	}
	if match := prefixedToolPattern.FindStringSubmatch(trimmed); len(match) == 2 {
		return strings.ToLower(strings.TrimSpace(match[1]))
	}
	if match := inlineToolPattern.FindStringSubmatch(trimmed); len(match) == 2 {
		return strings.ToLower(strings.TrimSpace(match[1]))
	}
	if match := nameToolPattern.FindStringSubmatch(trimmed); len(match) == 2 && strings.Contains(strings.ToLower(trimmed), "question") {
		return strings.ToLower(strings.TrimSpace(match[1]))
	}
	return ""
}
