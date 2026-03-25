package output

import "testing"

func TestDetectPromiseWhenFinalNonEmptyLineMatches(t *testing.T) {
	output := "working\n<promise> COMPLETE </promise>\n\n"
	if !DetectPromise(output, "COMPLETE") {
		t.Fatal("expected promise to be detected")
	}
}

func TestDetectPromiseIgnoresNonTerminalPromise(t *testing.T) {
	output := "<promise>COMPLETE</promise>\nstill working"
	if DetectPromise(output, "COMPLETE") {
		t.Fatal("did not expect non-terminal promise to count")
	}
}

func TestDetectPromiseRejectsWrongPromise(t *testing.T) {
	output := "done\n<promise>ABORT</promise>"
	if DetectPromise(output, "COMPLETE") {
		t.Fatal("did not expect wrong promise to count")
	}
}

func TestLastNonEmptyLineStripsANSIAndTrailingEmptyLines(t *testing.T) {
	output := "\x1b[32mnoise\x1b[0m\n\n  <promise>COMPLETE</promise>  \n\n"
	got := LastNonEmptyLine(output)
	want := "<promise>COMPLETE</promise>"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
