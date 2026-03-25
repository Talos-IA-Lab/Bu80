package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"bu80/internal/history"
)

type QuestionsFile struct {
	Records []QuestionRecord `json:"records"`
}

func EnsureDir() error {
	return os.MkdirAll(DirName, 0o755)
}

func LoadLoopState() (*LoopState, error) {
	var loop LoopState
	ok, err := readJSONFile(LoopStateFile, &loop)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return &loop, nil
}

func SaveLoopState(loop LoopState) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return writeJSONFile(LoopStateFile, loop)
}

func ClearLoopState() error {
	if err := os.Remove(LoopStateFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func LoadHistory() (*history.History, error) {
	var records history.History
	ok, err := readJSONFile(HistoryFile, &records)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return &records, nil
}

func SaveHistory(records history.History) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return writeJSONFile(HistoryFile, records)
}

func ClearHistory() error {
	if err := os.Remove(HistoryFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func LoadQuestions() (*QuestionsFile, error) {
	var questions QuestionsFile
	ok, err := readJSONFile(QuestionsFilePath, &questions)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return &questions, nil
}

func SaveQuestions(questions QuestionsFile) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return writeJSONFile(QuestionsFilePath, questions)
}

func ClearQuestions() error {
	if err := os.Remove(QuestionsFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func ReadContext() (string, error) {
	data, err := os.ReadFile(ContextFile)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func WriteContext(content string) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return os.WriteFile(ContextFile, []byte(content), 0o644)
}

func ClearContext() error {
	if err := os.Remove(ContextFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func ReadTasks() (string, error) {
	data, err := os.ReadFile(TasksFile)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func WriteTasks(content string) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return os.WriteFile(TasksFile, []byte(content), 0o644)
}

func readJSONFile(path string, dest any) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false, err
	}
	return true, nil
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
