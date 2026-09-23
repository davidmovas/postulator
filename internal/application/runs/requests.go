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
	RunID string `json:"runId" description:"The id of the run, exactly as runs_start or runs_list returned it"`
}

type GetResponse struct {
	Run Run `json:"run"`
}

type ListRequest struct {
	dto.ListRequest
	SiteID string `json:"siteId,omitempty"`
	Status string `json:"status,omitempty" enum:"pending,running,waiting,paused,completed,failed,cancelled" description:"Keep only runs in this state"`
	Kind   string `json:"kind,omitempty" enum:"generate,relink,audit,sync,import,repair,revert,custom" description:"Keep only runs of this kind"`
}

type ListItemsRequest struct {
	dto.ListRequest
	RunID  string `json:"runId" description:"The id of the run, exactly as runs_start or runs_list returned it"`
	Status string `json:"status,omitempty" enum:"pending,running,waiting,paused,completed,failed,cancelled" description:"Keep only items in this state"`
}

type ListEventsRequest struct {
	RunID    string `json:"runId" description:"The id of the run, exactly as runs_start or runs_list returned it"`
	SinceSeq int64  `json:"sinceSeq,omitempty" minimum:"0" description:"Return only events after this sequence number; leave it out to start at the beginning"`
	Limit    int    `json:"limit,omitempty" minimum:"0" description:"How many events to return; leave it out for the default"`
}

type ListEventsResponse struct {
	Events []Event `json:"events"`
}

type GetArtifactRequest struct {
	ItemID string `json:"itemId" description:"The id of the run item, exactly as runs_list_items returned it"`
	Kind   string `json:"kind" enum:"link_context,draft,body_html,meta,images,validation_report,judge_report,publish_result,relink_result,sync_result,final_report,revert_result" description:"Which artifact of the item to read"`
}

type GetArtifactResponse struct {
	Artifact Artifact `json:"artifact"`
}

type ListArtifactsRequest struct {
	ItemID string `json:"itemId" description:"The id of the run item, exactly as runs_list_items returned it"`
}

type ListArtifactsResponse struct {
	Artifacts []ArtifactSummary `json:"artifacts"`
}

type PauseRequest struct {
	RunID  string `json:"runId" description:"The id of the run to hold, exactly as runs_start or runs_list returned it"`
	Reason string `json:"reason,omitempty" enum:"budget_exceeded,awaiting_confirmation,needs_human,user,awaiting_parent" description:"Which of the five reasons holds the run; leave it out and it is recorded as user"`
}

type PauseResponse struct{}

type ResumeRequest struct {
	RunID string `json:"runId" description:"The id of the held run, exactly as runs_list returned it"`
}

type ResumeResponse struct{}

type CancelRequest struct {
	RunID string `json:"runId" description:"The id of the run to stop, exactly as runs_start or runs_list returned it"`
}

type CancelResponse struct{}

type RevertRequest struct {
	RunID string `json:"runId" description:"The id of the finished run to put back, exactly as runs_list returned it"`
}

type RevertResponse struct {
	RunID string `json:"runId"`
}

type RetryStepRequest struct {
	ItemID string `json:"itemId" description:"The id of the failed run item to try again, exactly as runs_list_items returned it"`
}

type RetryStepResponse struct{}

type RegenerateRequest struct {
	RunID   string   `json:"runId" description:"The run the items belong to"`
	ItemIDs []string `json:"itemIds" description:"Stopped items from runs_list_items; one that already wrote to the site is refused"`
}

type RegenerateResponse struct {
	Restarted int `json:"restarted"`
}
