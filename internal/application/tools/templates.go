package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	templatesName    = "templates"
	opOverride       = "override"
	opDeleteOverride = "delete_override"
	scopeSite        = "site"
)

type templatesArgs struct {
	Op         string            `json:"op" enum:"list,get,create,update,delete,override,delete_override" description:"get reads one with its overrides; override patches one for this site, or for one page with pageId"`
	TemplateID string            `json:"templateId,omitempty" description:"get, update, delete, override: the template"`
	OverrideID string            `json:"overrideId,omitempty" description:"delete_override: the override, from get"`
	Scope      string            `json:"scope,omitempty" enum:"global,site" description:"list: only this scope; create: global, the default, or this site's"`
	Name       *string           `json:"name,omitempty" description:"create, update: the name, 2-4 words"`
	Spec       *templateSpecArgs `json:"spec,omitempty" description:"create, update: the whole specification; override: only the parts to change"`
	PageID     string            `json:"pageId,omitempty" description:"override: one page of this site; omit for the whole site"`
	Cursor     string            `json:"cursor,omitempty" description:"list: nextCursor of the last call"`
	Limit      int               `json:"limit,omitempty" description:"list: default 50"`
}

func templatesTool(deps Deps) Tool {
	tool := checking(newSiteTool(Def{
		Name: templatesName,
		Description: "Read and change the content templates, global or this site's: sections, tone, length, keyword " +
			"and meta rules, images, product outputs, model per job and recipe, and their overrides for this site or a page.",
		Risk: RiskDangerous,
	}, func(ctx context.Context, b Binding, in templatesArgs) (any, error) {
		return deps.manageTemplates(ctx, b, &in)
	}), func(in templatesArgs) error {
		if (in.Op == opCreate || in.Op == opUpdate) && in.Spec != nil {
			return template.Validate(in.Spec.spec())
		}
		return nil
	})
	tool = describing(tool, func(ctx context.Context, _ Binding, in templatesArgs) (string, error) {
		return deps.templateChange(ctx, &in)
	})
	return riskedBy(tool, map[string]Risk{
		opList: RiskRead, opGet: RiskRead, opCreate: RiskWrite, opUpdate: RiskWrite, opOverride: RiskWrite,
	})
}

func (deps Deps) manageTemplates(ctx context.Context, b Binding, in *templatesArgs) (any, error) {
	switch in.Op {
	case opList:
		return fitted(deps, in.Limit, func(limit int) (paging.List[templates.Template], error) {
			return deps.Templates.ListTemplates(ctx, templates.ListTemplatesRequest{
				ListRequest: dto.ListRequest{Cursor: in.Cursor, Limit: limit}, Scope: in.Scope,
			})
		})
	case opGet:
		return deps.Templates.GetTemplate(ctx, templates.GetTemplateRequest{ID: in.TemplateID})
	case opCreate:
		if in.Spec == nil {
			return nil, errors.New(errors.Invalid, "create takes the whole specification").WithDetail("field", "spec")
		}
		req := templates.CreateTemplateRequest{Scope: in.Scope, Name: valueOf(in.Name), Spec: in.Spec.spec()}
		if in.Scope == scopeSite {
			req.SiteID = &b.SiteID
		}
		return deps.Templates.CreateTemplate(ctx, req)
	case opUpdate:
		req := templates.UpdateTemplateRequest{ID: in.TemplateID, Name: in.Name}
		if in.Spec != nil {
			spec := in.Spec.spec()
			req.Spec = &spec
		}
		return deps.Templates.UpdateTemplate(ctx, req)
	case opDelete:
		return deps.Templates.DeleteTemplate(ctx, templates.DeleteTemplateRequest{ID: in.TemplateID})
	case opOverride:
		if in.Spec == nil {
			return nil, errors.New(errors.Invalid, "an override takes the parts of the specification it changes").WithDetail("field", "spec")
		}
		patch, err := in.Spec.patch()
		if err != nil {
			return nil, err
		}
		req := templates.SetOverrideRequest{TemplateID: in.TemplateID, Scope: scopeSite, TargetID: b.SiteID, Patch: patch}
		if in.PageID != "" {
			page, pageErr := deps.ownPage(ctx, b, in.PageID)
			if pageErr != nil {
				return nil, pageErr
			}
			req.Scope, req.TargetID = "page", page.ID
		}
		return deps.Templates.SetOverride(ctx, req)
	case opDeleteOverride:
		return deps.Templates.DeleteOverride(ctx, templates.DeleteOverrideRequest{ID: in.OverrideID})
	default:
		return nil, errors.New(errors.Invalid, "templates takes the op list, get, create, update, delete, override or delete_override").
			WithDetail("field", "op")
	}
}

func (deps Deps) templateChange(ctx context.Context, in *templatesArgs) (string, error) {
	if in.Op == opCreate {
		where := "a global template"
		if in.Scope == scopeSite {
			where = "a template of this site"
		}
		return "Create " + where + " named " + quoted(valueOf(in.Name)) + specSaid(in.Spec), nil
	}
	if in.Op == opDeleteOverride {
		return "Remove the template override " + in.OverrideID, nil
	}
	held, err := deps.Templates.GetTemplate(ctx, templates.GetTemplateRequest{ID: in.TemplateID})
	if err != nil {
		return "", err
	}
	name := held.Template.Name
	switch in.Op {
	case opUpdate:
		return "Change the template " + name + specSaid(in.Spec), nil
	case opDelete:
		return "Delete the template " + name, nil
	default:
		where := "on this site"
		if in.PageID != "" {
			path, pathErr := deps.pathOf(ctx, in.PageID)
			if pathErr != nil {
				return "", pathErr
			}
			where = "on " + path
		}
		return "Override the template " + name + " " + where + specSaid(in.Spec), nil
	}
}

func specSaid(spec *templateSpecArgs) string {
	if spec == nil {
		return ""
	}
	parts := make([]string, 0, 9)
	if len(spec.Sections) > 0 {
		parts = append(parts, plural(len(spec.Sections), "section"))
	}
	named := []struct {
		set  bool
		name string
	}{
		{spec.Tone != nil, "tone"}, {spec.Length != nil, "length"}, {spec.KeywordRules != nil, "keyword rules"},
		{spec.MetaRules != nil, "meta rules"}, {spec.Images != nil, "images"}, {spec.Product != nil, "product outputs"},
		{len(spec.ModelProfiles) > 0, strconv.Itoa(len(spec.ModelProfiles)) + " model jobs"}, {len(spec.Recipe) > 0, "recipe"},
	}
	for _, held := range named {
		if held.set {
			parts = append(parts, held.name)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return ": " + strings.Join(parts, ", ")
}
