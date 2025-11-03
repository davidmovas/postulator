package entities

import "time"

type Prompt struct {
	ID           int64
	Name         string
	SystemPrompt string
	UserPrompt   string
	Placeholders []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
