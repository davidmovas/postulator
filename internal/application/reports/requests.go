package reports

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type SiteOverviewRequest struct {
	SiteID string `json:"siteId"`
}

type EntityTotals struct {
	Total             int `json:"total"`
	WithCanonicalPage int `json:"withCanonicalPage"`
	WithPublishedPage int `json:"withPublishedPage"`
}

type PageTotals struct {
	ByStatus map[string]int `json:"byStatus"`
	Total    int            `json:"total"`
	Unmapped int            `json:"unmapped"`
	Orphans  int            `json:"orphans"`
}

type EdgeTotals struct {
	Approved int `json:"approved"`
	Realized int `json:"realized"`
}

type DepthBucket struct {
	Depth int `json:"depth"`
	Pages int `json:"pages"`
}

type EntityScore struct {
	EntityID string  `json:"entityId"`
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Score    float64 `json:"score"`
}

type SiteOverviewResponse struct {
	SiteID   string        `json:"siteId"`
	Entities EntityTotals  `json:"entities"`
	Pages    PageTotals    `json:"pages"`
	Edges    EdgeTotals    `json:"edges"`
	Depth    []DepthBucket `json:"depth"`
	Top      []EntityScore `json:"top"`
}

type PageReportRequest struct {
	PageID string `json:"pageId"`
}

type PageReportResponse struct {
	FinishedAt *dto.Time       `json:"finishedAt,omitempty"`
	Validation json.RawMessage `json:"validation,omitempty"`
	Judge      json.RawMessage `json:"judge,omitempty"`
	Publish    json.RawMessage `json:"publish,omitempty"`
	Relink     json.RawMessage `json:"relink,omitempty"`
	PageID     string          `json:"pageId"`
	Path       string          `json:"path"`
	RunID      string          `json:"runId"`
	ItemID     string          `json:"itemId"`
	Status     string          `json:"status"`
}

type RunReportRequest struct {
	RunID string `json:"runId"`
}

type ItemReport struct {
	Report json.RawMessage `json:"report,omitempty"`
	ItemID string          `json:"itemId"`
	PageID string          `json:"pageId"`
	Status string          `json:"status"`
	Error  string          `json:"error,omitempty"`
}

type RunStats struct {
	Items  int     `json:"items"`
	Done   int     `json:"done"`
	Failed int     `json:"failed"`
	Tokens int     `json:"tokens"`
	USD    float64 `json:"usd"`
}

type RunReportResponse struct {
	RunID  string       `json:"runId"`
	SiteID string       `json:"siteId"`
	Kind   string       `json:"kind"`
	Status string       `json:"status"`
	Stats  RunStats     `json:"stats"`
	Items  []ItemReport `json:"items"`
}
