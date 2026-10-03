package runtime

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (e *Engine) Enqueue(ctx context.Context, record run.Run) (run.Run, error) {
	if err := run.ValidateRecipe(e.registry, record.Recipe); err != nil {
		return run.Run{}, err
	}

	enabled := run.Enabled(record.Recipe)
	now := e.now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.DeadlineAt.IsZero() {
		record.DeadlineAt = record.CreatedAt.Add(e.cfg.RunDeadline)
	}
	record.Stats.Items = len(record.Targets)
	pages := e.targetPages(ctx, record.Targets)
	record.Targets = inWorkingOrder(record.Kind, record.Targets, pages)

	validated, err := run.NewRun(record)
	if err != nil {
		return run.Run{}, err
	}
	queueBehindParents := validated.Kind != run.KindRevert && publishes(enabled)

	err = e.transact(ctx, func(c context.Context, box *outbox) error {
		if insertErr := e.deps.Runs.Insert(c, validated); insertErr != nil {
			return insertErr
		}
		if insertErr := e.insertItems(c, validated, enabled[0].Name, pages, queueBehindParents, now); insertErr != nil {
			return insertErr
		}
		box.add(c, validated.ID, events.RunQueued, events.RunQueuedPayload{
			RunID: validated.ID, Kind: string(validated.Kind), Items: len(validated.Targets),
		})
		return nil
	})
	if err != nil {
		return run.Run{}, err
	}

	e.nudge()
	return validated, nil
}

func (e *Engine) insertItems(ctx context.Context, record run.Run, first string, pages map[string]pagemap.Page,
	queueBehindParents bool, now time.Time) error {
	itemIDs := make(map[string]string, len(record.Targets))
	for seq, targetID := range record.Targets {
		item := run.Item{
			ID: id.New(), RunID: record.ID, SiteID: record.SiteID, TargetID: targetID,
			CurrentStep: first, Seq: seq, CreatedAt: now, UpdatedAt: now,
		}
		if queueBehindParents {
			item.BlockedBy = blockerOf(pages[targetID], pages, itemIDs)
		}
		item, err := run.NewItem(item)
		if err != nil {
			return err
		}
		if insertErr := e.deps.Items.Insert(ctx, item); insertErr != nil {
			return insertErr
		}
		itemIDs[targetID] = item.ID
	}
	return nil
}

func (e *Engine) targetPages(ctx context.Context, targets []string) map[string]pagemap.Page {
	pages := make(map[string]pagemap.Page, len(targets))
	if e.deps.Pages == nil {
		return pages
	}
	for _, targetID := range targets {
		page, err := e.deps.Pages.Get(ctx, targetID)
		if err != nil {
			continue
		}
		pages[targetID] = page
	}
	return pages
}

func inWorkingOrder(kind run.Kind, targets []string, pages map[string]pagemap.Page) []string {
	ordered := ancestorsFirst(targets, pages)
	if kind != run.KindRevert {
		return ordered
	}
	reversed := slices.Clone(ordered)
	slices.Reverse(reversed)
	return reversed
}

func ancestorsFirst(targets []string, pages map[string]pagemap.Page) []string {
	if len(targets) < 2 {
		return targets
	}

	ordered := slices.Clone(targets)
	slices.SortStableFunc(ordered, func(a, b string) int {
		left, leftKnown := pages[a]
		right, rightKnown := pages[b]
		switch {
		case !leftKnown && !rightKnown:
			return 0
		case !leftKnown:
			return 1
		case !rightKnown:
			return -1
		}
		if depth := strings.Count(left.Path, "/") - strings.Count(right.Path, "/"); depth != 0 {
			return depth
		}
		return strings.Compare(left.Path, right.Path)
	})
	return ordered
}

func publishes(recipe []template.StepSpec) bool {
	for i := range recipe {
		if recipe[i].Name == string(run.StepPublish) {
			return true
		}
	}
	return false
}

func blockerOf(page pagemap.Page, pages map[string]pagemap.Page, itemIDs map[string]string) string {
	if page.WPType != pagemap.WPPage || page.ParentPageID == nil {
		return ""
	}
	parent, targeted := pages[*page.ParentPageID]
	if !targeted || parent.WPID != nil {
		return ""
	}
	return itemIDs[parent.ID]
}
