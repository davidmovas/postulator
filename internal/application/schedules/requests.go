package schedules

import (
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type CreateRequest struct {
	SiteID          string   `json:"siteId"`
	Name            string   `json:"name" description:"What to call the schedule, two to four words"`
	Cron            string   `json:"cron,omitempty" description:"A five field cron expression read in UTC, or a descriptor such as @daily"`
	IntervalMinutes int      `json:"intervalMinutes,omitempty" minimum:"1" description:"Run every so many minutes instead of on a cron expression"`
	EntityID        string   `json:"entityId,omitempty" description:"Target only the pages of this entity, exactly as a read tool returned its id"`
	Status          string   `json:"status,omitempty" enum:"planned,exists,published,archived" description:"Target only pages in this state"`
	Limit           int      `json:"limit,omitempty" minimum:"1" description:"The most pages one firing may take"`
	TemplateID      string   `json:"templateId,omitempty" description:"The template to write with, left out to resolve each page's own"`
	Steps           []string `json:"steps,omitempty" enum:"resolve_context,generate_body,generate_meta,insert_links,repair_links,generate_images,validate,judge,publish,relink_neighbors,sync_back,report,sync_site" description:"The steps to run in order, left out to take the recipe of the resolved template"`
	PublishMode     string   `json:"publishMode,omitempty" enum:"draft,publish" description:"Whether each run leaves a draft in WordPress or publishes it; leave it out for draft"`
	MaxUSD          float64  `json:"maxUsd,omitempty" minimum:"0" description:"Stop a run once it has spent this many dollars"`
	MaxTokens       int      `json:"maxTokens,omitempty" minimum:"0" description:"Stop a run once it has used this many tokens"`
	Enabled         bool     `json:"enabled" description:"Arm the schedule at once; a disabled schedule is kept but never fires"`
}

type CreateResponse struct {
	Schedule Schedule `json:"schedule"`
}

type UpdateRequest struct {
	ID              string   `json:"id" description:"The id of the schedule, exactly as schedules_list returned it"`
	Name            *string  `json:"name,omitempty" description:"The new name, left out to keep the current one"`
	Cron            *string  `json:"cron,omitempty" description:"The new cron expression read in UTC, left out to keep the current timing"`
	IntervalMinutes *int     `json:"intervalMinutes,omitempty" minimum:"1" description:"The new interval in minutes, left out to keep the current timing"`
	EntityID        *string  `json:"entityId,omitempty" description:"The new entity to target, left out to keep the current targets"`
	Status          *string  `json:"status,omitempty" enum:"planned,exists,published,archived" description:"The new page state to target, left out to keep the current one"`
	Limit           *int     `json:"limit,omitempty" minimum:"1" description:"The new ceiling on pages per firing, left out to keep the current one"`
	TemplateID      *string  `json:"templateId,omitempty" description:"The new template, left out to keep the current one"`
	Steps           []string `json:"steps,omitempty" enum:"resolve_context,generate_body,generate_meta,insert_links,repair_links,generate_images,validate,judge,publish,relink_neighbors,sync_back,report,sync_site" description:"The whole new recipe in order, left out to keep the current one"`
	PublishMode     *string  `json:"publishMode,omitempty" enum:"draft,publish" description:"The new publish mode, left out to keep the current one"`
	MaxUSD          *float64 `json:"maxUsd,omitempty" minimum:"0" description:"The new dollar ceiling per run, left out to keep the current one"`
	MaxTokens       *int     `json:"maxTokens,omitempty" minimum:"0" description:"The new token ceiling per run, left out to keep the current one"`
}

type UpdateResponse struct {
	Schedule Schedule `json:"schedule"`
}

type DeleteRequest struct {
	ID string `json:"id" description:"The id of the schedule to remove, exactly as schedules_list returned it"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id" description:"The id of the schedule, exactly as schedules_list returned it"`
}

type GetResponse struct {
	Schedule Schedule `json:"schedule"`
}

type ListRequest struct {
	dto.ListRequest
	SiteID  string `json:"siteId,omitempty"`
	Enabled *bool  `json:"enabled,omitempty" description:"Keep only schedules that are armed, or only those that are not"`
}

type EnableRequest struct {
	ID string `json:"id" description:"The id of the schedule to arm, exactly as schedules_list returned it"`
}

type EnableResponse struct {
	Schedule Schedule `json:"schedule"`
}

type DisableRequest struct {
	ID string `json:"id" description:"The id of the schedule to stand down, exactly as schedules_list returned it"`
}

type DisableResponse struct {
	Schedule Schedule `json:"schedule"`
}

type RunNowRequest struct {
	ID string `json:"id" description:"The id of the schedule to fire once now, exactly as schedules_list returned it"`
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
