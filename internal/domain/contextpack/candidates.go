package contextpack

import "github.com/davidmovas/postulator/internal/domain/pagemap"

const candidateLimit = 20

type Candidate struct {
	Name string `json:"name"`
	Path string `json:"path"`
	WPID int64  `json:"wpId"`
}

func (s Snapshot) candidates(page pagemap.Page) []Candidate {
	if page.WPType != pagemap.WPProduct {
		return nil
	}
	out := make([]Candidate, 0)
	seen := map[string]struct{}{page.ID: {}}
	offer := func(pages []pagemap.Page) {
		ranked := s.ranked(page, pages)
		for i := range ranked {
			other := &ranked[i]
			if _, held := seen[other.ID]; held || other.WPType != pagemap.WPProduct || !other.Live() || len(out) == candidateLimit {
				continue
			}
			seen[other.ID] = struct{}{}
			out = append(out, Candidate{WPID: *other.WPID, Name: s.titleOf(*other), Path: other.Path})
		}
	}
	offer(s.siblings(page))
	if entity, mapped := s.entity(page); mapped {
		related := s.Graph.Related(entity.ID, 0)
		for i := range related {
			offer(s.Index.ByEntity(related[i].Entity.ID))
		}
	}
	return out
}
