package run

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
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
	KindCustom   Kind = "custom"
)

func (k Kind) Valid() bool {
	switch k {
	case KindGenerate, KindRelink, KindAudit, KindSync, KindImport, KindRepair, KindCustom:
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

func SyncRecipe() []template.StepSpec {
	return []template.StepSpec{{Name: string(StepSyncSite), Enabled: true}}
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

type EstimateFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Estimate struct {
	Tokens   int               `json:"tokens"`
	USD      float64           `json:"usd"`
	Findings []EstimateFinding `json:"findings"`
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

type Item struct {
	ID          string
	RunID       string
	SiteID      string
	TargetID    string
	Status      Status
	CurrentStep string
	Attempts    int
	Seq         int
	AdvanceSeq  int64
	Checkpoint  Checkpoint
	LeaseUntil  *time.Time
	WakeAt      *time.Time
	PauseReason PauseReason
	Error       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FinishedAt  *time.Time
}

func NewItem(i Item) (Item, error) {
	if i.Status == "" {
		i.Status = StatusPending
	}
	if i.Checkpoint == nil {
		i.Checkpoint = NewCheckpoint()
	}

	switch {
	case i.ID == "":
		return Item{}, invalid("item id must not be empty", "id")
	case i.RunID == "":
		return Item{}, invalid("item run id must not be empty", "runId")
	case i.SiteID == "":
		return Item{}, invalid("item site id must not be empty", "siteId")
	case i.TargetID == "":
		return Item{}, invalid("item target id must not be empty", "targetId")
	case !i.Status.Valid():
		return Item{}, invalid("item status is not recognized", "status")
	case i.CurrentStep == "":
		return Item{}, invalid("item current step must not be empty", "currentStep")
	case i.Attempts < 0:
		return Item{}, invalid("item attempts must not be negative", "attempts")
	case i.PauseReason != "" && !i.PauseReason.Valid():
		return Item{}, invalid("pause reason is not recognized", "pauseReason")
	}
	return i, nil
}

type ExecStatus string

const (
	ExecStarted ExecStatus = "started"
	ExecDone    ExecStatus = "done"
	ExecFailed  ExecStatus = "failed"
)

func (s ExecStatus) Valid() bool {
	switch s {
	case ExecStarted, ExecDone, ExecFailed:
		return true
	default:
		return false
	}
}

type StepExec struct {
	ID         string
	RunID      string
	ItemID     string
	Step       string
	Attempt    int
	Status     ExecStatus
	InputHash  string
	ArtifactID *string
	Tokens     int
	USD        float64
	StartedAt  time.Time
	FinishedAt *time.Time
	Error      string
}

type ArtifactKind string

const (
	ArtifactLinkContext      ArtifactKind = "link_context"
	ArtifactDraft            ArtifactKind = "draft"
	ArtifactBodyHTML         ArtifactKind = "body_html"
	ArtifactMeta             ArtifactKind = "meta"
	ArtifactImages           ArtifactKind = "images"
	ArtifactValidationReport ArtifactKind = "validation_report"
	ArtifactJudgeReport      ArtifactKind = "judge_report"
	ArtifactPublishResult    ArtifactKind = "publish_result"
	ArtifactRelinkResult     ArtifactKind = "relink_result"
	ArtifactSyncResult       ArtifactKind = "sync_result"
	ArtifactFinalReport      ArtifactKind = "final_report"
)

func ArtifactKinds() []ArtifactKind {
	return []ArtifactKind{
		ArtifactLinkContext, ArtifactDraft, ArtifactBodyHTML, ArtifactMeta, ArtifactImages,
		ArtifactValidationReport, ArtifactJudgeReport, ArtifactPublishResult, ArtifactRelinkResult,
		ArtifactSyncResult, ArtifactFinalReport,
	}
}

func (k ArtifactKind) Valid() bool {
	for _, known := range ArtifactKinds() {
		if k == known {
			return true
		}
	}
	return false
}

type Artifact struct {
	ID        string
	RunID     string
	ItemID    string
	Step      string
	Kind      ArtifactKind
	Blob      []byte
	Size      int
	Hash      string
	Purged    bool
	ExpiresAt *time.Time
	CreatedAt time.Time
}

func HashBlob(blob []byte) string {
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}

func NewArtifact(a Artifact) (Artifact, error) {
	switch {
	case a.ID == "":
		return Artifact{}, invalid("artifact id must not be empty", "id")
	case a.RunID == "":
		return Artifact{}, invalid("artifact run id must not be empty", "runId")
	case a.ItemID == "":
		return Artifact{}, invalid("artifact item id must not be empty", "itemId")
	case a.Step == "":
		return Artifact{}, invalid("artifact step must not be empty", "step")
	case !a.Kind.Valid():
		return Artifact{}, invalid("artifact kind is not recognized", "kind")
	}

	a.Size = len(a.Blob)
	a.Hash = HashBlob(a.Blob)
	return a, nil
}

type Event struct {
	RunID   string
	Seq     int64
	Type    string
	At      time.Time
	Payload json.RawMessage
}

func NewEvent(e Event) (Event, error) {
	e.Type = strings.TrimSpace(e.Type)
	switch {
	case e.RunID == "":
		return Event{}, invalid("event run id must not be empty", "runId")
	case e.Seq <= 0:
		return Event{}, invalid("event sequence must be positive", "seq")
	case e.Type == "":
		return Event{}, invalid("event type must not be empty", "type")
	case e.At.IsZero():
		return Event{}, invalid("event timestamp must not be empty", "at")
	}
	if len(e.Payload) == 0 {
		e.Payload = json.RawMessage("{}")
	}
	return e, nil
}
