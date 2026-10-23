package tools

import (
	"context"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	findingsName = "findings"
	opList       = "list"
	opResolve    = "resolve"
)

type findingsArgs struct {
	Op           string `json:"op,omitempty" enum:"list,resolve" description:"Default list"`
	Status       string `json:"status,omitempty" enum:"open,resolved,dismissed" description:"list: default open"`
	Code         string `json:"code,omitempty" description:"list: only this code"`
	PageID       string `json:"pageId,omitempty" description:"list: one page's; map_to_page: the row's page"`
	FindingID    string `json:"findingId,omitempty" description:"resolve: id from list"`
	Option       string `json:"option,omitempty" description:"resolve: one the finding offers"`
	PageTypeID   string `json:"pageTypeId,omitempty" description:"set_type: id from pagetypes_list"`
	ProposalHash string `json:"proposalHash,omitempty" description:"accept: from graph_propose_cards"`
	Cursor       string `json:"cursor,omitempty" description:"list: nextCursor of the last call"`
	Limit        int    `json:"limit,omitempty" description:"list: default 50"`
}

type FindingsResponse struct {
	List    *paging.List[findings.Finding] `json:"list,omitempty"`
	Resolve *findings.ResolveResponse      `json:"resolve,omitempty"`
}

func findingsTool(deps Deps) Tool {
	tool := confirming(newSiteTool(Def{
		Name: findingsName,
		Description: "List what the model found that disagrees, or resolve a finding: a model option changes the model, " +
			"a run option answers the request to pass to runs_start or sync_site.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in findingsArgs) (FindingsResponse, error) {
		switch in.Op {
		case opList, "":
			status := in.Status
			if status == "" {
				status = string(finding.Open)
			}
			listed, err := deps.Findings.List(ctx, findings.ListRequest{
				SiteID: b.SiteID, Status: status, Code: in.Code, PageID: in.PageID, Cursor: in.Cursor, Limit: in.Limit,
			})
			if err != nil {
				return FindingsResponse{}, err
			}
			return FindingsResponse{List: &listed}, nil
		case opResolve:
			resolved, err := deps.Findings.Resolve(ctx, findings.ResolveRequest{
				SiteID: b.SiteID, FindingID: in.FindingID, Option: in.Option, PageID: in.PageID,
				PageTypeID: in.PageTypeID, ProposalHash: in.ProposalHash, By: finding.ResolvedByAgent,
			})
			if err != nil {
				return FindingsResponse{}, err
			}
			return FindingsResponse{Resolve: &resolved}, nil
		default:
			return FindingsResponse{}, errors.New(errors.Invalid, "findings takes the op list or resolve").WithDetail("field", "op")
		}
	}), func(ctx context.Context, b Binding, in findingsArgs) (string, error) {
		if b.Mode != agent.ModeConfirm || in.Op != opResolve || finding.Option(in.Option).Kind() != finding.ModelChange {
			return "", nil
		}
		return deps.resolution(ctx, b, in)
	})
	tool.RiskOf = func(args json.RawMessage) Risk {
		var in findingsArgs
		if err := json.Unmarshal(args, &in); err == nil && in.Op == opResolve && finding.Option(in.Option).Kind() != finding.Run {
			return RiskWrite
		}
		return RiskRead
	}
	codes := make([]string, 0, len(finding.Codes()))
	for _, code := range finding.Codes() {
		codes = append(codes, string(code))
	}
	options := make([]string, 0, len(finding.Options()))
	for _, option := range finding.Options() {
		options = append(options, string(option))
	}
	return enumOn(enumOn(tool, "code", codes), "option", options)
}

const unwritten = "; nothing is written to the site"

func (deps Deps) resolution(ctx context.Context, b Binding, in findingsArgs) (string, error) {
	held, err := deps.Findings.Get(ctx, findings.GetRequest{SiteID: b.SiteID, FindingID: in.FindingID})
	if err != nil {
		return "", err
	}
	where := "the site"
	if held.PageID != "" {
		if where, err = deps.pathOf(ctx, held.PageID); err != nil {
			return "", err
		}
	}
	about := "the finding " + held.Code + " on " + where
	switch finding.Option(in.Option) {
	case finding.Ignore:
		return "Dismiss " + about + ": it stays dismissed while what it names stands" + unwritten, nil
	case finding.Accept:
		return "Write the entity card the agent proposed in " + about + unwritten, nil
	case finding.AcceptSite:
		return "Take what the site holds as the model's for " + about + unwritten, nil
	case finding.AcceptWorkbook:
		return "Pin the parent of " + where + " to the workbook's: the model's parent follows the sheet from the next sync" +
			unwritten, nil
	case finding.MapToPage:
		target, pathErr := deps.pathOf(ctx, in.PageID)
		if pathErr != nil {
			return "", pathErr
		}
		return "Map the workbook row of " + about + " onto " + target + ", and delete the planned page it replaces " +
			"when no run names it" + unwritten, nil
	case finding.SetType:
		kind, typeErr := deps.PageTypes.Get(ctx, pagetypes.GetRequest{ID: in.PageTypeID})
		if typeErr != nil {
			return "", typeErr
		}
		return "Type " + where + " as " + kind.Name + ", which settles " + about + unwritten, nil
	default:
		return "", nil
	}
}

func (deps Deps) pathOf(ctx context.Context, pageID string) (string, error) {
	held, err := deps.Pages.Get(ctx, pages.GetRequest{ID: pageID})
	if err != nil {
		return "", err
	}
	return held.Page.Path, nil
}
