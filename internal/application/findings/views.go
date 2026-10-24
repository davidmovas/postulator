package findings

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/application/runs"
	syncsite "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Option struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type Finding struct {
	ID         string          `json:"id"`
	SiteID     string          `json:"siteId"`
	PageID     string          `json:"pageId,omitempty"`
	PagePath   string          `json:"pagePath,omitempty"`
	PageTitle  string          `json:"pageTitle,omitempty"`
	EntityID   string          `json:"entityId,omitempty"`
	Code       string          `json:"code"`
	Severity   string          `json:"severity"`
	SubjectKey string          `json:"subjectKey"`
	Subject    json.RawMessage `json:"subject"`
	Options    []Option        `json:"options"`
	Status     string          `json:"status"`
	Resolution string          `json:"resolution,omitempty"`
	ResolvedBy string          `json:"resolvedBy,omitempty"`
	CreatedAt  dto.Time        `json:"createdAt"`
	UpdatedAt  dto.Time        `json:"updatedAt"`
}

type ListRequest struct {
	SiteID     string `json:"siteId"`
	Status     string `json:"status,omitempty"`
	Code       string `json:"code,omitempty"`
	PageID     string `json:"pageId,omitempty"`
	PageTypeID string `json:"pageTypeId,omitempty"`
	Under      string `json:"under,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type CountsRequest struct {
	SiteID     string `json:"siteId"`
	Status     string `json:"status,omitempty"`
	PageTypeID string `json:"pageTypeId,omitempty"`
	Under      string `json:"under,omitempty"`
}

type CodeCount struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

type CountsResponse struct {
	Codes []CodeCount `json:"codes"`
	Total int         `json:"total"`
}

type GetRequest struct {
	SiteID    string `json:"siteId,omitempty"`
	FindingID string `json:"findingId"`
}

type ResolveRequest struct {
	SiteID       string `json:"siteId,omitempty"`
	FindingID    string `json:"findingId"`
	Option       string `json:"option"`
	PageID       string `json:"pageId,omitempty"`
	PageTypeID   string `json:"pageTypeId,omitempty"`
	ProposalHash string `json:"proposalHash,omitempty"`
	By           string `json:"-"`
}

type ResolveResponse struct {
	Finding Finding                   `json:"finding"`
	Start   *runs.StartRequest        `json:"start,omitempty"`
	Sync    *syncsite.SyncSiteRequest `json:"sync,omitempty"`
}

func placed(f finding.Finding, places contextpack.Map) Finding {
	shown := view(f)
	if f.PageID == nil {
		return shown
	}
	if place, found := places.Place(*f.PageID); found {
		shown.PagePath, shown.PageTitle = place.Ref.Path, place.Ref.Title
	}
	return shown
}

func view(f finding.Finding) Finding {
	options := make([]Option, 0, len(f.Options))
	for _, option := range f.Options {
		options = append(options, Option{Name: string(option), Kind: string(option.Kind())})
	}
	shown := Finding{
		ID: f.ID, SiteID: f.SiteID, Code: string(f.Code), Severity: string(f.Severity), SubjectKey: f.SubjectKey,
		Subject: f.Subject, Options: options, Status: string(f.Status), Resolution: f.Resolution, ResolvedBy: f.ResolvedBy,
		CreatedAt: dto.NewTime(f.CreatedAt), UpdatedAt: dto.NewTime(f.UpdatedAt),
	}
	if f.PageID != nil {
		shown.PageID = *f.PageID
	}
	if f.EntityID != nil {
		shown.EntityID = *f.EntityID
	}
	if len(shown.Subject) == 0 {
		shown.Subject = json.RawMessage("{}")
	}
	return shown
}
