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
}

type GetRequest struct {
	RunID string `json:"runId"`
}

type GetResponse struct {
	Run Run `json:"run"`
}

type ListRequest struct {
	dto.ListRequest
	SiteID string `json:"siteId,omitempty"`
	Status string `json:"status,omitempty"`
	Kind   string `json:"kind,omitempty"`
}

type ListItemsRequest struct {
	dto.ListRequest
	RunID  string `json:"runId"`
	Status string `json:"status,omitempty"`
}

type ListEventsRequest struct {
	RunID    string `json:"runId"`
	SinceSeq int64  `json:"sinceSeq"`
	Limit    int    `json:"limit"`
}

type ListEventsResponse struct {
	Events []Event `json:"events"`
}

type GetArtifactRequest struct {
	ItemID string `json:"itemId"`
	Kind   string `json:"kind"`
}

type GetArtifactResponse struct {
	Artifact Artifact `json:"artifact"`
}

type PauseRequest struct {
	RunID  string `json:"runId"`
	Reason string `json:"reason,omitempty"`
}

type PauseResponse struct{}

type ResumeRequest struct {
	RunID string `json:"runId"`
}

type ResumeResponse struct{}

type CancelRequest struct {
	RunID string `json:"runId"`
}

type CancelResponse struct{}

type RetryStepRequest struct {
	ItemID string `json:"itemId"`
}

type RetryStepResponse struct{}
