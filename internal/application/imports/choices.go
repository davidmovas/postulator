package imports

import (
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type choices struct {
	pages   map[string]string
	used    map[string]struct{}
	refused []string
}

type chosenRow struct {
	pageID string
	key    string
	trail  []string
}

func newChoices(given []Choice) (*choices, error) {
	out := &choices{pages: make(map[string]string, len(given)), used: make(map[string]struct{}, len(given))}
	for _, choice := range given {
		key, pageID := strings.TrimSpace(choice.RowKey), strings.TrimSpace(choice.PageID)
		if key == "" || pageID == "" {
			return nil, errors.New(errors.Invalid, "a choice names its row and the page it maps to").WithDetail("field", "choices")
		}
		if _, twice := out.pages[key]; twice {
			return nil, errors.New(errors.Invalid, "a row is chosen twice").WithDetail("rowKey", key)
		}
		out.pages[key] = pageID
	}
	return out, nil
}

func (c *choices) pick(row *rowDraft) {
	if c == nil || row.key == "" {
		return
	}
	pageID, chosen := c.pages[row.key]
	if !chosen {
		return
	}
	c.used[row.key] = struct{}{}
	switch {
	case row.match.Found() && row.match.Page.ID == pageID:
	case row.match.Ambiguous():
		at := slices.IndexFunc(row.match.Candidates, func(page pagemap.Page) bool { return page.ID == pageID })
		if at < 0 {
			c.refused = append(c.refused, row.key)
			return
		}
		row.picked = row.match.Candidates
		picked := row.match.Candidates[at]
		row.match = pagemap.Match{Page: picked, Rule: pagemap.RuleMapped, Candidates: []pagemap.Page{picked}}
	default:
		c.refused = append(c.refused, row.key)
	}
}

func (c *choices) settle() error {
	if c == nil {
		return nil
	}
	refused := slices.Clone(c.refused)
	for key := range c.pages {
		if _, used := c.used[key]; !used {
			refused = append(refused, key)
		}
	}
	if len(refused) == 0 {
		return nil
	}
	slices.Sort(refused)
	return errors.New(errors.Invalid, "a choice names a page its row does not answer to, or a row the workbook does not hold").
		WithDetail("rowKeys", refused)
}
