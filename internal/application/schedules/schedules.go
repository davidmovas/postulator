package schedules

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	domainrun "github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/schedule"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		return CreateResponse{}, invalid("a schedule belongs to a site", "siteId")
	}
	if _, err := s.deps.Sites.Get(ctx, siteID); err != nil {
		return CreateResponse{}, err
	}

	status, err := statusOf(req.Status)
	if err != nil {
		return CreateResponse{}, err
	}

	actor, ok := kctx.ActorFrom(ctx)
	if !ok {
		actor = kctx.ActorUser
	}

	now := s.now()
	candidate := schedule.Schedule{
		ID:     id.New(),
		SiteID: siteID,
		Name:   req.Name,
		Cron:   req.Cron,
		Query: schedule.TargetQuery{
			EntityID: optional(req.EntityID), Status: status, Limit: req.Limit,
		},
		TemplateID:  strings.TrimSpace(req.TemplateID),
		Recipe:      recipeOf(req.Steps),
		PublishMode: domainrun.PublishMode(req.PublishMode),
		Budget:      domainrun.Budget{MaxUSD: req.MaxUSD, MaxTokens: req.MaxTokens},
		Enabled:     req.Enabled,
		CreatedBy:   actor,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.IntervalMinutes > 0 {
		interval := time.Duration(req.IntervalMinutes) * time.Minute
		candidate.Interval = &interval
	}

	created, err := s.arm(candidate, now)
	if err != nil {
		return CreateResponse{}, err
	}
	if insertErr := s.deps.Schedules.Insert(ctx, created); insertErr != nil {
		return CreateResponse{}, insertErr
	}
	if publishErr := s.changed(created.SiteID); publishErr != nil {
		return CreateResponse{}, publishErr
	}
	return CreateResponse{Schedule: view(created)}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (UpdateResponse, error) {
	current, err := s.require(ctx, req.ID)
	if err != nil {
		return UpdateResponse{}, err
	}

	next := current
	if req.Name != nil {
		next.Name = *req.Name
	}
	if req.Cron != nil {
		next.Cron = *req.Cron
		next.Interval = nil
	}
	if req.IntervalMinutes != nil {
		interval := time.Duration(*req.IntervalMinutes) * time.Minute
		next.Interval = &interval
		next.Cron = ""
	}
	if req.EntityID != nil {
		next.Query.EntityID = optional(*req.EntityID)
	}
	if req.Status != nil {
		status, statusErr := statusOf(*req.Status)
		if statusErr != nil {
			return UpdateResponse{}, statusErr
		}
		next.Query.Status = status
	}
	if req.Limit != nil {
		next.Query.Limit = *req.Limit
	}
	if req.TemplateID != nil {
		next.TemplateID = strings.TrimSpace(*req.TemplateID)
	}
	if req.Steps != nil {
		next.Recipe = recipeOf(req.Steps)
	}
	if req.PublishMode != nil {
		next.PublishMode = domainrun.PublishMode(*req.PublishMode)
	}
	if req.MaxUSD != nil {
		next.Budget.MaxUSD = *req.MaxUSD
	}
	if req.MaxTokens != nil {
		next.Budget.MaxTokens = *req.MaxTokens
	}

	now := s.now()
	next.UpdatedAt = now
	armed, err := s.arm(next, now)
	if err != nil {
		return UpdateResponse{}, err
	}
	if updateErr := s.deps.Schedules.Update(ctx, armed); updateErr != nil {
		return UpdateResponse{}, updateErr
	}
	if publishErr := s.changed(armed.SiteID); publishErr != nil {
		return UpdateResponse{}, publishErr
	}
	return UpdateResponse{Schedule: view(armed)}, nil
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (DeleteResponse, error) {
	current, err := s.require(ctx, req.ID)
	if err != nil {
		return DeleteResponse{}, err
	}
	if deleteErr := s.deps.Schedules.Delete(ctx, current.ID); deleteErr != nil {
		return DeleteResponse{}, deleteErr
	}
	return DeleteResponse{}, s.changed(current.SiteID)
}

func (s *Service) Get(ctx context.Context, req GetRequest) (GetResponse, error) {
	found, err := s.require(ctx, req.ID)
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Schedule: view(found)}, nil
}

func (s *Service) List(ctx context.Context, req ListRequest) (paging.List[Schedule], error) {
	found, err := s.deps.Schedules.List(ctx, schedule.Query{
		SiteID: strings.TrimSpace(req.SiteID), Enabled: req.Enabled,
	}, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Schedule]{}, err
	}
	return application.MapList(found, view), nil
}

func (s *Service) Enable(ctx context.Context, req EnableRequest) (EnableResponse, error) {
	switched, err := s.switchTo(ctx, req.ID, true)
	if err != nil {
		return EnableResponse{}, err
	}
	return EnableResponse{Schedule: view(switched)}, nil
}

func (s *Service) Disable(ctx context.Context, req DisableRequest) (DisableResponse, error) {
	switched, err := s.switchTo(ctx, req.ID, false)
	if err != nil {
		return DisableResponse{}, err
	}
	return DisableResponse{Schedule: view(switched)}, nil
}

func (s *Service) switchTo(ctx context.Context, scheduleID string, enabled bool) (schedule.Schedule, error) {
	current, err := s.require(ctx, scheduleID)
	if err != nil {
		return schedule.Schedule{}, err
	}

	now := s.now()
	current.Enabled = enabled
	current.UpdatedAt = now

	armed, err := s.arm(current, now)
	if err != nil {
		return schedule.Schedule{}, err
	}
	if updateErr := s.deps.Schedules.Update(ctx, armed); updateErr != nil {
		return schedule.Schedule{}, updateErr
	}
	if publishErr := s.changed(armed.SiteID); publishErr != nil {
		return schedule.Schedule{}, publishErr
	}
	return armed, nil
}

func (s *Service) require(ctx context.Context, scheduleID string) (schedule.Schedule, error) {
	trimmed := strings.TrimSpace(scheduleID)
	if trimmed == "" {
		return schedule.Schedule{}, invalid("a schedule is needed", "id")
	}
	return s.deps.Schedules.Get(ctx, trimmed)
}

func (s *Service) arm(candidate schedule.Schedule, now time.Time) (schedule.Schedule, error) {
	built, err := schedule.New(candidate)
	if err != nil {
		return schedule.Schedule{}, err
	}
	if !built.Enabled {
		built.NextRunAt = nil
		return built, nil
	}

	next, err := built.NextAfter(now)
	if err != nil {
		return schedule.Schedule{}, err
	}
	built.NextRunAt = &next
	return built, nil
}

func statusOf(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !pagemap.Status(trimmed).Valid() {
		return "", invalid("the target status is not recognized", "status")
	}
	return trimmed, nil
}

func optional(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
