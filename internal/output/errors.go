package output

import "strings"

func DetectOpenCodePlaceholderPlugin(outputText string) bool {
	lower := strings.ToLower(outputText)
	return strings.Contains(lower, "placeholder") && strings.Contains(lower, "plugin")
}

func DetectMissingModel(outputText string) bool {
	lower := strings.ToLower(outputText)
	patterns := []string{
		"no valid model configured",
		"no model configured",
		"model is not configured",
		"missing model",
	}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}
