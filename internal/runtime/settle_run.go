package runtime

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
)

func (e *Engine) settleRun(ctx context.Context, box *outbox, record run.Run, now time.Time) error {
	current, err := e.deps.Runs.Get(ctx, record.ID)
	if err != nil {
		return err
	}
	if atRest(current.Status) {
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
	current.Stats = statsOf(counts, spend)

	switch {
	case overBudget(current, spend):
		return e.pauseOverBudget(ctx, box, current, spend, now)
	case advanceable(counts):
		return e.deps.Runs.Update(ctx, current)
	case counts[run.StatusPaused] > 0:
		return e.pauseHeld(ctx, box, current)
	default:
		return e.conclude(ctx, box, current, now)
	}
}

func statsOf(counts map[run.Status]int, spend llm.Spend) run.Stats {
	return run.Stats{
		Items:  total(counts),
		Done:   counts[run.StatusCompleted],
		Failed: counts[run.StatusFailed],
		Tokens: spend.Usage.Total,
		USD:    spend.USD,
	}
}

func (e *Engine) pauseOverBudget(ctx context.Context, box *outbox, current run.Run, spend llm.Spend, now time.Time) error {
	if _, err := e.deps.Items.StopAll(ctx, current.ID, pausable, run.StatusPaused, run.PauseBudgetExceeded, now); err != nil {
		return err
	}

	box.add(ctx, current.ID, events.RunBudgetExceeded, events.RunBudgetExceededPayload{
		RunID: current.ID, SpentUSD: spend.USD, BudgetUSD: current.Budget.MaxUSD,
		SpentTokens: spend.Usage.Total, BudgetTokens: current.Budget.MaxTokens,
	})
	return e.pauseRun(ctx, box, current, run.PauseBudgetExceeded)
}

func (e *Engine) pauseHeld(ctx context.Context, box *outbox, current run.Run) error {
	reason, err := e.heldFor(ctx, current.ID)
	if err != nil {
		return err
	}
	return e.pauseRun(ctx, box, current, reason)
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

func (e *Engine) conclude(ctx context.Context, box *outbox, current run.Run, now time.Time) error {
	current.FinishedAt = &now

	if current.Stats.Done == 0 && current.Stats.Failed > 0 {
		current.Status = run.StatusFailed
		if err := e.deps.Runs.Update(ctx, current); err != nil {
			return err
		}
		box.add(ctx, current.ID, events.RunFailed, events.RunFailedPayload{
			RunID: current.ID, Code: run.CodeUnclassified, Message: "every item of the run failed",
		})
		return nil
	}

	current.Status = run.StatusCompleted
	if err := e.deps.Runs.Update(ctx, current); err != nil {
		return err
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

	if _, err = e.deps.Items.StopAll(ctx, current.ID, stoppable, run.StatusCancelled, "", now); err != nil {
		return err
	}

	current.Status = run.StatusFailed
	current.Error = "the run passed its deadline before every item finished"
	current.FinishedAt = &now
	if updateErr := e.deps.Runs.Update(ctx, current); updateErr != nil {
		return updateErr
	}

	box.add(ctx, current.ID, events.RunFailed, events.RunFailedPayload{
		RunID: current.ID, Code: run.CodeDeadlineExceeded, Message: current.Error,
	})
	return nil
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
