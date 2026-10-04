package schedules

import (
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type CreateRequest struct {
	SiteID          string   `json:"siteId"`
	Name            string   `json:"name" description:"Name, 2-4 words"`
	Cron            string   `json:"cron,omitempty" description:"Five-field cron in UTC, or a descriptor like @daily"`
	IntervalMinutes int      `json:"intervalMinutes,omitempty" minimum:"1" description:"Minutes between runs, instead of cron"`
	EntityID        string   `json:"entityId,omitempty" description:"Only pages of this entity"`
	Status          string   `json:"status,omitempty" enum:"planned,exists,published,archived" description:"Only pages in this state"`
	Limit           int      `json:"limit,omitempty" minimum:"1" description:"Max pages per firing"`
	TemplateID      string   `json:"templateId,omitempty" description:"Template id to move pages to; default their own"`
	Steps           []string `json:"steps,omitempty" enum:"resolve_context,generate_body,generate_meta,insert_links,repair_links,generate_images,validate,judge,publish,relink_neighbors,sync_back,report" description:"Steps in order; default the template's recipe"`
	PublishMode     string   `json:"publishMode,omitempty" enum:"draft,publish" description:"Default draft"`
	MaxUSD          float64  `json:"maxUsd,omitempty" minimum:"0" description:"Dollar ceiling per run"`
	MaxTokens       int      `json:"maxTokens,omitempty" minimum:"0" description:"Token ceiling per run"`
	Enabled         bool     `json:"enabled" description:"Arm now; a disabled schedule never fires"`
}

type CreateResponse struct {
	Schedule Schedule `json:"schedule"`
}

type UpdateRequest struct {
	ID              string   `json:"id" description:"Schedule id"`
	Name            *string  `json:"name,omitempty" description:"New name"`
	Cron            *string  `json:"cron,omitempty" description:"New cron in UTC"`
	IntervalMinutes *int     `json:"intervalMinutes,omitempty" minimum:"1" description:"New interval in minutes"`
	EntityID        *string  `json:"entityId,omitempty" description:"New target entity id"`
	Status          *string  `json:"status,omitempty" enum:"planned,exists,published,archived" description:"New target page state"`
	Limit           *int     `json:"limit,omitempty" minimum:"1" description:"New max pages per firing"`
	TemplateID      *string  `json:"templateId,omitempty" description:"New template id"`
	Steps           []string `json:"steps,omitempty" enum:"resolve_context,generate_body,generate_meta,insert_links,repair_links,generate_images,validate,judge,publish,relink_neighbors,sync_back,report" description:"Whole new recipe in order"`
	PublishMode     *string  `json:"publishMode,omitempty" enum:"draft,publish" description:"New publish mode"`
	MaxUSD          *float64 `json:"maxUsd,omitempty" minimum:"0" description:"New dollar ceiling per run"`
	MaxTokens       *int     `json:"maxTokens,omitempty" minimum:"0" description:"New token ceiling per run"`
}

type UpdateResponse struct {
	Schedule Schedule `json:"schedule"`
}

type DeleteRequest struct {
	ID string `json:"id" description:"Schedule id"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id" description:"Schedule id"`
}

type GetResponse struct {
	Schedule Schedule `json:"schedule"`
}

type ListRequest struct {
	dto.ListRequest
	SiteID  string `json:"siteId,omitempty"`
	Enabled *bool  `json:"enabled,omitempty" description:"Only armed, or only disarmed, schedules"`
}

type EnableRequest struct {
	ID string `json:"id" description:"Schedule id"`
}

type EnableResponse struct {
	Schedule Schedule `json:"schedule"`
}

type DisableRequest struct {
	ID string `json:"id" description:"Schedule id"`
}

type DisableResponse struct {
	Schedule Schedule `json:"schedule"`
}

type RunNowRequest struct {
	ID string `json:"id" description:"Schedule id"`
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
