package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/llm"
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
	Delete(ctx context.Context, id string) error
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

type completer interface {
	Complete(ctx context.Context, req llm.Request) (llm.Response, error)
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
	LLM           completer
	Registry      *tools.Registry
	Runner        Runner
	Turns         *Turns
	TurnTimeout   func() time.Duration
	Publisher     application.Publisher
	Clock         clock.Clock
	Allowed       []string
	LoopLimit     func() int
	HistoryBudget func() int
	MaxToolResult func() int
}

type Service struct {
	deps Deps
}

func New(deps Deps) *Service {
	if deps.Turns == nil {
		deps.Turns = NewTurns(deps.TurnTimeout)
	}
	service := &Service{deps: deps}
	deps.Turns.Resuming(service.resumeQueued)
	return service
}

func (s *Service) resumeQueued(conversationID, text string) {
	ctx := context.WithoutCancel(context.Background())
	conversation, err := s.deps.Conversations.Get(ctx, conversationID)
	if err != nil {
		s.deps.Turns.Note(err)
		return
	}
	if _, err = s.turn(ctx, conversation, text); err != nil {
		s.deps.Turns.Note(err)
	}
}

func (s *Service) Close() {
	s.deps.Turns.Close()
}

func chosen(read func() int, fallback int) int {
	if read == nil {
		return fallback
	}
	if value := read(); value > 0 {
		return value
	}
	return fallback
}

func (s *Service) loopLimit() int {
	return chosen(s.deps.LoopLimit, DefaultLoopLimit)
}

func (s *Service) historyBudget() int {
	return chosen(s.deps.HistoryBudget, DefaultHistoryBudgetChars)
}

func (s *Service) maxToolResult() int {
	return chosen(s.deps.MaxToolResult, DefaultMaxToolResultBytes)
}

func (s *Service) now() time.Time {
	return s.deps.Clock.Now().UTC().Truncate(time.Second)
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
