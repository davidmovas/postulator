package runs

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/run"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type EditRequest struct {
	SiteID string   `json:"siteId"`
	PageID string   `json:"pageId"`
	Edit   run.Edit `json:"edit"`
}

type EditResponse struct {
	RunID string `json:"runId"`
}

func (s *Service) StartEdit(ctx context.Context, req EditRequest) (EditResponse, error) {
	siteID, pageID := strings.TrimSpace(req.SiteID), strings.TrimSpace(req.PageID)
	if siteID == "" || pageID == "" {
		return EditResponse{}, invalid("an edit names its site and its page", "pageId")
	}
	if err := req.Edit.Validate(); err != nil {
		return EditResponse{}, err
	}
	page, err := s.pages.Get(ctx, pageID)
	if err != nil {
		return EditResponse{}, err
	}
	if page.SiteID != siteID {
		return EditResponse{}, errors.New(errors.NotFound, "the site holds no such page").WithDetail("pageId", pageID)
	}
	if page.WPID == nil {
		return EditResponse{}, invalid("the page "+page.Path+" is not on the site, so there is nothing to edit there", "pageId")
	}

	actor, ok := kctx.ActorFrom(ctx)
	if !ok {
		actor = kctx.ActorUser
	}
	queued, err := s.engine.Enqueue(ctx, run.Run{
		ID:          id.New(),
		SiteID:      siteID,
		Kind:        run.KindEdit,
		Targets:     []string{pageID},
		Recipe:      run.EditRecipe(req.Edit),
		PublishMode: run.PublishLive,
		CreatedBy:   actor,
	})
	if err != nil {
		return EditResponse{}, err
	}
	return EditResponse{RunID: queued.ID}, nil
}
