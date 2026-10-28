package runs

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/run"
)

type writable struct {
	targets  []string
	skipped  []run.Skipped
	resolved map[string]templates.ResolveForPageResponse
}

func (s *Service) writable(ctx context.Context, siteID string, targets []string, picked string) (writable, error) {
	set, err := s.types.Set(ctx, siteID)
	if err != nil {
		return writable{}, err
	}
	ready := writable{
		targets:  make([]string, 0, len(targets)),
		skipped:  make([]run.Skipped, 0),
		resolved: make(map[string]templates.ResolveForPageResponse, len(targets)),
	}
	for _, pageID := range targets {
		page, getErr := s.pages.Get(ctx, pageID)
		if getErr != nil {
			return writable{}, getErr
		}
		left := run.Skipped{PageID: pageID, Path: cmp.Or(page.Path, pageID), PageType: set.TypeOf(page).Name}
		if page.EntityID == nil || strings.TrimSpace(*page.EntityID) == "" {
			left.Reason = run.SkipNoEntity
			ready.skipped = append(ready.skipped, left)
			continue
		}
		resolved, resolveErr := s.resolve(ctx, siteID, pageID, picked)
		if templates.IsUnassigned(resolveErr) {
			left.Reason = run.SkipNoTemplate
			ready.skipped = append(ready.skipped, left)
			continue
		}
		if resolveErr != nil {
			return writable{}, resolveErr
		}
		ready.targets = append(ready.targets, pageID)
		ready.resolved[pageID] = resolved
	}
	if len(ready.targets) == 0 {
		return writable{}, nothingWritable(len(targets), ready.skipped)
	}
	return ready, nil
}

func nothingWritable(chosen int, skipped []run.Skipped) error {
	paths := make([]string, 0, len(skipped))
	byReason := make(map[run.SkipReason][]string, 2)
	for i := range skipped {
		paths = append(paths, skipped[i].Path)
		byReason[skipped[i].Reason] = append(byReason[skipped[i].Reason], skipped[i].Path)
	}
	reasons := make([]string, 0, 2)
	if untemplated := byReason[run.SkipNoTemplate]; len(untemplated) > 0 {
		one := len(untemplated) == 1
		reasons = append(reasons, named(untemplated)+" "+either(one, "has", "have")+" no template, as "+
			either(one, "its type has", "their types have")+" none and the site has no default; give "+
			either(one, "the type", "the types")+" a template on the Page types screen, or pick a template for this run")
	}
	if unmapped := byReason[run.SkipNoEntity]; len(unmapped) > 0 {
		one := len(unmapped) == 1
		reasons = append(reasons, named(unmapped)+" "+either(one, "is", "are")+
			" mapped to nothing in the graph, and a run plans a page's links from its entity; map "+
			either(one, "it", "them")+" on the Graph screen")
	}
	lead := "none of the " + strconv.Itoa(chosen) + " chosen pages can be written: "
	if chosen == 1 {
		lead = "the chosen page cannot be written: "
	}
	return invalid(lead+strings.Join(reasons, "; "), "pageIds").WithDetail("paths", paths).WithDetail("skipped", skipped)
}

func either(one bool, singular, plural string) string {
	if one {
		return singular
	}
	return plural
}

func leftOut(skipped []run.Skipped) []string {
	out := make([]string, 0, len(skipped))
	for i := range skipped {
		out = append(out, skipped[i].PageID)
	}
	return out
}

func withoutPages(pageIDs, left []string) []string {
	out := make([]string, 0, len(pageIDs))
	for _, pageID := range pageIDs {
		if !slices.Contains(left, strings.TrimSpace(pageID)) {
			out = append(out, pageID)
		}
	}
	return out
}

func withoutDirectives(directives []run.Directive, left []string) []run.Directive {
	out := make([]run.Directive, 0, len(directives))
	for i := range directives {
		if !slices.Contains(left, strings.TrimSpace(directives[i].PageID)) {
			out = append(out, directives[i])
		}
	}
	return out
}

func unwritableParent(parent AddedPage, err error) error {
	if !templates.IsUnassigned(err) {
		return err
	}
	return invalid("the parent "+parent.Path+" of "+parent.NeededBy+" is not on the site, so the run must write it first, "+
		"and it has no template: its type has none and the site has no default; give the type a template on the Page types "+
		"screen, or set the site's default template", "pageIds").
		WithDetail("parentPath", parent.Path).WithDetail("path", parent.NeededBy)
}
