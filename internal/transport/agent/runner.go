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

	"github.com/davidmovas/postulator/internal/adapters/llm/gollemclient"
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

const (
	promptName = "system"

	ChatStep = "chat"
)

type providerFactory interface {
	NewForTools(ctx context.Context, ref domainllm.ModelRef) (gollem.LLMClient, error)
}

type callStore interface {
	Insert(ctx context.Context, call domainllm.Call) error
}

type catalogReader interface {
	Lookup(ctx context.Context, ref domainllm.ModelRef) (domainllm.ModelInfo, error)
}

type Config struct {
	MaxToolResultBytes int
	Retries            int
	Backoff            time.Duration
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
	deps    Deps
	cfg     Config
	waiting *patience
}

func New(deps Deps, cfg Config) *Runner {
	if cfg.MaxToolResultBytes <= 0 {
		cfg.MaxToolResultBytes = agentapp.DefaultMaxToolResultBytes
	}
	return &Runner{deps: deps, cfg: cfg, waiting: newPatience(deps.Catalog, cfg.Retries, cfg.Backoff)}
}

func (r *Runner) resultCeiling(spec agentapp.RunSpec) int {
	if spec.MaxToolResult > 0 {
		return spec.MaxToolResult
	}
	return r.cfg.MaxToolResultBytes
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

	client, err := r.deps.Factory.NewForTools(ctx, spec.Ref)
	if err != nil {
		return agentapp.RunResult{}, err
	}

	system, user, err := prompts.Render(promptName, spec.Context)
	if err != nil {
		return agentapp.RunResult{}, err
	}

	guard := newGuard(spec, r.resultCeiling(spec))
	usage := &tally{}
	spoken := &answer{}

	result, err := r.execute(ctx, client, spec, system+"\n\n"+user, guard, usage, spoken)
	if err != nil {
		r.deps.Logger.Error("an agent turn failed",
			zap.String("conversationId", spec.Binding.ConversationID),
			zap.String("model", spec.Ref.String()),
			zap.String("code", errors.CodeOf(err).String()),
			zap.Error(err))
		return agentapp.RunResult{}, err
	}

	result.Usage = usage.total()
	result.ToolCalls = guard.calls()
	result.Calls = usage.calls()
	result.USD = usage.spent()
	return result, nil
}

func (r *Runner) execute(ctx context.Context, client gollem.LLMClient, spec agentapp.RunSpec, system string,
	guard *guard, usage *tally, spoken *answer) (result agentapp.RunResult, err error) {
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

	billing := context.WithoutCancel(ctx)
	record := func(done round) { r.bill(billing, spec, usage, done, guard.note) }

	options := []gollem.Option{
		gollem.WithLoopLimit(spec.LoopLimit),
		gollem.WithSystemPrompt(system),
		gollem.WithResponseMode(gollem.ResponseModeStreaming),
		gollem.WithStrategy(readTheStream{}),
		gollem.WithContentStreamMiddleware(r.waiting.middleware(spec.Ref)),
		gollem.WithContentStreamMiddleware(observe(spec.Stream, usage, spoken, guard.note, record)),
		gollem.WithContentStreamMiddleware(guard.unknown(r.deps.Registry.Names())),
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

	if _, err = gollem.New(client, options...).Execute(ctx, gollem.Text(spec.Input)); err != nil {
		return agentapp.RunResult{}, convert(err)
	}

	guard.settle(ctx)
	return agentapp.RunResult{Text: spoken.String()}, nil
}

func (r *Runner) bill(ctx context.Context, spec agentapp.RunSpec, usage *tally, done round, note func(error)) {
	counted := domainllm.Usage{
		Input:       done.usage.input,
		CachedInput: done.usage.cached,
		Output:      done.usage.output,
		Total:       done.usage.input + done.usage.output,
	}
	cost := r.cost(ctx, spec.Ref, counted)
	usage.charge(cost)

	r.record(ctx, spec, done, counted, cost)
	if spec.Stream == nil {
		return
	}
	note(spec.Stream.Spent(ctx, agentapp.RoundUsage{
		Provider: spec.Ref.Provider, Model: spec.Ref.Model, Round: done.index,
		Input: counted.Input, CachedInput: counted.CachedInput, Output: counted.Output, USD: cost,
	}))
}

func (r *Runner) record(ctx context.Context, spec agentapp.RunSpec, done round,
	counted domainllm.Usage, cost float64) {
	if r.deps.Calls == nil {
		return
	}

	call := domainllm.Call{
		ID:             id.New(),
		ConversationID: spec.Binding.ConversationID,
		RunID:          spec.Binding.RunID,
		Step:           ChatStep,
		Ref:            spec.Ref,
		Usage:          counted,
		USD:            cost,
		Latency:        done.latency,
		Status:         domainllm.CallOK,
		CreatedAt:      r.deps.Clock.Now().UTC().Truncate(time.Second),
	}
	if done.failure != nil {
		call.Status = domainllm.CallError
		call.ErrorCode = errors.CodeOf(convert(done.failure)).String()
	}

	if err := r.deps.Calls.Insert(ctx, call); err != nil {
		r.deps.Logger.Error("a model call could not be written to the spend ledger, so the totals under-count it",
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

type readTheStream struct{}

func (readTheStream) Init(context.Context, []gollem.Input) error {
	return nil
}

func (readTheStream) Handle(_ context.Context, state *gollem.StrategyState) (
	[]gollem.Input, *gollem.ExecuteResponse, error) {
	if state.Iteration == 0 {
		return state.InitInput, nil, nil
	}
	if state.LastResponse != nil && len(state.LastResponse.FunctionCalls) == 0 {
		return nil, &gollem.ExecuteResponse{}, nil
	}
	return state.NextInput, nil, nil
}

func (readTheStream) Tools(context.Context) ([]gollem.Tool, error) {
	return []gollem.Tool{}, nil
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
		if provider, ok := gollemclient.Provider(err); ok {
			return provider
		}
		return errors.Wrap(err, errors.External, "the model could not answer")
	}
}
