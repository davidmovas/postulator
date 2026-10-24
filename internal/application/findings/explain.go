package findings

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	unwritten  = "; nothing is written to the site"
	pagesNamed = 3
)

type ExplainRequest struct {
	SiteID     string   `json:"siteId,omitempty"`
	FindingIDs []string `json:"findingIds"`
	Option     string   `json:"option"`
	PageID     string   `json:"pageId,omitempty"`
	PageTypeID string   `json:"pageTypeId,omitempty"`
}

type Explanation struct {
	Sentence string `json:"sentence"`
	Kind     string `json:"kind"`
}

func (s *Service) Explain(ctx context.Context, req ExplainRequest) (Explanation, error) {
	option := finding.Option(req.Option)
	if option.Kind() == "" {
		return Explanation{}, invalid("the option is not recognized", "option")
	}
	held, err := s.gathered(ctx, req.SiteID, req.FindingIDs, option)
	if err != nil {
		return Explanation{}, err
	}
	places, err := s.deps.Places.Map(ctx, held[0].SiteID)
	if err != nil {
		return Explanation{}, err
	}
	said, err := sentence(held, option, req, places)
	if err != nil {
		return Explanation{}, err
	}
	return Explanation{Sentence: said, Kind: string(option.Kind())}, nil
}

func (s *Service) gathered(ctx context.Context, siteID string, ids []string, option finding.Option) ([]finding.Finding, error) {
	if len(ids) == 0 {
		return nil, invalid("name the findings", "findingIds")
	}
	if len(ids) > 1 && !option.Bulk() {
		return nil, invalid("the option "+string(option)+" is chosen one finding at a time", "option")
	}
	held := make([]finding.Finding, 0, len(ids))
	named := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, twice := named[id]; twice {
			return nil, invalid("the finding "+id+" is named twice", "findingIds")
		}
		named[id] = struct{}{}
		one, err := s.open(ctx, siteID, id, option)
		if err != nil {
			return nil, err
		}
		if len(held) > 0 && (one.Code != held[0].Code || one.SiteID != held[0].SiteID) {
			return nil, invalid("the findings resolved together share one code and one site", "findingIds")
		}
		held = append(held, one)
	}
	return held, nil
}

func sentence(held []finding.Finding, option finding.Option, req ExplainRequest, places contextpack.Map) (string, error) {
	where := pagesOf(held, places)
	about := "the finding " + string(held[0].Code) + " on " + where
	stays := ": it stays dismissed while what it names stands"
	if len(held) > 1 {
		about = "the " + strconv.Itoa(len(held)) + " findings " + string(held[0].Code) + " on " + where
		stays = ": each stays dismissed while what it names stands"
	}
	switch option {
	case finding.Ignore:
		return "Dismiss " + about + stays + unwritten, nil
	case finding.Accept:
		return "Write the entity card the agent proposed in " + about + unwritten, nil
	case finding.AcceptSite:
		return "Take what the site holds as the model's for " + about + unwritten, nil
	case finding.AcceptWorkbook:
		return "Pin the parent of " + where + " to the workbook's: the model's parent follows the sheet from the next sync" +
			unwritten, nil
	case finding.MapToPage:
		target, found := places.Place(req.PageID)
		if !found {
			return "", errors.New(errors.NotFound, "the site's map holds no such page").WithDetail("pageId", req.PageID)
		}
		return "Map the workbook row of " + about + " onto " + target.Ref.Path + ", and delete the planned page it replaces " +
			"when no run names it" + unwritten, nil
	case finding.SetType:
		if req.PageTypeID == "" {
			return "", invalid("name the page type to set", "pageTypeId")
		}
		kind, found := places.Type(req.PageTypeID)
		if !found {
			return "", errors.New(errors.NotFound, "the site has no such page type").WithDetail("pageTypeId", req.PageTypeID)
		}
		return "Type " + where + " as " + kind.Name + ", which settles " + about + unwritten, nil
	case finding.RewriteField:
		key, err := rewritten(held[0])
		if err != nil {
			return "", err
		}
		return "Open a run start that rewrites the " + string(key) + " of " + where +
			", the client's value included: the start shows the estimate first, and the finding closes itself once the run has written it", nil
	case finding.RelinkPage:
		return "Open a relink run start for " + where +
			": the start shows the estimate first, and the finding closes itself once the run has placed the links", nil
	case finding.ProposeEntity:
		return "Open the entity proposals named from the site for " + where +
			": the model takes only the ones a person accepts, and each finding closes itself once an entity carries its page" +
			unwritten, nil
	default:
		return "Start a full sync of the site that renders every page again: the finding closes itself once its cause is gone", nil
	}
}

func rewritten(held finding.Finding) (field.Key, error) {
	if held.Code != finding.FieldChangedOnSite {
		return field.Title, nil
	}
	values, err := valuesOf(held)
	if err != nil || values.Key == "" {
		return "", invalid("the finding names no field", "findingId")
	}
	return field.Key(values.Key), nil
}

func pagesOf(held []finding.Finding, places contextpack.Map) string {
	paths := make([]string, 0, len(held))
	for i := range held {
		if held[i].PageID == nil {
			continue
		}
		if place, found := places.Place(*held[i].PageID); found {
			paths = append(paths, place.Ref.Path)
		}
	}
	switch {
	case len(paths) == 0:
		return "the site"
	case len(paths) <= pagesNamed:
		return strings.Join(paths, ", ")
	default:
		return strings.Join(paths[:pagesNamed], ", ") + " and " + strconv.Itoa(len(paths)-pagesNamed) + " more"
	}
}
