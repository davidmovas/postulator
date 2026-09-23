package agent

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/gollem-dev/gollem"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
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
	cap    int
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
	return trim(shorten(&history, b.cap), b.budget), nil
}

func (b bounded) Save(ctx context.Context, conversationID string, history *gollem.History) error {
	if history == nil {
		return nil
	}

	trimmed := trim(shorten(history, b.cap), b.budget)
	body, err := json.Marshal(trimmed)
	if err != nil {
		return errors.Wrap(err, errors.Internal, "encode the conversation history")
	}
	return b.store.Save(ctx, conversationID, body, trimmed.Version, b.clock.Now().UTC().Truncate(time.Second))
}

func shorten(history *gollem.History, limit int) *gollem.History {
	if history == nil || limit <= 0 || len(history.Messages) == 0 {
		return history
	}

	out := *history
	out.Messages = slices.Clone(history.Messages)
	for i := range out.Messages {
		out.Messages[i].Contents = shortenContents(out.Messages[i].Contents, limit)
	}
	return &out
}

func shortenContents(contents []gollem.MessageContent, limit int) []gollem.MessageContent {
	out := contents
	for i := range contents {
		if contents[i].Type != gollem.MessageContentTypeToolResponse {
			continue
		}
		held, err := contents[i].GetToolResponseContent()
		if err != nil || held == nil {
			continue
		}
		capped, cut := compact(held.Response, limit)
		if !cut {
			continue
		}
		content, err := gollem.NewToolResponseContent(held.ToolCallID, held.Name, capped, held.IsError)
		if err != nil {
			continue
		}
		content.Meta = contents[i].Meta
		if &out[0] == &contents[0] {
			out = slices.Clone(contents)
		}
		out[i] = content
	}
	return out
}

func compact(response map[string]any, limit int) (map[string]any, bool) {
	fenced, marked := response[agentapp.UntrustedMarker].(bool)
	held, wrapped := response[agentapp.UntrustedData].(map[string]any)
	if !marked || !fenced || !wrapped {
		return agentapp.Cap(response, limit)
	}

	capped, cut := agentapp.Cap(held, limit)
	if !cut {
		return response, false
	}
	return agentapp.Fence(capped), true
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
