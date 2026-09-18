package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type conversationStore interface {
	Insert(ctx context.Context, c domainagent.Conversation) error
	Update(ctx context.Context, c domainagent.Conversation) error
	Get(ctx context.Context, id string) (domainagent.Conversation, error)
	List(ctx context.Context, q domainagent.ConversationQuery, page paging.Request) (paging.List[domainagent.Conversation], error)
}

type messageStore interface {
	Append(ctx context.Context, m domainagent.Message) (domainagent.Message, error)
	List(ctx context.Context, q domainagent.MessageQuery, page paging.Request) (paging.List[domainagent.Message], error)
}

type actionStore interface {
	Get(ctx context.Context, id string) (domainagent.PendingAction, error)
	Transition(ctx context.Context, id string, from, to domainagent.ActionStatus, result json.RawMessage,
		failure string, now time.Time) (bool, error)
	List(ctx context.Context, q domainagent.ActionQuery, page paging.Request) (paging.List[domainagent.PendingAction], error)
}

type callStore interface {
	Insert(ctx context.Context, c domainagent.ToolCall) error
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type overviewReader interface {
	SiteOverview(ctx context.Context, req reports.SiteOverviewRequest) (reports.SiteOverviewResponse, error)
}

type templateReader interface {
	ListTemplates(ctx context.Context, req templates.ListTemplatesRequest) (paging.List[templates.Template], error)
}

type profileResolver interface {
	Resolve(ctx context.Context, siteID string, role domainllm.Role, templateProfiles map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error)
}

type Runner interface {
	Run(ctx context.Context, spec RunSpec) (RunResult, error)
}

type Deps struct {
	Conversations conversationStore
	Messages      messageStore
	Actions       actionStore
	Calls         callStore
	Sites         siteReader
	Reports       overviewReader
	Templates     templateReader
	Profiles      profileResolver
	Registry      *tools.Registry
	Runner        Runner
	Turns         *Turns
	Publisher     application.Publisher
	Clock         clock.Clock
	Allowed       []string
	LoopLimit     int
	HistoryBudget int
	MaxToolResult int
}

type Service struct {
	deps Deps
}

func New(deps Deps) *Service {
	if deps.Turns == nil {
		deps.Turns = NewTurns()
	}
	if deps.LoopLimit <= 0 {
		deps.LoopLimit = DefaultLoopLimit
	}
	if deps.HistoryBudget <= 0 {
		deps.HistoryBudget = DefaultHistoryBudgetChars
	}
	if deps.MaxToolResult <= 0 {
		deps.MaxToolResult = DefaultMaxToolResultBytes
	}
	return &Service{deps: deps}
}

func (s *Service) Close() {
	s.deps.Turns.Close()
}

func (s *Service) now() time.Time {
	return s.deps.Clock.Now().UTC().Truncate(time.Second)
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
