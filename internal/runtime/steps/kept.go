package steps

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ReasonClientChanged = "the page changed on the site after the writer read it, and its owner's text is kept, " +
		"so nothing was written; sync the site and run the page again"
	ReasonKeptNeedsPlugin = "the page holds its owner's own text, which is kept as it stands, and only the companion " +
		"plugin's raw route can write beside it without changing it; install the plugin or let the run rewrite the page"
	ReasonSlugTaken = "the site already holds a page under this page's slug, with text the writer never read; " +
		"sync the site so the model reads that page, then run it again"
)

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
		return liveBody{raw: raw.Content, hash: raw.ContentHash, plugin: true}, nil
	case wp.IsPluginMissing(err), errors.IsCode(err, errors.NotFound):
		return liveBody{raw: existing.Content}, nil
	default:
		return liveBody{}, err
	}
}

type guarded struct {
	final    string
	theirs   string
	previous string
	client   bool
}

func (d Deps) guard(ctx context.Context, pageID string, kept *content.KeptBody, site liveBody, rendered string) (guarded, bool, error) {
	written, err := d.writtenOf(ctx, pageID, kept)
	if err != nil {
		return guarded{}, false, err
	}
	read := block.Client(kept.Raw, kept.Written)
	theirs := block.Client(site.raw, written)
	if block.Hash(read) != block.Hash(theirs) {
		return guarded{}, false, nil
	}
	ours, err := withoutHeadingOne(rendered)
	if err != nil {
		return guarded{}, false, err
	}
	final, client := block.Keep(site.raw, written, ours)
	if !client {
		final = rendered
	}
	previous := site.raw
	if site.hashed() != kept.Hash {
		previous = kept.Raw
	}
	return guarded{final: final, theirs: theirs, previous: previous, client: client}, true, nil
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

func adoptable(site liveBody, final string) bool {
	return block.Parse(site.raw).Empty() || block.Hash(site.raw) == block.Hash(final)
}

func recordOurBlocks(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, itemType wp.ItemType, id int64,
	sent, theirs string) error {
	if deps.Bodies == nil {
		return nil
	}
	stored := sent
	raw, err := client.GetRaw(ctx, itemType, id)
	switch {
	case err == nil:
		stored = raw.Content
	case !wp.IsPluginMissing(err):
		return err
	}
	return deps.Bodies.ReplaceWritten(ctx, sc.Page.ID, block.Ours(stored, theirs), sc.Run.ID, deps.now())
}
