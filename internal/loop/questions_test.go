package loop

import "testing"

func TestDetectQuestionFromColonLine(t *testing.T) {
	got := detectQuestion("tool question: What timeout should we use?")
	if got != "What timeout should we use?" {
		t.Fatalf("unexpected question: %q", got)
	}
}

func TestDetectQuestionFromJSONLine(t *testing.T) {
	got := detectQuestion(`{"tool":"question","question":"Which config path should I use?"}`)
	if got != "Which config path should I use?" {
		t.Fatalf("unexpected question: %q", got)
	}
}
