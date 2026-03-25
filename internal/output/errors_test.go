package output

import "testing"

func TestDetectOpenCodePlaceholderPlugin(t *testing.T) {
	if !DetectOpenCodePlaceholderPlugin("Loaded legacy placeholder plugin package") {
		t.Fatal("expected placeholder plugin output to be detected")
	}
	if DetectOpenCodePlaceholderPlugin("Loaded auth plugin package") {
		t.Fatal("did not expect unrelated plugin output to match")
	}
}

func TestDetectMissingModel(t *testing.T) {
	if !DetectMissingModel("Error: no valid model configured") {
		t.Fatal("expected missing model output to be detected")
	}
	if DetectMissingModel("Error: permission denied") {
		t.Fatal("did not expect unrelated error to match")
	}
}
