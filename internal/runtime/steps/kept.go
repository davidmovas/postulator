package steps

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
)

const (
	ReasonClientChanged = "the page changed on the site after the writer read it, and its owner's text is kept, " +
		"so nothing was written; sync the site and run the page again"
	ReasonKeptNeedsPlugin = "the page holds its owner's own text, which is kept as it stands, and only the companion " +
		"plugin's raw route can write beside it without changing it; install the plugin or let the run rewrite the page"
)

func keptBody(sc *run.StepContext, seen sitemodel.Picture) *content.KeptBody {
	row, held := seen.Facts.Field(bodyField(sc))
	if !held || seen.Scope.Rewrites || sc.Rewrites(sc.Page.ID) {
		return nil
	}
	if !block.ClientText(row.Current, seen.Facts.Written) {
		return nil
	}
	return &content.KeptBody{Raw: row.Current, Hash: block.BodyHash(row.Current), Written: slices.Clone(seen.Facts.Written)}
}

func composedBody(kept *content.KeptBody, rendered string) (string, error) {
	if kept == nil {
		return rendered, nil
	}
	doc, err := content.Parse(rendered)
	if err != nil {
		return "", err
	}
	ours, err := doc.RenderWithoutHeadingOne()
	if err != nil {
		return "", err
	}
	raw, _ := block.Keep(kept.Raw, kept.Written, ours)
	return raw, nil
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

func recordOurBlocks(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, itemType wp.ItemType, id int64,
	sent string, kept *content.KeptBody) error {
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
	theirs := ""
	if kept != nil {
		theirs = block.Client(kept.Raw, kept.Written)
	}
	return deps.Bodies.ReplaceWritten(ctx, sc.Page.ID, block.Ours(stored, theirs), sc.Run.ID, deps.now())
}
