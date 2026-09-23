package runs

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func (s *Service) Get(ctx context.Context, req GetRequest) (GetResponse, error) {
	record, err := s.runs.Get(ctx, req.RunID)
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Run: runView(record)}, nil
}

func (s *Service) List(ctx context.Context, req ListRequest) (paging.List[Run], error) {
	status, err := statusFilter(req.Status)
	if err != nil {
		return paging.List[Run]{}, err
	}
	kind, err := kindFilter(req.Kind)
	if err != nil {
		return paging.List[Run]{}, err
	}
	key, desc, err := sortOf(req.Sort)
	if err != nil {
		return paging.List[Run]{}, err
	}

	q := run.Query{SiteID: strings.TrimSpace(req.SiteID), Status: status, Kind: kind, Sort: key, Desc: desc}
	list, err := s.runs.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Run]{}, err
	}
	return application.MapList(list, runView), nil
}

func (s *Service) ListItems(ctx context.Context, req ListItemsRequest) (paging.List[Item], error) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return paging.List[Item]{}, invalid("an item listing needs a run", "runId")
	}
	status, err := statusFilter(req.Status)
	if err != nil {
		return paging.List[Item]{}, err
	}

	desc := req.Sort != nil && req.Sort.Desc
	list, err := s.items.List(ctx, run.ItemQuery{RunID: runID, Status: status, Desc: desc},
		application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Item]{}, err
	}

	blocked, err := s.retryBlocks(ctx, list.Items)
	if err != nil {
		return paging.List[Item]{}, err
	}
	awaited, err := s.awaitedParents(ctx, runID, list.Items)
	if err != nil {
		return paging.List[Item]{}, err
	}
	return application.MapList(list, func(item run.Item) Item {
		return itemView(item, blocked[item.ID], awaited[item.ID])
	}), nil
}

func (s *Service) awaitedParents(ctx context.Context, runID string, items []run.Item) (map[string]*AwaitedParent, error) {
	awaited := make(map[string]*AwaitedParent)

	var siblings []run.Item
	for i := range items {
		if items[i].PauseReason != run.PauseAwaitingParent {
			continue
		}
		page, err := s.pages.Get(ctx, items[i].TargetID)
		if err != nil {
			return nil, err
		}
		if page.ParentPageID == nil {
			continue
		}
		parent, err := s.pages.Get(ctx, *page.ParentPageID)
		if err != nil {
			return nil, err
		}

		if siblings == nil {
			if siblings, err = s.items.ByRun(ctx, runID); err != nil {
				return nil, err
			}
		}
		ref := &AwaitedParent{PageID: parent.ID, Path: parent.Path}
		for j := range siblings {
			if siblings[j].TargetID != parent.ID {
				continue
			}
			ref.ItemID = siblings[j].ID
			ref.ItemStatus = string(siblings[j].Status)
			ref.Step = siblings[j].CurrentStep
		}
		awaited[items[i].ID] = ref
	}
	return awaited, nil
}

func (s *Service) retryBlocks(ctx context.Context, items []run.Item) (map[string]run.RetryBlockedReason, error) {
	blocked := make(map[string]run.RetryBlockedReason, len(items))
	if len(items) == 0 {
		return blocked, nil
	}

	consumed := false
	ids := make([]string, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
		if def, known := s.steps.Lookup(items[i].CurrentStep); known && len(def.Requires) > 0 {
			consumed = true
		}
	}
	if !consumed {
		return blocked, nil
	}

	purged, err := s.artifacts.PurgedByItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		def, known := s.steps.Lookup(items[i].CurrentStep)
		if !known {
			continue
		}
		if reason := run.RetryBlocked(def.Requires, purged[items[i].ID]); reason != "" {
			blocked[items[i].ID] = reason
		}
	}
	return blocked, nil
}

func (s *Service) ListEvents(ctx context.Context, req ListEventsRequest) (ListEventsResponse, error) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return ListEventsResponse{}, invalid("an event listing needs a run", "runId")
	}
	if req.SinceSeq < 0 {
		return ListEventsResponse{}, invalid("the sequence to read from must not be negative", "sinceSeq")
	}

	limit := req.Limit
	switch {
	case limit <= 0:
		limit = dto.DefaultLimit
	case limit > dto.MaxLimit:
		limit = dto.MaxLimit
	}

	stored, err := s.events.List(ctx, runID, req.SinceSeq, limit)
	if err != nil {
		return ListEventsResponse{}, err
	}

	out := make([]Event, 0, len(stored))
	for i := range stored {
		out = append(out, eventView(stored[i]))
	}
	return ListEventsResponse{Events: out}, nil
}

func (s *Service) ListArtifacts(ctx context.Context, req ListArtifactsRequest) (ListArtifactsResponse, error) {
	itemID := strings.TrimSpace(req.ItemID)
	if itemID == "" {
		return ListArtifactsResponse{}, invalid("an artifact listing needs a run item", "itemId")
	}

	stored, err := s.artifacts.ByItem(ctx, itemID)
	if err != nil {
		return ListArtifactsResponse{}, err
	}

	out := make([]ArtifactSummary, 0, len(stored))
	for i := range stored {
		out = append(out, artifactSummaryView(stored[i]))
	}
	return ListArtifactsResponse{Artifacts: out}, nil
}

func (s *Service) GetArtifact(ctx context.Context, req GetArtifactRequest) (GetArtifactResponse, error) {
	itemID := strings.TrimSpace(req.ItemID)
	if itemID == "" {
		return GetArtifactResponse{}, invalid("an artifact belongs to a run item", "itemId")
	}
	kind := run.ArtifactKind(req.Kind)
	if !kind.Valid() {
		return GetArtifactResponse{}, invalid("artifact kind is not recognized", "kind")
	}

	stored, err := s.artifacts.ByItem(ctx, itemID)
	if err != nil {
		return GetArtifactResponse{}, err
	}

	var latest *run.Artifact
	for i := range stored {
		if stored[i].Kind != kind {
			continue
		}
		if latest == nil || !stored[i].CreatedAt.Before(latest.CreatedAt) {
			latest = &stored[i]
		}
	}
	if latest == nil {
		return GetArtifactResponse{}, errors.New(errors.NotFound, "the run item has no artifact of this kind").
			WithDetail("itemId", itemID).WithDetail("kind", req.Kind)
	}
	return GetArtifactResponse{Artifact: artifactView(*latest)}, nil
}
