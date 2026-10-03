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
	Aggregate(ctx context.Context, q llm.SpendQuery) ([]llm.SpendSlice, error)
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

type outcome struct {
	failure  error
	usage    llm.Usage
	tier     llm.ServiceTier
	latency  time.Duration
	finished bool
}

func (o *outcome) absorb(delta port.Delta) {
	if delta.Usage != nil {
		o.usage = *delta.Usage
	}
	if delta.Tier != "" {
		o.tier = delta.Tier
	}
	if delta.Err != nil {
		o.failure = delta.Err
	}
	if delta.Done {
		o.finished = true
	}
}

func (o *outcome) cancelled(ctx context.Context) {
	o.failure = errors.New(errors.Cancelled, "the caller cancelled the stream before it ended").
		WithInternal(context.Cause(ctx))
}

func (l *Ledger) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	started := time.Now()
	resp, err := l.next.Complete(ctx, req)
	done := outcome{usage: resp.Usage, tier: resp.Tier, latency: time.Since(started), failure: err}
	if recordErr := l.record(ctx, req, done); recordErr != nil {
		return port.Response{}, recordErr
	}
	return resp, err
}

func (l *Ledger) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	started := time.Now()

	deltas, err := l.next.Stream(ctx, req)
	if err != nil {
		if recordErr := l.record(ctx, req, outcome{latency: time.Since(started), failure: err}); recordErr != nil {
			return nil, recordErr
		}
		return nil, err
	}

	out := make(chan port.Delta)
	go func() {
		defer close(out)

		done := relay(ctx, deltas, out)
		done.latency = time.Since(started)
		if recordErr := l.record(ctx, req, done); recordErr != nil {
			select {
			case out <- port.Delta{Err: recordErr}:
			case <-ctx.Done():
			}
		}
	}()
	return out, nil
}

func relay(ctx context.Context, deltas <-chan port.Delta, out chan<- port.Delta) outcome {
	var done outcome
	for delta := range deltas {
		done.absorb(delta)
		select {
		case out <- delta:
		case <-ctx.Done():
			for rest := range deltas {
				done.absorb(rest)
			}
			done.cancelled(ctx)
			return done
		}
	}
	if done.failure == nil && !done.finished && ctx.Err() != nil {
		done.cancelled(ctx)
	}
	return done
}

func (l *Ledger) record(ctx context.Context, req port.Request, done outcome) error {
	tier := served(done.tier)
	call := llm.Call{
		ID:             id.New(),
		RunID:          req.Meta.RunID,
		ItemID:         req.Meta.ItemID,
		Step:           req.Meta.Step,
		ConversationID: req.Meta.ConversationID,
		Ref:            req.Ref,
		Usage:          done.usage,
		USD:            l.cost(ctx, req.Ref, done.usage, tier),
		Latency:        done.latency,
		Status:         llm.CallOK,
		Tier:           tier,
		CreatedAt:      l.clock.Now().UTC().Truncate(time.Second),
	}
	if done.failure != nil {
		call.Status = llm.CallError
		call.ErrorCode = errors.CodeOf(done.failure).String()
	}

	if err := l.store.Insert(context.WithoutCancel(ctx), call); err != nil {
		return err
	}
	if call.Usage.Total == 0 {
		return nil
	}
	return l.events.Publish(events.LLMUsage, events.LLMUsagePayload{
		RunID:            call.RunID,
		ItemID:           call.ItemID,
		Provider:         call.Ref.Provider,
		Model:            call.Ref.Model,
		Tier:             string(call.Tier),
		PromptTokens:     call.Usage.Input,
		CompletionTokens: call.Usage.Output,
		ReasoningTokens:  call.Usage.Reasoning,
		CacheWriteTokens: call.Usage.CacheWrite,
		USD:              call.USD,
	})
}

func served(tier llm.ServiceTier) llm.ServiceTier {
	if tier.Valid() {
		return tier
	}
	return llm.TierDefault
}

func (l *Ledger) cost(ctx context.Context, ref llm.ModelRef, usage llm.Usage, tier llm.ServiceTier) float64 {
	if usage.Total == 0 {
		return 0
	}
	info, err := l.catalog.Lookup(context.WithoutCancel(ctx), ref)
	if err != nil {
		return 0
	}
	return llm.Cost(usage, info, tier)
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

func (l *Ledger) Aggregate(ctx context.Context, q llm.SpendQuery) ([]llm.SpendSlice, error) {
	return l.store.Aggregate(ctx, q)
}
