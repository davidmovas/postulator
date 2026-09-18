package templates

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Template struct {
	ID        string                `json:"id"`
	Scope     string                `json:"scope"`
	SiteID    *string               `json:"siteId"`
	Name      string                `json:"name"`
	PageKind  string                `json:"pageKind"`
	Version   int                   `json:"version"`
	Spec      template.TemplateSpec `json:"spec"`
	CreatedAt dto.Time              `json:"createdAt"`
	UpdatedAt dto.Time              `json:"updatedAt"`
}

type Override struct {
	ID         string          `json:"id"`
	TemplateID string          `json:"templateId"`
	Scope      string          `json:"scope"`
	TargetID   string          `json:"targetId"`
	Patch      json.RawMessage `json:"patch"`
	CreatedAt  dto.Time        `json:"createdAt"`
	UpdatedAt  dto.Time        `json:"updatedAt"`
}

type LinkPolicy struct {
	ID             string             `json:"id"`
	Scope          string             `json:"scope"`
	SiteID         *string            `json:"siteId"`
	Name           string             `json:"name"`
	Rules          template.LinkRules `json:"rules"`
	ForbidExternal bool               `json:"forbidExternal"`
	ForbidSelf     bool               `json:"forbidSelf"`
	AnchorStrategy string             `json:"anchorStrategy"`
	CreatedAt      dto.Time           `json:"createdAt"`
	UpdatedAt      dto.Time           `json:"updatedAt"`
}

func templateView(t template.Template) Template {
	return Template{
		ID:        t.ID,
		Scope:     string(t.Scope),
		SiteID:    t.SiteID,
		Name:      t.Name,
		PageKind:  t.PageKind,
		Version:   t.Version,
		Spec:      t.Spec,
		CreatedAt: dto.NewTime(t.CreatedAt),
		UpdatedAt: dto.NewTime(t.UpdatedAt),
	}
}

func overrideView(o template.Override) Override {
	return Override{
		ID:         o.ID,
		TemplateID: o.TemplateID,
		Scope:      string(o.Scope),
		TargetID:   o.TargetID,
		Patch:      o.Patch,
		CreatedAt:  dto.NewTime(o.CreatedAt),
		UpdatedAt:  dto.NewTime(o.UpdatedAt),
	}
}

func overrideViews(overrides []template.Override) []Override {
	out := make([]Override, 0, len(overrides))
	for i := range overrides {
		out = append(out, overrideView(overrides[i]))
	}
	return out
}
