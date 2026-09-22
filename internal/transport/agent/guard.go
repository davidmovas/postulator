package agent

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gollem-dev/gollem"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	notFoundSuffix = " is not found"

	auditCallID = "audit"
	auditTool   = "agent_audit"
	auditFailed = "the record of this turn is incomplete: "
)

type guard struct {
	mu      sync.Mutex
	stream  agentapp.Stream
	allowed []string
	seen    map[string]bool
	cap     int
	count   int
	dropped []string
}

func newGuard(spec agentapp.RunSpec, resultCap int) *guard {
	return &guard{
		stream:  spec.Stream,
		allowed: slices.Clone(spec.Allowed),
		seen:    map[string]bool{},
		cap:     resultCap,
	}
}

func (g *guard) middlewares() []gollem.ToolMiddleware {
	return []gollem.ToolMiddleware{g.fence(), g.audit(), g.capResult(), g.permission()}
}

func (g *guard) calls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.count
}

func (g *guard) note(err error) {
	if err == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	g.dropped = append(g.dropped, err.Error())
}

func (g *guard) settle(ctx context.Context) {
	g.mu.Lock()
	held := slices.Clone(g.dropped)
	g.dropped = nil
	g.mu.Unlock()

	if len(held) == 0 || g.stream == nil {
		return
	}
	g.note(g.stream.ToolFinished(context.WithoutCancel(ctx), agentapp.ToolOutcome{
		CallID: auditCallID, Tool: auditTool, Args: json.RawMessage("{}"),
		Status: string(domainagent.CallError), Error: auditFailed + strings.Join(held, "; "),
	}))
}

func (g *guard) unknown(known []string) gollem.ContentStreamMiddleware {
	names := make(map[string]bool, len(known))
	for _, name := range known {
		names[name] = true
	}

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
					g.invented(ctx, names, chunk)
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

func (g *guard) invented(ctx context.Context, known map[string]bool, chunk *gollem.ContentResponse) {
	if chunk == nil || g.stream == nil {
		return
	}

	for _, call := range chunk.FunctionCalls {
		if call == nil || known[call.Name] || !g.first(call.ID) {
			continue
		}

		args := tools.Redact(encode(call.Arguments))
		g.note(g.stream.ToolStarted(ctx, call.ID, call.Name, args))
		g.note(g.stream.ToolFinished(ctx, agentapp.ToolOutcome{
			CallID: call.ID, Tool: call.Name, Args: args,
			Status: string(domainagent.CallError), Error: call.Name + notFoundSuffix,
		}))
	}
}

func (g *guard) first(callID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.seen[callID] {
		return false
	}
	g.seen[callID] = true
	g.count++
	return true
}

func (g *guard) fence() gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			resp, err := next(ctx, req)
			if err != nil || resp == nil || resp.Result == nil {
				return resp, err
			}

			fenced := *resp
			fenced.Result = agentapp.Fence(resp.Result)
			return &fenced, err
		}
	}
}

func (g *guard) audit() gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			args := tools.Redact(encode(req.Tool.Arguments))
			if g.stream != nil {
				g.note(g.stream.ToolStarted(ctx, req.Tool.ID, req.Tool.Name, args))
			}

			started := time.Now()
			resp, err := next(ctx, req)
			elapsed := time.Since(started).Milliseconds()

			outcome := agentapp.ToolOutcome{
				CallID: req.Tool.ID, Tool: req.Tool.Name, Args: args,
				Status: string(domainagent.CallOK), DurationMS: elapsed,
			}
			switch {
			case err != nil:
				outcome.Status, outcome.Error = string(domainagent.CallError), err.Error()
			case resp != nil && resp.Error != nil:
				outcome.Status, outcome.Error = string(domainagent.CallError), resp.Error.Error()
				if errors.IsCode(resp.Error, errors.Unauthorized) {
					outcome.Status = string(domainagent.CallDenied)
				}
			case resp != nil:
				outcome.Result = encode(resp.Result)
			}

			g.mu.Lock()
			g.count++
			g.mu.Unlock()

			if g.stream != nil {
				g.note(g.stream.ToolFinished(ctx, outcome))
			}
			return resp, err
		}
	}
}

func (g *guard) capResult() gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			resp, err := next(ctx, req)
			if err != nil || resp == nil || resp.Result == nil {
				return resp, err
			}

			capped, shortened := agentapp.Cap(resp.Result, g.cap)
			if !shortened {
				return resp, err
			}

			out := *resp
			out.Result = capped
			return &out, err
		}
	}
}

func (g *guard) permission() gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			if err := agentapp.Permit(g.allowed, req.Tool.Name); err != nil {
				return &gollem.ToolExecResponse{Error: err}, nil
			}
			return next(ctx, req)
		}
	}
}

func encode(value map[string]any) json.RawMessage {
	if value == nil {
		return json.RawMessage("{}")
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("{}")
	}
	return encoded
}
