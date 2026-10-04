package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	CapabilitySEOMeta  = "seo_meta"
	CodeSEOMetaSkipped = "seo_meta_skipped"

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

func applySEO(ctx context.Context, client *wp.Client, sc *run.StepContext, itemType wp.ItemType, wpID int64, updating bool) (seoWrite, error) {
	meta, found, err := decodeArtifact[Meta](sc, run.ArtifactMeta)
	if err != nil {
		return seoWrite{}, err
	}
	if !found {
		if artifact, held := sc.Artifacts[run.ArtifactMeta]; held && artifact.Purged {
			return metaPurged(sc.Page), nil
		}
		return noMetaGenerated(), nil
	}

	capabilities, err := client.Capabilities(ctx)
	if err != nil {
		if wp.IsPluginMissing(err) {
			return metaNotWritten(sc.Page, ReasonNoPlugin), nil
		}
		return seoWrite{}, err
	}
	if !capabilities.Has(CapabilitySEOMeta) {
		return metaNotWritten(sc.Page, ReasonNoSEOWriter), nil
	}

	previous, err := metaBeingReplaced(ctx, client, capabilities, itemType, wpID, updating)
	if err != nil {
		return seoWrite{}, err
	}

	result, err := client.SetSEOMeta(ctx, itemType, wpID, wp.SEOMeta{
		Title:         meta.Title,
		Description:   meta.Description,
		Canonical:     meta.Canonical,
		OGTitle:       meta.OGTitle,
		OGDescription: meta.OGDescription,
	})
	if err != nil {
		if wp.IsPluginMissing(err) {
			return metaNotWritten(sc.Page, ReasonNoPlugin), nil
		}
		return seoWrite{}, err
	}
	return seoWrite{
		previous: previous,
		applied:  append(make([]string, 0, len(result.Applied)), result.Applied...),
		skipped:  []string{},
		findings: nil,
	}, nil
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

func noMetaGenerated() seoWrite {
	return seoWrite{applied: []string{}, skipped: []string{CodeSEOMetaSkipped}}
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
