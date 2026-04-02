package state

import "time"

const (
	DirName            = ".loop"
	LoopStateFile      = ".loop/state.json"
	HistoryFile        = ".loop/history.json"
	ContextFile        = ".loop/context.md"
	TasksFile          = ".loop/tasks.md"
	QuestionsFilePath  = ".loop/questions.json"
	OpenCodeConfigFile = ".loop/opencode.config.json"
	DefaultConfigPath  = "~/.config/bu80/agents.json"
	DefaultTaskPromise = "READY_FOR_NEXT_TASK"
	DefaultDonePromise = "COMPLETE"
)

type LoopState struct {
	Active            bool      `json:"active"`
	Iteration         int       `json:"iteration"`
	MinIterations     int       `json:"minIterations"`
	MaxIterations     int       `json:"maxIterations"`
	CompletionPromise string    `json:"completionPromise"`
	AbortPromise      string    `json:"abortPromise,omitempty"`
	TasksMode         bool      `json:"tasksMode"`
	TaskPromise       string    `json:"taskPromise"`
	Prompt            string    `json:"prompt"`
	PromptTemplate    string    `json:"promptTemplate,omitempty"`
	StartedAt         time.Time `json:"startedAt"`
	Model             string    `json:"model,omitempty"`
	Agent             string    `json:"agent"`
	Rotation          []string  `json:"rotation,omitempty"`
	RotationIndex     int       `json:"rotationIndex"`
}

type QuestionRecord struct {
	Question   string    `json:"question"`
	Answer     string    `json:"answer,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	AnsweredAt time.Time `json:"answeredAt,omitempty"`
	Pending    bool      `json:"pending"`
}
