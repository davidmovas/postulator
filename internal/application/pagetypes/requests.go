package pagetypes

import (
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

type ListRulesRequest struct {
	SiteID string `json:"siteId"`
}

type RulesResponse struct {
	Rules    []Rule `json:"rules"`
	Fallback *Rule  `json:"fallback"`
}

type SaveRuleRequest struct {
	ID         string              `json:"id,omitempty"`
	SiteID     string              `json:"siteId"`
	Name       string              `json:"name"`
	Enabled    bool                `json:"enabled"`
	TypeID     string              `json:"typeId"`
	Conditions pagetype.Conditions `json:"conditions"`
}

type DeleteRuleRequest struct {
	ID string `json:"id"`
}

type ReorderRulesRequest struct {
	SiteID  string   `json:"siteId"`
	RuleIDs []string `json:"ruleIds"`
}

type ListRequest struct {
	SiteID string `json:"siteId"`
}

type ListResponse struct {
	Types  []Type  `json:"types"`
	Fields []Field `json:"fields"`
}

type GetRequest struct {
	ID string `json:"id"`
}

type CreateRequest struct {
	SiteID     string  `json:"siteId"`
	Name       string  `json:"name"`
	Preset     string  `json:"preset"`
	TemplateID *string `json:"templateId,omitempty"`
}

type SaveRequest struct {
	ID         string  `json:"id"`
	Name       *string `json:"name,omitempty"`
	TemplateID *string `json:"templateId,omitempty"`
}

type DeleteRequest struct {
	ID         string `json:"id"`
	ReassignTo string `json:"reassignTo"`
}

type DeleteResponse struct {
	Moved int `json:"moved"`
}

type SetScopeRequest struct {
	PageTypeID  string           `json:"pageTypeId"`
	Field       string           `json:"field"`
	Enabled     bool             `json:"enabled"`
	AIWrites    bool             `json:"aiWrites"`
	Instruction string           `json:"instruction"`
	MinLength   int              `json:"minLength"`
	MaxLength   int              `json:"maxLength"`
	Rewrites    bool             `json:"rewrites"`
	Links       field.LinkPolicy `json:"links"`
	Label       string           `json:"label"`
	Provider    string           `json:"provider"`
}
