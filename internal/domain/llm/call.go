package llm

import "time"

type CallStatus string

const (
	CallOK    CallStatus = "ok"
	CallError CallStatus = "error"
)

func (s CallStatus) Valid() bool {
	return s == CallOK || s == CallError
}

type Call struct {
	CreatedAt      time.Time
	ID             string
	RunID          string
	ItemID         string
	Step           string
	ConversationID string
	ErrorCode      string
	Ref            ModelRef
	Usage          Usage
	USD            float64
	Latency        time.Duration
	Status         CallStatus
}

func (c Call) Validate() error {
	if c.ID == "" {
		return invalid("a call needs an identifier", "id")
	}
	if !c.Ref.Valid() {
		return invalid("a call must name a provider and a model", "ref")
	}
	if !c.Status.Valid() {
		return invalid("a call status must be ok or error", "status")
	}
	return nil
}

type CallQuery struct {
	RunID          string
	ConversationID string
	Desc           bool
}

type Spend struct {
	Usage Usage
	USD   float64
	Calls int
}
