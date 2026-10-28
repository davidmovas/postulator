package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	pagetypesName = "pagetypes"
	opSave        = "save"
	opScope       = "scope"
)

type pagetypesArgs struct {
	Op          string            `json:"op" enum:"list,get,save,delete,scope" description:"list the types in brief; get one whole with its scopes and the field catalog; save creates one without pageTypeId or changes one; scope changes how one field of a type is written"`
	PageTypeID  string            `json:"pageTypeId,omitempty" description:"get, save, delete, scope: the type"`
	Name        *string           `json:"name,omitempty" description:"save: its name; required to create"`
	Preset      string            `json:"preset,omitempty" enum:"landing,category,guide,info,product,store_category,catalog,technical" description:"save: the role it takes; required to create, fixed after"`
	TemplateID  *string           `json:"templateId,omitempty" description:"save: the template of its pages; empty clears"`
	ReassignTo  string            `json:"reassignTo,omitempty" description:"delete: the type its pages move to"`
	Field       string            `json:"field,omitempty" enum:"title,excerpt,body,seo_title,seo_description,product_description,product_short_description,variation_description,image_alt,image_title,upsells,cross_sells" description:"scope: the field; omit for a custom field"`
	MetaKey     string            `json:"metaKey,omitempty" description:"scope: a custom field's meta key, in place of field"`
	Enabled     *bool             `json:"enabled,omitempty" description:"scope: write this field on the type's pages"`
	AIWrites    *bool             `json:"aiWrites,omitempty" description:"scope: the model writes it; false leaves it to a person"`
	Instruction *string           `json:"instruction,omitempty" description:"scope: what the writer is told about the field"`
	MinLength   *int              `json:"minLength,omitempty" minimum:"0" description:"scope: fewest characters; 0 for none"`
	MaxLength   *int              `json:"maxLength,omitempty" minimum:"0" description:"scope: most characters; 0 for none"`
	Links       *field.LinkPolicy `json:"links,omitempty" description:"scope: the whole new link policy"`
	Label       *string           `json:"label,omitempty" description:"scope: a custom field's label"`
	Provider    *string           `json:"provider,omitempty" description:"scope: where a custom field is stored: plugin or store"`
}

func (in *pagetypesArgs) key() string {
	if in.MetaKey != "" {
		return string(field.Instance(field.Meta, in.MetaKey))
	}
	return in.Field
}

func (in *pagetypesArgs) over(current pagetypes.Scope) pagetypes.SetScopeRequest {
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

type PageTypeBrief struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Preset              string   `json:"preset"`
	Role                string   `json:"role"`
	Writable            bool     `json:"writable"`
	TemplateID          *string  `json:"templateId"`
	SuggestedTemplateID *string  `json:"suggestedTemplateId"`
	Pages               int      `json:"pages"`
	Overrides           int      `json:"overrides"`
	Writes              []string `json:"writes"`
}

type PageTypesResponse struct {
	Types  []PageTypeBrief   `json:"types,omitempty"`
	Type   *pagetypes.Type   `json:"type,omitempty"`
	Fields []pagetypes.Field `json:"fields,omitempty"`
}

func briefOf(kind *pagetypes.Type) PageTypeBrief {
	writes := make([]string, 0, len(kind.Scopes))
	for i := range kind.Scopes {
		if kind.Scopes[i].Enabled {
			writes = append(writes, kind.Scopes[i].Field)
		}
	}
	return PageTypeBrief{
		ID: kind.ID, Name: kind.Name, Preset: kind.Preset, Role: kind.Role, Writable: kind.Writable, TemplateID: kind.TemplateID,
		SuggestedTemplateID: kind.Suggested, Pages: kind.Pages, Overrides: kind.Overrides, Writes: writes,
	}
}

func scopeOf(kind pagetypes.Type, key string) pagetypes.Scope {
	for i := range kind.Scopes {
		if kind.Scopes[i].Field == key {
			return kind.Scopes[i]
		}
	}
	return pagetypes.Scope{Field: key, AIWrites: true, Links: field.NoLinks()}
}

func pagetypesTool(deps Deps) Tool {
	tool := newSiteTool(Def{
		Name: pagetypesName,
		Description: "Read and change the site's page types: their roles, templates and field scopes, each field's " +
			"instruction, lengths and link policy.",
		Risk: RiskDangerous,
	}, func(ctx context.Context, b Binding, in pagetypesArgs) (any, error) {
		return deps.managePageTypes(ctx, b, &in)
	})
	tool = confirming(tool, func(ctx context.Context, b Binding, in pagetypesArgs) (string, error) {
		switch in.Op {
		case opDelete:
			return deps.deleteRewrites(ctx, b, &in)
		case opScope:
			return deps.scopeRewrites(ctx, b, &in)
		default:
			return "", nil
		}
	})
	tool = describing(tool, func(ctx context.Context, b Binding, in pagetypesArgs) (string, error) {
		return deps.typeChange(ctx, b, &in)
	})
	return riskedBy(tool, map[string]Risk{opList: RiskRead, opGet: RiskRead, opSave: RiskWrite, opScope: RiskWrite})
}

func (deps Deps) ownType(ctx context.Context, b Binding, typeID string) (pagetypes.Type, error) {
	kind, err := deps.PageTypes.Get(ctx, pagetypes.GetRequest{ID: strings.TrimSpace(typeID)})
	if err != nil {
		return pagetypes.Type{}, err
	}
	if kind.SiteID != b.SiteID {
		return pagetypes.Type{}, errors.New(errors.NotFound, "page type not found").WithDetail("pageTypeId", typeID)
	}
	return kind, nil
}

func (deps Deps) managePageTypes(ctx context.Context, b Binding, in *pagetypesArgs) (any, error) {
	switch in.Op {
	case opList, opGet:
		listed, err := deps.PageTypes.List(ctx, pagetypes.ListRequest{SiteID: b.SiteID})
		if err != nil {
			return nil, err
		}
		if in.Op == opGet {
			kind, kindErr := deps.ownType(ctx, b, in.PageTypeID)
			if kindErr != nil {
				return nil, kindErr
			}
			return PageTypesResponse{Type: &kind, Fields: listed.Fields}, nil
		}
		briefs := make([]PageTypeBrief, 0, len(listed.Types))
		for i := range listed.Types {
			briefs = append(briefs, briefOf(&listed.Types[i]))
		}
		return PageTypesResponse{Types: briefs}, nil
	case opSave:
		if in.PageTypeID == "" {
			return deps.PageTypes.Create(ctx, pagetypes.CreateRequest{
				SiteID: b.SiteID, Name: valueOf(in.Name), Preset: in.Preset, TemplateID: in.TemplateID,
			})
		}
		if _, err := deps.ownType(ctx, b, in.PageTypeID); err != nil {
			return nil, err
		}
		return deps.PageTypes.Save(ctx, pagetypes.SaveRequest{ID: in.PageTypeID, Name: in.Name, TemplateID: in.TemplateID})
	case opDelete:
		if _, err := deps.ownType(ctx, b, in.PageTypeID); err != nil {
			return nil, err
		}
		return deps.PageTypes.Delete(ctx, pagetypes.DeleteRequest{ID: in.PageTypeID, ReassignTo: in.ReassignTo})
	case opScope:
		kind, err := deps.ownType(ctx, b, in.PageTypeID)
		if err != nil {
			return nil, err
		}
		return deps.PageTypes.SetScope(ctx, in.over(scopeOf(kind, in.key())))
	default:
		return nil, errors.New(errors.Invalid, "pagetypes takes the op list, get, save, delete or scope").WithDetail("field", "op")
	}
}

func (deps Deps) deleteRewrites(ctx context.Context, b Binding, in *pagetypesArgs) (string, error) {
	target, err := deps.ownType(ctx, b, in.ReassignTo)
	if err != nil {
		return "", err
	}
	fields := rewrittenFields(target)
	if len(fields) == 0 {
		return "", nil
	}
	leaving, err := deps.ownType(ctx, b, in.PageTypeID)
	if err != nil || leaving.Pages == 0 {
		return "", err
	}
	return "Delete the page type " + leaving.Name + " and move its " + strconv.Itoa(leaving.Pages) + " pages to " +
		whoseScopeRewrites(target, fields) + ": their own text will be rewritten by the next run", nil
}

func (deps Deps) scopeRewrites(ctx context.Context, b Binding, in *pagetypesArgs) (string, error) {
	kind, err := deps.ownType(ctx, b, in.PageTypeID)
	if err != nil {
		return "", err
	}
	current := scopeOf(kind, in.key())
	switching := (in.Enabled != nil && *in.Enabled && !current.Enabled) || (in.AIWrites != nil && *in.AIWrites && !current.AIWrites)
	if !current.Rewrites || !switching {
		return "", nil
	}
	return "Switch on the " + in.key() + " of " + kind.Name + ", which a person set to rewrite: the pages' own text " +
		"in it will be rewritten by the next run", nil
}

func (deps Deps) typeChange(ctx context.Context, b Binding, in *pagetypesArgs) (string, error) {
	if in.Op == opSave && in.PageTypeID == "" {
		return "Create the page type " + valueOf(in.Name) + " over the " + in.Preset + " role", nil
	}
	kind, err := deps.ownType(ctx, b, in.PageTypeID)
	if err != nil {
		return "", err
	}
	switch in.Op {
	case opSave:
		return "Change the page type " + kind.Name + "'s " + strings.Join(savedParts(in), " and "), nil
	case opDelete:
		target, targetErr := deps.ownType(ctx, b, in.ReassignTo)
		if targetErr != nil {
			return "", targetErr
		}
		return "Delete the page type " + kind.Name + " and move its " + strconv.Itoa(kind.Pages) + " pages to " + target.Name, nil
	default:
		return "Change how the " + in.key() + " of " + kind.Name + " pages is written", nil
	}
}

func savedParts(in *pagetypesArgs) []string {
	parts := make([]string, 0, 2)
	if in.Name != nil {
		parts = append(parts, "name")
	}
	if in.TemplateID != nil {
		parts = append(parts, "template")
	}
	if len(parts) == 0 {
		return []string{"nothing"}
	}
	return parts
}
