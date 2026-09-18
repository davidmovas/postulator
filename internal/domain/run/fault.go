package run

import "github.com/davidmovas/postulator/internal/kernel/errors"

type FaultClass string

const (
	FaultTransient FaultClass = "transient"
	FaultExhausted FaultClass = "exhausted"
	FaultInvalid   FaultClass = "invalid"
	FaultFatal     FaultClass = "fatal"
)

func (c FaultClass) Valid() bool {
	switch c {
	case FaultTransient, FaultExhausted, FaultInvalid, FaultFatal:
		return true
	default:
		return false
	}
}

type FaultAction string

const (
	ActionRetry FaultAction = "retry"
	ActionPause FaultAction = "pause"
	ActionFail  FaultAction = "fail"
)

const (
	CodeUnclassified      = "UNCLASSIFIED"
	CodeStepTimeout       = "STEP_TIMEOUT"
	CodeAttemptsExhausted = "ATTEMPTS_EXHAUSTED"
	CodeDeadlineExceeded  = "RUN_DEADLINE_EXCEEDED"
	CodeUnknownStep       = "UNKNOWN_STEP"
	CodeCancelled         = "CANCELLED_BY_REQUEST"
)

type Fault struct {
	Class   FaultClass  `json:"class"`
	Code    string      `json:"code"`
	Action  FaultAction `json:"action"`
	Reason  PauseReason `json:"reason,omitempty"`
	Message string      `json:"message"`
}

func DefaultAction(class FaultClass) FaultAction {
	switch class {
	case FaultTransient:
		return ActionRetry
	case FaultExhausted:
		return ActionPause
	case FaultInvalid, FaultFatal:
		return ActionFail
	default:
		return ActionFail
	}
}

func Classify(err error) Fault {
	if err == nil {
		return Fault{}
	}

	code := errors.CodeOf(err)
	fault := Fault{Code: code.String(), Message: err.Error()}

	switch code {
	case errors.RateLimited, errors.External:
		fault.Class = FaultTransient
	case errors.Cancelled:
		fault.Class = FaultTransient
		fault.Code = CodeStepTimeout
	case errors.BudgetExceeded:
		fault.Class = FaultExhausted
		fault.Reason = PauseBudgetExceeded
	case errors.NeedsHuman:
		fault.Class = FaultExhausted
		fault.Reason = PauseNeedsHuman
	case errors.Invalid, errors.Unauthorized, errors.NotFound, errors.Conflict:
		fault.Class = FaultInvalid
	default:
		fault.Class = FaultFatal
	}

	fault.Action = DefaultAction(fault.Class)
	return fault
}
