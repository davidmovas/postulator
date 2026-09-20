package agent

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/events"
	llmport "github.com/davidmovas/postulator/internal/application/llm"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
)

const (
	titleTokens  = 32
	titleContext = 2000
)

type titlePrompt struct {
	Question string
	Answer   string
}

func clip(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
}

func (s *Service) settle(ctx context.Context, conversationID, question, answer string) {
	conversation, err := s.deps.Conversations.Get(ctx, conversationID)
	if err != nil {
		s.deps.Turns.Note(err)
		return
	}
	if conversation.TitleSettled {
		return
	}

	conversation.TitleSettled = true
	if suggested, ok := s.suggestTitle(ctx, conversation, question, answer); ok {
		conversation.Title = suggested
	}
	conversation.UpdatedAt = s.now()

	if err = s.deps.Conversations.Update(ctx, conversation); err != nil {
		s.deps.Turns.Note(err)
		return
	}
	s.emit(events.AgentTitled, events.AgentTitledPayload{
		ConversationID: conversation.ID, Title: conversation.Title,
	})
}

func (s *Service) suggestTitle(ctx context.Context, conversation domainagent.Conversation, question, answer string) (string, bool) {
	ref, err := s.deps.Profiles.Resolve(ctx, siteOf(conversation), domainllm.RoleTitler, nil)
	if err != nil {
		return "", false
	}

	system, user, err := prompts.Render("title", titlePrompt{
		Question: clip(question, titleContext), Answer: clip(answer, titleContext),
	})
	if err != nil {
		return "", false
	}

	response, err := s.deps.LLM.Complete(ctx, llmport.Request{
		Ref:       ref,
		System:    system,
		Messages:  []llmport.Message{{Role: llmport.RoleUser, Text: user}},
		MaxTokens: titleTokens,
		Meta:      llmport.CallMeta{ConversationID: conversation.ID, Step: "title"},
	})
	if err != nil {
		return "", false
	}
	return domainagent.SuggestedTitle(response.Text)
}
