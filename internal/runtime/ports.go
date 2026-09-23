package runtime

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
)

type runStore interface {
	Insert(ctx context.Context, record run.Run) error
	Get(ctx context.Context, id string) (run.Run, error)
	Update(ctx context.Context, record run.Run) error
	Active(ctx context.Context) ([]run.Run, error)
	PastDeadline(ctx context.Context, now time.Time, limit int) ([]run.Run, error)
}

type itemStore interface {
	Insert(ctx context.Context, item run.Item) error
	Get(ctx context.Context, id string) (run.Item, error)
	Claim(ctx context.Context, id string, expectSeq int64, leaseUntil, now time.Time) (run.Item, error)
	Persist(ctx context.Context, item run.Item, expectSeq int64) (bool, error)
	Requeue(ctx context.Context, id string, expectSeq int64, from run.Status, now time.Time) (bool, error)
	ByRun(ctx context.Context, runID string) ([]run.Item, error)
	Due(ctx context.Context, now time.Time, limit int) ([]run.Item, error)
	Stalled(ctx context.Context, now time.Time, limit int) ([]run.Item, error)
	Runnable(ctx context.Context, now time.Time, limit int) ([]run.Item, error)
	Counts(ctx context.Context, runID string) (map[run.Status]int, error)
	StopAll(ctx context.Context, runID string, from []run.Status, to run.Status, reason run.PauseReason, now time.Time) (int64, error)
	ResumeAll(ctx context.Context, runID string, now time.Time) (int64, error)
}

type artifactStore interface {
	ReplaceStep(ctx context.Context, itemID, step string, artifacts []run.Artifact) error
	ByItem(ctx context.Context, itemID string) ([]run.Artifact, error)
	PurgePublishedBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type execStore interface {
	Insert(ctx context.Context, exec run.StepExec) error
	Done(ctx context.Context, itemID, step, inputHash string) (run.StepExec, error)
	CountByStep(ctx context.Context, itemID, step string) (int, error)
}

type eventStore interface {
	Append(ctx context.Context, runID, eventType string, at time.Time, payload []byte) (run.Event, error)
	PurgeTerminalBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type pageReader interface {
	Get(ctx context.Context, id string) (pagemap.Page, error)
}

type specResolver interface {
	ResolveForPage(ctx context.Context, req templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error)
}

type spendReader interface {
	SumByRun(ctx context.Context, runID string) (llm.Spend, error)
}

type modelCatalog interface {
	Lookup(ctx context.Context, ref llm.ModelRef) (llm.ModelInfo, error)
}

type profileResolver interface {
	Resolve(ctx context.Context, siteID string, role llm.Role, templateProfiles map[llm.Role]llm.ModelRef) (llm.ModelRef, error)
}

type publisher interface {
	PublishRun(runID string, seq int64, eventType events.Type, payload any) error
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
