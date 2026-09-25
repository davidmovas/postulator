package reports

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/template"
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
	PageID string `json:"pageId" description:"The id of the page, exactly as a read tool returned it"`
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
	RunID string `json:"runId" description:"The id of the run, exactly as runs_list or runs_start returned it"`
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

type SkipReason string

const (
	SkipUnmapped   SkipReason = "unmapped"
	SkipNoTemplate SkipReason = "no_template"
)

type LinkState string

const (
	LinkPlaced            LinkState = "placed"
	LinkTargetUnpublished LinkState = "target_unpublished"
	LinkMissing           LinkState = "missing"
	LinkAwaitingTarget    LinkState = "awaiting_target"
	LinkAwaitingPage      LinkState = "awaiting_page"
	LinkBlocked           LinkState = "blocked"
)

type LinkAuditRequest struct {
	SiteID string `json:"siteId"`
}

type LinkPolicySummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ForbidExternal bool   `json:"forbidExternal"`
	ForbidSelf     bool   `json:"forbidSelf"`
	AnchorStrategy string `json:"anchorStrategy"`
}

type LinkTotals struct {
	Pages           int `json:"pages"`
	Audited         int `json:"audited"`
	Targets         int `json:"targets"`
	Required        int `json:"required"`
	Satisfied       int `json:"satisfied"`
	Missing         int `json:"missing"`
	MissingRequired int `json:"missingRequired"`
	Blocked         int `json:"blocked"`
	OffGraph        int `json:"offGraph"`
	Orphans         int `json:"orphans"`
	Pending         int `json:"pending"`
	Unpublished     int `json:"unpublished"`
}

type PageAudit struct {
	PageID          string `json:"pageId"`
	Path            string `json:"path"`
	Status          string `json:"status"`
	EntityID        string `json:"entityId"`
	EntityName      string `json:"entityName"`
	SkipReason      string `json:"skipReason"`
	OnSite          bool   `json:"onSite"`
	Targets         int    `json:"targets"`
	Required        int    `json:"required"`
	Satisfied       int    `json:"satisfied"`
	Missing         int    `json:"missing"`
	MissingRequired int    `json:"missingRequired"`
	Blocked         int    `json:"blocked"`
	OffGraph        int    `json:"offGraph"`
	Pending         int    `json:"pending"`
	Unpublished     int    `json:"unpublished"`
	Inbound         int    `json:"inbound"`
	Orphan          bool   `json:"orphan"`
}

type LinkAuditResponse struct {
	SiteID string            `json:"siteId"`
	Policy LinkPolicySummary `json:"policy"`
	Totals LinkTotals        `json:"totals"`
	Pages  []PageAudit       `json:"pages"`
}

type LinkAuditPageRequest struct {
	PageID string `json:"pageId" description:"The id of the mapped page to audit, exactly as a read tool returned it"`
}

type RequiredLink struct {
	Relation         string   `json:"relation"`
	Required         bool     `json:"required"`
	TargetEntityID   string   `json:"targetEntityId"`
	TargetEntityName string   `json:"targetEntityName"`
	TargetPageID     string   `json:"targetPageId"`
	TargetPath       string   `json:"targetPath"`
	Satisfied        bool     `json:"satisfied"`
	Anchor           string   `json:"anchor"`
	AnchorAllowed    bool     `json:"anchorAllowed"`
	AnchorsAllowed   []string `json:"anchorsAllowed"`
	Weight           float64  `json:"weight"`
	Depth            int      `json:"depth"`
	BlockedReason    string   `json:"blockedReason"`
	State            string   `json:"state"`
	TargetOnSite     bool     `json:"targetOnSite"`
}

type ExtraLink struct {
	ToURL    string `json:"toUrl"`
	ToPageID string `json:"toPageId"`
	Anchor   string `json:"anchor"`
	Kind     string `json:"kind"`
	Origin   string `json:"origin"`
}

type LinkAuditPageResponse struct {
	Page       PageAudit          `json:"page"`
	TemplateID string             `json:"templateId"`
	Rules      template.LinkRules `json:"rules"`
	Required   []RequiredLink     `json:"required"`
	Extra      []ExtraLink        `json:"extra"`
}
