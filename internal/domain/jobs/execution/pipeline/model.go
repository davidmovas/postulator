package pipeline

type Command interface {
	Name() string
	Execute(ctx *Context) error
	CanExecute(ctx *Context) bool
	RequiredState() State
	NextState() State
	OnError(pctx *Context, err error) error
	IsRetryable() bool
	MaxRetries() int
}
