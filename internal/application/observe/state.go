package observe

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/site"
)

type Stage string

const (
	StageInventory Stage = "inventory"
	StageContent   Stage = "content"
	StageStore     Stage = "store"
	StageRender    Stage = "render"
	StageDerive    Stage = "derive"
	StageFields    Stage = "fields"
	StageDone      Stage = "done"
)

const (
	SourcePlugin = "plugin"
	SourceCore   = "core"
)

const CheckpointKey = "sync"

type Progress struct {
	State   State `json:"state"`
	Batches int   `json:"batches"`
}

type Options struct {
	Full         bool
	RenderRescan bool
}

type Counts struct {
	Pulled      int `json:"pulled"`
	Created     int `json:"created"`
	Updated     int `json:"updated"`
	Products    int `json:"products"`
	Rendered    int `json:"rendered"`
	Unreachable int `json:"unreachable"`
	Archived    int `json:"archived"`
	Typed       int `json:"typed"`
	Custom      int `json:"custom"`
	Closed      int `json:"closed"`
}

type State struct {
	StartedAt    time.Time           `json:"startedAt"`
	ModifiedSeen time.Time           `json:"modifiedSeen"`
	Since        *time.Time          `json:"since,omitempty"`
	SiteID       string              `json:"siteId"`
	Source       string              `json:"source"`
	Stage        Stage               `json:"stage"`
	Types        []string            `json:"types"`
	Cursor       string              `json:"cursor"`
	CoreType     string              `json:"coreType"`
	CorePage     int                 `json:"corePage"`
	StorePage    int                 `json:"storePage"`
	RenderAfter  string              `json:"renderAfter"`
	FieldsAfter  string              `json:"fieldsAfter"`
	Technical    site.TechnicalPages `json:"technical"`
	Findings     []content.Finding   `json:"findings,omitempty"`
	Counts       Counts              `json:"counts"`
	Full         bool                `json:"full"`
	Rescan       bool                `json:"rescan"`
}

func (s State) Done() bool {
	return s.Stage == StageDone
}

func (s *State) sawModified(at time.Time) {
	if at.After(s.ModifiedSeen) {
		s.ModifiedSeen = at.UTC()
	}
}
