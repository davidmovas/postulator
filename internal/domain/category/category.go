package category

import (
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Category struct {
	ID        string
	SiteID    string
	Name      string
	Key       string
	ParentID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func New(c Category) (Category, error) {
	c.Name = strings.TrimSpace(c.Name)
	c.Key = Key(c.Name)

	switch {
	case c.ID == "":
		return Category{}, invalid("category id must not be empty", "id")
	case c.SiteID == "":
		return Category{}, invalid("category site id must not be empty", "siteId")
	case c.Key == "":
		return Category{}, invalid("category name must not be empty", "name")
	case c.ParentID == c.ID:
		return Category{}, invalid("a category cannot sit under itself", "parentId")
	}
	return c, nil
}

func Chain(all []Category, leafID string) []Category {
	byID := make(map[string]Category, len(all))
	for i := range all {
		byID[all[i].ID] = all[i]
	}

	chain := make([]Category, 0)
	walked := make(map[string]struct{})
	for at, held := lookup(byID, leafID); held; at, held = lookup(byID, at.ParentID) {
		if _, again := walked[at.ID]; again {
			break
		}
		walked[at.ID] = struct{}{}
		chain = append(chain, at)
	}
	slices.Reverse(chain)
	return chain
}

func lookup(byID map[string]Category, id string) (Category, bool) {
	if id == "" {
		return Category{}, false
	}
	found, held := byID[id]
	return found, held
}
