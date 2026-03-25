package tasks

import "testing"

func TestParseRecognizesTopLevelTasksAndSubtasks(t *testing.T) {
	input := "- [ ] one\n  - [x] child\n- [/] two"
	got := Parse(input)

	if len(got) != 2 {
		t.Fatalf("expected 2 top-level tasks, got %d", len(got))
	}
	if got[0].Status != StatusTodo {
		t.Fatalf("expected first task to be todo, got %s", got[0].Status)
	}
	if len(got[0].Subtasks) != 1 || got[0].Subtasks[0].Status != StatusComplete {
		t.Fatal("expected parsed complete subtask")
	}
	if got[1].Status != StatusInProgress {
		t.Fatalf("expected second task to be in-progress, got %s", got[1].Status)
	}
}

func TestAllCompleteRequiresAtLeastOneCheckbox(t *testing.T) {
	if AllComplete("# Bu80 Tasks\n") {
		t.Fatal("expected empty task list to be incomplete")
	}
}

func TestAllCompleteRejectsTodoOrInProgressTasks(t *testing.T) {
	input := "- [x] done\n  - [ ] child"
	if AllComplete(input) {
		t.Fatal("expected incomplete subtasks to block completion")
	}
}

func TestAllCompleteAcceptsAllCompletedTasks(t *testing.T) {
	input := "- [x] done\n  - [X] child\n- [x] second"
	if !AllComplete(input) {
		t.Fatal("expected all completed tasks to pass")
	}
}
