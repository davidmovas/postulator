package run

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Kind string

const (
	KindGenerate Kind = "generate"
	KindRelink   Kind = "relink"
	KindAudit    Kind = "audit"
	KindSync     Kind = "sync"
	KindImport   Kind = "import"
	KindRepair   Kind = "repair"
	KindRevert   Kind = "revert"
	KindCustom   Kind = "custom"
)

func (k Kind) Valid() bool {
	switch k {
	case KindGenerate, KindRelink, KindAudit, KindSync, KindImport, KindRepair, KindRevert, KindCustom:
		return true
	default:
		return false
	}
}

func (k Kind) PageScoped() bool {
	switch k {
	case KindGenerate, KindRelink, KindAudit, KindRepair, KindCustom:
		return true
	default:
		return false
	}
}

type PublishMode string

const (
	PublishDraft   PublishMode = "draft"
	PublishLive    PublishMode = "publish"
	defaultPublish             = PublishDraft
)

func (m PublishMode) Valid() bool {
	switch m {
	case PublishDraft, PublishLive:
		return true
	default:
		return false
	}
}

type Budget struct {
	MaxUSD    float64 `json:"maxUsd"`
	MaxTokens int     `json:"maxTokens"`
}

type Stats struct {
	Items  int     `json:"items"`
	Done   int     `json:"done"`
	Failed int     `json:"failed"`
	Tokens int     `json:"tokens"`
	USD    float64 `json:"usd"`
}

type Run struct {
	ID              string
	SiteID          string
	Kind            Kind
	Status          Status
	Targets         []string
	Recipe          []template.StepSpec
	TemplateID      string
	TemplateVersion int
	PublishMode     PublishMode
	Budget          Budget
	Stats           Stats
	CreatedBy       kctx.Actor
	ParentRunID     *string
	PauseReason     PauseReason
	Error           string
	DeadlineAt      time.Time
	CreatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func NewRun(r Run) (Run, error) {
	if r.PublishMode == "" {
		r.PublishMode = defaultPublish
	}
	if r.Status == "" {
		r.Status = StatusPending
	}
	if r.CreatedBy == "" {
		r.CreatedBy = kctx.ActorUser
	}

	switch {
	case r.ID == "":
		return Run{}, invalid("run id must not be empty", "id")
	case r.SiteID == "":
		return Run{}, invalid("run site id must not be empty", "siteId")
	case !r.Kind.Valid():
		return Run{}, invalid("run kind is not recognized", "kind")
	case !r.Status.Valid():
		return Run{}, invalid("run status is not recognized", "status")
	case !r.PublishMode.Valid():
		return Run{}, invalid("publish mode is not recognized", "publishMode")
	case len(r.Targets) == 0:
		return Run{}, invalid("a run needs at least one target page", "targets")
	case len(r.Recipe) == 0:
		return Run{}, invalid("a run needs a recipe", "recipe")
	case r.Budget.MaxUSD < 0 || r.Budget.MaxTokens < 0:
		return Run{}, invalid("a budget must not be negative", "budget")
	case r.PauseReason != "" && !r.PauseReason.Valid():
		return Run{}, invalid("pause reason is not recognized", "pauseReason")
	case r.ParentRunID != nil && *r.ParentRunID == "":
		return Run{}, invalid("parent run id must not be empty when set", "parentRunId")
	case r.DeadlineAt.IsZero():
		return Run{}, invalid("a run needs a deadline", "deadlineAt")
	}
	return r, nil
}
