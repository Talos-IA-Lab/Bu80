package output

import (
	"reflect"
	"testing"
)

func TestParseToolCountsFromMixedOutput(t *testing.T) {
	input := "tool edit: changed file\n{\"tool\":\"question\",\"question\":\"Which path?\"}\ntool=edit\n"
	got := ParseToolCounts(input)
	if got["edit"] != 2 {
		t.Fatalf("expected edit count 2, got %v", got)
	}
	if got["question"] != 1 {
		t.Fatalf("expected question count 1, got %v", got)
	}
}

func TestParseToolsReturnsSortedUniqueNames(t *testing.T) {
	input := "tool write: a\ntool question: b\ntool write: c\n"
	got := ParseTools(input)
	want := []string{"question", "write"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected tools: got=%v want=%v", got, want)
	}
}
