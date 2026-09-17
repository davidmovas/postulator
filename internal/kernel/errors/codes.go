package errors

type Code string

const (
	NotFound       Code = "NOT_FOUND"
	Conflict       Code = "CONFLICT"
	Invalid        Code = "INVALID"
	Unauthorized   Code = "UNAUTHORIZED"
	RateLimited    Code = "RATE_LIMITED"
	BudgetExceeded Code = "BUDGET_EXCEEDED"
	External       Code = "EXTERNAL"
	Internal       Code = "INTERNAL"
	Cancelled      Code = "CANCELLED"
	NeedsHuman     Code = "NEEDS_HUMAN"
	Locked         Code = "LOCKED"
)

func (c Code) String() string {
	return string(c)
}

func Codes() []Code {
	return []Code{
		NotFound,
		Conflict,
		Invalid,
		Unauthorized,
		RateLimited,
		BudgetExceeded,
		External,
		Internal,
		Cancelled,
		NeedsHuman,
		Locked,
	}
}
