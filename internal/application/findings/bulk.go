package findings

import (
	"context"
	"strconv"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/domain/finding"
)

const bulkLimit = 1000

type ResolveManyRequest struct {
	SiteID     string   `json:"siteId"`
	Code       string   `json:"code"`
	Option     string   `json:"option"`
	FindingIDs []string `json:"findingIds"`
	PageTypeID string   `json:"pageTypeId,omitempty"`
	By         string   `json:"-"`
}

type ResolveManyResponse struct {
	Findings  []Finding                      `json:"findings"`
	Proposals *graph.PreviewFromPagesRequest `json:"proposals,omitempty"`
}

func (s *Service) ResolveMany(ctx context.Context, req ResolveManyRequest) (ResolveManyResponse, error) {
	by, err := chooser(req.By)
	if err != nil {
		return ResolveManyResponse{}, err
	}
	if req.SiteID == "" {
		return ResolveManyResponse{}, invalid("name the site whose findings to resolve", "siteId")
	}
	option := finding.Option(req.Option)
	if !option.Bulk() {
		return ResolveManyResponse{}, invalid("the option "+req.Option+" is chosen one finding at a time", "option")
	}
	code := finding.Code(req.Code)
	if !code.Valid() {
		return ResolveManyResponse{}, invalid("finding code is not recognized", "code")
	}
	if len(req.FindingIDs) > bulkLimit {
		return ResolveManyResponse{}, invalid("at most "+strconv.Itoa(bulkLimit)+" findings are resolved at once", "findingIds")
	}
	held, err := s.gathered(ctx, req.SiteID, req.FindingIDs, option)
	if err != nil {
		return ResolveManyResponse{}, err
	}
	if held[0].Code != code {
		return ResolveManyResponse{}, invalid("the findings are not of the code "+req.Code, "code")
	}
	places, err := s.deps.Places.Map(ctx, req.SiteID)
	if err != nil {
		return ResolveManyResponse{}, err
	}
	if option.Kind() == finding.Proposal {
		proposals, proposeErr := proposed(held)
		if proposeErr != nil {
			return ResolveManyResponse{}, proposeErr
		}
		out := ResolveManyResponse{Findings: make([]Finding, 0, len(held)), Proposals: proposals}
		for i := range held {
			out.Findings = append(out.Findings, placed(held[i], places))
		}
		return out, nil
	}
	chosen := choice{option: option, pageTypeID: req.PageTypeID, by: by}
	if refused := applicable(chosen, places); refused != nil {
		return ResolveManyResponse{}, refused
	}
	err = s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		for i := range held {
			current, openErr := s.open(c, req.SiteID, held[i].ID, option)
			if openErr != nil {
				return openErr
			}
			if changeErr := s.change(c, current, chosen); changeErr != nil {
				return changeErr
			}
		}
		return nil
	})
	if err != nil {
		return ResolveManyResponse{}, err
	}
	out := ResolveManyResponse{Findings: make([]Finding, 0, len(held))}
	for i := range held {
		settled, getErr := s.deps.Findings.Get(ctx, held[i].ID)
		if getErr != nil {
			return ResolveManyResponse{}, getErr
		}
		out.Findings = append(out.Findings, placed(settled, places))
	}
	return out, s.changed(req.SiteID)
}
