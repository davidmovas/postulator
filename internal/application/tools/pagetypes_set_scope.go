package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/domain/field"
)

const pagetypesSetScopeName = "pagetypes_set_scope"

type pagetypesSetScopeArgs struct {
	PageTypeID  string            `json:"pageTypeId" description:"Page type id from pagetypes_list"`
	Field       string            `json:"field,omitempty" enum:"title,excerpt,body,seo_title,seo_description,product_description,product_short_description,variation_description,image_alt,image_title,upsells,cross_sells" description:"Field to scope; omit for a custom field"`
	MetaKey     string            `json:"metaKey,omitempty" description:"Custom field's meta key, in place of field"`
	Enabled     *bool             `json:"enabled,omitempty" description:"Write this field on the type's pages"`
	AIWrites    *bool             `json:"aiWrites,omitempty" description:"The model writes it; false leaves it to a person"`
	Instruction *string           `json:"instruction,omitempty" description:"What the writer is told about the field"`
	MinLength   *int              `json:"minLength,omitempty" minimum:"0" description:"Fewest characters; 0 for none"`
	MaxLength   *int              `json:"maxLength,omitempty" minimum:"0" description:"Most characters; 0 for none"`
	Links       *field.LinkPolicy `json:"links,omitempty" description:"Whole new link policy"`
	Label       *string           `json:"label,omitempty" description:"A custom field's label"`
	Provider    *string           `json:"provider,omitempty" description:"Where a custom field is stored: plugin or store"`
}

func (in *pagetypesSetScopeArgs) key() string {
	if in.MetaKey != "" {
		return string(field.Instance(field.Meta, in.MetaKey))
	}
	return in.Field
}

func (in *pagetypesSetScopeArgs) over(current pagetypes.Scope) pagetypes.SetScopeRequest {
	next := pagetypes.SetScopeRequest{
		PageTypeID: in.PageTypeID, Field: in.key(), Enabled: current.Enabled, AIWrites: current.AIWrites,
		Instruction: current.Instruction, MinLength: current.MinLength, MaxLength: current.MaxLength,
		Rewrites: current.Rewrites, Links: current.Links, Label: current.Label, Provider: current.Provider,
	}
	if in.Enabled != nil {
		next.Enabled = *in.Enabled
	}
	if in.AIWrites != nil {
		next.AIWrites = *in.AIWrites
	}
	if in.Instruction != nil {
		next.Instruction = *in.Instruction
	}
	if in.MinLength != nil {
		next.MinLength = *in.MinLength
	}
	if in.MaxLength != nil {
		next.MaxLength = *in.MaxLength
	}
	if in.Links != nil {
		next.Links = *in.Links
	}
	if in.Label != nil {
		next.Label = *in.Label
	}
	if in.Provider != nil {
		next.Provider = *in.Provider
	}
	return next
}

func scopeOf(kind pagetypes.Type, key string) pagetypes.Scope {
	for i := range kind.Scopes {
		if kind.Scopes[i].Field == key {
			return kind.Scopes[i]
		}
	}
	return pagetypes.Scope{Field: key, AIWrites: true, Links: field.NoLinks()}
}

func pagetypesSetScope(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagetypesSetScopeName,
		Description: "Change how one field of a page type is written: on or off, instruction, lengths, link policy.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in pagetypesSetScopeArgs) (pagetypes.Scope, error) {
		kind, err := deps.PageTypes.Get(ctx, pagetypes.GetRequest{ID: in.PageTypeID})
		if err != nil {
			return pagetypes.Scope{}, err
		}
		return deps.PageTypes.SetScope(ctx, in.over(scopeOf(kind, in.key())))
	})
}
