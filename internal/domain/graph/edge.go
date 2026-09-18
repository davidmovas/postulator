package graph

import "time"

type EdgeKind string

const (
	EdgeParent  EdgeKind = "parent"
	EdgeRelated EdgeKind = "related"
)

func (k EdgeKind) Valid() bool {
	switch k {
	case EdgeParent, EdgeRelated:
		return true
	default:
		return false
	}
}

type EdgeStatus string

const (
	StatusApproved EdgeStatus = "approved"
	StatusProposed EdgeStatus = "proposed"
	StatusRejected EdgeStatus = "rejected"
)

func (s EdgeStatus) Valid() bool {
	switch s {
	case StatusApproved, StatusProposed, StatusRejected:
		return true
	default:
		return false
	}
}

type Edge struct {
	ID           string
	SiteID       string
	FromEntityID string
	ToEntityID   string
	Kind         EdgeKind
	Weight       float64
	Source       Source
	Status       EdgeStatus
	CreatedAt    time.Time
}

func NewEdge(e Edge) (Edge, error) {
	switch {
	case e.ID == "":
		return Edge{}, invalid("edge id must not be empty", "id")
	case e.SiteID == "":
		return Edge{}, invalid("edge site id must not be empty", "siteId")
	case e.FromEntityID == "" || e.ToEntityID == "":
		return Edge{}, invalid("edge endpoints must not be empty", "fromEntityId")
	case e.FromEntityID == e.ToEntityID:
		return Edge{}, invalid("edge endpoints must differ", "toEntityId")
	case !e.Kind.Valid():
		return Edge{}, invalid("edge kind is not recognized", "kind")
	case !e.Source.Valid():
		return Edge{}, invalid("edge source is not recognized", "source")
	case !e.Status.Valid():
		return Edge{}, invalid("edge status is not recognized", "status")
	case e.Weight < 0 || e.Weight > 1:
		return Edge{}, invalid("edge weight must be between 0 and 1", "weight")
	}

	if e.Kind == EdgeParent {
		e.Weight = 1
	}
	if e.Kind == EdgeRelated && e.FromEntityID > e.ToEntityID {
		e.FromEntityID, e.ToEntityID = e.ToEntityID, e.FromEntityID
	}
	return e, nil
}
