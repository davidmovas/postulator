package agent

import (
	"context"
	"strings"
	"sync"
	"time"

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

type round struct {
	failure error
	usage   spend
	latency time.Duration
	index   int
}

type tally struct {
	mu     sync.Mutex
	input  int
	cached int
	output int
	usd    float64
	seq    int64
	rounds int
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

func (t *tally) enter() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rounds++
	return t.rounds
}

func (t *tally) charge(usd float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usd += usd
}

func (t *tally) calls() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.rounds
}

func (t *tally) spent() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.usd
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

func observe(stream agentapp.Stream, usage *tally, spoken *answer, note func(error),
	record func(round)) gollem.ContentStreamMiddleware {
	return func(next gollem.ContentStreamHandler) gollem.ContentStreamHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (<-chan *gollem.ContentResponse, error) {
			index := usage.enter()
			started := time.Now()

			chunks, err := next(ctx, req)
			if err != nil {
				record(round{index: index, latency: time.Since(started), failure: err})
				return nil, err
			}

			out := make(chan *gollem.ContentResponse)
			go func() {
				watched := &watch{stream: stream, usage: usage, spoken: spoken, note: note}
				stopped := false
				defer func() {
					watched.settle()
					record(round{
						index: index, usage: watched.round, latency: time.Since(started),
						failure: stoppedBy(ctx, stopped),
					})
					close(out)
				}()

				for chunk := range chunks {
					watched.chunk(ctx, chunk)
					select {
					case out <- chunk:
					case <-ctx.Done():
						stopped = true
						drain(chunks)
						return
					}
				}
			}()
			return out, nil
		}
	}
}

func stoppedBy(ctx context.Context, stopped bool) error {
	if !stopped {
		return nil
	}
	return ctx.Err()
}

func drain(chunks <-chan *gollem.ContentResponse) {
	for range chunks {
	}
}
