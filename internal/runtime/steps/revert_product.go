package steps

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ReasonRevertNoStore      = "the site answers no WooCommerce store, so the product cannot be put back"
	ReasonRevertStoreRefused = "the WordPress user may not edit products, so the product cannot be put back"
	ReasonRevertShortEdited  = "the short description was edited in the store since the run wrote it, " +
		"so putting the old one back would lose that edit"
)

func putTheProductBack(ctx context.Context, deps Deps, sc *run.StepContext, result *RevertResult,
	work revertWork) (string, bool) {
	held, err := work.client.GetProduct(ctx, work.published.WPID)
	if err != nil {
		return storeReason(err), false
	}
	update, reason := productRestore(*work.published.PreviousProduct, held)
	if reason != "" {
		return reason, false
	}
	if kept, changed := categoriesTakenBack(held.Categories, work.published.Categories); changed {
		update.Categories = &kept
	}
	raw, reason, ok := bodyToRestore(ctx, work)
	if !ok {
		return reason, false
	}

	if hasFields(update) {
		if _, err = work.client.UpdateProduct(ctx, held.ID, update); err != nil {
			return storeReason(err), false
		}
		if raw, reason, ok = readRawBack(ctx, work); !ok {
			return reason, false
		}
	}
	hash, reason, ok := writeBodyBack(ctx, work, raw)
	if !ok {
		return reason, false
	}
	return bodyRestored(ctx, deps, sc, result, work, hash,
		"the description, the short description and the attributes the run replaced were written back")
}

func storeReason(err error) string {
	switch {
	case wp.StoreAbsent(err):
		return ReasonRevertNoStore
	case wp.Forbidden(err):
		return ReasonRevertStoreRefused
	case errors.IsCode(err, errors.NotFound):
		return ReasonRevertGone
	default:
		return err.Error()
	}
}

func productRestore(snapshot ProductSnapshot, held wp.Product) (update wp.UpdateProduct, reason string) {
	if snapshot.ShortWritten {
		if strings.TrimSpace(held.ShortDescription) != strings.TrimSpace(snapshot.WrittenShort) {
			return wp.UpdateProduct{}, ReasonRevertShortEdited
		}
		short := snapshot.ShortDescription
		update.ShortDescription = &short
	}
	if snapshot.AttributesSent {
		attributes, changed, edited := attributesRestored(snapshot, held.Attributes)
		if edited != "" {
			return wp.UpdateProduct{}, edited
		}
		if changed {
			update.Attributes = &attributes
		}
	}
	if snapshot.ImageID != 0 {
		ids := imageIDs(held.Images)
		kept := slices.DeleteFunc(slices.Clone(ids), func(id int64) bool { return id == snapshot.ImageID })
		if len(kept) != len(ids) {
			update.Images = &kept
		}
	}
	return update, ""
}

func attributesRestored(snapshot ProductSnapshot, held []wp.ProductAttribute) (restored []wp.ProductAttribute, changed bool, reason string) {
	before := snapshot.StoreAttributes()
	restored = make([]wp.ProductAttribute, 0, len(held))
	for _, attribute := range held {
		written := slices.IndexFunc(snapshot.Written, func(w SnapshotAttribute) bool {
			return strings.EqualFold(strings.TrimSpace(w.Name), strings.TrimSpace(attribute.Name))
		})
		if written < 0 {
			attribute.Options = slices.Clone(attribute.Options)
			restored = append(restored, attribute)
			continue
		}
		if !slices.Equal(attribute.Options, snapshot.Written[written].Options) {
			return nil, false, "the attribute " + attribute.Name + " was edited in the store since the run wrote it, " +
				"so taking it back would lose that edit"
		}
		changed = true
		if at := attributeNamed(before, attribute.Name); at >= 0 {
			restored = append(restored, before[at])
		}
	}
	return restored, changed, ""
}
