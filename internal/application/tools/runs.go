package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	runsName     = "runs"
	viewList     = "list"
	viewItems    = "items"
	viewEvents   = "events"
	viewArtifact = "artifact"
	viewReport   = "report"
)

type runsArgs struct {
	View     string `json:"view" enum:"list,get,items,events,artifact,report" description:"list the site's runs, newest first; get one with its budget and totals; items and the step each is on; events after sinceSeq; one item's artifact; report per item"`
	RunID    string `json:"runId,omitempty" description:"get, items, events, report: the run"`
	ItemID   string `json:"itemId,omitempty" description:"artifact: the item, from items"`
	Status   string `json:"status,omitempty" enum:"pending,running,waiting,paused,completed,failed,cancelled" description:"list, items: only this state"`
	Kind     string `json:"kind,omitempty" enum:"generate,relink,audit,sync,import,repair,revert,edit,custom" description:"list: only this kind"`
	Artifact string `json:"artifact,omitempty" enum:"link_context,draft,body_html,meta,fields,images,validation_report,judge_report,publish_result,relink_result,sync_result,final_report,revert_result" description:"artifact: which one"`
	SinceSeq int64  `json:"sinceSeq,omitempty" minimum:"0" description:"events: only those after this sequence number"`
	Cursor   string `json:"cursor,omitempty" description:"list, items: nextCursor of the last call"`
	Limit    int    `json:"limit,omitempty" description:"Default 50"`
}

type RunsResponse struct {
	Runs     *paging.List[runs.Run]     `json:"runs,omitempty"`
	Run      *runs.Run                  `json:"run,omitempty"`
	Items    *paging.List[runs.Item]    `json:"items,omitempty"`
	Events   *runs.ListEventsResponse   `json:"events,omitempty"`
	Artifact *runs.Artifact             `json:"artifact,omitempty"`
	Report   *reports.RunReportResponse `json:"report,omitempty"`
}

func runsTool(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        runsName,
		Description: "Read the site's runs: one with its status, budget and totals, its items and their steps, its event log, an item's artifact, its report.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in runsArgs) (RunsResponse, error) {
		switch in.View {
		case viewList:
			listed, err := fitted(deps, in.Limit, func(limit int) (paging.List[runs.Run], error) {
				return deps.Runs.List(ctx, runs.ListRequest{
					ListRequest: dto.ListRequest{Cursor: in.Cursor, Limit: limit}, SiteID: b.SiteID, Status: in.Status, Kind: in.Kind,
				})
			})
			return RunsResponse{Runs: &listed}, err
		case opGet:
			held, err := deps.Runs.Get(ctx, runs.GetRequest{RunID: in.RunID})
			return RunsResponse{Run: &held.Run}, err
		case viewItems:
			listed, err := fitted(deps, in.Limit, func(limit int) (paging.List[runs.Item], error) {
				return deps.Runs.ListItems(ctx, runs.ListItemsRequest{
					ListRequest: dto.ListRequest{Cursor: in.Cursor, Limit: limit}, RunID: in.RunID, Status: in.Status,
				})
			})
			return RunsResponse{Items: &listed}, err
		case viewEvents:
			listed, err := fitted(deps, in.Limit, func(limit int) (runs.ListEventsResponse, error) {
				return deps.Runs.ListEvents(ctx, runs.ListEventsRequest{RunID: in.RunID, SinceSeq: in.SinceSeq, Limit: limit})
			})
			return RunsResponse{Events: &listed}, err
		case viewArtifact:
			held, err := deps.Runs.GetArtifact(ctx, runs.GetArtifactRequest{ItemID: in.ItemID, Kind: in.Artifact})
			return RunsResponse{Artifact: &held.Artifact}, err
		case viewReport:
			report, err := deps.Reports.RunReport(ctx, reports.RunReportRequest{RunID: in.RunID})
			return RunsResponse{Report: &report}, err
		default:
			return RunsResponse{}, errors.New(errors.Invalid, "runs takes the view list, get, items, events, artifact or report").
				WithDetail("field", "view")
		}
	})
}
