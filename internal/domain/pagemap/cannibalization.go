package pagemap

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
)

type Reason string

const (
	ReasonSamePrimaryKeyword  Reason = "same_primary_keyword"
	ReasonSameEntityCanonical Reason = "same_entity_canonical"
	ReasonPathConflict        Reason = "path_conflict"
)

type Evidence struct {
	PageID   string
	Path     string
	Reason   Reason
	EntityID string
}

type Verdict struct {
	Allowed  bool
	Evidence []Evidence
}

func Cannibalization(candidate Page, entity graph.Entity, index Index, g graph.Graph) Verdict {
	var evidence []Evidence

	if path, err := NormalizePath(candidate.Path); err == nil {
		if other, ok := index.byPath[path]; ok && other.ID != candidate.ID {
			evidence = append(evidence, Evidence{PageID: other.ID, Path: other.Path, Reason: ReasonPathConflict})
		}
	}

	if entity.ID == "" {
		return Verdict{Allowed: len(evidence) == 0, Evidence: evidence}
	}

	if entity.CanonicalPageID != nil && *entity.CanonicalPageID != candidate.ID {
		if owner, ok := index.byID[*entity.CanonicalPageID]; ok {
			evidence = append(evidence, Evidence{PageID: owner.ID, Path: owner.Path, Reason: ReasonSameEntityCanonical, EntityID: entity.ID})
		}
	}

	if entity.PrimaryKeyword != "" {
		others := g.Entities()
		for i := range others {
			other := &others[i]
			if other.ID == entity.ID || other.CanonicalPageID == nil || !strings.EqualFold(other.PrimaryKeyword, entity.PrimaryKeyword) {
				continue
			}
			owner, ok := index.byID[*other.CanonicalPageID]
			if !ok || owner.ID == candidate.ID {
				continue
			}
			evidence = append(evidence, Evidence{PageID: owner.ID, Path: owner.Path, Reason: ReasonSamePrimaryKeyword, EntityID: other.ID})
		}
	}

	return Verdict{Allowed: len(evidence) == 0, Evidence: evidence}
}
