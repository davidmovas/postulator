package steps

import (
	"context"
	"slices"

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
	ParamRevertFields = "fields"

	OutcomeKept = "kept"

	CodeRevertFieldKept = "revert_field_kept"

	ReasonRevertCreatedEdited = "the page the run created was edited on the site since, so it stays; " +
		"take it down by hand if it should go"
	ReasonRevertFieldMoved = "it changed on the site since the run wrote it, so the edit is kept"
	ReasonRevertBlocksKept = "the blocks the run wrote were edited on the site since, so they stay"
	ReasonRevertNothing    = "every field the run wrote was changed on the site since, so nothing was put back"
)

type RevertedField struct {
	Key     field.Key `json:"key"`
	Outcome string    `json:"outcome"`
	Detail  string    `json:"detail,omitempty"`
}

func revertFields(sc *run.StepContext) []field.Key {
	listed, ok := sc.Params[ParamRevertFields].([]any)
	if !ok {
		return nil
	}
	out := make([]field.Key, 0, len(listed))
	for _, held := range listed {
		if key, named := held.(string); named && key != "" {
			out = append(out, field.Key(key))
		}
	}
	return out
}

func wantedField(wanted []field.Key, key field.Key) bool {
	return len(wanted) == 0 || slices.Contains(wanted, key)
}

func bodyKeyOf(page pagemap.Page) field.Key {
	if page.WPType == pagemap.WPProduct {
		return field.ProductDescription
	}
	return field.Body
}

func revertByField(ctx context.Context, deps Deps, sc *run.StepContext, result *RevertResult, work revertWork) (string, bool) {
	if work.published.Created {
		return revertCreated(ctx, deps, result, work)
	}
	wanted := revertFields(sc)
	result.Fields = make([]RevertedField, 0, len(work.published.Fields)+1)
	target := edit.Target{Type: string(onSiteType(work.page)), ID: work.published.WPID, Product: work.page.WPType == pagemap.WPProduct}
	editor := editorFor(work.client)
	body := bodyKeyOf(work.page)
	for _, change := range work.published.Fields {
		if change.Key == body || !change.Moves() || !wantedField(wanted, change.Key) {
			continue
		}
		reverted := revertField(ctx, editor, target, change, work.published)
		if reverted.Outcome == OutcomeRestored {
			if err := recordField(ctx, deps, sc, work.page.ID, change.Key, change.Before); err != nil {
				return err.Error(), false
			}
		}
		result.Fields = append(result.Fields, reverted)
	}
	if work.page.WPType == pagemap.WPProduct && wantedField(wanted, field.FeaturedImage) && work.published.PreviousProduct != nil &&
		work.published.PreviousProduct.ImageID != 0 {
		result.Fields = append(result.Fields, revertImage(ctx, work))
	}
	switch {
	case !wantedField(wanted, body) || work.published.PreviousContentHash == "":
	case work.published.ContentHash == "":
		result.Fields = append(result.Fields, RevertedField{Key: body, Outcome: OutcomeKept, Detail: ReasonRevertBodyUnproven})
	default:
		result.Fields = append(result.Fields, revertBody(ctx, deps, sc, work, body))
	}
	return settleReverted(result)
}

func settleReverted(result *RevertResult) (string, bool) {
	restored := 0
	for _, reverted := range result.Fields {
		switch reverted.Outcome {
		case OutcomeRestored:
			restored++
		case OutcomeKept:
			result.Findings = append(result.Findings, content.Finding{
				Severity: content.SeverityWarn, Code: CodeRevertFieldKept,
				Message: "the " + string(reverted.Key) + " of " + result.Path + " was not put back: " + reverted.Detail,
				Details: map[string]any{"pageId": result.PageID, "path": result.Path, "field": string(reverted.Key), "reason": reverted.Detail},
			})
		}
	}
	if restored == 0 && len(result.Fields) > 0 {
		return ReasonRevertNothing, false
	}
	result.Outcome = OutcomeRestored
	result.Detail = "each field the run wrote that the site still holds as it wrote it was put back"
	return "", true
}

func revertField(ctx context.Context, editor edit.Service, target edit.Target, change field.Change, published PublishResult) RevertedField {
	before := change.Before
	write := edit.Write{Key: change.Key, Value: &before, Expected: expectedOf(change.After), Provider: providerOf(change.Key, published)}
	if change.Key.Kind() == field.Meta && before == "" && write.Provider != field.ProviderStore {
		write.Value = nil
	}
	write.Social = socialBack(change.Key, published)
	results, err := editor.Write(ctx, target, []edit.Write{write})
	if err != nil {
		return RevertedField{Key: change.Key, Outcome: OutcomeKept, Detail: err.Error()}
	}
	for _, held := range results {
		if held.Key == change.Key && held.Outcome == edit.OutcomeWritten {
			return RevertedField{Key: change.Key, Outcome: OutcomeRestored}
		}
	}
	return RevertedField{Key: change.Key, Outcome: OutcomeKept, Detail: ReasonRevertFieldMoved}
}

func providerOf(key field.Key, published PublishResult) field.Provider {
	for _, written := range published.PreviousFields {
		if written.Key == key {
			return written.Provider
		}
	}
	return field.ProviderNone
}

func socialBack(key field.Key, published PublishResult) *edit.Social {
	if published.WrittenMeta == nil {
		return nil
	}
	previous := wp.SEOMeta{}
	if published.PreviousMeta != nil {
		previous = *published.PreviousMeta
	}
	switch {
	case key == field.SEOTitle && slices.Contains(published.SEOApplied, "ogTitle"):
		return &edit.Social{Value: previous.OGTitle, Expected: block.BodyHash(published.WrittenMeta.OGTitle)}
	case key == field.SEODescription && slices.Contains(published.SEOApplied, "ogDescription"):
		return &edit.Social{Value: previous.OGDescription, Expected: block.BodyHash(published.WrittenMeta.OGDescription)}
	default:
		return nil
	}
}

func revertImage(ctx context.Context, work revertWork) RevertedField {
	ours := work.published.PreviousProduct.ImageID
	held, err := work.client.GetProduct(ctx, work.published.WPID)
	if err != nil {
		return RevertedField{Key: field.FeaturedImage, Outcome: OutcomeKept, Detail: storeReason(err)}
	}
	ids := imageIDs(held.Images)
	if len(ids) == 0 || ids[0] != ours {
		return RevertedField{Key: field.FeaturedImage, Outcome: OutcomeKept, Detail: ReasonRevertFieldMoved}
	}
	kept := ids[1:]
	if _, err = work.client.UpdateProduct(ctx, held.ID, wp.UpdateProduct{Images: &kept}); err != nil {
		return RevertedField{Key: field.FeaturedImage, Outcome: OutcomeKept, Detail: storeReason(err)}
	}
	return RevertedField{Key: field.FeaturedImage, Outcome: OutcomeRestored}
}

func revertBody(ctx context.Context, deps Deps, sc *run.StepContext, work revertWork, key field.Key) RevertedField {
	raw, reason, ok := readRawBack(ctx, work)
	if !ok {
		return RevertedField{Key: key, Outcome: OutcomeKept, Detail: reason}
	}
	detail := ""
	body := work.published.PreviousContent
	switch raw.ContentHash {
	case work.published.PreviousContentHash:
		return RevertedField{Key: key, Outcome: OutcomeRestored}
	case work.published.ContentHash:
	default:
		undone := block.Undo(raw.Content, work.published.Blocks)
		if undone.Undone == 0 {
			return RevertedField{Key: key, Outcome: OutcomeKept, Detail: ReasonRevertBlocksKept}
		}
		body = undone.Body
		if undone.Kept > 0 {
			detail = ReasonRevertBlocksKept
		}
	}
	written, err := work.client.PutRaw(ctx, onSiteType(work.page), work.published.WPID, body, raw.ContentHash)
	switch {
	case errors.IsCode(err, errors.Conflict):
		return RevertedField{Key: key, Outcome: OutcomeKept, Detail: writtenUnderUs(raw.ContentHash)}
	case err != nil:
		return RevertedField{Key: key, Outcome: OutcomeKept, Detail: err.Error()}
	}
	if adoptErr := readopt(ctx, deps, sc, work.page, raw.Content, written.Content, uncoveredOf(work.published.Blocks)); adoptErr != nil {
		return RevertedField{Key: key, Outcome: OutcomeKept, Detail: adoptErr.Error()}
	}
	return RevertedField{Key: key, Outcome: OutcomeRestored, Detail: detail}
}

func uncoveredOf(changes []block.Change) []string {
	out := make([]string, 0)
	for i := range changes {
		if changes[i].Owner == block.OwnerOurs && changes[i].Change == block.ChangeRemoved {
			out = append(out, changes[i].Hash)
		}
	}
	return out
}

func revertCreated(ctx context.Context, deps Deps, result *RevertResult, work revertWork) (string, bool) {
	hash, err := liveHash(ctx, work)
	switch {
	case errors.IsCode(err, errors.NotFound):
	case err != nil:
		return err.Error(), false
	case hash != work.published.ContentHash:
		return ReasonRevertCreatedEdited, false
	}
	return takeOffTheSite(ctx, deps, result, work)
}

func liveHash(ctx context.Context, work revertWork) (string, error) {
	raw, err := work.client.GetRaw(ctx, onSiteType(work.page), work.published.WPID)
	if err == nil {
		return raw.ContentHash, nil
	}
	if !wp.IsPluginMissing(err) {
		return "", err
	}
	itemType, err := itemTypeOf(work.page)
	if err != nil {
		return "", err
	}
	item, err := work.client.GetItem(ctx, itemType, work.published.WPID)
	if err != nil {
		return "", err
	}
	return wp.ContentHash(item.Content), nil
}
