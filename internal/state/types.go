package state

import "time"

const (
	DirName            = ".bu80"
	LoopStateFile      = ".bu80/bu80-loop.state.json"
	HistoryFile        = ".bu80/bu80-history.json"
	ContextFile        = ".bu80/bu80-context.md"
	TasksFile          = ".bu80/bu80-tasks.md"
	QuestionsFilePath  = ".bu80/bu80-questions.json"
	OpenCodeConfigFile = ".bu80/bu80-opencode.config.json"
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
