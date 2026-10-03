package agent

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"go.uber.org/zap"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = llm.MustPrompts(promptFS, "prompts/*.tmpl")

const (
	instructionsPrompt = "chat.instructions"
	contextPrompt      = "chat.context"
	cacheFamily        = "chat:"

	ChatStep = domainllm.StepChat
)

type streamer interface {
	Stream(ctx context.Context, req llm.Request) (<-chan llm.Delta, error)
}

type catalogReader interface {
	Lookup(ctx context.Context, ref domainllm.ModelRef) (domainllm.ModelInfo, error)
}

type Deps struct {
	Client   streamer
	Registry *tools.Registry
	History  historyStore
	Catalog  catalogReader
	Clock    clock.Clock
	Logger   *zap.Logger
}

type Runner struct {
	deps Deps
}

func New(deps Deps) *Runner {
	return &Runner{deps: deps}
}

func resultCeiling(spec agentapp.RunSpec) int {
	if spec.MaxToolResult > 0 {
		return spec.MaxToolResult
	}
	return agentapp.DefaultMaxToolResultBytes
}

func historyCeiling(spec agentapp.RunSpec) int {
	if spec.HistoryToolResult > 0 {
		return spec.HistoryToolResult
	}
	return agentapp.DefaultHistoryToolResultBytes
}

func (r *Runner) Run(ctx context.Context, spec agentapp.RunSpec) (agentapp.RunResult, error) {
	if strings.TrimSpace(spec.Input) == "" {
		return agentapp.RunResult{}, errors.New(errors.Invalid, "an agent turn needs something to answer")
	}
	if spec.LoopLimit <= 0 {
		return agentapp.RunResult{}, errors.New(errors.Invalid, "an agent turn needs a positive loop limit")
	}
	if !spec.Ref.Valid() {
		return agentapp.RunResult{}, errors.New(errors.Invalid, "an agent turn needs a model")
	}

	instructions, err := prompts.One(instructionsPrompt, spec.Context)
	if err != nil {
		return agentapp.RunResult{}, err
	}
	situation, err := prompts.One(contextPrompt, spec.Context)
	if err != nil {
		return agentapp.RunResult{}, err
	}

	current := r.open(spec, instructions, situation)
	if err = r.execute(ctx, current); err != nil {
		r.deps.Logger.Error("an agent turn failed",
			zap.String("conversationId", spec.Binding.ConversationID),
			zap.String("model", spec.Ref.String()),
			zap.String("code", errors.CodeOf(err).String()),
			zap.Error(err))
		return agentapp.RunResult{}, err
	}

	current.guard.settle(ctx)
	return agentapp.RunResult{
		Text:      current.spoken.String(),
		ToolCalls: current.guard.calls(),
		Calls:     current.usage.answered(),
		Usage:     current.usage.total(),
		USD:       current.usage.spent(),
	}, nil
}

func (r *Runner) open(spec agentapp.RunSpec, instructions, situation string) *turn {
	built := r.deps.Registry.Build(spec.Binding)
	offered := make([]llm.Tool, 0, len(built))
	for i := range built {
		offered = append(offered, llm.Tool{
			Name: built[i].Def.Name, Description: built[i].Def.Description, Schema: built[i].Def.Schema,
		})
	}

	return &turn{
		spec:    spec,
		client:  r.deps.Client,
		catalog: r.deps.Catalog,
		memory: memory{
			store: r.deps.History, clock: r.deps.Clock, conversationID: spec.Binding.ConversationID,
			budget: spec.HistoryBudget, cap: historyCeiling(spec),
		},
		guard:  newGuard(spec, built, resultCeiling(spec)),
		usage:  &tally{},
		spoken: &answer{},
		asked: llm.Request{
			Ref:    spec.Ref,
			System: instructions,
			Tools:  offered,
			Meta: llm.CallMeta{
				RunID: spec.Binding.RunID, ConversationID: spec.Binding.ConversationID,
				Step: ChatStep, Role: domainllm.RoleChat,
			},
			Effort:   domainllm.EffortNone,
			Tier:     domainllm.TierDefault,
			CacheKey: cacheFamily + spec.Context.Mode,
		},
		opening: []llm.Message{
			{Role: llm.RoleDeveloper, Text: situation},
			{Role: llm.RoleUser, Text: spec.Input},
		},
	}
}

func (r *Runner) execute(ctx context.Context, current *turn) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			r.deps.Logger.Error("an agent turn panicked",
				zap.String("conversationId", current.spec.Binding.ConversationID))
			err = errors.New(errors.Internal, fmt.Sprintf("the agent turn panicked: %v", recovered))
		}
	}()
	return current.run(ctx)
}
