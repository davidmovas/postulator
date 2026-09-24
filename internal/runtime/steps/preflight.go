package steps

import (
	"context"
	"strconv"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const (
	CodeEntityMissing          = "entity_missing"
	CodeRequiredTargetUnplaced = "required_target_unplaced"
	CodeImageSourceUnavailable = "image_source_unavailable"
	CodePluginMissing          = "plugin_missing"
)

func effectivePolicyFor(ctx context.Context, deps Deps, siteID string, spec template.TemplateSpec) (template.LinkPolicy, error) {
	resp, err := deps.Policies.GetEffectivePolicy(ctx, templates.GetEffectivePolicyRequest{SiteID: siteID})
	if err != nil {
		return template.LinkPolicy{}, err
	}

	return template.LinkPolicy{
		Rules:          templates.EffectiveRules(resp.Policy.Rules, spec.LinkRules),
		ForbidExternal: resp.Policy.ForbidExternal,
		ForbidSelf:     resp.Policy.ForbidSelf,
		AnchorStrategy: template.AnchorStrategy(resp.Policy.AnchorStrategy),
	}, nil
}

func pageFinding(severity content.Severity, code string, page pagemap.Page, message string) run.EstimateFinding {
	return run.EstimateFinding{Severity: severity, Code: code, Message: message, PageID: page.ID, Path: page.Path}
}

func graphPreflight(deps Deps) run.Preflight {
	return func(ctx context.Context, record run.Run, targets map[string]run.Target) ([]run.EstimateFinding, error) {
		findings := make([]run.EstimateFinding, 0)
		if len(record.Targets) == 0 {
			return findings, nil
		}

		entities, err := deps.Entities.ListBySite(ctx, record.SiteID)
		if err != nil {
			return nil, err
		}
		edges, err := deps.Edges.ListBySite(ctx, record.SiteID)
		if err != nil {
			return nil, err
		}
		pages, err := deps.Pages.ListBySite(ctx, record.SiteID)
		if err != nil {
			return nil, err
		}
		owner, err := deps.Sites.Get(ctx, record.SiteID)
		if err != nil {
			return nil, err
		}
		g, err := graph.New(entities, edges)
		if err != nil {
			return nil, err
		}
		index := pagemap.NewIndex(pages)

		unplacedSeverity := content.SeverityError
		if allowed, ok := run.ParamsFor(record.Recipe, NameValidate)[ParamAllowErrors].(bool); ok && allowed {
			unplacedSeverity = content.SeverityWarn
		}

		for _, targetID := range record.Targets {
			target := targets[targetID]
			page := target.Page
			if page.EntityID == nil || *page.EntityID == "" {
				findings = append(findings, pageFinding(content.SeverityError, CodeEntityMissing, page,
					page.Path+" is mapped to no entity, so nothing says what it must link to; map it on the Graph screen"))
				continue
			}
			entity, held := g.Entity(*page.EntityID)
			if !held {
				findings = append(findings, pageFinding(content.SeverityError, CodeEntityMissing, page,
					page.Path+" is mapped to an entity the graph no longer holds; map it again on the Graph screen"))
				continue
			}

			policy, policyErr := effectivePolicyFor(ctx, deps, record.SiteID, target.Spec)
			if policyErr != nil {
				return nil, policyErr
			}
			plan := content.PlanLinks(g, index, content.Subject{
				Site: pagemap.NewSite(owner.BaseURL), PageID: page.ID, PagePath: page.Path, EntityID: entity.ID,
			}, policy)
			for _, blocked := range plan.Blocked {
				if !blocked.Required {
					continue
				}
				name := blocked.EntityID
				if missing, known := g.Entity(blocked.EntityID); known {
					name = missing.Name
				}
				findings = append(findings, pageFinding(unplacedSeverity, CodeRequiredTargetUnplaced, page,
					page.Path+" must link to "+name+", which has no page yet, so validate would hold it; "+
						"plan a page for "+name+" or accept the finding when the page is held"))
			}
		}
		return findings, nil
	}
}

func imagesPreflight(deps Deps) run.Preflight {
	return func(_ context.Context, record run.Run, targets map[string]run.Target) ([]run.EstimateFinding, error) {
		findings := make([]run.EstimateFinding, 0)
		for _, targetID := range record.Targets {
			target := targets[targetID]
			wanted := wantedImages(target.Spec.Images)
			if wanted == 0 {
				continue
			}
			count := strconv.Itoa(wanted)
			source := target.Spec.Images.Source
			switch {
			case source == template.ImagesAI && deps.ImageProvider == nil:
				findings = append(findings, pageFinding(content.SeverityWarn, CodeImageSourceUnavailable, target.Page,
					"no model is configured to draw images, so "+target.Page.Path+" will be written without the "+
						count+" images its template asks for"))
			case source != template.ImagesAI && deps.ImageSources[source] == nil:
				findings = append(findings, pageFinding(content.SeverityWarn, CodeImageSourceUnavailable, target.Page,
					"no image library named "+string(source)+" is configured, so "+target.Page.Path+
						" will be written without the "+count+" images its template asks for"))
			}
		}
		return findings, nil
	}
}

func pluginPreflight(deps Deps, step, consequence string) run.Preflight {
	return func(ctx context.Context, record run.Run, _ map[string]run.Target) ([]run.EstimateFinding, error) {
		owner, err := deps.Sites.Get(ctx, record.SiteID)
		if err != nil {
			return nil, err
		}
		if owner.Plugin.Installed {
			return []run.EstimateFinding{}, nil
		}
		return []run.EstimateFinding{{
			Severity: content.SeverityWarn, Code: CodePluginMissing,
			Message: "the site has no Postulator companion plugin, so " + step + " " + consequence +
				"; install the plugin and sync the site to change that",
		}}, nil
	}
}
