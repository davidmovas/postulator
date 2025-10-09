package entities

import "time"

type AIProvider struct {
	ID        int64
	Name      string
	APIKey    string
	Model     string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
