package contextcmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"bu80/internal/state"
)

func Add(text string, now time.Time) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("context text is required")
	}

	existing, err := state.ReadContext()
	if err != nil {
		return err
	}
	section := fmt.Sprintf("## %s\n\n%s\n", now.Format(time.RFC3339), text)
	if strings.TrimSpace(existing) == "" {
		return state.WriteContext(section)
	}
	combined := strings.TrimRight(existing, "\n") + "\n\n" + section
	return state.WriteContext(combined)
}

func Clear(w io.Writer) error {
	_, err := os.Stat(state.ContextFile)
	if errors.Is(err, os.ErrNotExist) {
		_, writeErr := fmt.Fprintln(w, "No context to clear.")
		return writeErr
	}
	if err != nil {
		return err
	}
	return state.ClearContext()
}
