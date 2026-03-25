package loop

import (
	"regexp"
	"strings"
)

var (
	jsonQuestionPattern   = regexp.MustCompile(`(?i)"question"\s*:\s*"([^"]+)"`)
	quotedQuestionPattern = regexp.MustCompile(`"([^"]+\?)"`)
)

func detectQuestion(outputText string) string {
	for _, line := range strings.Split(outputText, "\n") {
		question := extractQuestion(line)
		if question != "" {
			return question
		}
	}
	return ""
}

func extractQuestion(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if !strings.Contains(lower, "question") {
		return ""
	}
	if match := jsonQuestionPattern.FindStringSubmatch(trimmed); len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	for _, marker := range []string{"question:", "tool question:", "tool=question", "name=question"} {
		idx := strings.Index(lower, marker)
		if idx >= 0 {
			suffix := strings.TrimSpace(trimmed[idx+len(marker):])
			suffix = strings.Trim(suffix, " \t-:=\"'")
			if suffix != "" {
				return suffix
			}
		}
	}
	if match := quotedQuestionPattern.FindStringSubmatch(trimmed); len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}
