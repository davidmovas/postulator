package models

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	defaultSpendDays = 30
	maxSpendDays     = 366
	callSortField    = "createdAt"
)

func (s *Service) SpendReport(ctx context.Context, req SpendReportRequest) (SpendReportResponse, error) {
	if req.Days < 0 || req.Days > maxSpendDays {
		return SpendReportResponse{}, errors.New(errors.Invalid, "a spend report covers 1 to 366 days").
			WithDetail("field", "days")
	}

	resp := SpendReportResponse{RunID: strings.TrimSpace(req.RunID)}
	query := llm.SpendQuery{RunID: resp.RunID}
	if resp.RunID == "" {
		resp.Days = req.Days
		if resp.Days == 0 {
			resp.Days = defaultSpendDays
		}
		query.Since = s.now().AddDate(0, 0, -resp.Days)
		resp.Since = dto.NewTime(query.Since)
	}

	slices, err := s.spend.Aggregate(ctx, query)
	if err != nil {
		return SpendReportResponse{}, err
	}

	resp.Totals = totalsOf(slices)
	resp.Slices = make([]SpendSlice, 0, len(slices))
	for i := range slices {
		resp.Slices = append(resp.Slices, sliceView(slices[i]))
	}
	return resp, nil
}

func (s *Service) ListCalls(ctx context.Context, req ListCallsRequest) (paging.List[Call], error) {
	q := llm.CallQuery{
		RunID:          strings.TrimSpace(req.RunID),
		ConversationID: strings.TrimSpace(req.ConversationID),
		Desc:           true,
	}
	if q.RunID != "" && q.ConversationID != "" {
		return paging.List[Call]{}, errors.New(errors.Invalid, "name the run or the conversation, not both").
			WithDetail("field", "conversationId")
	}
	if req.Sort != nil {
		if req.Sort.Field != callSortField {
			return paging.List[Call]{}, errors.New(errors.Invalid, "calls are sorted by createdAt").
				WithDetail("field", "sort.field")
		}
		q.Desc = req.Sort.Desc
	}

	list, err := s.spend.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Call]{}, err
	}
	return application.MapList(list, callView), nil
}
