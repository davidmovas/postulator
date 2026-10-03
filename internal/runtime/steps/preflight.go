package steps

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const (
	CodeEntityMissing          = "entity_missing"
	CodeRequiredTargetUnplaced = "required_target_unplaced"
	CodeImageSourceUnavailable = "image_source_unavailable"
	CodePluginMissing          = "plugin_missing"
)

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

			policy, policyErr := effectivePolicy(ctx, deps, record.SiteID, target.Spec)
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
			wanted := target.Spec.Images.Wanted()
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

func preflights(checks ...run.Preflight) run.Preflight {
	return func(ctx context.Context, record run.Run, targets map[string]run.Target) ([]run.EstimateFinding, error) {
		found := make([]run.EstimateFinding, 0)
		for _, check := range checks {
			more, err := check(ctx, record, targets)
			if err != nil {
				return nil, err
			}
			found = append(found, more...)
		}
		return found, nil
	}
}

func storePreflight(deps Deps) run.Preflight {
	return func(ctx context.Context, record run.Run, targets map[string]run.Target) ([]run.EstimateFinding, error) {
		findings := make([]run.EstimateFinding, 0)
		var owner *site.Site
		for _, targetID := range record.Targets {
			target := targets[targetID]
			page := target.Page
			switch page.WPType {
			case pagemap.WPProductCategory:
				findings = append(findings, pageFinding(content.SeverityError, CodeProductCategoryUnwritable, page,
					page.Path+" is a product category, whose description the companion plugin does not write; "+
						"write it in WooCommerce"))
				continue
			case pagemap.WPProduct:
			default:
				if target.Spec.Product != nil {
					findings = append(findings, pageFinding(content.SeverityWarn, CodeProductOutputsIgnored, page,
						"the template of "+page.Path+" declares product outputs and "+page.Path+" is a "+
							string(page.WPType)+", so they are not written"))
				}
				continue
			}

			if owner == nil {
				held, err := deps.Sites.Get(ctx, record.SiteID)
				if err != nil {
					return nil, err
				}
				owner = &held
			}
			findings = append(findings, productFindings(*owner, record, target)...)
		}
		return findings, nil
	}
}

func productFindings(owner site.Site, record run.Run, target run.Target) []run.EstimateFinding {
	page := target.Page
	found := make([]run.EstimateFinding, 0, 2)
	switch owner.Commerce {
	case site.CommerceUnknown:
		found = append(found, pageFinding(content.SeverityError, CodeCommerceUnknown, page,
			page.Path+" is a product, and "+owner.Name+" has not been checked for a store yet; "+
				"open it on the Sites screen and press Recheck, or sync the site"))
	case site.CommerceAbsent:
		found = append(found, pageFinding(content.SeverityError, CodeCommerceAbsent, page,
			page.Path+" is a product, and "+owner.Name+" answers no WooCommerce store, so there is nothing to write it to; "+
				"activate WooCommerce and recheck the site, or change the row's type to page"))
	case site.CommerceForbidden:
		found = append(found, pageFinding(content.SeverityError, CodeCommerceForbidden, page,
			page.Path+" is a product, and the WordPress user "+owner.Username+" may not edit products on "+owner.Name+
				"; use the application password of an administrator or a shop manager and recheck the site"))
	case site.CommerceReady:
	}
	if !owner.Plugin.Installed || !slices.Contains(owner.Plugin.Capabilities, wp.CapabilityRaw) {
		found = append(found, pageFinding(content.SeverityError, CodeProductNeedsPlugin, page,
			page.Path+" is a product, and its description is written through the Postulator companion plugin, "+
				"which "+owner.Name+" does not carry; install the plugin and recheck the site"))
	}
	if page.WPID == nil {
		found = append(found, pageFinding(content.SeverityError, CodeProductNotInStore, page, page.Path+" "+notInStore))
	}
	if record.PublishMode != run.PublishLive {
		found = append(found, pageFinding(content.SeverityError, CodeProductEditedLive, page,
			page.Path+" is a product, which has no draft copy, so its text goes live the moment it is written; "+
				"start the run in live mode"))
	}
	if target.Spec.Product == nil {
		found = append(found, pageFinding(content.SeverityWarn, CodeProductOutputsMissing, page,
			"the template of "+page.Path+" declares no product outputs, so the product gets its description and nothing else"))
	}
	return found
}

func categoryPreflight(deps Deps) run.Preflight {
	return func(ctx context.Context, record run.Run, targets map[string]run.Target) ([]run.EstimateFinding, error) {
		findings := make([]run.EstimateFinding, 0)
		pages := make([]pagemap.Page, 0, len(record.Targets))
		for _, targetID := range record.Targets {
			page := targets[targetID].Page
			if page.WPType == pagemap.WPPage && page.EntityID != nil && *page.EntityID != "" {
				pages = append(pages, page)
			}
		}
		if len(pages) == 0 {
			return findings, nil
		}

		owner, err := deps.Sites.Get(ctx, record.SiteID)
		if err != nil {
			return nil, err
		}
		if owner.Plugin.Installed && slices.Contains(owner.Plugin.Capabilities, wp.CapabilityPageCategories) {
			return findings, nil
		}
		entities, err := deps.Entities.ListBySite(ctx, record.SiteID)
		if err != nil {
			return nil, err
		}
		for i := range pages {
			chain := graph.CategoryChain(entities, *pages[i].EntityID)
			if len(chain) == 0 {
				continue
			}
			findings = append(findings, pageFinding(content.SeverityWarn, CodePageCategoriesNeedPlugin, pages[i],
				pages[i].Path+" is filed under "+strings.Join(chainNames(chain), chainSeparator)+", and WordPress pages "+
					"carry categories only through the Postulator companion plugin "+pageCategoriesPlugin+", which "+
					owner.Name+" does not carry; update the plugin and sync the site, or the page goes up without them"))
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
