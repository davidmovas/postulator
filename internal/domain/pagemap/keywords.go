package pagemap

import (
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
)

func Keywords(page Page, entity graph.Entity) keyword.List {
	if len(page.Keywords) > 0 {
		return keyword.New(page.Keywords)
	}
	return keyword.New(entity.Keywords)
}
