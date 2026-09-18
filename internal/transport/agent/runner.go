package agent

import (
	"context"
	"embed"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/gollem-dev/gollem"
	"go.uber.org/zap"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = llm.MustPrompts(promptFS, "prompts/*.tmpl")

const promptName = "system"

type providerFactory interface {
	New(ctx context.Context, ref domainllm.ModelRef) (gollem.LLMClient, error)
}

type callStore interface {
	Insert(ctx context.Context, call domainllm.Call) error
}

type catalogReader interface {
	Lookup(ctx context.Context, ref domainllm.ModelRef) (domainllm.ModelInfo, error)
}

type Config struct {
	MaxToolResultBytes int
}

type Deps struct {
	Factory  providerFactory
	Registry *tools.Registry
	History  historyStore
	Calls    callStore
	Catalog  catalogReader
	Clock    clock.Clock
	Logger   *zap.Logger
}

type Runner struct {
	deps Deps
	cfg  Config
}

func New(deps Deps, cfg Config) *Runner {
	if cfg.MaxToolResultBytes <= 0 {
		cfg.MaxToolResultBytes = agentapp.DefaultMaxToolResultBytes
	}
	return &Runner{deps: deps, cfg: cfg}
}

func (r *Runner) Run(ctx context.Context, spec agentapp.RunSpec) (agentapp.RunResult, error) {
	if strings.TrimSpace(spec.Input) == "" {
		return agentapp.RunResult{}, errors.New(errors.Invalid, "an agent turn needs something to answer")
	}
	if spec.LoopLimit <= 0 {
		return agentapp.RunResult{}, errors.New(errors.Invalid,
			"an agent turn needs a positive loop limit; gollem reads a zero as no limit at all")
	}
	if !spec.Ref.Valid() {
		return agentapp.RunResult{}, errors.New(errors.Invalid, "an agent turn needs a model")
	}

	client, err := r.deps.Factory.New(ctx, spec.Ref)
	if err != nil {
		return agentapp.RunResult{}, err
	}

	system, user, err := prompts.Render(promptName, spec.Context)
	if err != nil {
		return agentapp.RunResult{}, err
	}

	guard := newGuard(spec, r.cfg.MaxToolResultBytes)
	usage := &tally{}
	started := time.Now()

	result, err := r.execute(ctx, client, spec, system+"\n\n"+user, guard, usage)
	r.record(ctx, spec, usage, time.Since(started), err)
	if err != nil {
		return agentapp.RunResult{}, err
	}

	result.Usage = usage.total()
	result.ToolCalls = guard.calls()
	result.USD = r.cost(ctx, spec.Ref, result.Usage)
	return result, nil
}

func (r *Runner) execute(ctx context.Context, client gollem.LLMClient, spec agentapp.RunSpec, system string,
	guard *guard, usage *tally) (result agentapp.RunResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			r.deps.Logger.Error("an agent turn panicked", zap.String("conversationId", spec.Binding.ConversationID))
			result = agentapp.RunResult{}
			err = errors.New(errors.Internal, fmt.Sprintf("the agent turn panicked: %v", recovered))
		}
	}()

	built := r.deps.Registry.Build(spec.Binding)
	adapted := make([]gollem.Tool, 0, len(built))
	for i := range built {
		adapted = append(adapted, &gollemTool{tool: built[i], binding: spec.Binding})
	}

	options := []gollem.Option{
		gollem.WithLoopLimit(spec.LoopLimit),
		gollem.WithSystemPrompt(system),
		gollem.WithResponseMode(gollem.ResponseModeStreaming),
		gollem.WithContentStreamMiddleware(observe(spec.Stream, usage, guard.note)),
		gollem.WithTools(adapted...),
	}
	for _, middleware := range guard.middlewares() {
		options = append(options, gollem.WithToolMiddleware(middleware))
	}
	if r.deps.History != nil && spec.Binding.ConversationID != "" {
		options = append(options, gollem.WithHistoryRepository(
			bounded{store: r.deps.History, clock: r.deps.Clock, budget: spec.HistoryBudget},
			spec.Binding.ConversationID,
		))
	}

	answered, err := gollem.New(client, options...).Execute(ctx, gollem.Text(spec.Input))
	if err != nil {
		return agentapp.RunResult{}, convert(err)
	}

	text := ""
	if answered != nil {
		text = answered.String()
	}
	return agentapp.RunResult{Text: text}, guard.failure()
}

func (r *Runner) record(ctx context.Context, spec agentapp.RunSpec, usage *tally, latency time.Duration, failure error) {
	if r.deps.Calls == nil {
		return
	}

	observed := usage.total()
	call := domainllm.Call{
		ID:             id.New(),
		ConversationID: spec.Binding.ConversationID,
		RunID:          spec.Binding.RunID,
		Step:           promptName,
		Ref:            spec.Ref,
		Usage:          observed,
		USD:            r.cost(ctx, spec.Ref, observed),
		Latency:        latency,
		Status:         domainllm.CallOK,
		CreatedAt:      r.deps.Clock.Now().UTC().Truncate(time.Second),
	}
	if failure != nil {
		call.Status = domainllm.CallError
		call.ErrorCode = errors.CodeOf(failure).String()
	}

	if err := r.deps.Calls.Insert(context.WithoutCancel(ctx), call); err != nil {
		r.deps.Logger.Error("an agent turn could not be written to the spend ledger, so the totals under-count it",
			zap.String("conversationId", spec.Binding.ConversationID), zap.Error(err))
	}
}

func (r *Runner) cost(ctx context.Context, ref domainllm.ModelRef, usage domainllm.Usage) float64 {
	if usage.Total == 0 || r.deps.Catalog == nil {
		return 0
	}

	info, err := r.deps.Catalog.Lookup(ctx, ref)
	if err != nil {
		return 0
	}
	return domainllm.Cost(usage, info)
}

func convert(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.CodeOf(err) != errors.Internal:
		return err
	case stderrors.Is(err, context.Canceled), stderrors.Is(err, context.DeadlineExceeded):
		return errors.Wrap(err, errors.Cancelled, "the agent turn was stopped")
	case stderrors.Is(err, gollem.ErrLoopLimitExceeded):
		return errors.Wrap(err, errors.BudgetExceeded, "the agent used up its tool call budget for this turn")
	default:
		return errors.Wrap(err, errors.External, "the model could not answer")
	}
}
