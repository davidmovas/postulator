package run

import "time"

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
	BlockedBy   string
	Checkpoint  Checkpoint
	LeaseUntil  *time.Time
	WakeAt      *time.Time
	PauseReason PauseReason
	Error       string
	Note        string
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
	case i.BlockedBy == i.ID:
		return Item{}, invalid("an item cannot wait for itself", "blockedBy")
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
