package metered_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/images/metered"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type drawing struct {
	image images.Image
	err   error
}

func (d drawing) Generate(context.Context, images.Prompt) (images.Image, error) {
	return d.image, d.err
}

type calls struct {
	mu   sync.Mutex
	rows []llm.Call
}

func (c *calls) Insert(_ context.Context, call llm.Call) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows = append(c.rows, call)
	return nil
}

type prices struct{}

func (prices) Lookup(context.Context, llm.ModelRef) (llm.ModelInfo, error) {
	return llm.ModelInfo{InputUSDPerM: 8, CachedInputUSDPerM: 1.25, OutputUSDPerM: 30}, nil
}

type bus struct {
	mu        sync.Mutex
	published []events.LLMUsagePayload
}

func (b *bus) Publish(_ events.Type, payload any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if usage, ok := payload.(events.LLMUsagePayload); ok {
		b.published = append(b.published, usage)
	}
	return nil
}

func TestEveryImageIsACallInTheLedger(t *testing.T) {
	t.Parallel()

	ref := llm.ModelRef{Provider: "openai", Model: "gpt-image-2"}
	stamp := time.Date(2026, time.September, 23, 18, 0, 0, 0, time.UTC)
	prompt := images.Prompt{RunID: "run-1", ItemID: "item-1", Step: "generate_images", Subject: "Kettle"}

	cases := []struct {
		name   string
		inner  drawing
		status llm.CallStatus
		code   string
		usd    float64
		heard  int
	}{
		{
			name:   "a drawn image is priced from the catalog",
			inner:  drawing{image: images.Image{Usage: llm.Usage{Input: 1_000_000, Output: 1_000_000, Total: 2_000_000}}},
			status: llm.CallOK,
			usd:    38,
			heard:  1,
		},
		{
			name:   "a refused image is a failed call",
			inner:  drawing{err: errors.New(errors.RateLimited, "slow down")},
			status: llm.CallError,
			code:   errors.RateLimited.String(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ledger := &calls{}
			heard := &bus{}
			generator := metered.New(tc.inner, ref, ledger, prices{}, heard, clock.NewFake(stamp))

			_, err := generator.Generate(t.Context(), prompt)
			if (err != nil) != (tc.inner.err != nil) {
				t.Fatalf("Generate = %v, want the inner answer passed through", err)
			}

			if len(ledger.rows) != 1 {
				t.Fatalf("the ledger holds %d calls, want one per image", len(ledger.rows))
			}
			row := ledger.rows[0]
			if row.RunID != "run-1" || row.ItemID != "item-1" || row.Step != "generate_images" || row.Ref != ref {
				t.Fatalf("the call = %+v, want it attributed to the run, the item and the step", row)
			}
			if row.Status != tc.status || row.ErrorCode != tc.code {
				t.Fatalf("the call ended %s/%q, want %s/%q", row.Status, row.ErrorCode, tc.status, tc.code)
			}
			if row.USD != tc.usd {
				t.Fatalf("the call cost %v, want %v", row.USD, tc.usd)
			}
			if !row.CreatedAt.Equal(stamp) || row.ID == "" {
				t.Fatalf("the call = %+v", row)
			}
			if len(heard.published) != tc.heard {
				t.Fatalf("llm.usage was published %d times, want %d", len(heard.published), tc.heard)
			}
		})
	}
}
