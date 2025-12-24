package linking

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/infra/events"
)

const (
	// Apply events
	EventApplyStarted   events.EventType = "linking.apply.started"
	EventApplyProgress  events.EventType = "linking.apply.progress"
	EventApplyCompleted events.EventType = "linking.apply.completed"
	EventApplyFailed    events.EventType = "linking.apply.failed"

	EventPageProcessing events.EventType = "linking.page.processing"
	EventPageCompleted  events.EventType = "linking.page.completed"
	EventPageFailed     events.EventType = "linking.page.failed"

	// Suggest events
	EventSuggestStarted   events.EventType = "linking.suggest.started"
	EventSuggestProgress  events.EventType = "linking.suggest.progress"
	EventSuggestCompleted events.EventType = "linking.suggest.completed"
	EventSuggestFailed    events.EventType = "linking.suggest.failed"
)

type ApplyStartedEvent struct {
	TaskID     string
	TotalLinks int
	TotalPages int // Number of unique source pages
}

type ApplyProgressEvent struct {
	TaskID         string
	ProcessedPages int
	TotalPages     int
	AppliedLinks   int
	FailedLinks    int
	CurrentPage    *PageInfo
}

type PageInfo struct {
	NodeID int64
	Title  string
	Path   string
}

type ApplyCompletedEvent struct {
	TaskID       string
	TotalLinks   int
	AppliedLinks int
	FailedLinks  int
	DurationMs   int64
}

type ApplyFailedEvent struct {
	TaskID string
	Error  string
}

type PageProcessingEvent struct {
	TaskID    string
	NodeID    int64
	Title     string
	LinkCount int
}

type PageCompletedEvent struct {
	TaskID       string
	NodeID       int64
	Title        string
	AppliedLinks int
	FailedLinks  int
}

type PageFailedEvent struct {
	TaskID string
	NodeID int64
	Title  string
	Error  string
}

type ApplyEventEmitter struct {
	eventBus *events.EventBus
}

func NewApplyEventEmitter(eventBus *events.EventBus) *ApplyEventEmitter {
	return &ApplyEventEmitter{eventBus: eventBus}
}

func (e *ApplyEventEmitter) EmitApplyStarted(ctx context.Context, taskID string, totalLinks, totalPages int) {
	e.eventBus.Publish(ctx, events.NewEvent(EventApplyStarted, ApplyStartedEvent{
		TaskID:     taskID,
		TotalLinks: totalLinks,
		TotalPages: totalPages,
	}))
}

func (e *ApplyEventEmitter) EmitApplyProgress(ctx context.Context, taskID string, processedPages, totalPages, appliedLinks, failedLinks int, currentPage *PageInfo) {
	e.eventBus.Publish(ctx, events.NewEvent(EventApplyProgress, ApplyProgressEvent{
		TaskID:         taskID,
		ProcessedPages: processedPages,
		TotalPages:     totalPages,
		AppliedLinks:   appliedLinks,
		FailedLinks:    failedLinks,
		CurrentPage:    currentPage,
	}))
}

func (e *ApplyEventEmitter) EmitApplyCompleted(ctx context.Context, taskID string, totalLinks, appliedLinks, failedLinks int, startTime time.Time) {
	e.eventBus.Publish(ctx, events.NewEvent(EventApplyCompleted, ApplyCompletedEvent{
		TaskID:       taskID,
		TotalLinks:   totalLinks,
		AppliedLinks: appliedLinks,
		FailedLinks:  failedLinks,
		DurationMs:   time.Since(startTime).Milliseconds(),
	}))
}

func (e *ApplyEventEmitter) EmitApplyFailed(ctx context.Context, taskID, errMsg string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventApplyFailed, ApplyFailedEvent{
		TaskID: taskID,
		Error:  errMsg,
	}))
}

func (e *ApplyEventEmitter) EmitPageProcessing(ctx context.Context, taskID string, nodeID int64, title string, linkCount int) {
	e.eventBus.Publish(ctx, events.NewEvent(EventPageProcessing, PageProcessingEvent{
		TaskID:    taskID,
		NodeID:    nodeID,
		Title:     title,
		LinkCount: linkCount,
	}))
}

func (e *ApplyEventEmitter) EmitPageCompleted(ctx context.Context, taskID string, nodeID int64, title string, appliedLinks, failedLinks int) {
	e.eventBus.Publish(ctx, events.NewEvent(EventPageCompleted, PageCompletedEvent{
		TaskID:       taskID,
		NodeID:       nodeID,
		Title:        title,
		AppliedLinks: appliedLinks,
		FailedLinks:  failedLinks,
	}))
}

func (e *ApplyEventEmitter) EmitPageFailed(ctx context.Context, taskID string, nodeID int64, title, errMsg string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventPageFailed, PageFailedEvent{
		TaskID: taskID,
		NodeID: nodeID,
		Title:  title,
		Error:  errMsg,
	}))
}

// =========================================================================
// Suggest Events
// =========================================================================

type SuggestStartedEvent struct {
	TaskID       string
	TotalNodes   int
	TotalBatches int
}

type SuggestProgressEvent struct {
	TaskID           string
	CurrentBatch     int
	TotalBatches     int
	ProcessedNodes   int
	TotalNodes       int
	LinksCreated     int
	CurrentBatchSize int
}

type SuggestCompletedEvent struct {
	TaskID       string
	TotalNodes   int
	LinksCreated int
	DurationMs   int64
}

type SuggestFailedEvent struct {
	TaskID string
	Error  string
}

type SuggestEventEmitter struct {
	eventBus *events.EventBus
}

func NewSuggestEventEmitter(eventBus *events.EventBus) *SuggestEventEmitter {
	return &SuggestEventEmitter{eventBus: eventBus}
}

func (e *SuggestEventEmitter) EmitSuggestStarted(ctx context.Context, taskID string, totalNodes, totalBatches int) {
	e.eventBus.Publish(ctx, events.NewEvent(EventSuggestStarted, SuggestStartedEvent{
		TaskID:       taskID,
		TotalNodes:   totalNodes,
		TotalBatches: totalBatches,
	}))
}

func (e *SuggestEventEmitter) EmitSuggestProgress(ctx context.Context, taskID string, currentBatch, totalBatches, processedNodes, totalNodes, linksCreated, currentBatchSize int) {
	e.eventBus.Publish(ctx, events.NewEvent(EventSuggestProgress, SuggestProgressEvent{
		TaskID:           taskID,
		CurrentBatch:     currentBatch,
		TotalBatches:     totalBatches,
		ProcessedNodes:   processedNodes,
		TotalNodes:       totalNodes,
		LinksCreated:     linksCreated,
		CurrentBatchSize: currentBatchSize,
	}))
}

func (e *SuggestEventEmitter) EmitSuggestCompleted(ctx context.Context, taskID string, totalNodes, linksCreated int, startTime time.Time) {
	e.eventBus.Publish(ctx, events.NewEvent(EventSuggestCompleted, SuggestCompletedEvent{
		TaskID:       taskID,
		TotalNodes:   totalNodes,
		LinksCreated: linksCreated,
		DurationMs:   time.Since(startTime).Milliseconds(),
	}))
}

func (e *SuggestEventEmitter) EmitSuggestFailed(ctx context.Context, taskID, errMsg string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventSuggestFailed, SuggestFailedEvent{
		TaskID: taskID,
		Error:  errMsg,
	}))
}
