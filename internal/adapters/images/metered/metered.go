package metered

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type generator interface {
	Generate(ctx context.Context, prompt images.Prompt) (images.Image, error)
}

type callStore interface {
	Insert(ctx context.Context, call llm.Call) error
}

type catalogReader interface {
	Lookup(ctx context.Context, ref llm.ModelRef) (llm.ModelInfo, error)
}

type publisher interface {
	Publish(eventType events.Type, payload any) error
}

type Images struct {
	next    generator
	store   callStore
	catalog catalogReader
	events  publisher
	clock   clock.Clock
	ref     llm.ModelRef
}

func New(next generator, ref llm.ModelRef, store callStore, catalog catalogReader, publisher publisher,
	clk clock.Clock) *Images {
	return &Images{next: next, ref: ref, store: store, catalog: catalog, events: publisher, clock: clk}
}

func (m *Images) Generate(ctx context.Context, prompt images.Prompt) (images.Image, error) {
	started := time.Now()
	image, err := m.next.Generate(ctx, prompt)

	call := llm.Call{
		ID:        id.New(),
		RunID:     prompt.RunID,
		ItemID:    prompt.ItemID,
		Step:      prompt.Step,
		Ref:       m.ref,
		Usage:     image.Usage,
		USD:       m.cost(ctx, image.Usage),
		Latency:   time.Since(started),
		Status:    llm.CallOK,
		CreatedAt: m.clock.Now().UTC().Truncate(time.Second),
	}
	if err != nil {
		call.Status = llm.CallError
		call.ErrorCode = errors.CodeOf(err).String()
	}

	if insertErr := m.store.Insert(context.WithoutCancel(ctx), call); insertErr != nil {
		return images.Image{}, insertErr
	}
	if err != nil {
		return images.Image{}, err
	}
	if call.Usage.Total > 0 {
		if publishErr := m.events.Publish(events.LLMUsage, events.LLMUsagePayload{
			RunID: call.RunID, ItemID: call.ItemID, Provider: m.ref.Provider, Model: m.ref.Model,
			PromptTokens: call.Usage.Input, CompletionTokens: call.Usage.Output, USD: call.USD,
		}); publishErr != nil {
			return images.Image{}, publishErr
		}
	}
	return image, nil
}

func (m *Images) cost(ctx context.Context, usage llm.Usage) float64 {
	if usage.Total == 0 {
		return 0
	}
	info, err := m.catalog.Lookup(ctx, m.ref)
	if err != nil {
		return 0
	}
	return llm.Cost(usage, info)
}
