package state

import (
	"fmt"
	"strings"
	"time"
)

func AddPendingQuestion(question string, now time.Time) error {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil
	}
	questions, err := LoadQuestions()
	if err != nil {
		return err
	}
	if questions == nil {
		questions = &QuestionsFile{}
	}
	for _, record := range questions.Records {
		if record.Pending && strings.EqualFold(strings.TrimSpace(record.Question), question) {
			return nil
		}
	}
	questions.Records = append(questions.Records, QuestionRecord{
		Question:  question,
		CreatedAt: now,
		Pending:   true,
	})
	return SaveQuestions(*questions)
}

func SaveAnswer(question string, answer string, now time.Time) error {
	questions, err := LoadQuestions()
	if err != nil {
		return err
	}
	if questions == nil {
		questions = &QuestionsFile{}
	}
	question = strings.TrimSpace(question)
	answer = strings.TrimSpace(answer)
	for i := range questions.Records {
		if strings.EqualFold(strings.TrimSpace(questions.Records[i].Question), question) {
			questions.Records[i].Answer = answer
			questions.Records[i].Pending = false
			questions.Records[i].AnsweredAt = now
			return SaveQuestions(*questions)
		}
	}
	questions.Records = append(questions.Records, QuestionRecord{
		Question:   question,
		Answer:     answer,
		CreatedAt:  now,
		AnsweredAt: now,
		Pending:    false,
	})
	return SaveQuestions(*questions)
}

func ConsumeAnsweredQuestions() ([]QuestionRecord, error) {
	questions, err := LoadQuestions()
	if err != nil {
		return nil, err
	}
	if questions == nil {
		return nil, nil
	}
	answered := make([]QuestionRecord, 0, len(questions.Records))
	remaining := make([]QuestionRecord, 0, len(questions.Records))
	for _, record := range questions.Records {
		if !record.Pending && strings.TrimSpace(record.Answer) != "" {
			answered = append(answered, record)
			continue
		}
		remaining = append(remaining, record)
	}
	if len(answered) == 0 {
		return nil, nil
	}
	if len(remaining) == 0 {
		if err := ClearQuestions(); err != nil {
			return nil, err
		}
	} else {
		questions.Records = remaining
		if err := SaveQuestions(*questions); err != nil {
			return nil, err
		}
	}
	return answered, nil
}

func FormatAnswersBlock(records []QuestionRecord) string {
	if len(records) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Answered questions:\n")
	for _, record := range records {
		b.WriteString(fmt.Sprintf("Q: %s\n", strings.TrimSpace(record.Question)))
		b.WriteString(fmt.Sprintf("A: %s\n", strings.TrimSpace(record.Answer)))
	}
	return strings.TrimSpace(b.String())
}

func MergeContext(existing string, addition string) string {
	existing = strings.TrimSpace(existing)
	addition = strings.TrimSpace(addition)
	if addition == "" {
		return existing
	}
	if existing == "" {
		return addition
	}
	if strings.Contains(existing, addition) {
		return existing
	}
	return existing + "\n\n" + addition
}
