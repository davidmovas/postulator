package runs

import (
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type StartRequest struct {
	SiteID      string              `json:"siteId"`
	PageIDs     []string            `json:"pageIds"`
	TemplateID  string              `json:"templateId,omitempty"`
	Recipe      []template.StepSpec `json:"recipe,omitempty"`
	PublishMode string              `json:"publishMode,omitempty"`
	Kind        string              `json:"kind,omitempty"`
	Budget      run.Budget          `json:"budget"`
}

type StartResponse struct {
	RunID    string       `json:"runId"`
	Estimate run.Estimate `json:"estimate"`
	Added    []AddedPage  `json:"added"`
}

type EstimateResponse struct {
	Estimate run.Estimate `json:"estimate"`
	Added    []AddedPage  `json:"added"`
}

type AddedPage struct {
	PageID   string `json:"pageId"`
	Path     string `json:"path"`
	NeededBy string `json:"neededBy"`
}

type GetRequest struct {
	RunID string `json:"runId" description:"Run id"`
}

type GetResponse struct {
	Run Run `json:"run"`
}

type ListRequest struct {
	dto.ListRequest
	SiteID string `json:"siteId,omitempty"`
	Status string `json:"status,omitempty" enum:"pending,running,waiting,paused,completed,failed,cancelled" description:"Only this state"`
	Kind   string `json:"kind,omitempty" enum:"generate,relink,audit,sync,import,repair,revert,custom" description:"Only this kind"`
}

type ListItemsRequest struct {
	dto.ListRequest
	RunID  string `json:"runId" description:"Run id"`
	Status string `json:"status,omitempty" enum:"pending,running,waiting,paused,completed,failed,cancelled" description:"Only this state"`
}

type ListEventsRequest struct {
	RunID    string `json:"runId" description:"Run id"`
	SinceSeq int64  `json:"sinceSeq,omitempty" minimum:"0" description:"Only events after this sequence number"`
	Limit    int    `json:"limit,omitempty" minimum:"0" description:"Page size, default 50, max 500"`
}

type ListEventsResponse struct {
	Events []Event `json:"events"`
}

type GetArtifactRequest struct {
	ItemID string `json:"itemId" description:"Item id from runs_list_items"`
	Kind   string `json:"kind" enum:"link_context,draft,body_html,meta,images,validation_report,judge_report,publish_result,relink_result,sync_result,final_report,revert_result" description:"Artifact to read"`
}

type GetArtifactResponse struct {
	Artifact Artifact `json:"artifact"`
}

type ListArtifactsRequest struct {
	ItemID string `json:"itemId" description:"Item id from runs_list_items"`
}

type ListArtifactsResponse struct {
	Artifacts []ArtifactSummary `json:"artifacts"`
}

type PauseRequest struct {
	RunID  string `json:"runId" description:"Run id"`
	Reason string `json:"reason,omitempty" enum:"budget_exceeded,awaiting_confirmation,needs_human,user,awaiting_parent" description:"Why it is held; default user"`
}

type PauseResponse struct{}

type ResumeRequest struct {
	RunID string `json:"runId" description:"Paused run id"`
}

type ResumeResponse struct{}

type CancelRequest struct {
	RunID string `json:"runId" description:"Run id"`
}

type CancelResponse struct{}

type RevertRequest struct {
	RunID string `json:"runId" description:"Finished run id"`
}

type RevertResponse struct {
	RunID string `json:"runId"`
}

type RetryStepRequest struct {
	ItemID         string `json:"itemId" description:"Stopped item id from runs_list_items"`
	AcceptFindings bool   `json:"acceptFindings,omitempty" description:"Let a page held at validate go on as it is"`
}

type RetryStepResponse struct{}

type RegenerateRequest struct {
	RunID   string   `json:"runId" description:"Run id of the items"`
	ItemIDs []string `json:"itemIds" description:"Stopped item ids; one that already wrote to the site is refused"`
}

type RegenerateResponse struct {
	Restarted int `json:"restarted"`
}
