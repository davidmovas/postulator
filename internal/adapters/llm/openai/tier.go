package openai

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"
	"time"

	port "github.com/davidmovas/postulator/internal/application/llm"
)

const codeResourceUnavailable = "resource_unavailable"

var (
	errImpatient   = stderrors.New("the flex tier did not answer within its patience")
	capacityTokens = []string{"resource unavailable", codeResourceUnavailable, "capacity"}
)

func WithFlexPatience(patience time.Duration) Option {
	return func(c *Client) { c.patience = max(patience, 0) }
}

func (e exchange) flex() bool {
	return e.body.ServiceTier == tierFlex
}

func (e exchange) onDefault() exchange {
	e.body.ServiceTier = tierDefault
	return e
}

type attempt struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	timer  *time.Timer
}

func (c *Client) begin(call context.Context, ex exchange) attempt {
	ctx, cancel := context.WithCancelCause(call)
	started := attempt{ctx: ctx, cancel: cancel}
	if ex.flex() && c.patience > 0 {
		started.timer = time.AfterFunc(c.patience, func() { cancel(errImpatient) })
	}
	return started
}

func (a attempt) settle() bool {
	if a.timer == nil {
		return false
	}
	return !a.timer.Stop() && stderrors.Is(context.Cause(a.ctx), errImpatient)
}

func (a attempt) release() {
	a.cancel(nil)
}

func fallsBack(call context.Context, ex exchange, impatient bool, err error) bool {
	if err == nil || !ex.flex() || call.Err() != nil {
		return false
	}
	return impatient || capacityRefusal(err)
}

func capacityRefusal(err error) bool {
	refused := refusalIn(err)
	return refused != nil && refused.effectiveStatus() == http.StatusTooManyRequests && outOfCapacity(refused.fault)
}

func outOfCapacity(fault wireFault) bool {
	if exhausted(fault) {
		return false
	}
	told := strings.ToLower(fault.Code + " " + fault.Type + " " + fault.Message)
	for _, token := range capacityTokens {
		if strings.Contains(told, token) {
			return true
		}
	}
	return false
}

func (c *Client) complete(call context.Context, ex exchange) (port.Response, error) {
	resp, impatient, err := c.completeAttempt(call, ex)
	if !fallsBack(call, ex, impatient, err) {
		return resp, err
	}
	resp, _, err = c.completeAttempt(call, ex.onDefault())
	return resp, err
}

func (c *Client) completeAttempt(call context.Context, ex exchange) (port.Response, bool, error) {
	started := c.begin(call, ex)
	defer started.release()

	resp, err := c.completeOnce(started.ctx, ex)
	return resp, started.settle(), err
}

func (c *Client) open(call context.Context, ex exchange) (*liveStream, error) {
	live, impatient, err := c.openAttempt(call, ex)
	if !fallsBack(call, ex, impatient, err) {
		return live, err
	}
	live, _, err = c.openAttempt(call, ex.onDefault())
	return live, err
}

func (c *Client) openAttempt(call context.Context, ex exchange) (*liveStream, bool, error) {
	started := c.begin(call, ex)
	live, err := c.openOnce(started.ctx, started.release, ex)
	impatient := started.settle()
	if err == nil && !impatient {
		return live, false, nil
	}

	if live != nil {
		live.close()
	}
	started.release()
	if err == nil {
		err = errImpatient
	}
	return nil, impatient, err
}
