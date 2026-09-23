package events

import "github.com/davidmovas/postulator/internal/kernel/dto"

type Envelope struct {
	Type    Type     `json:"type"`
	Seq     int64    `json:"seq"`
	RunID   *string  `json:"runId,omitempty"`
	At      dto.Time `json:"at"`
	Payload any      `json:"payload"`
}
