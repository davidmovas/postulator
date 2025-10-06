package prompt

import "time"

type Prompt struct {
	ID           int64
	Name         string
	SystemPrompt string
	UserPrompt   string
	Placeholders []string // ["title", "words", "category"]
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
