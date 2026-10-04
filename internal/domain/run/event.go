package run

import (
	"encoding/json"
	"strings"
	"time"
)

type Event struct {
	RunID   string
	Seq     int64
	Type    string
	At      time.Time
	Payload json.RawMessage
}

func NewEvent(e Event) (Event, error) {
	e.Type = strings.TrimSpace(e.Type)
	switch {
	case e.RunID == "":
		return Event{}, invalid("event run id must not be empty", "runId")
	case e.Seq <= 0:
		return Event{}, invalid("event sequence must be positive", "seq")
	case e.Type == "":
		return Event{}, invalid("event type must not be empty", "type")
	case e.At.IsZero():
		return Event{}, invalid("event timestamp must not be empty", "at")
	}
	if len(e.Payload) == 0 {
		e.Payload = json.RawMessage("{}")
	}
	return e, nil
}
