package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type siteBody struct {
	raw wp.RawContent
	doc *content.Document
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

	doc, err := content.Parse(raw.Content)
	if err != nil {
		return siteBody{}, unreadable, nil
	}
	return siteBody{raw: raw, doc: doc}, "", nil
}

func (b siteBody) put(ctx context.Context, client *wp.Client, page pagemap.Page, rendered string) (string, error) {
	return client.PutRaw(ctx, onSiteType(page), *page.WPID, rendered, b.raw.ContentHash)
}

func backfill(doc *content.Document, lc content.LinkContext, policy template.LinkPolicy,
	target content.LinkTarget) (placement content.InsertResult, sentence string) {
	placement = content.InsertTarget(doc, lc, policy, target)
	if _, inserted := insertedAnchor(placement); inserted || !writable(placement) || len(target.Anchors) == 0 {
		return placement, ""
	}
	sentence = templated(owedPhrase{text: target.Anchors[0]})
	if err := settleSentence(doc, placeFor(doc, policy, target), sentence); err != nil {
		return placement, ""
	}
	return content.InsertTarget(doc, lc, policy, target), sentence
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
