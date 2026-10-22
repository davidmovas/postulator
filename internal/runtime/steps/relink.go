package steps

import (
	"context"
	stdhtml "html"
	"slices"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type siteBody struct {
	raw wp.RawContent
}

func readSiteBody(ctx context.Context, client *wp.Client, page pagemap.Page,
	gone, unreadable string) (body siteBody, standDown string, err error) {
	raw, err := client.GetRaw(ctx, onSiteType(page), *page.WPID)
	if err != nil {
		switch {
		case wp.IsPluginMissing(err):
			return siteBody{}, ReasonNoPlugin, nil
		case errors.IsCode(err, errors.NotFound):
			return siteBody{}, gone, nil
		default:
			return siteBody{}, "", err
		}
	}

	if _, err = content.Parse(raw.Content); err != nil {
		return siteBody{}, unreadable, nil
	}
	return siteBody{raw: raw}, "", nil
}

type splicedBody struct {
	raw    string
	ours   []string
	placed []block.Placed
}

const unwritableDetail = "the anchor sits where the link cannot be written into its block's own bytes"

type linkWriter struct {
	body   block.Body
	ours   map[string]bool
	placed []block.Placed
}

func (b siteBody) writer(ctx context.Context, deps Deps, page pagemap.Page) (*linkWriter, error) {
	written, err := deps.ourBlocks(ctx, page.ID)
	if err != nil {
		return nil, err
	}
	ours := make(map[string]bool, len(written))
	for _, hash := range written {
		ours[hash] = true
	}
	return &linkWriter{body: block.Parse(b.raw.Content), ours: ours}, nil
}

func (w *linkWriter) document() (*content.Document, error) {
	return content.Parse(w.body.Serialize())
}

func (w *linkWriter) insert(lc content.LinkContext, policy field.LinkPolicy, target content.LinkTarget) (content.InsertResult, error) {
	doc, err := w.document()
	if err != nil {
		return content.InsertResult{}, err
	}
	placement := content.InsertTarget(doc, lc, policy, target)
	anchor, inserted := insertedAnchor(placement)
	if !inserted {
		return placement, nil
	}
	oracle, err := doc.Render()
	if err != nil {
		return content.InsertResult{}, err
	}
	if w.write(anchor, lc.Href(target), target.URL, oracle) {
		return placement, nil
	}
	return content.InsertResult{
		Missing:   []content.LinkTarget{target},
		Decisions: []content.Decision{{Target: target, Outcome: content.OutcomeAnchorNotFound, Detail: unwritableDetail}},
	}, nil
}

func (w *linkWriter) write(anchor, href, to, oracle string) bool {
	at, single := block.ChangedBlock(w.body.Serialize(), oracle)
	if !single {
		return false
	}
	for nth := 0; ; nth++ {
		placed, err := w.body.PlaceLink(at, block.Link{Anchor: anchor, Href: href, Nth: nth})
		if err != nil {
			return false
		}
		if block.SameRender(placed.Serialize(), oracle) {
			before, after := w.body.Blocks[at].Hash, placed.Blocks[at].Hash
			if w.ours[before] {
				w.ours[after] = true
			}
			for i := range w.placed {
				if w.placed[i].Block == before {
					w.placed[i].Block = after
				}
			}
			w.placed = append(w.placed, block.Placed{To: to, Anchor: anchor, Block: after})
			w.body = placed
			return true
		}
	}
}

func (w *linkWriter) backfill(lc content.LinkContext, policy field.LinkPolicy,
	target content.LinkTarget) (placement content.InsertResult, sentence string, err error) {
	if placement, err = w.insert(lc, policy, target); err != nil {
		return content.InsertResult{}, "", err
	}
	if _, inserted := insertedAnchor(placement); inserted || !writable(placement) || len(target.Anchors) == 0 {
		return placement, "", nil
	}
	sentence = templated(owedPhrase{text: target.Anchors[0]})
	added, err := w.addSentence(policy, target, sentence)
	if err != nil || !added {
		return placement, "", err
	}
	placement, err = w.insert(lc, policy, target)
	return placement, sentence, err
}

func (w *linkWriter) addSentence(policy field.LinkPolicy, target content.LinkTarget, sentence string) (bool, error) {
	doc, err := w.document()
	if err != nil {
		return false, err
	}
	at := w.blockOfParagraph(placeFor(doc, policy, target))
	if target.Relation == content.RelationUp {
		at--
	}
	raw := block.Part(w.body.Serialize(), nil, "<p>"+stdhtml.EscapeString(sentence)+"</p>")
	next, err := w.body.InsertAfter(at, raw)
	if err != nil {
		return false, nil
	}
	w.body = next
	w.ours[block.Hash(raw)] = true
	return true, nil
}

func (w *linkWriter) blockOfParagraph(paragraph int) int {
	seen := 0
	for i := range w.body.Blocks {
		doc, err := content.Parse(w.body.Blocks[i].Raw)
		if err != nil {
			continue
		}
		seen += len(doc.Paragraphs())
		if paragraph < seen {
			return i
		}
	}
	return len(w.body.Blocks) - 1
}

func (w *linkWriter) result() splicedBody {
	ours := make([]string, 0, len(w.ours))
	for i := range w.body.Blocks {
		if hash := w.body.Blocks[i].Hash; w.ours[hash] && !slices.Contains(ours, hash) {
			ours = append(ours, hash)
		}
	}
	return splicedBody{raw: w.body.Serialize(), ours: ours, placed: slices.Clone(w.placed)}
}

func recordSpliced(ctx context.Context, deps Deps, pageID, runID string, body splicedBody) error {
	if err := recordBody(ctx, deps, pageID, body.raw); err != nil {
		return err
	}
	if deps.Bodies == nil {
		return nil
	}
	return deps.Bodies.ReplaceWritten(ctx, pageID, body.ours, runID, deps.now())
}

func writable(placement content.InsertResult) bool {
	if len(placement.Decisions) == 0 {
		return false
	}
	outcome := placement.Decisions[0].Outcome
	return outcome == content.OutcomeAnchorNotFound || outcome == content.OutcomePositionRule
}

func alreadyLinked(placement content.InsertResult) bool {
	return len(placement.Decisions) > 0 && placement.Decisions[0].Outcome == content.OutcomeAlreadyLinked
}

func insertedAnchor(placement content.InsertResult) (anchor string, inserted bool) {
	for i := range placement.Decisions {
		if placement.Decisions[i].Outcome == content.OutcomeInserted {
			return placement.Decisions[i].Anchor, true
		}
	}
	return "", false
}

func firstDetail(placement content.InsertResult) string {
	if len(placement.Decisions) == 0 {
		return ""
	}
	return string(placement.Decisions[0].Outcome) + ": " + placement.Decisions[0].Detail
}
