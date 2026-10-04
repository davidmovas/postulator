package agent

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	notFoundSuffix = " is not found"
	notRunMessage  = "the turn was stopped before this call ran"
	errorField     = "error"
	emptyArguments = "{}"

	auditCallID = "audit"
	auditTool   = "agent_audit"
	auditFailed = "the record of this turn is incomplete: "
)

type guard struct {
	stream  agentapp.Stream
	binding tools.Binding
	allowed []string
	tools   map[string]tools.Tool
	cap     int
	count   int
	dropped []string
}

func newGuard(spec agentapp.RunSpec, built []tools.Tool, resultCap int) *guard {
	byName := make(map[string]tools.Tool, len(built))
	for _, tool := range built {
		byName[tool.Def.Name] = tool
	}
	return &guard{
		stream:  spec.Stream,
		binding: spec.Binding,
		allowed: slices.Clone(spec.Allowed),
		tools:   byName,
		cap:     resultCap,
	}
}

func (g *guard) calls() int {
	return g.count
}

func (g *guard) note(err error) {
	if err == nil {
		return
	}
	g.dropped = append(g.dropped, err.Error())
}

func (g *guard) settle(ctx context.Context) {
	held := g.dropped
	g.dropped = nil
	if len(held) == 0 || g.stream == nil {
		return
	}
	g.note(g.stream.ToolFinished(context.WithoutCancel(ctx), agentapp.ToolOutcome{
		CallID: auditCallID, Tool: auditTool, Args: json.RawMessage(emptyArguments),
		Status: string(domainagent.CallError), Error: auditFailed + strings.Join(held, "; "),
	}))
}

func (g *guard) dispatch(ctx context.Context, call llm.ToolCall) llm.Message {
	return llm.Message{Role: llm.RoleTool, Result: &llm.ToolResult{CallID: call.ID, Output: g.answer(ctx, call)}}
}

func (g *guard) answer(ctx context.Context, call llm.ToolCall) json.RawMessage {
	if ctx.Err() != nil {
		return failed(errors.New(errors.Cancelled, notRunMessage))
	}

	args := argumentsOf(call.Args)
	shown := tools.Redact(args)
	g.count++
	g.started(ctx, call, shown)

	tool, known := g.tools[call.Name]
	if !known {
		missing := errors.New(errors.NotFound, call.Name+notFoundSuffix)
		g.finished(ctx, agentapp.ToolOutcome{
			CallID: call.ID, Tool: call.Name, Args: shown,
			Status: string(domainagent.CallError), Error: missing.Error(),
		})
		return failed(missing)
	}

	began := time.Now()
	result, err := g.run(ctx, tool, args)
	outcome := agentapp.ToolOutcome{
		CallID: call.ID, Tool: call.Name, Args: shown,
		Status: string(domainagent.CallOK), DurationMS: time.Since(began).Milliseconds(),
	}
	if err != nil {
		outcome.Status, outcome.Error = string(domainagent.CallError), err.Error()
		if errors.IsCode(err, errors.Unauthorized) {
			outcome.Status = string(domainagent.CallDenied)
		}
		g.finished(ctx, outcome)
		return failed(err)
	}

	outcome.Result = encode(result)
	g.finished(ctx, outcome)
	return fenced(result)
}

func (g *guard) run(ctx context.Context, tool tools.Tool, args json.RawMessage) (map[string]any, error) {
	if err := agentapp.Permit(g.allowed, tool.Def.Name); err != nil {
		return nil, err
	}
	if err := check(tool.Def.Name, tool.Def.Schema, args); err != nil {
		return nil, err
	}
	if tool.Authorize != nil {
		if err := tool.Authorize(ctx, g.binding); err != nil {
			return nil, explain(err)
		}
	}

	out, err := tool.Run(ctx, g.binding, args)
	if err != nil {
		return nil, explain(err)
	}
	object, err := objectOf(out)
	if err != nil {
		return nil, err
	}
	capped, _ := agentapp.Cap(object, g.cap)
	return capped, nil
}

func (g *guard) started(ctx context.Context, call llm.ToolCall, shown json.RawMessage) {
	if g.stream != nil {
		g.note(g.stream.ToolStarted(ctx, call.ID, call.Name, shown))
	}
}

func (g *guard) finished(ctx context.Context, outcome agentapp.ToolOutcome) {
	if g.stream != nil {
		g.note(g.stream.ToolFinished(ctx, outcome))
	}
}

func argumentsOf(raw json.RawMessage) json.RawMessage {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return json.RawMessage(emptyArguments)
	}
	return raw
}

func failed(err error) json.RawMessage {
	encoded, marshalErr := json.Marshal(map[string]string{errorField: err.Error()})
	if marshalErr != nil {
		return json.RawMessage(`{"error":"the tool failed"}`)
	}
	return encoded
}

func fenced(result map[string]any) json.RawMessage {
	encoded, err := json.Marshal(agentapp.Fence(result))
	if err != nil {
		return failed(errors.Wrap(err, errors.Internal, "the tool answered something that cannot be encoded"))
	}
	return encoded
}
