package steps

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	CodeRevertTermsKept      = "revert_terms_kept"
	CodeRevertCategoriesKept = "revert_categories_kept"
)

func putTheCategoriesBack(ctx context.Context, work revertWork) (string, bool) {
	written := work.published.Categories
	if written == nil || len(written.Added) == 0 {
		return "", true
	}

	itemType := onSiteType(work.page)
	held, err := work.client.GetItem(ctx, itemType, work.published.WPID)
	if err != nil {
		return itemReason(err), false
	}
	kept, changed := categoriesTakenBack(held.Categories, written)
	if !changed {
		return "", true
	}
	if _, err = work.client.UpdateItem(ctx, itemType, work.published.WPID, wp.UpdateItem{Categories: kept}); err != nil {
		return itemReason(err), false
	}
	return "", true
}

func categoriesTakenBack(carried []int64, written *CategoryWrite) ([]int64, bool) {
	if written == nil || len(written.Added) == 0 {
		return nil, false
	}
	kept := make([]int64, 0, len(carried))
	for _, id := range carried {
		if !slices.Contains(written.Added, id) {
			kept = append(kept, id)
		}
	}
	return kept, len(kept) != len(carried)
}

func itemReason(err error) string {
	if errors.IsCode(err, errors.NotFound) {
		return ReasonRevertGone
	}
	return err.Error()
}

func categoriesKept(page pagemap.Page, written *CategoryWrite, reason string) content.Finding {
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeRevertCategoriesKept,
		Message:  "the categories the run added to " + page.Path + " stay on it: " + reason,
		Details: map[string]any{
			"pageId": page.ID, "path": page.Path, "taxonomy": string(written.Taxonomy),
			"termIds": slices.Clone(written.Added), "reason": reason,
		},
	}
}

func termsKept(page pagemap.Page, written *CategoryWrite) []content.Finding {
	if written == nil {
		return nil
	}
	names := make([]string, 0, len(written.Terms))
	ids := make([]int64, 0, len(written.Terms))
	for i := range written.Terms {
		if written.Terms[i].Created {
			names = append(names, written.Terms[i].Name)
			ids = append(ids, written.Terms[i].TermID)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return []content.Finding{{
		Severity: content.SeverityInfo,
		Code:     CodeRevertTermsKept,
		Message: "the categories the run created for " + page.Path + " stay on the site, since other content may be " +
			"filed under them: " + strings.Join(names, ", ") + "; delete them in WordPress if nothing needs them",
		Details: map[string]any{
			"pageId": page.ID, "path": page.Path, "taxonomy": string(written.Taxonomy),
			"categories": names, "termIds": ids,
		},
	}}
}
