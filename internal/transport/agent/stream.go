package agent

import (
	"context"
	"sync"

	"github.com/gollem-dev/gollem"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
)

type tally struct {
	mu     sync.Mutex
	input  int
	output int
	seq    int64
}

func (t *tally) add(input, output int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.input += input
	t.output += output
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
	return domainllm.Usage{Input: t.input, Output: t.output, Total: t.input + t.output}
}

func observe(stream agentapp.Stream, usage *tally, note func(error)) gollem.ContentStreamMiddleware {
	return func(next gollem.ContentStreamHandler) gollem.ContentStreamHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (<-chan *gollem.ContentResponse, error) {
			chunks, err := next(ctx, req)
			if err != nil {
				return nil, err
			}

			out := make(chan *gollem.ContentResponse)
			go func() {
				defer close(out)
				for chunk := range chunks {
					observeChunk(ctx, stream, usage, chunk, note)
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

func observeChunk(ctx context.Context, stream agentapp.Stream, usage *tally, chunk *gollem.ContentResponse,
	note func(error)) {
	if chunk == nil {
		return
	}

	usage.add(chunk.InputToken, chunk.OutputToken)
	if stream == nil {
		return
	}
	for _, text := range chunk.Texts {
		if text == "" {
			continue
		}
		if err := stream.Delta(ctx, usage.next(), text); err != nil {
			note(err)
			return
		}
	}
}

func drain(chunks <-chan *gollem.ContentResponse) {
	for range chunks {
	}
}
