package bindings

import "Postulator/internal/handlers"

type Binder struct {
	handler *handlers.Handler
}

func NewBinder(handler *handlers.Handler) *Binder {
	return &Binder{
		handler: handler,
	}
}

// SetHandler sets the handler for the binder
func (b *Binder) SetHandler(handler *handlers.Handler) {
	b.handler = handler
}
