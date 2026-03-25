package history

import "time"

type IterationRecord struct {
	Iteration          int       `json:"iteration"`
	StartedAt          time.Time `json:"startedAt"`
	EndedAt            time.Time `json:"endedAt"`
	DurationMs         int64     `json:"durationMs"`
	Agent              string    `json:"agent"`
	Model              string    `json:"model,omitempty"`
	ToolsUsed          []string  `json:"toolsUsed,omitempty"`
	FilesModified      []string  `json:"filesModified,omitempty"`
	ExitCode           int       `json:"exitCode"`
	CompletionDetected bool      `json:"completionDetected"`
	Errors             []string  `json:"errors,omitempty"`
}

type StruggleIndicators struct {
	RepeatedErrors  int `json:"repeatedErrors"`
	NoProgressIters int `json:"noProgressIterations"`
	ShortIterations int `json:"shortIterations"`
}

type History struct {
	Iterations         []IterationRecord  `json:"iterations"`
	TotalDurationMs    int64              `json:"totalDurationMs"`
	StruggleIndicators StruggleIndicators `json:"struggleIndicators"`
}
