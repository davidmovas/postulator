package runtime

import "context"

func (e *Engine) advance(parent context.Context, itemID string) (bool, error) {
	held, err := e.claim(parent, itemID)
	if err != nil || held == nil {
		return false, err
	}
	e.hold(held.item.ID, held.expectSeq)

	out, err := e.work(parent, held)
	if err != nil {
		return false, err
	}
	if out.stopped && e.stopping() {
		return false, nil
	}

	again, err := e.settle(parent, held, out)
	if err != nil {
		return false, err
	}
	e.forget(held.item.ID)
	if !out.status.Advanceable() {
		e.nudge()
	}
	return again, nil
}
