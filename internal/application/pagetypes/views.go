package pagetypes

import (
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Type struct {
	ID         string   `json:"id"`
	SiteID     string   `json:"siteId"`
	Name       string   `json:"name"`
	Preset     string   `json:"preset"`
	Role       string   `json:"role"`
	Writable   bool     `json:"writable"`
	TemplateID *string  `json:"templateId"`
	Pages      int      `json:"pages"`
	Scopes     []Scope  `json:"scopes"`
	CreatedAt  dto.Time `json:"createdAt"`
	UpdatedAt  dto.Time `json:"updatedAt"`
}

type Scope struct {
	Field       string           `json:"field"`
	Label       string           `json:"label"`
	Provider    string           `json:"provider"`
	Enabled     bool             `json:"enabled"`
	AIWrites    bool             `json:"aiWrites"`
	Instruction string           `json:"instruction"`
	MinLength   int              `json:"minLength"`
	MaxLength   int              `json:"maxLength"`
	Rewrites    bool             `json:"rewrites"`
	Links       field.LinkPolicy `json:"links"`
	Stored      bool             `json:"stored"`
}

type Rule struct {
	ID         string              `json:"id"`
	SiteID     string              `json:"siteId"`
	Kind       string              `json:"kind"`
	Number     int                 `json:"number"`
	Name       string              `json:"name"`
	Enabled    bool                `json:"enabled"`
	TypeID     string              `json:"typeId"`
	Conditions pagetype.Conditions `json:"conditions"`
	CreatedAt  dto.Time            `json:"createdAt"`
	UpdatedAt  dto.Time            `json:"updatedAt"`
}

func ruleView(rule pagetype.Rule, number int) Rule {
	return Rule{
		ID: rule.ID, SiteID: rule.SiteID, Kind: string(rule.Kind), Number: number, Name: rule.Name, Enabled: rule.Enabled,
		TypeID: rule.TypeID, Conditions: rule.Conditions, CreatedAt: dto.NewTime(rule.CreatedAt), UpdatedAt: dto.NewTime(rule.UpdatedAt),
	}
}

type Field struct {
	Key        string `json:"key"`
	Read       string `json:"read"`
	Write      string `json:"write"`
	Text       bool   `json:"text"`
	Instanced  bool   `json:"instanced"`
	Denied     bool   `json:"denied"`
	Scopable   bool   `json:"scopable"`
	AIWritable bool   `json:"aiWritable"`
}

func scopeView(s field.Scope, stored bool) Scope {
	return Scope{
		Field: string(s.Key), Label: s.Label, Provider: string(s.Provider), Enabled: s.Enabled, AIWrites: s.AIWrites,
		Instruction: s.Instruction, MinLength: s.MinLength, MaxLength: s.MaxLength, Rewrites: s.Rewrites, Links: s.Links, Stored: stored,
	}
}

func scopableKeys() []field.Key {
	out := make([]field.Key, 0)
	for _, spec := range field.Catalog() {
		if spec.Key != field.Meta && field.CheckScope(spec.Key) == nil {
			out = append(out, spec.Key)
		}
	}
	return out
}

func typeView(t pagetype.Type, set pagetype.Set, stored []field.Scope, pages int) Type {
	keys := scopableKeys()
	scopes := make([]Scope, 0, len(stored)+len(keys))
	for _, key := range keys {
		scope, held := set.Scope(t.ID, key)
		scopes = append(scopes, scopeView(scope, held))
	}
	for i := range stored {
		if stored[i].PageTypeID == t.ID && stored[i].Key.Kind() == field.Meta {
			scopes = append(scopes, scopeView(stored[i], true))
		}
	}
	return Type{
		ID: t.ID, SiteID: t.SiteID, Name: t.Name, Preset: string(t.Preset), Role: string(t.Preset.Role()), Writable: t.Preset.Writable(),
		TemplateID: t.TemplateID, Pages: pages, Scopes: scopes, CreatedAt: dto.NewTime(t.CreatedAt), UpdatedAt: dto.NewTime(t.UpdatedAt),
	}
}

func catalogView() []Field {
	specs := field.Catalog()
	out := make([]Field, 0, len(specs))
	for _, spec := range specs {
		out = append(out, Field{
			Key: string(spec.Key), Read: string(spec.Read), Write: string(spec.Write), Text: spec.Text, Instanced: spec.Instanced,
			Denied: spec.Denied, Scopable: spec.Key == field.Meta || field.CheckScope(spec.Key) == nil, AIWritable: spec.AIWritable(),
		})
	}
	return out
}
