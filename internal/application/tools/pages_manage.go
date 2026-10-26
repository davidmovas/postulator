package tools

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	pagesManageName = "pages_manage"
	opCreate        = "create"
	opUpdate        = "update"
	opDelete        = "delete"
	opMap           = "map"
	opUnmap         = "unmap"
	opCanonical     = "canonical"
	opPreviewLink   = "preview_link"
	opSetTypes      = "set_types"
	opSetTemplates  = "set_templates"
)

type pagesManageArgs struct {
	Op              string         `json:"op" enum:"create,update,delete,map,unmap,canonical,preview_link,set_types,set_templates" description:"create plans a page in the map; preview_link signs a one-hour link to a draft; set_types and set_templates change many pages"`
	PageID          string         `json:"pageId,omitempty" description:"The page; create and the set ops take none"`
	Path            *string        `json:"path,omitempty" description:"create, update: site path with both slashes, e.g. /supplements/creatine/"`
	WPType          *string        `json:"wpType,omitempty" enum:"page,post,product,product_cat" description:"create, update: WordPress type; create's default page"`
	Title           *string        `json:"title,omitempty" description:"create, update: WordPress title"`
	H1              *string        `json:"h1,omitempty" description:"create, update: H1; create's default the title"`
	MetaTitle       *string        `json:"metaTitle,omitempty" description:"create, update: SEO title; default from the template"`
	MetaDescription *string        `json:"metaDescription,omitempty" description:"create, update: SEO description"`
	Keywords        *[]dto.Keyword `json:"keywords,omitempty" description:"create, update: the whole keyword list; empty follows its entity"`
	Status          *string        `json:"status,omitempty" enum:"planned,exists,published,archived" description:"create, update: the page map's state; default planned"`
	EntityID        *string        `json:"entityId,omitempty" description:"create, map, canonical: the entity"`
	TemplateID      *string        `json:"templateId,omitempty" description:"create, update, set_templates: template id"`
	PageTypeID      *string        `json:"pageTypeId,omitempty" description:"update, set_types: type id from pagetypes"`
	OnSite          bool           `json:"onSite,omitempty" description:"delete: also move it to the WordPress trash, still restorable"`
	Reset           bool           `json:"reset,omitempty" description:"set_types: back to the rules; set_templates: back to each type's template"`
	Select          *selectArgs    `json:"select,omitempty" description:"set_types, set_templates: the pages, at most 10,000"`
}

func pagesManage(deps Deps) Tool {
	tool := newSiteTool(Def{
		Name: pagesManageName,
		Description: "Change the page map: plan, change or delete a page, map it to its entity or make it the " +
			"entity's canonical, sign a preview link, or retype or re-template many pages at once.",
		Risk: RiskDangerous,
	}, func(ctx context.Context, b Binding, in pagesManageArgs) (any, error) {
		return deps.managePage(ctx, b, &in)
	})
	tool = confirming(tool, func(ctx context.Context, b Binding, in pagesManageArgs) (string, error) {
		return deps.mustAsk(ctx, b, &in)
	})
	tool = describing(tool, func(ctx context.Context, b Binding, in pagesManageArgs) (string, error) {
		return deps.pageChange(ctx, b, &in)
	})
	return riskedBy(tool, map[string]Risk{
		opCreate: RiskWrite, opUpdate: RiskWrite, opMap: RiskWrite, opUnmap: RiskWrite, opCanonical: RiskWrite,
		opPreviewLink: RiskWrite, opSetTypes: RiskWrite, opSetTemplates: RiskWrite,
	})
}

func (deps Deps) managePage(ctx context.Context, b Binding, in *pagesManageArgs) (any, error) {
	if in.Op == opCreate {
		return deps.Pages.Create(ctx, pages.CreateRequest{
			SiteID: b.SiteID, Path: valueOf(in.Path), WPType: valueOf(in.WPType), Title: valueOf(in.Title), H1: valueOf(in.H1),
			MetaTitle: valueOf(in.MetaTitle), MetaDescription: valueOf(in.MetaDescription), Keywords: valueOf(in.Keywords),
			Status: valueOf(in.Status), EntityID: in.EntityID, TemplateID: in.TemplateID,
		})
	}
	if in.Op == opSetTypes || in.Op == opSetTemplates {
		return deps.manageMany(ctx, b, in)
	}
	if !knownPageOp(in.Op) {
		return nil, errors.New(errors.Invalid, "pages_manage takes the op create, update, delete, map, unmap, canonical, "+
			"preview_link, set_types or set_templates").WithDetail("field", "op")
	}
	page, err := deps.ownPage(ctx, b, in.PageID)
	if err != nil {
		return nil, err
	}
	switch in.Op {
	case opUpdate:
		return deps.updatePage(ctx, page.ID, in)
	case opDelete:
		return deps.Pages.Delete(ctx, pages.DeleteRequest{ID: page.ID, OnSite: in.OnSite})
	case opMap:
		return deps.Pages.MapToEntity(ctx, pages.MapToEntityRequest{PageID: page.ID, EntityID: valueOf(in.EntityID)})
	case opUnmap:
		return deps.Pages.Unmap(ctx, pages.UnmapRequest{PageID: page.ID})
	case opCanonical:
		return deps.Pages.SetCanonical(ctx, pages.SetCanonicalRequest{PageID: page.ID, EntityID: valueOf(in.EntityID)})
	default:
		return deps.Pages.PreviewLink(ctx, pages.PreviewLinkRequest{PageID: page.ID})
	}
}

func knownPageOp(op string) bool {
	switch op {
	case opUpdate, opDelete, opMap, opUnmap, opCanonical, opPreviewLink:
		return true
	default:
		return false
	}
}

func (in *pagesManageArgs) touchesPage() bool {
	return in.Path != nil || in.WPType != nil || in.Title != nil || in.H1 != nil || in.MetaTitle != nil ||
		in.MetaDescription != nil || in.Keywords != nil || in.Status != nil || in.TemplateID != nil
}

func (deps Deps) updatePage(ctx context.Context, pageID string, in *pagesManageArgs) (pages.UpdateResponse, error) {
	if in.PageTypeID == nil || in.touchesPage() {
		updated, err := deps.Pages.Update(ctx, pages.UpdateRequest{
			ID: pageID, Path: in.Path, WPType: in.WPType, Title: in.Title, H1: in.H1, MetaTitle: in.MetaTitle,
			MetaDescription: in.MetaDescription, Keywords: in.Keywords, Status: in.Status, TemplateID: in.TemplateID,
		})
		if err != nil || in.PageTypeID == nil {
			return updated, err
		}
	}
	typed, err := deps.Pages.SetType(ctx, pages.SetTypeRequest{PageID: pageID, PageTypeID: *in.PageTypeID, By: pagemap.TypeByAI})
	if err != nil {
		return pages.UpdateResponse{}, err
	}
	return pages.UpdateResponse(typed), nil
}

func (deps Deps) manageMany(ctx context.Context, b Binding, in *pagesManageArgs) (any, error) {
	if in.Select.empty() {
		return nil, errors.New(errors.Invalid, "name the pages: pageIds, under or ofType").WithDetail("field", "select")
	}
	sel := in.Select.selection(b.SiteID)
	if in.Op == opSetTypes {
		typeID, err := chosen(in.PageTypeID, in.Reset, "pageTypeId", "a type")
		if err != nil {
			return nil, err
		}
		return deps.Pages.SetTypes(ctx, pages.SetTypesRequest{Selection: sel, PageTypeID: typeID, By: pagemap.TypeByAI})
	}
	templateID, err := chosen(in.TemplateID, in.Reset, "templateId", "a template")
	if err != nil {
		return nil, err
	}
	return deps.Pages.SetTemplates(ctx, pages.SetTemplatesRequest{Selection: sel, TemplateID: templateID})
}

func chosen(id *string, reset bool, field, what string) (*string, error) {
	switch {
	case reset && id != nil:
		return nil, errors.New(errors.Invalid, "give "+field+" or reset, not both").WithDetail("field", field)
	case reset:
		return nil, nil
	case id == nil || strings.TrimSpace(*id) == "":
		return nil, errors.New(errors.Invalid, "name "+what+" by "+field+", or reset").WithDetail("field", field)
	default:
		return id, nil
	}
}

func (deps Deps) mustAsk(ctx context.Context, b Binding, in *pagesManageArgs) (string, error) {
	switch {
	case in.Op == opDelete && in.OnSite:
		page, err := deps.ownPage(ctx, b, in.PageID)
		if err != nil {
			return "", err
		}
		return "Move " + page.Path + " to the WordPress trash: it leaves the site for its visitors until it is " +
			"restored there", nil
	case in.Op == opUpdate && in.PageTypeID != nil:
		return deps.rewritingType(ctx, b, *in.PageTypeID, func() (string, error) {
			page, err := deps.ownPage(ctx, b, in.PageID)
			return page.Path, err
		})
	case in.Op == opSetTypes && in.PageTypeID != nil && !in.Reset:
		return deps.rewritingType(ctx, b, *in.PageTypeID, func() (string, error) {
			picked, err := deps.selectPages(ctx, b, in.Select)
			if err != nil {
				return "", err
			}
			return picked.named(), nil
		})
	default:
		return "", nil
	}
}

func (deps Deps) rewritingType(ctx context.Context, b Binding, typeID string, what func() (string, error)) (string, error) {
	kind, err := deps.PageTypes.Get(ctx, pagetypes.GetRequest{ID: strings.TrimSpace(typeID)})
	if err != nil {
		return "", err
	}
	if kind.SiteID != b.SiteID {
		return "", errors.New(errors.NotFound, "page type not found").WithDetail("pageTypeId", typeID)
	}
	fields := rewrittenFields(kind)
	if len(fields) == 0 {
		return "", nil
	}
	named, err := what()
	if err != nil {
		return "", err
	}
	return "Type " + named + " as " + whoseScopeRewrites(kind, fields) + ": the page's own text will be rewritten by the next run", nil
}

func (deps Deps) pageChange(ctx context.Context, b Binding, in *pagesManageArgs) (string, error) {
	switch in.Op {
	case opCreate:
		return "Plan the page " + valueOf(in.Path) + " in the page map; nothing reaches the site until a run publishes it", nil
	case opSetTypes, opSetTemplates:
		return deps.manyChange(ctx, b, in)
	}
	page, err := deps.ownPage(ctx, b, in.PageID)
	if err != nil {
		return "", err
	}
	switch in.Op {
	case opUpdate:
		return "Change the " + strings.Join(in.changed(), ", ") + " of " + page.Path + " in the page map", nil
	case opDelete:
		return "Delete " + page.Path + " from the page map", nil
	case opMap, opCanonical:
		name, err := deps.entityName(ctx, valueOf(in.EntityID))
		if err != nil {
			return "", err
		}
		if in.Op == opMap {
			return "Map " + page.Path + " to the entity " + name, nil
		}
		return "Make " + page.Path + " the canonical page of " + name, nil
	case opUnmap:
		return "Detach " + page.Path + " from its entity", nil
	default:
		return "Sign a one-hour link that shows " + page.Path + " as the theme renders it to anyone holding it", nil
	}
}

func (deps Deps) manyChange(ctx context.Context, b Binding, in *pagesManageArgs) (string, error) {
	picked, err := deps.selectPages(ctx, b, in.Select)
	if err != nil {
		return "", err
	}
	switch {
	case in.Op == opSetTypes && in.Reset:
		return "Hand " + picked.named() + " back to the classification rules", nil
	case in.Op == opSetTypes:
		kind, err := deps.PageTypes.Get(ctx, pagetypes.GetRequest{ID: valueOf(in.PageTypeID)})
		if err != nil {
			return "", err
		}
		return "Type " + picked.named() + " as " + kind.Name + "; a page whose WordPress type the role refuses keeps its type", nil
	case in.Reset:
		return "Hand " + picked.named() + " back to their type's template", nil
	default:
		held, err := deps.Templates.GetTemplate(ctx, templates.GetTemplateRequest{ID: valueOf(in.TemplateID)})
		if err != nil {
			return "", err
		}
		return "Give " + picked.named() + " the template " + held.Template.Name, nil
	}
}

func (in *pagesManageArgs) changed() []string {
	named := []struct {
		set  bool
		name string
	}{
		{in.Path != nil, "path"}, {in.WPType != nil, "WordPress type"}, {in.Title != nil, "title"}, {in.H1 != nil, "H1"},
		{in.MetaTitle != nil, "SEO title"}, {in.MetaDescription != nil, "SEO description"}, {in.Keywords != nil, "keywords"},
		{in.Status != nil, "state"}, {in.TemplateID != nil, "template"}, {in.PageTypeID != nil, "type"},
	}
	out := make([]string, 0, len(named))
	for _, held := range named {
		if held.set {
			out = append(out, held.name)
		}
	}
	if len(out) == 0 {
		return []string{"nothing"}
	}
	return out
}

func valueOf[T any](held *T) T {
	var zero T
	if held == nil {
		return zero
	}
	return *held
}
