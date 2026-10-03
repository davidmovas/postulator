package runtime

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const defaultRetries = 3

type outcome struct {
	wakeAt     *time.Time
	startedAt  time.Time
	duration   time.Duration
	checkpoint run.Checkpoint
	artifacts  []run.Artifact
	fault      *run.Fault
	status     run.Status
	nextStep   string
	reason     run.PauseReason
	message    string
	notice     string
	attempts   int
	tokens     int
	usd        float64
	again      bool
	reuse      bool
	stopped    bool
}

func (e *Engine) work(parent context.Context, held *claim) (outcome, error) {
	started := e.clock.Now().UTC()

	reused, ok, err := e.reuse(parent, held)
	if err != nil {
		return outcome{}, err
	}
	if ok {
		reused.startedAt = started
		return reused, nil
	}

	ctx, done := e.runContext(parent, held.record.ID, held.item.ID)
	result, stepErr := e.execute(ctx, held)
	done()

	out := e.outcomeOf(held, result, stepErr)
	out.startedAt = started
	out.duration = e.clock.Now().UTC().Sub(started)
	return out, nil
}

func (e *Engine) reuse(parent context.Context, held *claim) (outcome, bool, error) {
	done, err := e.deps.Execs.Done(parent, held.item.ID, held.step.Name, held.inputHash)
	if errors.IsCode(err, errors.NotFound) {
		return outcome{}, false, nil
	}
	if err != nil {
		return outcome{}, false, err
	}

	result := run.Result{Next: run.TransitionContinue, Checkpoint: run.NewCheckpoint(), Message: "reused"}
	reused := e.outcomeOf(held, result, nil)
	reused.reuse = true
	reused.tokens = done.Tokens
	reused.usd = done.USD
	return reused, true, nil
}

func (e *Engine) execute(ctx context.Context, held *claim) (result run.Result, err error) {
	stepCtx, cancel := context.WithTimeout(ctx, held.timeout)
	defer cancel()

	defer func() {
		if recovered := recover(); recovered != nil {
			e.logger.Error("a run step panicked and was turned into a retryable fault")
			err = errors.New(errors.External, fmt.Sprintf("step %s panicked: %v", held.step.Name, recovered))
		}
	}()

	sc := &run.StepContext{
		Run: held.record, Item: held.item, Page: held.page, Spec: held.spec,
		Params: held.params, Artifacts: held.artifacts, Check: held.item.Checkpoint.Clone(),
	}
	return held.step.Run(stepCtx, sc)
}

func (e *Engine) outcomeOf(held *claim, result run.Result, stepErr error) outcome {
	out := outcome{
		checkpoint: result.Checkpoint,
		artifacts:  result.Artifacts,
		message:    result.Message,
		notice:     result.Notice,
		nextStep:   held.step.Name,
		tokens:     result.Tokens,
		usd:        result.USD,
	}
	if stepErr != nil {
		out.stopped = stderrors.Is(stepErr, context.Canceled) || errors.IsCode(stepErr, errors.Cancelled)
		return e.faultOutcome(held, out, stepErr)
	}
	return e.transitionOutcome(held, out, result)
}

func (e *Engine) transitionOutcome(held *claim, out outcome, result run.Result) outcome {
	switch result.Next {
	case run.TransitionWait:
		wakeAt := result.WakeAt
		if wakeAt.IsZero() {
			wakeAt = e.now().Add(e.cfg.SweepInterval)
		}
		out.status = run.StatusWaiting
		out.wakeAt = &wakeAt
		out.again = !wakeAt.After(e.now())
	case run.TransitionPause:
		reason := result.Reason
		if reason == "" {
			reason = run.PauseNeedsHuman
		}
		out.status = run.StatusPaused
		out.reason = reason
	case run.TransitionFail:
		message := out.message
		if message == "" {
			message = "step " + held.step.Name + " stopped the item without naming a reason"
		}
		out.status = run.StatusFailed
		out.message = message
		out.fault = &run.Fault{
			Class: run.FaultInvalid, Code: run.CodeUnclassified, Action: run.ActionFail, Message: message,
		}
	case run.TransitionComplete:
		out.status = run.StatusCompleted
	default:
		if held.index+1 >= len(held.def.Steps) {
			out.status = run.StatusCompleted
			break
		}
		out.status = run.StatusPending
		out.nextStep = held.def.Steps[held.index+1].Name
		out.again = true
	}
	return out
}

func (e *Engine) faultOutcome(held *claim, out outcome, stepErr error) outcome {
	fault := run.Classify(stepErr)
	out.fault = &fault
	out.message = fault.Message

	switch fault.Action {
	case run.ActionRetry:
		limit := held.step.Retry.Max
		if limit <= 0 {
			limit = defaultRetries
		}
		attempts := held.item.Attempts + 1
		if attempts >= limit {
			exhausted := fault
			exhausted.Code = run.CodeAttemptsExhausted
			exhausted.Action = run.ActionFail
			exhausted.Message = fmt.Sprintf("step %s failed %d times and has no attempts left: %s",
				held.step.Name, attempts, fault.Message)
			out.fault = &exhausted
			out.message = exhausted.Message
			out.status = run.StatusFailed
			out.attempts = attempts
			return out
		}

		wakeAt := e.now().Add(backoff(held.step.Retry, attempts))
		out.status = run.StatusWaiting
		out.wakeAt = &wakeAt
		out.attempts = attempts
	case run.ActionPause:
		out.status = run.StatusPaused
		out.reason = fault.Reason
	default:
		out.status = run.StatusFailed
	}
	return out
}

func backoff(policy run.RetryPolicy, attempt int) time.Duration {
	if policy.Backoff != nil {
		if delay := policy.Backoff(attempt); delay > 0 {
			return min(delay, maxRetryBackoff)
		}
	}

	delay := DefaultRetryBackoff
	for range attempt - 1 {
		delay *= 2
		if delay >= maxRetryBackoff {
			return maxRetryBackoff
		}
	}
	return delay
}

func (e *Engine) record(ctx context.Context, held *claim, out outcome, now time.Time) error {
	if err := e.deps.Artifacts.ReplaceStep(ctx, held.item.ID, held.step.Name, stamped(held, out.artifacts, now)); err != nil {
		return err
	}

	attempt, err := e.deps.Execs.CountByStep(ctx, held.item.ID, held.step.Name)
	if err != nil {
		return err
	}

	startedAt := out.startedAt
	if startedAt.IsZero() {
		startedAt = now
	}

	status, message := execOf(out)
	return e.deps.Execs.Insert(ctx, run.StepExec{
		ID: id.New(), RunID: held.record.ID, ItemID: held.item.ID, Step: held.step.Name,
		Attempt: attempt + 1, Status: status, InputHash: held.inputHash,
		Tokens: out.tokens, USD: out.usd, StartedAt: startedAt, FinishedAt: &now, Error: message,
	})
}

func execOf(out outcome) (status run.ExecStatus, message string) {
	status = run.ExecDone
	if out.status == run.StatusWaiting && out.fault == nil && len(out.artifacts) == 0 && len(out.checkpoint) == 0 {
		status = run.ExecStarted
	}
	if out.status == run.StatusFailed || (out.fault != nil && out.fault.Action != run.ActionPause) {
		status = run.ExecFailed
		message = out.fault.Message
	}
	if out.status == run.StatusPaused {
		status = run.ExecFailed
		message = out.message
		if out.fault != nil {
			message = out.fault.Message
		}
	}
	return status, message
}

func stamped(held *claim, artifacts []run.Artifact, now time.Time) []run.Artifact {
	out := make([]run.Artifact, 0, len(artifacts))
	for i := range artifacts {
		artifact := artifacts[i]
		artifact.ID = id.New()
		artifact.RunID = held.record.ID
		artifact.ItemID = held.item.ID
		artifact.Step = held.step.Name
		artifact.CreatedAt = now
		artifact.Size = len(artifact.Blob)
		artifact.Hash = run.HashBlob(artifact.Blob)
		out = append(out, artifact)
	}
	return out
}
