package output

import (
	"regexp"
	"strings"
)

var (
	ansiPattern    = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	promisePattern = regexp.MustCompile(`(?i)^<\s*promise\s*>\s*(.*?)\s*<\s*/\s*promise\s*>$`)
)

func StripANSI(input string) string {
	return ansiPattern.ReplaceAllString(input, "")
}

func NormalizeLineEndings(input string) string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	return strings.ReplaceAll(input, "\r", "\n")
}

func LastNonEmptyLine(input string) string {
	normalized := NormalizeLineEndings(StripANSI(input))
	lines := strings.Split(normalized, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return ""
}

func DetectPromise(output string, promise string) bool {
	if strings.TrimSpace(promise) == "" {
		return false
	}

	lastLine := LastNonEmptyLine(output)
	matches := promisePattern.FindStringSubmatch(lastLine)
	if len(matches) != 2 {
		return false
	}

	return strings.TrimSpace(matches[1]) == strings.TrimSpace(promise)
}
