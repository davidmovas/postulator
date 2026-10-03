package agent

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	historyFormat  = "responses/1"
	historyVersion = 1
)

type historyStore interface {
	Load(ctx context.Context, conversationID string) ([]byte, int, error)
	Save(ctx context.Context, conversationID string, body []byte, version int, at time.Time) error
}

type history struct {
	Format string        `json:"format"`
	Items  []llm.Message `json:"items"`
}

type memory struct {
	store          historyStore
	clock          clock.Clock
	conversationID string
	budget         int
	cap            int
}

func (m memory) kept() bool {
	return m.store != nil && m.conversationID != ""
}

func (m memory) load(ctx context.Context) ([]llm.Message, error) {
	if !m.kept() {
		return nil, nil
	}

	body, _, err := m.store.Load(ctx, m.conversationID)
	if errors.IsCode(err, errors.NotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var held history
	if json.Unmarshal(body, &held) != nil || held.Format != historyFormat {
		return nil, nil
	}
	return trim(shorten(held.Items, m.cap), m.budget), nil
}

func (m memory) save(ctx context.Context, items []llm.Message) error {
	if !m.kept() {
		return nil
	}

	kept := trim(shorten(masked(items), m.cap), m.budget)
	body, err := json.Marshal(history{Format: historyFormat, Items: kept})
	if err != nil {
		return errors.Wrap(err, errors.Internal, "encode the conversation history")
	}
	return m.store.Save(context.WithoutCancel(ctx), m.conversationID, body, historyVersion,
		m.clock.Now().UTC().Truncate(time.Second))
}

func masked(items []llm.Message) []llm.Message {
	out := slices.Clone(items)
	for i := range out {
		if held := out[i].Call; held != nil {
			call := *held
			call.Args = tools.Redact(call.Args)
			out[i].Call = &call
		}
	}
	return out
}

func shorten(items []llm.Message, limit int) []llm.Message {
	if limit <= 0 || len(items) == 0 {
		return items
	}

	out := items
	for i := range items {
		held := items[i].Result
		if held == nil {
			continue
		}
		output, cut := compact(held.Output, limit)
		if !cut {
			continue
		}
		if &out[0] == &items[0] {
			out = slices.Clone(items)
		}
		out[i].Result = &llm.ToolResult{CallID: held.CallID, Output: output}
	}
	return out
}

func compact(output json.RawMessage, limit int) (json.RawMessage, bool) {
	value, err := decoded(output)
	if err != nil {
		return output, false
	}
	object, ok := value.(map[string]any)
	if !ok {
		return output, false
	}

	fenced, marked := object[agentapp.UntrustedMarker].(bool)
	data, wrapped := object[agentapp.UntrustedData].(map[string]any)
	if !marked || !fenced || !wrapped {
		capped, cut := agentapp.Cap(object, limit)
		return reencoded(output, capped, cut)
	}

	capped, cut := agentapp.Cap(data, limit)
	return reencoded(output, agentapp.Fence(capped), cut)
}

func reencoded(original json.RawMessage, value map[string]any, cut bool) (json.RawMessage, bool) {
	if !cut {
		return original, false
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return original, false
	}
	return encoded, true
}

func trim(items []llm.Message, budget int) []llm.Message {
	if len(items) == 0 || budget <= 0 {
		return items
	}

	suffixes := suffixSizes(items)
	if suffixes[0] <= budget {
		return items
	}

	starts := turnStarts(items)
	if len(starts) == 0 {
		return nil
	}
	for _, start := range starts[:len(starts)-1] {
		if suffixes[start] <= budget {
			return items[start:]
		}
	}
	return items[starts[len(starts)-1]:]
}

func suffixSizes(items []llm.Message) []int {
	sizes := make([]int, len(items)+1)
	for i := len(items) - 1; i >= 0; i-- {
		encoded, err := json.Marshal(items[i])
		if err != nil {
			sizes[i] = sizes[i+1]
			continue
		}
		sizes[i] = sizes[i+1] + len(encoded)
	}
	return sizes
}

func turnStarts(items []llm.Message) []int {
	starts := make([]int, 0, len(items))
	for i := range items {
		if items[i].Role == llm.RoleDeveloper {
			starts = append(starts, i)
		}
	}
	return starts
}
