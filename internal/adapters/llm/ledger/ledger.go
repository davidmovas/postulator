package ledger

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type callStore interface {
	Insert(ctx context.Context, call llm.Call) error
	SumByRun(ctx context.Context, runID string) (llm.Spend, error)
	SumByConversation(ctx context.Context, conversationID string) (llm.Spend, error)
	SumAll(ctx context.Context) (llm.Spend, error)
	List(ctx context.Context, q llm.CallQuery, page paging.Request) (paging.List[llm.Call], error)
}

type catalogReader interface {
	Lookup(ctx context.Context, ref llm.ModelRef) (llm.ModelInfo, error)
}

type publisher interface {
	Publish(eventType events.Type, payload any) error
}

type Ledger struct {
	next    port.Client
	store   callStore
	catalog catalogReader
	events  publisher
	clock   clock.Clock
}

func New(next port.Client, store callStore, catalog catalogReader, publisher publisher, clk clock.Clock) *Ledger {
	return &Ledger{next: next, store: store, catalog: catalog, events: publisher, clock: clk}
}

func (l *Ledger) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	started := time.Now()
	resp, err := l.next.Complete(ctx, req)
	if recordErr := l.record(ctx, req, resp.Usage, time.Since(started), err); recordErr != nil {
		return port.Response{}, recordErr
	}
	return resp, err
}

func (l *Ledger) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	started := time.Now()

	deltas, err := l.next.Stream(ctx, req)
	if err != nil {
		if recordErr := l.record(ctx, req, llm.Usage{}, time.Since(started), err); recordErr != nil {
			return nil, recordErr
		}
		return nil, err
	}

	out := make(chan port.Delta)
	go func() {
		defer close(out)

		var usage llm.Usage
		var failure error
		for delta := range deltas {
			if delta.Usage != nil {
				usage = *delta.Usage
			}
			if delta.Err != nil {
				failure = delta.Err
			}
			select {
			case out <- delta:
			case <-ctx.Done():
				return
			}
		}
		if recordErr := l.record(ctx, req, usage, time.Since(started), failure); recordErr != nil {
			select {
			case out <- port.Delta{Err: recordErr}:
			case <-ctx.Done():
			}
		}
	}()
	return out, nil
}

func (l *Ledger) record(ctx context.Context, req port.Request, usage llm.Usage, latency time.Duration, failure error) error {
	call := llm.Call{
		ID:             id.New(),
		RunID:          req.Meta.RunID,
		ItemID:         req.Meta.ItemID,
		Step:           req.Meta.Step,
		ConversationID: req.Meta.ConversationID,
		Ref:            req.Ref,
		Usage:          usage,
		USD:            l.cost(ctx, req.Ref, usage),
		Latency:        latency,
		Status:         llm.CallOK,
		CreatedAt:      l.clock.Now().UTC().Truncate(time.Second),
	}
	if failure != nil {
		call.Status = llm.CallError
		call.ErrorCode = errors.CodeOf(failure).String()
	}

	if err := l.store.Insert(ctx, call); err != nil {
		return err
	}
	if usage.Total == 0 {
		return nil
	}
	return l.events.Publish(events.LLMUsage, events.LLMUsagePayload{
		RunID:            call.RunID,
		ItemID:           call.ItemID,
		Provider:         call.Ref.Provider,
		Model:            call.Ref.Model,
		PromptTokens:     usage.Input,
		CompletionTokens: usage.Output,
		USD:              call.USD,
	})
}

func (l *Ledger) cost(ctx context.Context, ref llm.ModelRef, usage llm.Usage) float64 {
	if usage.Total == 0 {
		return 0
	}
	info, err := l.catalog.Lookup(ctx, ref)
	if err != nil {
		return 0
	}
	return llm.Cost(usage, info)
}

func (l *Ledger) SumByRun(ctx context.Context, runID string) (llm.Spend, error) {
	return l.store.SumByRun(ctx, runID)
}

func (l *Ledger) SumByConversation(ctx context.Context, conversationID string) (llm.Spend, error) {
	return l.store.SumByConversation(ctx, conversationID)
}

func (l *Ledger) SumAll(ctx context.Context) (llm.Spend, error) {
	return l.store.SumAll(ctx)
}

func (l *Ledger) List(ctx context.Context, q llm.CallQuery, page paging.Request) (paging.List[llm.Call], error) {
	return l.store.List(ctx, q, page)
}
