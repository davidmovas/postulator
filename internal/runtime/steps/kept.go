package steps

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ReasonClientChanged = "the page changed on the site after the writer read it, and its owner's text is kept, " +
		"so nothing was written; sync the site, then regenerate the page so the writer reads it again"
	ReasonKeptNeedsPlugin = "the page holds its owner's own text, which is kept as it stands, and only the companion " +
		"plugin's raw route can write beside it without changing it; install the plugin or let the run rewrite the page"
	ReasonSlugTaken = "the site already holds a page under this page's slug that this run did not create; " +
		"give the planned page another slug, or map that page to the entity in its place, then run it again"
	ReasonSaveAltersClient = "a save of this page by the site's user would change its owner's content, because the user " +
		"lacks unfiltered_html, so nothing that saves the post again was written; give that user unfiltered_html, " +
		"or edit the page by hand"
	ReasonPluginTooOld = "the companion plugin on the site is too old to write under the hash it read, so nothing was " +
		"written; upload the new zip over the old one and run the page again"
	ReasonRewriteRefused = "the rewrite would have changed a block it must keep, so nothing was written"
)

func fieldsMoved(page string, keys []string) string {
	return page + ": " + strings.Join(keys, ", ") + " changed on the site after it was read, so nothing more was " +
		"written; sync the site and run the page again"
}

func unread() *content.KeptBody {
	return &content.KeptBody{Raw: "", Hash: block.BodyHash(""), Written: []string{}}
}

func keptBody(sc *run.StepContext, seen sitemodel.Picture) *content.KeptBody {
	if sc.Page.WPID == nil || seen.Scope.Rewrites || sc.Rewrites(sc.Page.ID) {
		return nil
	}
	read := ""
	if row, held := seen.Facts.Field(bodyField(sc)); held {
		read = row.Current
	}
	return &content.KeptBody{Raw: read, Hash: block.BodyHash(read), Written: slices.Clone(seen.Facts.Written)}
}

func composedBody(kept *content.KeptBody, rendered string) (string, error) {
	if kept == nil {
		return rendered, nil
	}
	ours, err := withoutHeadingOne(rendered)
	if err != nil {
		return "", err
	}
	raw, _ := block.Keep(kept.Raw, kept.Written, ours)
	return raw, nil
}

func withoutHeadingOne(rendered string) (string, error) {
	doc, err := content.Parse(rendered)
	if err != nil {
		return "", err
	}
	return doc.RenderWithoutHeadingOne()
}

func wholeBody(ours *content.Document, kept *content.KeptBody) (*content.Document, error) {
	if kept == nil {
		return ours, nil
	}
	rendered, err := ours.Render()
	if err != nil {
		return nil, err
	}
	raw, err := composedBody(kept, rendered)
	if err != nil {
		return nil, err
	}
	return content.Parse(raw)
}

func clientDocument(sc *run.StepContext) (*content.Document, error) {
	draft, drafted, err := decodeArtifact[content.ContentDraft](sc, run.ArtifactDraft)
	if err != nil || !drafted || draft.Kept == nil {
		return nil, err
	}
	return content.Parse(block.Client(draft.Kept.Raw, draft.Kept.Written))
}

func unlinkedByClient(sc *run.StepContext, lc content.LinkContext) (content.LinkContext, error) {
	theirs, err := clientDocument(sc)
	if err != nil {
		return content.LinkContext{}, err
	}
	return lc.Unlinked(theirs), nil
}

type liveBody struct {
	raw    string
	hash   string
	plugin bool
	stable bool
}

func (b liveBody) hashed() string {
	if b.hash != "" {
		return b.hash
	}
	return wp.ContentHash(b.raw)
}

func readLiveBody(ctx context.Context, client *wp.Client, itemType wp.ItemType, existing wp.Item) (liveBody, error) {
	raw, err := client.GetRaw(ctx, itemType, existing.ID)
	switch {
	case err == nil:
		return liveBody{raw: raw.Content, hash: raw.ContentHash, plugin: true, stable: raw.SaveStable}, nil
	case wp.IsPluginMissing(err), errors.IsCode(err, errors.NotFound):
		return liveBody{raw: existing.Content, stable: true}, nil
	default:
		return liveBody{}, err
	}
}

type guarded struct {
	final    string
	previous string
	client   bool
}

func composed(raw string, written []string, ours, rendered string) (final, part string, client bool) {
	final, client = block.Keep(raw, written, ours)
	if !client {
		return rendered, rendered, false
	}
	return final, block.Part(raw, written, ours), true
}

func (d Deps) guard(ctx context.Context, pageID string, kept *content.KeptBody, site liveBody, rendered string) (guarded, bool, error) {
	written, err := d.writtenOf(ctx, pageID, kept)
	if err != nil {
		return guarded{}, false, err
	}
	ours, err := withoutHeadingOne(rendered)
	if err != nil {
		return guarded{}, false, err
	}
	read := block.Client(kept.Raw, kept.Written)
	theirs := block.Client(site.raw, written)
	if block.Hash(read) != block.Hash(theirs) {
		earlier, part, _ := composed(kept.Raw, kept.Written, ours, rendered)
		if !block.SameRender(site.raw, earlier) {
			return guarded{}, false, nil
		}
		written = append(slices.Clone(written), block.Ours(site.raw, read, part)...)
		theirs = block.Client(site.raw, written)
		if block.Hash(read) != block.Hash(theirs) {
			return guarded{}, false, nil
		}
	}
	final, _, client := composed(site.raw, written, ours, rendered)
	previous := site.raw
	if site.hashed() != kept.Hash {
		previous = kept.Raw
	}
	return guarded{final: final, previous: previous, client: client}, true, nil
}

func (d Deps) writtenOf(ctx context.Context, pageID string, kept *content.KeptBody) ([]string, error) {
	if d.Bodies == nil {
		return kept.Written, nil
	}
	held, err := d.Bodies.ListWritten(ctx, pageID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(held))
	for i := range held {
		out = append(out, held[i].BlockHash)
	}
	return out, nil
}

func recordOurBlocks(ctx context.Context, deps Deps, sc *run.StepContext, landed []block.Change) error {
	if deps.Bodies == nil {
		return nil
	}
	return deps.Bodies.ReplaceWritten(ctx, sc.Page.ID, block.OursOf(landed), sc.Run.ID, deps.now())
}
