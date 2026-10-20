package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/edit"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	CapabilitySEOMeta  = "seo_meta"
	CodeSEOMetaSkipped = "seo_meta_skipped"
	CodeSEOMetaKept    = "seo_meta_kept"

	ReasonNoSEOWriter = "the companion plugin on this site cannot write SEO meta"
)

type seoWrite struct {
	previous *wp.SEOMeta
	applied  []string
	skipped  []string
	findings []content.Finding
}

func (r *PublishResult) took(seo seoWrite) {
	r.SEOApplied = seo.applied
	r.Skipped = seo.skipped
	r.PreviousMeta = seo.previous
	r.Findings = append(r.Findings, seo.findings...)
}

func unsent(skipped seoWrite) seoPlan {
	return seoPlan{Skipped: skipped.skipped, Findings: skipped.findings}
}

func planSEO(ctx context.Context, client *wp.Client, sc *run.StepContext, itemType wp.ItemType, wpID int64, updating bool,
	kept keptFields, canonical bool) (seoPlan, error) {
	meta, found, err := decodeArtifact[Meta](sc, run.ArtifactMeta)
	if err != nil {
		return seoPlan{}, err
	}
	if !found {
		if artifact, held := sc.Artifacts[run.ArtifactMeta]; held && artifact.Purged {
			return unsent(metaPurged(sc.Page)), nil
		}
		return unsent(noMetaGenerated()), nil
	}
	if kept[field.SEOTitle] {
		meta.Title, meta.OGTitle = "", ""
	}
	if kept[field.SEODescription] {
		meta.Description, meta.OGDescription = "", ""
	}
	if kept.all(field.SEOTitle, field.SEODescription) && !canonical {
		return unsent(snippetKept()), nil
	}

	capabilities, err := client.Capabilities(ctx)
	if err != nil {
		if wp.IsPluginMissing(err) {
			return unsent(metaNotWritten(sc.Page, ReasonNoPlugin)), nil
		}
		return seoPlan{}, err
	}
	if !capabilities.Has(CapabilitySEOMeta) {
		return unsent(metaNotWritten(sc.Page, ReasonNoSEOWriter)), nil
	}

	previous, err := metaBeingReplaced(ctx, client, capabilities, itemType, wpID, updating)
	if err != nil {
		return seoPlan{}, err
	}
	if previous != nil {
		if previous.OGTitle != "" && previous.OGTitle != previous.Title {
			meta.OGTitle = ""
		}
		if previous.OGDescription != "" && previous.OGDescription != previous.Description {
			meta.OGDescription = ""
		}
	}
	return seoPlan{
		Previous: previous, Send: true, Canonical: canonical,
		Meta: wp.SEOMeta{
			Title: meta.Title, Description: meta.Description, OGTitle: meta.OGTitle, OGDescription: meta.OGDescription,
		},
	}, nil
}

func (p seoPlan) before() wp.SEOMeta {
	if p.Previous == nil {
		return wp.SEOMeta{}
	}
	return *p.Previous
}

func (p seoPlan) writes() []edit.Write {
	before := p.before()
	out := make([]edit.Write, 0, 3)
	for _, sent := range []struct {
		key                     field.Key
		value, held, og, heldOG string
	}{
		{key: field.SEOTitle, value: p.Meta.Title, held: before.Title, og: p.Meta.OGTitle, heldOG: before.OGTitle},
		{key: field.SEODescription, value: p.Meta.Description, held: before.Description, og: p.Meta.OGDescription, heldOG: before.OGDescription},
		{key: field.SEOCanonical, value: p.Meta.Canonical, held: before.Canonical},
	} {
		if sent.value == "" {
			continue
		}
		write := edit.Write{Key: sent.key, Value: new(sent.value), Expected: new(block.BodyHash(sent.held))}
		if sent.og != "" {
			write.Social = &edit.Social{Value: sent.og, Expected: block.BodyHash(sent.heldOG)}
		}
		out = append(out, write)
	}
	return out
}

func (p seoPlan) movedSince(held wp.SEOMeta) []string {
	before := p.before()
	moved := make([]string, 0, 2)
	if p.Meta.Title != "" && (held.Title != before.Title || (p.Meta.OGTitle != "" && held.OGTitle != before.OGTitle)) {
		moved = append(moved, string(field.SEOTitle))
	}
	if p.Meta.Description != "" && (held.Description != before.Description ||
		(p.Meta.OGDescription != "" && held.OGDescription != before.OGDescription)) {
		moved = append(moved, string(field.SEODescription))
	}
	return moved
}

func appliedOf(plan seoPlan, results []edit.Result) []string {
	written := make(map[field.Key]bool, len(results))
	for i := range results {
		written[results[i].Key] = results[i].Outcome == edit.OutcomeWritten
	}
	applied := make([]string, 0, 5)
	for _, name := range []struct {
		field string
		key   field.Key
		sent  bool
	}{
		{field: "title", key: field.SEOTitle, sent: true},
		{field: "description", key: field.SEODescription, sent: true},
		{field: "canonical", key: field.SEOCanonical, sent: true},
		{field: "ogTitle", key: field.SEOTitle, sent: plan.Meta.OGTitle != ""},
		{field: "ogDescription", key: field.SEODescription, sent: plan.Meta.OGDescription != ""},
	} {
		if name.sent && written[name.key] {
			applied = append(applied, name.field)
		}
	}
	return applied
}

func (d Deps) canonicalWritten(ctx context.Context, sc *run.StepContext, creating bool) (bool, error) {
	if !creating || sc.Run.PublishMode != run.PublishLive {
		return false, nil
	}
	set, err := d.Types.Set(ctx, sc.Run.SiteID)
	if err != nil {
		return false, err
	}
	scope, _ := set.Scope(claimedType(set, sc).ID, field.SEOCanonical)
	return scope.Enabled, nil
}

func sendSEO(ctx context.Context, client *wp.Client, sc *run.StepContext, itemType wp.ItemType, wpID int64,
	plan seoPlan) (seo seoWrite, moved []string, err error) {
	results, err := editorFor(client).Write(ctx, edit.Target{Type: string(itemType), ID: wpID}, plan.writes())
	switch {
	case wp.IsPluginMissing(err), refusedFor(err, edit.ReasonNoPlugin):
		return metaNotWritten(sc.Page, ReasonNoPlugin), nil, nil
	case wp.IsPluginOutdated(err):
		return metaNotWritten(sc.Page, ReasonPluginTooOld), nil, nil
	case err != nil:
		return seoWrite{}, nil, err
	}
	if moved = movedKeys(results); len(moved) > 0 {
		return seoWrite{}, moved, nil
	}
	return seoWrite{previous: plan.Previous, applied: appliedOf(plan, results), skipped: []string{}}, nil, nil
}

func metaBeingReplaced(ctx context.Context, client *wp.Client, capabilities wp.Capabilities,
	itemType wp.ItemType, wpID int64, updating bool) (*wp.SEOMeta, error) {
	if !updating || !capabilities.Has(wp.CapabilitySEOMetaRead) {
		return nil, nil
	}

	held, err := client.GetSEOMeta(ctx, itemType, wpID)
	switch {
	case err == nil:
		return &held, nil
	case wp.IsPluginMissing(err), wp.IsPluginOutdated(err), errors.IsCode(err, errors.NotFound):
		return nil, nil
	default:
		return nil, err
	}
}

func liveSnippet(ctx context.Context, client *wp.Client, itemType wp.ItemType, wpID int64, updating bool,
	into map[field.Key]string) error {
	if !updating {
		return nil
	}
	capabilities, err := client.Capabilities(ctx)
	switch {
	case wp.IsPluginMissing(err):
		return nil
	case err != nil:
		return err
	}
	held, err := metaBeingReplaced(ctx, client, capabilities, itemType, wpID, updating)
	if err != nil || held == nil {
		return err
	}
	into[field.SEOTitle], into[field.SEODescription] = held.Title, held.Description
	return nil
}

func noMetaGenerated() seoWrite {
	return seoWrite{applied: []string{}, skipped: []string{CodeSEOMetaSkipped}}
}

func snippetKept() seoWrite {
	return seoWrite{applied: []string{}, skipped: []string{CodeSEOMetaKept}}
}

func metaPurged(page pagemap.Page) seoWrite {
	return seoWrite{
		applied: []string{},
		skipped: []string{CodeSEOMetaSkipped},
		findings: []content.Finding{{
			Severity: content.SeverityWarn,
			Code:     CodeArtifactPurged,
			Message: "the meta of " + page.Path + " was dropped by retention before it could be written, " +
				"so the search snippet on the site is whatever was there before",
			Details: map[string]any{
				"pageId": page.ID, "path": page.Path, "kind": string(run.ArtifactMeta),
			},
		}},
	}
}

func metaNotWritten(page pagemap.Page, reason string) seoWrite {
	return seoWrite{
		applied: []string{},
		skipped: []string{CodeSEOMetaSkipped},
		findings: []content.Finding{{
			Severity: content.SeverityWarn,
			Code:     CodeSEOMetaSkipped,
			Message:  "the SEO meta of " + page.Path + " was generated but not written: " + reason,
			Details:  map[string]any{"pageId": page.ID, "path": page.Path, "reason": reason},
		}},
	}
}
