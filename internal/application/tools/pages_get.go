package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	pagesGetName  = "pages_get"
	viewPack      = "pack"
	viewReports   = "reports"
	bytesPerToken = 4
)

type pagesGetArgs struct {
	View    string `json:"view,omitempty" enum:"model,pack,reports" description:"Default model; pack is what the writer reads for a field, reports the page's newest validation, judge, publish and relink reports"`
	PageID  string `json:"pageId,omitempty" description:"Page id, or give path"`
	Path    string `json:"path,omitempty" description:"Page path"`
	Field   string `json:"field,omitempty" enum:"title,excerpt,body,seo_title,seo_description,product_description,product_short_description,variation_description,image_alt,image_title" description:"pack: default the body"`
	MetaKey string `json:"metaKey,omitempty" description:"pack: a custom field's meta key, in place of field"`
}

type TemplateApplied struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Source   string   `json:"source"`
	Version  int      `json:"version"`
	Headings []string `json:"headings"`
}

type PagesGetResponse struct {
	Page     *sitemodel.PageModel        `json:"page,omitempty"`
	Template *TemplateApplied            `json:"template,omitempty"`
	Pack     *sitemodel.PackText         `json:"pack,omitempty"`
	Reports  *reports.PageReportResponse `json:"reports,omitempty"`
}

func pagesGet(deps Deps) Tool {
	return newSiteTool(Def{
		Name: pagesGetName,
		Description: "Read one page: its model (identity, type and why, parent or why none, fields with current, base " +
			"and owner, the body in sum, neighbors, open findings, the template a run would use), the writer's pack, or its reports.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in pagesGetArgs) (PagesGetResponse, error) {
		switch in.View {
		case viewPack:
			req := sitemodel.PackTextRequest{SiteID: b.SiteID, PageID: in.PageID, Path: in.Path, Field: in.Field}
			if in.MetaKey != "" {
				req.Field = string(field.Instance(field.Meta, in.MetaKey))
			}
			if deps.ResultBytes != nil {
				req.Budget = deps.ResultBytes() / bytesPerToken
			}
			text, err := deps.SiteModel.PackText(ctx, req)
			if err != nil {
				return PagesGetResponse{}, err
			}
			return PagesGetResponse{Pack: &text}, nil
		case viewReports:
			model, err := deps.SiteModel.Page(ctx, sitemodel.PageRequest{SiteID: b.SiteID, PageID: in.PageID, Path: in.Path})
			if err != nil {
				return PagesGetResponse{}, err
			}
			report, err := deps.Reports.PageReport(ctx, reports.PageReportRequest{PageID: model.Page.PageID})
			if err != nil {
				return PagesGetResponse{}, err
			}
			return PagesGetResponse{Reports: &report}, nil
		default:
			return deps.pageModel(ctx, b, in)
		}
	})
}

func (deps Deps) pageModel(ctx context.Context, b Binding, in pagesGetArgs) (PagesGetResponse, error) {
	model, err := deps.SiteModel.Page(ctx, sitemodel.PageRequest{SiteID: b.SiteID, PageID: in.PageID, Path: in.Path})
	if err != nil {
		return PagesGetResponse{}, err
	}
	answered := PagesGetResponse{Page: &model}
	resolved, err := deps.Templates.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: model.Page.PageID})
	switch {
	case errors.IsCode(err, errors.NotFound):
	case err != nil:
		return PagesGetResponse{}, err
	default:
		applied := &TemplateApplied{
			ID: resolved.TemplateID, Name: resolved.Name, Source: resolved.Source, Version: resolved.Version,
			Headings: make([]string, 0, len(resolved.Spec.Sections)),
		}
		for _, section := range resolved.Spec.Sections {
			applied.Headings = append(applied.Headings, section.Heading)
		}
		answered.Template = applied
	}
	return answered, nil
}
