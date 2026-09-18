package schedules

import (
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type CreateRequest struct {
	SiteID          string   `json:"siteId"`
	Name            string   `json:"name"`
	Cron            string   `json:"cron,omitempty" description:"a five field cron expression or a descriptor such as @daily"`
	IntervalMinutes int      `json:"intervalMinutes,omitempty" description:"run every so many minutes instead of on a cron expression"`
	EntityID        string   `json:"entityId,omitempty" description:"target only the pages of this entity"`
	Status          string   `json:"status,omitempty" enum:"planned,exists,published,archived"`
	Limit           int      `json:"limit,omitempty"`
	TemplateID      string   `json:"templateId,omitempty"`
	Steps           []string `json:"steps,omitempty" description:"the step names to run; the template recipe is used when empty"`
	PublishMode     string   `json:"publishMode,omitempty" enum:"draft,publish"`
	MaxUSD          float64  `json:"maxUsd,omitempty"`
	MaxTokens       int      `json:"maxTokens,omitempty"`
	Enabled         bool     `json:"enabled"`
}

type CreateResponse struct {
	Schedule Schedule `json:"schedule"`
}

type UpdateRequest struct {
	ID              string   `json:"id"`
	Name            *string  `json:"name,omitempty"`
	Cron            *string  `json:"cron,omitempty"`
	IntervalMinutes *int     `json:"intervalMinutes,omitempty"`
	EntityID        *string  `json:"entityId,omitempty"`
	Status          *string  `json:"status,omitempty" enum:"planned,exists,published,archived"`
	Limit           *int     `json:"limit,omitempty"`
	TemplateID      *string  `json:"templateId,omitempty"`
	Steps           []string `json:"steps,omitempty"`
	PublishMode     *string  `json:"publishMode,omitempty" enum:"draft,publish"`
	MaxUSD          *float64 `json:"maxUsd,omitempty"`
	MaxTokens       *int     `json:"maxTokens,omitempty"`
}

type UpdateResponse struct {
	Schedule Schedule `json:"schedule"`
}

type DeleteRequest struct {
	ID string `json:"id"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id"`
}

type GetResponse struct {
	Schedule Schedule `json:"schedule"`
}

type ListRequest struct {
	dto.ListRequest
	SiteID  string `json:"siteId,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type EnableRequest struct {
	ID string `json:"id"`
}

type EnableResponse struct {
	Schedule Schedule `json:"schedule"`
}

type DisableRequest struct {
	ID string `json:"id"`
}

type DisableResponse struct {
	Schedule Schedule `json:"schedule"`
}

type RunNowRequest struct {
	ID string `json:"id"`
}

type RunNowResponse struct {
	RunID   string `json:"runId"`
	Targets int    `json:"targets"`
	Skipped string `json:"skipped,omitempty"`
}

type TickResponse struct {
	Started []string `json:"started"`
	Skipped int      `json:"skipped"`
}
