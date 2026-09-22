package agent

import (
	"context"
	"strings"
	"sync"

	"github.com/gollem-dev/gollem"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
)

type spend struct {
	input  int
	cached int
	output int
}

func (s *spend) reading(chunk *gollem.ContentResponse) {
	if chunk.InputToken > 0 {
		s.input = chunk.InputToken
	}
	if chunk.CacheReadInputToken > 0 {
		s.cached = chunk.CacheReadInputToken
	}
	if chunk.OutputToken > 0 {
		s.output = chunk.OutputToken
	}
}

type tally struct {
	mu     sync.Mutex
	input  int
	cached int
	output int
	seq    int64
}

func (t *tally) add(round spend) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.input += round.input
	t.cached += round.cached
	t.output += round.output
}

func (t *tally) next() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.seq++
	return t.seq
}

func (t *tally) total() domainllm.Usage {
	t.mu.Lock()
	defer t.mu.Unlock()
	return domainllm.Usage{
		Input:       t.input,
		CachedInput: t.cached,
		Output:      t.output,
		Total:       t.input + t.output,
	}
}

type answer struct {
	mu     sync.Mutex
	rounds []string
}

func (a *answer) spoke(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.rounds = append(a.rounds, text)
}

func (a *answer) String() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return strings.Join(a.rounds, "\n\n")
}

type watch struct {
	stream agentapp.Stream
	usage  *tally
	spoken *answer
	note   func(error)
	round  spend
	said   strings.Builder
}

func (w *watch) chunk(ctx context.Context, chunk *gollem.ContentResponse) {
	if chunk == nil {
		return
	}

	w.round.reading(chunk)
	for _, text := range chunk.Texts {
		if text == "" {
			continue
		}
		w.said.WriteString(text)
		if w.stream == nil {
			continue
		}
		if err := w.stream.Delta(ctx, w.usage.next(), text); err != nil {
			w.note(err)
			return
		}
	}
}

func (w *watch) settle() {
	w.usage.add(w.round)
	w.spoken.spoke(w.said.String())
}

func observe(stream agentapp.Stream, usage *tally, spoken *answer, note func(error)) gollem.ContentStreamMiddleware {
	return func(next gollem.ContentStreamHandler) gollem.ContentStreamHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (<-chan *gollem.ContentResponse, error) {
			chunks, err := next(ctx, req)
			if err != nil {
				return nil, err
			}

			out := make(chan *gollem.ContentResponse)
			go func() {
				round := &watch{stream: stream, usage: usage, spoken: spoken, note: note}
				defer func() {
					round.settle()
					close(out)
				}()

				for chunk := range chunks {
					round.chunk(ctx, chunk)
					select {
					case out <- chunk:
					case <-ctx.Done():
						drain(chunks)
						return
					}
				}
			}()
			return out, nil
		}
	}
}

func drain(chunks <-chan *gollem.ContentResponse) {
	for range chunks {
	}
}
