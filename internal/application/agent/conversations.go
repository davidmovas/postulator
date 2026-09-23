package agent

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/templates"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const templatesInPrompt = 20

func (s *Service) CreateConversation(ctx context.Context, req CreateConversationRequest) (CreateConversationResponse, error) {
	now := s.now()
	record := domainagent.Conversation{
		ID: id.New(), Title: req.Title, Mode: domainagent.Mode(req.Mode),
		TitleSettled: strings.TrimSpace(req.Title) != "", CreatedAt: now, UpdatedAt: now,
	}
	if siteID := strings.TrimSpace(req.SiteID); siteID != "" {
		if _, err := s.deps.Sites.Get(ctx, siteID); err != nil {
			return CreateConversationResponse{}, err
		}
		record.SiteID = &siteID
	}

	created, err := domainagent.NewConversation(record)
	if err != nil {
		return CreateConversationResponse{}, err
	}
	if insertErr := s.deps.Conversations.Insert(ctx, created); insertErr != nil {
		return CreateConversationResponse{}, insertErr
	}
	return CreateConversationResponse{Conversation: conversationView(created)}, nil
}

func (s *Service) SetMode(ctx context.Context, req SetModeRequest) (SetModeResponse, error) {
	conversationID := strings.TrimSpace(req.ConversationID)
	if conversationID == "" {
		return SetModeResponse{}, invalid("a conversation is needed", "conversationId")
	}

	mode := domainagent.Mode(req.Mode)
	if !mode.Valid() {
		return SetModeResponse{}, invalid("the mode must be confirm or autonomous", "mode")
	}

	current, err := s.deps.Conversations.Get(ctx, conversationID)
	if err != nil {
		return SetModeResponse{}, err
	}

	current.Mode = mode
	current.UpdatedAt = s.now()
	if updateErr := s.deps.Conversations.Update(ctx, current); updateErr != nil {
		return SetModeResponse{}, updateErr
	}
	return SetModeResponse{Conversation: conversationView(current)}, nil
}

func (s *Service) RenameConversation(ctx context.Context, req RenameConversationRequest) (RenameConversationResponse, error) {
	conversationID := strings.TrimSpace(req.ConversationID)
	if conversationID == "" {
		return RenameConversationResponse{}, invalid("a conversation is needed", "conversationId")
	}
	title := domainagent.Title(req.Title)
	if title == "" {
		return RenameConversationResponse{}, invalid("a conversation needs a title", "title")
	}

	current, err := s.deps.Conversations.Get(ctx, conversationID)
	if err != nil {
		return RenameConversationResponse{}, err
	}

	current.Title = title
	current.TitleSettled = true
	current.UpdatedAt = s.now()
	if updateErr := s.deps.Conversations.Update(ctx, current); updateErr != nil {
		return RenameConversationResponse{}, updateErr
	}
	return RenameConversationResponse{Conversation: conversationView(current)}, nil
}

func (s *Service) DeleteConversation(ctx context.Context, req DeleteConversationRequest) (DeleteConversationResponse, error) {
	conversationID := strings.TrimSpace(req.ConversationID)
	if conversationID == "" {
		return DeleteConversationResponse{}, invalid("a conversation is needed", "conversationId")
	}

	s.deps.Turns.Cancel(conversationID)
	if err := s.deps.Conversations.Delete(ctx, conversationID); err != nil {
		return DeleteConversationResponse{}, err
	}
	return DeleteConversationResponse{}, nil
}

func (s *Service) ListConversations(ctx context.Context, req ListConversationsRequest) (paging.List[Conversation], error) {
	found, err := s.deps.Conversations.List(ctx, domainagent.ConversationQuery{
		SiteID: strings.TrimSpace(req.SiteID), Desc: true,
	}, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Conversation]{}, err
	}
	return application.MapList(found, conversationView), nil
}

func (s *Service) ListMessages(ctx context.Context, req ListMessagesRequest) (paging.List[Message], error) {
	conversationID := strings.TrimSpace(req.ConversationID)
	if conversationID == "" {
		return paging.List[Message]{}, invalid("a conversation is needed", "conversationId")
	}

	found, err := s.deps.Messages.List(ctx, domainagent.MessageQuery{ConversationID: conversationID},
		application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Message]{}, err
	}

	settled, err := s.toolStatuses(ctx, conversationID, found.Items)
	if err != nil {
		return paging.List[Message]{}, err
	}
	return application.MapList(found, func(m domainagent.Message) Message {
		return messageView(m, settled[m.CallID])
	}), nil
}

func (s *Service) toolStatuses(ctx context.Context, conversationID string,
	listed []domainagent.Message) (map[string]string, error) {
	answered := false
	for i := range listed {
		if listed[i].Role == domainagent.RoleTool && listed[i].CallID != "" {
			answered = true
			break
		}
	}
	if !answered || s.deps.Calls == nil {
		return nil, nil
	}

	recorded, err := s.deps.Calls.ByConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	settled := make(map[string]string, len(recorded))
	for i := range recorded {
		settled[recorded[i].CallID] = string(recorded[i].Status)
	}
	return settled, nil
}

func (s *Service) ListPendingActions(ctx context.Context, req ListPendingActionsRequest) (paging.List[PendingAction], error) {
	query := domainagent.ActionQuery{ConversationID: strings.TrimSpace(req.ConversationID), Desc: true}
	if raw := strings.TrimSpace(req.Status); raw != "" {
		status := domainagent.ActionStatus(raw)
		if !status.Valid() {
			return paging.List[PendingAction]{}, invalid("the action status is not recognized", "status")
		}
		query.Status = &status
	}

	found, err := s.deps.Actions.List(ctx, query, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[PendingAction]{}, err
	}
	return application.MapList(found, actionView), nil
}

func (s *Service) Cancel(_ context.Context, req CancelRequest) (CancelResponse, error) {
	conversationID := strings.TrimSpace(req.ConversationID)
	if conversationID == "" {
		return CancelResponse{}, invalid("a conversation is needed", "conversationId")
	}
	return CancelResponse{Cancelled: s.deps.Turns.Cancel(conversationID)}, nil
}

func (s *Service) siteContext(ctx context.Context, conversation domainagent.Conversation) (SiteContext, error) {
	built := SiteContext{
		Mode:      string(conversation.Mode),
		Templates: []string{},
	}

	listed, err := s.deps.Templates.ListTemplates(ctx, templates.ListTemplatesRequest{})
	if err != nil {
		return SiteContext{}, err
	}
	for i := range listed.Items {
		if i >= templatesInPrompt {
			break
		}
		built.Templates = append(built.Templates, listed.Items[i].Name+" ("+listed.Items[i].PageKind+")")
	}

	if conversation.SiteID == nil {
		return built, nil
	}

	owner, err := s.deps.Sites.Get(ctx, *conversation.SiteID)
	if err != nil {
		return SiteContext{}, err
	}
	built.SiteName = owner.Name

	overview, err := s.deps.Reports.SiteOverview(ctx, reports.SiteOverviewRequest{SiteID: *conversation.SiteID})
	if err != nil {
		return SiteContext{}, err
	}
	built.Entities = overview.Entities.Total
	built.Pages = overview.Pages.Total
	built.Published = overview.Pages.ByStatus["published"]
	built.Unmapped = overview.Pages.Unmapped
	return built, nil
}

func (s *Service) conversation(ctx context.Context, conversationID string) (domainagent.Conversation, error) {
	trimmed := strings.TrimSpace(conversationID)
	if trimmed == "" {
		return domainagent.Conversation{}, invalid("a conversation is needed", "conversationId")
	}

	found, err := s.deps.Conversations.Get(ctx, trimmed)
	if err != nil {
		return domainagent.Conversation{}, err
	}
	if s.deps.Runner == nil {
		return domainagent.Conversation{}, errors.New(errors.Invalid, "no model runner is wired for the agent")
	}
	return found, nil
}
