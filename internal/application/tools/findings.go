package tools

import (
	"context"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	findingsName  = "findings"
	opList        = "list"
	opResolve     = "resolve"
	opResolveMany = "resolve_many"
)

type findingsArgs struct {
	Op           string   `json:"op,omitempty" enum:"list,resolve,resolve_many" description:"Default list; resolve_many settles many findings of one code with one model option, at most 1,000"`
	Status       string   `json:"status,omitempty" enum:"open,resolved,dismissed" description:"list: default open"`
	Code         string   `json:"code,omitempty" description:"list: only this code; resolve_many: their code"`
	PageID       string   `json:"pageId,omitempty" description:"list: one page's; map_to_page: the row's page"`
	FindingID    string   `json:"findingId,omitempty" description:"resolve: id from list"`
	FindingIDs   []string `json:"findingIds,omitempty" description:"resolve_many: ids from list"`
	Option       string   `json:"option,omitempty" description:"resolve: one the finding offers"`
	PageTypeID   string   `json:"pageTypeId,omitempty" description:"set_type: the type"`
	ProposalHash string   `json:"proposalHash,omitempty" description:"accept: from the card proposal"`
	Cursor       string   `json:"cursor,omitempty" description:"list: nextCursor of the last call"`
	Limit        int      `json:"limit,omitempty" description:"list: default 50"`
}

type FindingsResponse struct {
	List        *paging.List[findings.Finding] `json:"list,omitempty"`
	Resolve     *findings.ResolveResponse      `json:"resolve,omitempty"`
	ResolveMany *findings.ResolveManyResponse  `json:"resolveMany,omitempty"`
}

func findingsTool(deps Deps) Tool {
	tool := confirming(newSiteTool(Def{
		Name: findingsName,
		Description: "List what the model found that disagrees, or resolve a finding or many: a model option changes the model, " +
			"a run option answers the request to pass to runs_control or sites_manage sync.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in findingsArgs) (FindingsResponse, error) {
		switch in.Op {
		case opList, "":
			status := in.Status
			if status == "" {
				status = string(finding.Open)
			}
			listed, err := fitted(deps, in.Limit, func(limit int) (paging.List[findings.Finding], error) {
				return deps.Findings.List(ctx, findings.ListRequest{
					SiteID: b.SiteID, Status: status, Code: in.Code, PageID: in.PageID, Cursor: in.Cursor, Limit: limit,
				})
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
		case opResolveMany:
			resolved, err := deps.Findings.ResolveMany(ctx, findings.ResolveManyRequest{
				SiteID: b.SiteID, Code: in.Code, Option: in.Option, FindingIDs: in.FindingIDs, PageTypeID: in.PageTypeID,
				By: string(finding.ResolvedByAgent),
			})
			if err != nil {
				return FindingsResponse{}, err
			}
			return FindingsResponse{ResolveMany: &resolved}, nil
		default:
			return FindingsResponse{}, errors.New(errors.Invalid, "findings takes the op list, resolve or resolve_many").WithDetail("field", "op")
		}
	}), func(ctx context.Context, b Binding, in findingsArgs) (string, error) {
		if b.Mode != agent.ModeConfirm || !resolving(in) || finding.Option(in.Option).Kind() != finding.ModelChange {
			return "", nil
		}
		return deps.resolution(ctx, b, in)
	})
	tool = describing(tool, func(ctx context.Context, b Binding, in findingsArgs) (string, error) {
		if !resolving(in) {
			return "", nil
		}
		return deps.resolution(ctx, b, in)
	})
	tool.RiskOf = func(args json.RawMessage) Risk {
		var in findingsArgs
		if err := json.Unmarshal(args, &in); err == nil && resolving(in) && finding.Option(in.Option).Kind() == finding.ModelChange {
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

func resolving(in findingsArgs) bool {
	return in.Op == opResolve || in.Op == opResolveMany
}

func (deps Deps) resolution(ctx context.Context, b Binding, in findingsArgs) (string, error) {
	ids := in.FindingIDs
	if in.Op == opResolve {
		ids = []string{in.FindingID}
	}
	said, err := deps.Findings.Explain(ctx, findings.ExplainRequest{
		SiteID: b.SiteID, FindingIDs: ids, Option: in.Option, PageID: in.PageID, PageTypeID: in.PageTypeID,
	})
	return said.Sentence, err
}

func (deps Deps) pathOf(ctx context.Context, pageID string) (string, error) {
	held, err := deps.Pages.Get(ctx, pages.GetRequest{ID: pageID})
	if err != nil {
		return "", err
	}
	return held.Page.Path, nil
}
