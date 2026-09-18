package limiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/llm/limiter"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type catalog struct {
	rpm     map[string]int
	lookups int
}

func (c *catalog) Lookup(_ context.Context, ref llm.ModelRef) (llm.ModelInfo, error) {
	c.lookups++
	rpm, ok := c.rpm[ref.String()]
	if !ok {
		return llm.ModelInfo{}, errors.New(errors.NotFound, "the model catalog does not carry this model")
	}
	return llm.ModelInfo{Ref: ref, RPM: rpm}, nil
}

func request(model string) port.Request {
	return port.Request{
		Ref:      llm.ModelRef{Provider: "openai", Model: model},
		Messages: []port.Message{{Role: port.RoleUser, Text: "ANSWER: ok"}},
	}
}

func TestLimiterPassesCallsThrough(t *testing.T) {
	t.Parallel()

	inner := fake.New()
	models := &catalog{rpm: map[string]int{"openai:fast": 6000, "openai:slow": 60}}
	client := limiter.New(inner, models)

	for range 3 {
		if _, err := client.Complete(t.Context(), request("fast")); err != nil {
			t.Fatalf("Complete: %v", err)
		}
	}
	if _, err := client.Stream(t.Context(), request("slow")); err != nil {
		t.Fatalf("Stream: %v", err)
	}

	if len(inner.Requests()) != 4 {
		t.Errorf("inner calls = %d, want 4", len(inner.Requests()))
	}
	if models.lookups != 4 {
		t.Errorf("lookups = %d, want one per call", models.lookups)
	}
}

func TestLimiterRefusesAnUnknownModel(t *testing.T) {
	t.Parallel()

	client := limiter.New(fake.New(), &catalog{rpm: map[string]int{}})
	if _, err := client.Complete(t.Context(), request("ghost")); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Complete error = %v, want %s", err, errors.NotFound)
	}
	if _, err := client.Stream(t.Context(), request("ghost")); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Stream error = %v, want %s", err, errors.NotFound)
	}
}

func TestLimiterThrottlesPastTheBurst(t *testing.T) {
	t.Parallel()

	models := &catalog{rpm: map[string]int{"openai:trickle": 1}}
	client := limiter.New(fake.New(), models)

	if _, err := client.Complete(t.Context(), request("trickle")); err != nil {
		t.Fatalf("first Complete: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if _, err := client.Complete(ctx, request("trickle")); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("second Complete error = %v, want %s", err, errors.Cancelled)
	}
}

func TestLimiterToleratesAMissingRate(t *testing.T) {
	t.Parallel()

	models := &catalog{rpm: map[string]int{"openai:unrated": 0}}
	client := limiter.New(fake.New(), models)

	if _, err := client.Complete(t.Context(), request("unrated")); err != nil {
		t.Fatalf("Complete: %v", err)
	}
}
