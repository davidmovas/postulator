package agent

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gollem-dev/gollem"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	untrustedMarker = "untrustedContent"
	untrustedData   = "data"

	truncatedKey  = "truncated"
	previewKey    = "preview"
	totalBytesKey = "totalBytes"

	minPreviewBytes = 256
)

type guard struct {
	mu      sync.Mutex
	stream  agentapp.Stream
	allowed []string
	cap     int
	count   int
	failed  error
}

func newGuard(spec agentapp.RunSpec, resultCap int) *guard {
	return &guard{stream: spec.Stream, allowed: slices.Clone(spec.Allowed), cap: resultCap}
}

func (g *guard) middlewares() []gollem.ToolMiddleware {
	return []gollem.ToolMiddleware{g.fence(), g.audit(), g.capResult(), g.permission()}
}

func (g *guard) calls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.count
}

func (g *guard) failure() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.failed
}

func (g *guard) note(err error) {
	if err == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.failed == nil {
		g.failed = errors.Wrap(err, errors.CodeOf(err), "a tool call could not be audited")
	}
}

func (g *guard) fence() gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			resp, err := next(ctx, req)
			if err != nil || resp == nil || resp.Result == nil {
				return resp, err
			}

			fenced := *resp
			fenced.Result = map[string]any{untrustedMarker: true, untrustedData: resp.Result}
			return &fenced, err
		}
	}
}

func (g *guard) audit() gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			args := encode(req.Tool.Arguments)
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

			capped, cut := truncate(resp.Result, g.cap)
			if !cut {
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
			if len(g.allowed) > 0 && !slices.Contains(g.allowed, req.Tool.Name) {
				return &gollem.ToolExecResponse{
					Error: errors.New(errors.Unauthorized, "the tool "+req.Tool.Name+" is not open to this conversation"),
				}, nil
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

func truncate(result map[string]any, limit int) (map[string]any, bool) {
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) <= limit {
		return result, false
	}

	preview := max(limit/2, minPreviewBytes)
	return map[string]any{
		truncatedKey:  true,
		totalBytesKey: len(encoded),
		previewKey:    cutAtRune(string(encoded), preview),
	}, true
}

func cutAtRune(text string, limit int) string {
	if len(text) <= limit {
		return text
	}

	cut := limit
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut]
}
