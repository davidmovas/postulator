package runtime

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const defaultRetries = 3

type claim struct {
	item      run.Item
	record    run.Run
	page      pagemap.Page
	spec      template.TemplateSpec
	def       run.Definition
	step      run.StepDef
	artifacts map[run.ArtifactKind]run.Artifact
	params    map[string]any
	index     int
	version   int
	inputHash string
	timeout   time.Duration
	expectSeq int64
}

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

func (e *Engine) claim(parent context.Context, itemID string) (*claim, error) {
	var held *claim

	err := e.transact(parent, func(ctx context.Context, out *outbox) error {
		item, err := e.deps.Items.Get(ctx, itemID)
		if err != nil {
			return err
		}
		record, err := e.deps.Runs.Get(ctx, item.RunID)
		if err != nil {
			return err
		}

		now := e.now()
		switch {
		case record.Status.Terminal(), record.Status == run.StatusPaused, !item.Status.Advanceable():
			return nil
		case item.Status == run.StatusWaiting && item.WakeAt != nil && now.Before(*item.WakeAt):
			return nil
		case now.After(record.DeadlineAt):
			return e.expire(ctx, out, record, now)
		}

		scoped := record.Kind.PageScoped()

		var resolved templates.ResolveForPageResponse
		if scoped {
			resolved, err = e.deps.Specs.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: item.TargetID})
			if err != nil {
				return e.abandon(ctx, out, record, item, run.Classify(err), now)
			}
		}

		def, err := run.Plan(e.registry, record.Kind, record.Recipe, e.cfg.RunDeadline)
		if err != nil {
			return e.abandon(ctx, out, record, item, run.Classify(err), now)
		}

		index, step, found := def.StepByName(item.CurrentStep)
		if !found {
			fault := run.Fault{
				Class: run.FaultFatal, Code: run.CodeUnknownStep, Action: run.ActionFail,
				Message: "step " + item.CurrentStep + " is not part of the recipe of this run",
			}
			return e.abandon(ctx, out, record, item, fault, now)
		}

		var page pagemap.Page
		if scoped {
			page, err = e.deps.Pages.Get(ctx, item.TargetID)
			if err != nil {
				return e.abandon(ctx, out, record, item, run.Classify(err), now)
			}
		}

		stored, err := e.deps.Artifacts.ByItem(ctx, item.ID)
		if err != nil {
			return err
		}

		timeout := step.Timeout
		if timeout <= 0 {
			timeout = e.cfg.StepTimeout
		}

		expectSeq := item.AdvanceSeq
		claimed, err := e.deps.Items.Claim(ctx, item.ID, expectSeq, now.Add(timeout+e.cfg.LeaseDuration), now)
		if errors.IsCode(err, errors.Conflict) {
			return nil
		}
		if err != nil {
			return err
		}

		available := artifactsByKind(def, index, stored)
		params := run.ParamsFor(record.Recipe, step.Name)
		inputHash, err := run.InputHash(step.Name, params, required(step, available), resolved.Version, claimed.Checkpoint)
		if err != nil {
			return err
		}

		if record.Status == run.StatusPending {
			record.Status = run.StatusRunning
			record.StartedAt = &now
			if updateErr := e.deps.Runs.Update(ctx, record); updateErr != nil {
				return updateErr
			}
			out.add(ctx, record.ID, events.RunStarted, events.RunStartedPayload{RunID: record.ID})
		}
		if claimed.AdvanceSeq == 1 {
			out.add(ctx, record.ID, events.ItemStarted, events.ItemStartedPayload{RunID: record.ID, ItemID: claimed.ID})
		}
		out.add(ctx, record.ID, events.StepStarted,
			events.StepStartedPayload{RunID: record.ID, ItemID: claimed.ID, Step: step.Name})

		held = &claim{
			item: claimed, record: record, page: page, spec: resolved.Spec, def: def, step: step,
			artifacts: available, params: params, index: index, version: resolved.Version,
			inputHash: inputHash, timeout: timeout, expectSeq: claimed.AdvanceSeq,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return held, nil
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

	transition := result.Next
	if transition == "" {
		transition = run.TransitionContinue
	}

	switch transition {
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

func (e *Engine) settle(parent context.Context, held *claim, out outcome) (bool, error) {
	var again bool

	err := e.transact(context.WithoutCancel(parent), func(ctx context.Context, box *outbox) error {
		now := e.now()
		next := held.item
		next.Checkpoint = held.item.Checkpoint.MergedWith(out.checkpoint)
		next.Status = out.status
		next.CurrentStep = out.nextStep
		next.Attempts = out.attempts
		next.PauseReason = out.reason
		next.Error = ""
		next.Note = ""
		next.LeaseUntil = nil
		next.WakeAt = out.wakeAt
		next.UpdatedAt = now
		if out.fault != nil {
			next.Error = out.fault.Message
		}
		if out.fault == nil && (out.status == run.StatusPaused || out.status == run.StatusWaiting) {
			next.Note = out.message
		}
		if out.status == run.StatusCompleted {
			next.Note = out.notice
		}
		if next.Status.Terminal() {
			next.FinishedAt = &now
		}

		if !out.reuse {
			if err := e.record(ctx, held, out, now); err != nil {
				return err
			}
		}

		persisted, err := e.deps.Items.Persist(ctx, next, held.expectSeq)
		if err != nil {
			return err
		}
		if !persisted {
			return nil
		}

		e.announce(ctx, box, held, out, next)
		again = out.again
		return e.settleRun(ctx, box, held.record, now)
	})
	if err != nil {
		return false, err
	}
	return again, nil
}

func (e *Engine) record(ctx context.Context, held *claim, out outcome, now time.Time) error {
	if replaceErr := e.deps.Artifacts.ReplaceStep(ctx, held.item.ID, held.step.Name, stamped(held, out.artifacts, now)); replaceErr != nil {
		return replaceErr
	}

	status := run.ExecDone
	message := ""
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

	attempt, err := e.deps.Execs.CountByStep(ctx, held.item.ID, held.step.Name)
	if err != nil {
		return err
	}

	startedAt := out.startedAt
	if startedAt.IsZero() {
		startedAt = now
	}

	return e.deps.Execs.Insert(ctx, run.StepExec{
		ID: id.New(), RunID: held.record.ID, ItemID: held.item.ID, Step: held.step.Name,
		Attempt: attempt + 1, Status: status, InputHash: held.inputHash,
		Tokens: out.tokens, USD: out.usd, StartedAt: startedAt, FinishedAt: &now, Error: message,
	})
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

func (e *Engine) announce(ctx context.Context, box *outbox, held *claim, out outcome, next run.Item) {
	runID, itemID, step := held.record.ID, held.item.ID, held.step.Name

	switch {
	case out.status == run.StatusWaiting && out.attempts > 0:
		box.add(ctx, runID, events.StepRetrying, events.StepRetryingPayload{
			RunID: runID, ItemID: itemID, Step: step, Attempt: out.attempts,
			AfterMs: e.delayMs(out.wakeAt), Code: faultCode(out.fault), Message: out.message,
		})
	case out.fault != nil:
		box.add(ctx, runID, events.StepFailed, events.StepFailedPayload{
			RunID: runID, ItemID: itemID, Step: step, Code: out.fault.Code, Message: out.fault.Message,
		})
	default:
		box.add(ctx, runID, events.StepDone, events.StepDonePayload{
			RunID: runID, ItemID: itemID, Step: step, DurationMs: out.duration.Milliseconds(), Message: out.message,
		})
	}

	switch next.Status {
	case run.StatusCompleted:
		box.add(ctx, runID, events.ItemDone, events.ItemDonePayload{RunID: runID, ItemID: itemID, Note: next.Note})
	case run.StatusFailed:
		box.add(ctx, runID, events.ItemFailed, events.ItemFailedPayload{
			RunID: runID, ItemID: itemID, Code: faultCode(out.fault), Message: out.message,
		})
	case run.StatusPaused:
		message := next.Note
		if message == "" {
			message = out.message
		}
		box.add(ctx, runID, events.ItemNeedsHuman, events.ItemNeedsHumanPayload{
			RunID: runID, ItemID: itemID, Reason: string(next.PauseReason), Message: message,
		})
	}
}

func (e *Engine) delayMs(wakeAt *time.Time) int64 {
	if wakeAt == nil {
		return 0
	}
	return max(wakeAt.Sub(e.now()).Milliseconds(), 0)
}

func faultCode(fault *run.Fault) string {
	if fault == nil {
		return run.CodeUnclassified
	}
	return fault.Code
}

func (e *Engine) abandon(ctx context.Context, box *outbox, record run.Run, item run.Item, fault run.Fault, now time.Time) error {
	next := item
	next.Status = run.StatusFailed
	next.Error = fault.Message
	next.LeaseUntil = nil
	next.WakeAt = nil
	next.UpdatedAt = now
	next.FinishedAt = &now

	persisted, err := e.deps.Items.Persist(ctx, next, item.AdvanceSeq)
	if err != nil || !persisted {
		return err
	}

	box.add(ctx, record.ID, events.ItemFailed, events.ItemFailedPayload{
		RunID: record.ID, ItemID: item.ID, Code: fault.Code, Message: fault.Message,
	})
	return e.settleRun(ctx, box, record, now)
}

func (e *Engine) settleRun(ctx context.Context, box *outbox, record run.Run, now time.Time) error {
	current, err := e.deps.Runs.Get(ctx, record.ID)
	if err != nil {
		return err
	}
	if current.Status.Terminal() || current.Status == run.StatusPaused {
		return nil
	}

	counts, err := e.deps.Items.Counts(ctx, current.ID)
	if err != nil {
		return err
	}
	spend, err := e.deps.Spend.SumByRun(ctx, current.ID)
	if err != nil {
		return err
	}

	current.Stats = run.Stats{
		Items:  total(counts),
		Done:   counts[run.StatusCompleted],
		Failed: counts[run.StatusFailed],
		Tokens: spend.Usage.Total,
		USD:    spend.USD,
	}

	if overBudget(current, spend) {
		current.Status = run.StatusPaused
		current.PauseReason = run.PauseBudgetExceeded
		if _, stopErr := e.deps.Items.StopAll(ctx, current.ID, pausable, run.StatusPaused, run.PauseBudgetExceeded, now); stopErr != nil {
			return stopErr
		}
		if updateErr := e.deps.Runs.Update(ctx, current); updateErr != nil {
			return updateErr
		}
		box.add(ctx, current.ID, events.RunBudgetExceeded, events.RunBudgetExceededPayload{
			RunID: current.ID, SpentUSD: spend.USD, BudgetUSD: current.Budget.MaxUSD,
			SpentTokens: spend.Usage.Total, BudgetTokens: current.Budget.MaxTokens,
		})
		box.add(ctx, current.ID, events.RunPaused, events.RunPausedPayload{
			RunID: current.ID, Reason: string(run.PauseBudgetExceeded),
		})
		return nil
	}

	if advanceable(counts) {
		return e.deps.Runs.Update(ctx, current)
	}

	if counts[run.StatusPaused] > 0 {
		reason, reasonErr := e.heldFor(ctx, current.ID)
		if reasonErr != nil {
			return reasonErr
		}
		current.Status = run.StatusPaused
		current.PauseReason = reason
		if updateErr := e.deps.Runs.Update(ctx, current); updateErr != nil {
			return updateErr
		}
		box.add(ctx, current.ID, events.RunPaused, events.RunPausedPayload{
			RunID: current.ID, Reason: string(reason),
		})
		return nil
	}

	current.FinishedAt = &now
	if current.Stats.Done == 0 && current.Stats.Failed > 0 {
		current.Status = run.StatusFailed
		if updateErr := e.deps.Runs.Update(ctx, current); updateErr != nil {
			return updateErr
		}
		box.add(ctx, current.ID, events.RunFailed, events.RunFailedPayload{
			RunID: current.ID, Code: run.CodeUnclassified, Message: "every item of the run failed",
		})
		return nil
	}

	current.Status = run.StatusCompleted
	if updateErr := e.deps.Runs.Update(ctx, current); updateErr != nil {
		return updateErr
	}
	box.add(ctx, current.ID, events.RunCompleted, events.RunCompletedPayload{
		RunID: current.ID, Succeeded: current.Stats.Done, Failed: current.Stats.Failed,
	})
	return nil
}

func (e *Engine) expire(ctx context.Context, box *outbox, record run.Run, now time.Time) error {
	current, err := e.deps.Runs.Get(ctx, record.ID)
	if err != nil {
		return err
	}
	if current.Status.Terminal() {
		return nil
	}
	record = current

	if _, err = e.deps.Items.StopAll(ctx, record.ID, stoppable, run.StatusCancelled, "", now); err != nil {
		return err
	}

	record.Status = run.StatusFailed
	record.Error = "the run passed its deadline before every item finished"
	record.FinishedAt = &now
	if updateErr := e.deps.Runs.Update(ctx, record); updateErr != nil {
		return updateErr
	}

	box.add(ctx, record.ID, events.RunFailed, events.RunFailedPayload{
		RunID: record.ID, Code: run.CodeDeadlineExceeded, Message: record.Error,
	})
	return nil
}

func (e *Engine) heldFor(ctx context.Context, runID string) (run.PauseReason, error) {
	items, err := e.deps.Items.ByRun(ctx, runID)
	if err != nil {
		return "", err
	}
	for i := range items {
		if items[i].Status == run.StatusPaused && items[i].PauseReason != run.PauseAwaitingParent {
			return run.PauseNeedsHuman, nil
		}
	}
	return run.PauseAwaitingParent, nil
}

func (e *Engine) revive(record *run.Run, now time.Time) {
	record.Status = run.StatusRunning
	record.PauseReason = ""
	record.Error = ""
	record.FinishedAt = nil
	record.DeadlineAt = now.Add(e.cfg.RunDeadline)
}

func overBudget(record run.Run, spend llm.Spend) bool {
	overMoney := record.Budget.MaxUSD > 0 && spend.USD > record.Budget.MaxUSD
	overTokens := record.Budget.MaxTokens > 0 && spend.Usage.Total > record.Budget.MaxTokens
	return overMoney || overTokens
}

func total(counts map[run.Status]int) int {
	sum := 0
	for _, count := range counts {
		sum += count
	}
	return sum
}

func advanceable(counts map[run.Status]int) bool {
	for status, count := range counts {
		if count > 0 && status.Advanceable() {
			return true
		}
	}
	return false
}

func required(step run.StepDef, available map[run.ArtifactKind]run.Artifact) []run.Artifact {
	out := make([]run.Artifact, 0, len(step.Requires))
	for _, kind := range step.Requires {
		if artifact, ok := available[kind]; ok {
			out = append(out, artifact)
		}
	}
	return out
}

func artifactsByKind(def run.Definition, upto int, stored []run.Artifact) map[run.ArtifactKind]run.Artifact {
	order := make(map[string]int, len(def.Steps))
	for i := range def.Steps {
		order[def.Steps[i].Name] = i
	}

	byKind := make(map[run.ArtifactKind]run.Artifact, len(stored))
	rank := make(map[run.ArtifactKind]int, len(stored))
	for i := range stored {
		artifact := stored[i]
		position, known := order[artifact.Step]
		if !known || position > upto {
			continue
		}
		if seen, ok := rank[artifact.Kind]; ok && seen >= position {
			continue
		}
		byKind[artifact.Kind] = artifact
		rank[artifact.Kind] = position
	}
	return byKind
}
