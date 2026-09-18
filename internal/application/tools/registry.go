package tools

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/content"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const maxSummaryArgs = 160

type actionStore interface {
	Insert(ctx context.Context, action agent.PendingAction) error
}

type Deps struct {
	Sites     *sites.Service
	Graph     *graph.Service
	Pages     *pages.Service
	Templates *templates.Service
	Runs      *runs.Service
	Sync      *sync.Service
	Reports   *reports.Service
	Imports   *imports.Service
	Models    *models.Service
	Content   *content.Service
	Schedules *schedules.Service
	Actions   actionStore
	Publisher application.Publisher
	Clock     clock.Clock
}

type factory func(Deps) Tool

type Registry struct {
	deps   Deps
	byName map[string]factory
	order  []string
}

func New(deps Deps) *Registry {
	built := factories()
	registry := &Registry{deps: deps, byName: make(map[string]factory, len(built)), order: make([]string, 0, len(built))}
	for _, build := range built {
		name := build(deps).Def.Name
		registry.byName[name] = build
		registry.order = append(registry.order, name)
	}
	return registry
}

func (r *Registry) Names() []string {
	return slices.Clone(r.order)
}

func (r *Registry) Build(b Binding) []Tool {
	built := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		tool := r.byName[name](r.deps)
		if b.Mode == agent.ModeConfirm && tool.Def.Risk.NeedsConfirmation() {
			tool.Run = r.proposal(tool.Def)
		}
		built = append(built, tool)
	}
	return built
}

func (r *Registry) Call(ctx context.Context, b Binding, name string, args json.RawMessage) (any, error) {
	build, known := r.byName[name]
	if !known {
		return nil, errors.New(errors.NotFound, "no tool is registered under this name").WithDetail("tool", name)
	}

	tool := build(r.deps)
	if tool.Authorize != nil {
		if err := tool.Authorize(ctx, b); err != nil {
			return nil, err
		}
	}
	return tool.Run(ctx, b, args)
}

func (r *Registry) Lookup(name string) (Def, bool) {
	build, known := r.byName[name]
	if !known {
		return Def{}, false
	}
	return build(r.deps).Def, true
}

func (r *Registry) proposal(def Def) func(context.Context, Binding, json.RawMessage) (any, error) {
	return func(ctx context.Context, b Binding, args json.RawMessage) (any, error) {
		if b.ConversationID == "" {
			return nil, errors.New(errors.Invalid, "a confirmation needs the conversation that asked for it").
				WithDetail("tool", def.Name)
		}

		now := r.now()
		action, err := agent.NewPendingAction(agent.PendingAction{
			ID: id.New(), ConversationID: b.ConversationID, Tool: def.Name, Args: args,
			Summary: Summary(def, args), CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return nil, err
		}
		if insertErr := r.deps.Actions.Insert(ctx, action); insertErr != nil {
			return nil, insertErr
		}
		if publishErr := r.deps.Publisher.Publish(events.AgentConfirmRequested, events.AgentConfirmRequestedPayload{
			ConversationID: b.ConversationID, ConfirmationID: action.ID, Tool: def.Name,
			Args: Redact(action.Args), Risk: string(def.Risk), Summary: action.Summary,
		}); publishErr != nil {
			return nil, publishErr
		}
		return Confirmation{Status: agent.ConfirmationRequired, ActionID: action.ID, Summary: action.Summary}, nil
	}
}

func (r *Registry) now() time.Time {
	return r.deps.Clock.Now().UTC().Truncate(time.Second)
}

func Summary(def Def, args json.RawMessage) string {
	summary := strings.TrimSuffix(strings.TrimSpace(def.Description), ".")
	if summary == "" {
		summary = def.Name
	}

	compact := strings.TrimSpace(string(Redact(args)))
	if compact == "" || compact == "{}" || compact == "null" {
		return summary
	}
	if len(compact) > maxSummaryArgs {
		compact = compact[:maxSummaryArgs] + "…"
	}
	return summary + " with " + compact
}
