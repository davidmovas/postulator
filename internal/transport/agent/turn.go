package agent

import (
	"context"
	stderrors "errors"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/llm/retry"
	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/llm"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type turn struct {
	spec    agentapp.RunSpec
	client  streamer
	catalog catalogReader
	memory  memory
	guard   *guard
	usage   *tally
	spoken  *answer
	asked   llm.Request
	opening []llm.Message
}

type heard struct {
	text    string
	calls   []llm.ToolCall
	carried []llm.Message
}

func (h *heard) carry(delta llm.Delta) {
	if delta.Search != nil {
		search := *delta.Search
		h.carried = append(h.carried, llm.Message{Role: llm.RoleAssistant, Search: &search})
	}
	if delta.Call != nil {
		call := *delta.Call
		h.calls = append(h.calls, call)
		h.carried = append(h.carried, llm.Message{Role: llm.RoleAssistant, Call: &call})
	}
}

func (h heard) said() []llm.Message {
	out := make([]llm.Message, 0, len(h.carried)+1)
	if h.text != "" {
		out = append(out, llm.Message{Role: llm.RoleAssistant, Text: h.text})
	}
	return append(out, h.carried...)
}

func (t *turn) run(ctx context.Context) error {
	past, err := t.memory.load(ctx)
	if err != nil {
		return err
	}

	current := slices.Clone(t.opening)
	for round := 1; round <= t.spec.LoopLimit; round++ {
		answered, askErr := t.ask(ctx, round, joined(past, current))
		if askErr != nil {
			return askErr
		}
		current = append(current, answered.said()...)
		if len(answered.calls) == 0 {
			return t.memory.save(ctx, joined(past, current))
		}

		for _, call := range answered.calls {
			current = append(current, t.guard.dispatch(ctx, call))
		}
		if saveErr := t.memory.save(ctx, joined(past, current)); saveErr != nil {
			return saveErr
		}
		if ctx.Err() != nil {
			return stopped(ctx, ctx.Err())
		}
	}
	return errors.New(errors.BudgetExceeded, "the agent used up its tool call budget for this turn").
		WithDetail("loopLimit", t.spec.LoopLimit)
}

func joined(past, current []llm.Message) []llm.Message {
	return append(slices.Clip(past), current...)
}

func (t *turn) ask(ctx context.Context, round int, messages []llm.Message) (heard, error) {
	req := t.asked
	req.Messages = messages

	deltas, err := t.client.Stream(retry.Observing(ctx, t.waited(ctx)), req)
	if err != nil {
		return heard{}, convert(ctx, err)
	}

	var (
		answered heard
		said     strings.Builder
		done     *llm.Delta
		failure  error
		muted    bool
	)
	for delta := range deltas {
		if delta.Err != nil && failure == nil {
			failure = delta.Err
		}
		if delta.Text != "" {
			said.WriteString(delta.Text)
			if !muted {
				muted = !t.show(ctx, delta.Text)
			}
		}
		answered.carry(delta)
		if delta.Done {
			ended := delta
			done = &ended
		}
	}

	switch {
	case failure != nil:
		return heard{}, convert(ctx, failure)
	case done == nil:
		return heard{}, unfinished(ctx)
	}

	answered.text = said.String()
	t.spoken.spoke(answered.text)
	t.bill(ctx, round, *done)
	return answered, nil
}

func (t *turn) show(ctx context.Context, text string) bool {
	if t.spec.Stream == nil {
		return true
	}
	if err := t.spec.Stream.Delta(ctx, t.usage.next(), text); err != nil {
		t.guard.note(err)
		return false
	}
	return true
}

func (t *turn) waited(ctx context.Context) func(retry.Wait) {
	return func(held retry.Wait) {
		if t.spec.Stream == nil {
			return
		}
		t.guard.note(t.spec.Stream.Waiting(ctx, agentapp.Wait{
			Reason: errors.CodeOf(held.Cause).String(), Attempt: held.Attempt, AfterMS: held.Delay.Milliseconds(),
		}))
	}
}

func (t *turn) bill(ctx context.Context, round int, done llm.Delta) {
	var used domainllm.Usage
	if done.Usage != nil {
		used = *done.Usage
	}
	cost := t.cost(ctx, used, served(done.Tier))
	t.usage.add(used, cost)

	if t.spec.Stream == nil {
		return
	}
	t.guard.note(t.spec.Stream.Spent(ctx, agentapp.RoundUsage{
		Provider: t.spec.Ref.Provider, Model: t.spec.Ref.Model, Round: round,
		Input: used.Input, CachedInput: used.CachedInput, Output: used.Output, USD: cost,
	}))
}

func (t *turn) cost(ctx context.Context, used domainllm.Usage, tier domainllm.ServiceTier) float64 {
	if used.Total == 0 || t.catalog == nil {
		return 0
	}
	info, err := t.catalog.Lookup(context.WithoutCancel(ctx), t.spec.Ref)
	if err != nil {
		return 0
	}
	return domainllm.Cost(used, info, tier)
}

func served(tier domainllm.ServiceTier) domainllm.ServiceTier {
	if tier.Valid() {
		return tier
	}
	return domainllm.TierDefault
}

func convert(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		return stopped(ctx, err)
	case stderrors.Is(err, context.Canceled), stderrors.Is(err, context.DeadlineExceeded):
		return stopped(ctx, err)
	case errors.CodeOf(err) != errors.Internal:
		return err
	default:
		return errors.Wrap(err, errors.External, "the model could not answer")
	}
}

func stopped(ctx context.Context, err error) error {
	cause := err
	if reason := context.Cause(ctx); reason != nil {
		cause = reason
	}
	return errors.Wrap(cause, errors.Cancelled, "the agent turn was stopped")
}

func unfinished(ctx context.Context) error {
	if ctx.Err() != nil {
		return stopped(ctx, ctx.Err())
	}
	return errors.New(errors.External, "the model stopped streaming before its answer was complete")
}
