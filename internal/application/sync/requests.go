package sync

import "github.com/davidmovas/postulator/internal/kernel/dto"

const Filename = "postulator-companion.zip"

type ProgressRequest struct {
	SiteID string `json:"siteId"`
	RunID  string `json:"runId,omitempty"`
}

type ProgressResponse struct {
	WakeAt      *dto.Time      `json:"wakeAt"`
	StartedAt   *dto.Time      `json:"startedAt"`
	CreatedAt   dto.Time       `json:"createdAt"`
	RunID       string         `json:"runId"`
	Status      string         `json:"status"`
	Stage       string         `json:"stage"`
	Source      string         `json:"source"`
	Error       string         `json:"error"`
	Batches     int            `json:"batches"`
	Pulled      int            `json:"pulled"`
	Created     int            `json:"created"`
	Updated     int            `json:"updated"`
	Products    int            `json:"products"`
	Rendered    int            `json:"rendered"`
	Unreachable int            `json:"unreachable"`
	Archived    int            `json:"archived"`
	Entities    int            `json:"entities"`
	Placed      int            `json:"placed"`
	Unplaced    int            `json:"unplaced"`
	UnplacedBy  map[string]int `json:"unplacedBy"`
	Attempts    int            `json:"attempts"`
	Found       bool           `json:"found"`
	Full        bool           `json:"full"`
}

type SyncSiteRequest struct {
	SiteID       string `json:"siteId"`
	Full         bool   `json:"full,omitempty"`
	RenderRescan bool   `json:"renderRescan,omitempty"`
}

type SyncSiteResponse struct {
	RunID string `json:"runId"`
}

type CheckPluginRequest struct {
	SiteID string `json:"siteId"`
}

type CheckPluginResponse struct {
	Plugin   Plugin `json:"plugin"`
	Commerce string `json:"commerce"`
}

type PluginPackageRequest struct{}

type PluginPackageResponse struct {
	Filename string `json:"filename"`
	Bytes    []byte `json:"bytes"`
}
