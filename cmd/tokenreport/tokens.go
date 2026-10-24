package main

import (
	"context"
	"slices"
	"strings"

	"github.com/tiktoken-go/tokenizer"

	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/domain/llm"
)

var (
	o200kModels   = []string{"gpt-4.1", "gpt-4o", "o1", "o3", "o4-mini"}
	o200kPrefixes = []string{"gpt-5", "gpt-4.5-", "gpt-4.1-", "chatgpt-4o-", "gpt-4o-", "o1-", "o3-", "o4-mini-", "ft:gpt-4o"}
)

type counter struct {
	codec tokenizer.Codec
}

func newCounter(encoding tokenizer.Encoding) (counter, error) {
	codec, err := tokenizer.Get(encoding)
	if err != nil {
		return counter{}, err
	}
	return counter{codec: codec}, nil
}

func (c counter) count(text string) (int, error) {
	return c.codec.Count(text)
}

type counters struct {
	held map[tokenizer.Encoding]counter
}

func newCounters() *counters {
	return &counters{held: map[tokenizer.Encoding]counter{}}
}

func (cs *counters) of(ref llm.ModelRef) (counter, error) {
	mapped := encodingFor(ref)
	if held, ok := cs.held[mapped.encoding]; ok {
		return held, nil
	}
	made, err := newCounter(mapped.encoding)
	if err != nil {
		return counter{}, err
	}
	cs.held[mapped.encoding] = made
	return made, nil
}

type encoding struct {
	encoding tokenizer.Encoding
	fallback bool
}

func encodingFor(ref llm.ModelRef) encoding {
	known := slices.Contains(o200kModels, ref.Model) ||
		slices.ContainsFunc(o200kPrefixes, func(prefix string) bool { return strings.HasPrefix(ref.Model, prefix) })
	return encoding{encoding: tokenizer.O200kBase, fallback: !known}
}

type modelEncoding struct {
	model    string
	encoding tokenizer.Encoding
	fallback bool
}

type noOverrides struct{}

func (noOverrides) List(context.Context) ([]llm.ModelOverride, error) {
	return nil, nil
}

func catalogEncodings(ctx context.Context, models *catalog.Catalog) ([]modelEncoding, error) {
	listed, err := models.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]modelEncoding, 0, len(listed))
	for i := range listed {
		mapped := encodingFor(listed[i].Ref)
		out = append(out, modelEncoding{model: listed[i].Ref.Model, encoding: mapped.encoding, fallback: mapped.fallback})
	}
	return out, nil
}
