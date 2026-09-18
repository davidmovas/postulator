package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gollem-dev/gollem"

	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type historyStore interface {
	Load(ctx context.Context, conversationID string) ([]byte, int, error)
	Save(ctx context.Context, conversationID string, body []byte, version int, at time.Time) error
}

type bounded struct {
	store  historyStore
	clock  clock.Clock
	budget int
}

func (b bounded) Load(ctx context.Context, conversationID string) (*gollem.History, error) {
	body, _, err := b.store.Load(ctx, conversationID)
	if errors.IsCode(err, errors.NotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var history gollem.History
	if unmarshalErr := json.Unmarshal(body, &history); unmarshalErr != nil {
		return nil, nil
	}
	return trim(&history, b.budget), nil
}

func (b bounded) Save(ctx context.Context, conversationID string, history *gollem.History) error {
	if history == nil {
		return nil
	}

	trimmed := trim(history, b.budget)
	body, err := json.Marshal(trimmed)
	if err != nil {
		return errors.Wrap(err, errors.Internal, "encode the conversation history")
	}
	return b.store.Save(ctx, conversationID, body, trimmed.Version, b.clock.Now().UTC().Truncate(time.Second))
}

func trim(history *gollem.History, budget int) *gollem.History {
	if history == nil || len(history.Messages) == 0 || budget <= 0 || size(history) <= budget {
		return history
	}

	starts := turnStarts(history.Messages)
	if len(starts) == 0 {
		return &gollem.History{LLType: history.LLType, Version: history.Version}
	}

	suffixes := suffixSizes(history.Messages)
	newest := starts[len(starts)-1]

	for _, start := range starts[:len(starts)-1] {
		if suffixes[start] > budget {
			continue
		}
		candidate := from(history, start)
		if size(candidate) <= budget {
			return candidate
		}
	}
	return from(history, newest)
}

func from(history *gollem.History, start int) *gollem.History {
	return &gollem.History{LLType: history.LLType, Version: history.Version, Messages: history.Messages[start:]}
}

func size(history *gollem.History) int {
	encoded, err := json.Marshal(history)
	if err != nil {
		return 0
	}
	return len(encoded)
}

func suffixSizes(messages []gollem.Message) []int {
	sizes := make([]int, len(messages)+1)
	for i := len(messages) - 1; i >= 0; i-- {
		encoded, err := json.Marshal(messages[i])
		if err != nil {
			sizes[i] = sizes[i+1]
			continue
		}
		sizes[i] = sizes[i+1] + len(encoded)
	}
	return sizes
}

func turnStarts(messages []gollem.Message) []int {
	starts := make([]int, 0, len(messages))
	for i := range messages {
		if messages[i].Role != gollem.RoleUser || carriesToolResponse(messages[i]) {
			continue
		}
		starts = append(starts, i)
	}
	return starts
}

func carriesToolResponse(message gollem.Message) bool {
	for _, content := range message.Contents {
		if content.Type == gollem.MessageContentTypeToolResponse {
			return true
		}
	}
	return false
}
