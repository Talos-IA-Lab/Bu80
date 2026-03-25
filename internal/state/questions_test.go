package state

import (
	"os"
	"testing"
	"time"
)

func TestAddPendingQuestionDeduplicatesPending(t *testing.T) {
	inTempRepoState(t)
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	if err := AddPendingQuestion("What should the timeout be?", now); err != nil {
		t.Fatalf("add pending question: %v", err)
	}
	if err := AddPendingQuestion("What should the timeout be?", now.Add(time.Minute)); err != nil {
		t.Fatalf("add duplicate pending question: %v", err)
	}
	questions, err := LoadQuestions()
	if err != nil {
		t.Fatalf("load questions: %v", err)
	}
	if questions == nil || len(questions.Records) != 1 {
		t.Fatalf("expected exactly one pending question, got %+v", questions)
	}
}

func TestConsumeAnsweredQuestionsRemovesAnsweredRecords(t *testing.T) {
	inTempRepoState(t)
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	if err := SaveAnswer("Question one?", "Answer one", now); err != nil {
		t.Fatalf("save answer: %v", err)
	}
	if err := AddPendingQuestion("Question two?", now); err != nil {
		t.Fatalf("add pending question: %v", err)
	}
	answered, err := ConsumeAnsweredQuestions()
	if err != nil {
		t.Fatalf("consume answered questions: %v", err)
	}
	if len(answered) != 1 || answered[0].Question != "Question one?" {
		t.Fatalf("unexpected answered records: %+v", answered)
	}
	questions, err := LoadQuestions()
	if err != nil {
		t.Fatalf("load questions: %v", err)
	}
	if questions == nil || len(questions.Records) != 1 || !questions.Records[0].Pending {
		t.Fatalf("expected pending record to remain, got %+v", questions)
	}
}

func TestMergeContextAppendsOnlyOnce(t *testing.T) {
	base := "Existing context"
	addition := "Answered questions:\nQ: One\nA: Two"
	got := MergeContext(base, addition)
	if got != base+"\n\n"+addition {
		t.Fatalf("unexpected merged context: %q", got)
	}
	got = MergeContext(got, addition)
	if got != base+"\n\n"+addition {
		t.Fatalf("expected duplicate addition to be ignored, got %q", got)
	}
}

func inTempRepoState(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}
