package run

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusWaiting   Status = "waiting"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

func (s Status) Valid() bool {
	return s.Active() || s.Terminal()
}

func (s Status) Active() bool {
	switch s {
	case StatusPending, StatusRunning, StatusWaiting, StatusPaused:
		return true
	default:
		return false
	}
}

func (s Status) Terminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

func (s Status) Advanceable() bool {
	switch s {
	case StatusPending, StatusRunning, StatusWaiting:
		return true
	default:
		return false
	}
}

type Transition string

const (
	TransitionContinue Transition = "continue"
	TransitionWait     Transition = "wait"
	TransitionPause    Transition = "pause"
	TransitionComplete Transition = "complete"
	TransitionFail     Transition = "fail"
)

func (t Transition) Valid() bool {
	switch t {
	case TransitionContinue, TransitionWait, TransitionPause, TransitionComplete, TransitionFail:
		return true
	default:
		return false
	}
}

type PauseReason string

const (
	PauseBudgetExceeded       PauseReason = "budget_exceeded"
	PauseAwaitingConfirmation PauseReason = "awaiting_confirmation"
	PauseNeedsHuman           PauseReason = "needs_human"
	PauseUser                 PauseReason = "user"
)

func (r PauseReason) Valid() bool {
	switch r {
	case PauseBudgetExceeded, PauseAwaitingConfirmation, PauseNeedsHuman, PauseUser:
		return true
	default:
		return false
	}
}
