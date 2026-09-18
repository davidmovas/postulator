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

const maxEvents = 500

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
	return application.MapList(list, itemView), nil
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
	case limit > maxEvents:
		limit = maxEvents
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
