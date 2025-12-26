package generation

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/infra/events"
)

const (
	EventTaskStarted   events.EventType = "pagegeneration.task.started"
	EventTaskProgress  events.EventType = "pagegeneration.task.progress"
	EventTaskPaused    events.EventType = "pagegeneration.task.paused"
	EventTaskResumed   events.EventType = "pagegeneration.task.resumed"
	EventTaskCompleted events.EventType = "pagegeneration.task.completed"
	EventTaskFailed    events.EventType = "pagegeneration.task.failed"
	EventTaskCancelled events.EventType = "pagegeneration.task.cancelled"

	EventNodeQueued     events.EventType = "pagegeneration.node.queued"
	EventNodeGenerating events.EventType = "pagegeneration.node.generating"
	EventNodeGenerated  events.EventType = "pagegeneration.node.generated"
	EventNodePublishing events.EventType = "pagegeneration.node.publishing"
	EventNodeCompleted  events.EventType = "pagegeneration.node.completed"
	EventNodeFailed     events.EventType = "pagegeneration.node.failed"
	EventNodeSkipped    events.EventType = "pagegeneration.node.skipped"

	// Linking phase events
	EventLinkingPhaseStarted   events.EventType = "pagegeneration.linking.started"
	EventLinkingPhaseCompleted events.EventType = "pagegeneration.linking.completed"
)

type TaskStartedEvent struct {
	TaskID     string
	SitemapID  int64
	TotalNodes int
}

type TaskProgressEvent struct {
	TaskID         string
	ProcessedNodes int
	TotalNodes     int
	FailedNodes    int
	SkippedNodes   int
	CurrentNode    *NodeInfo
}

type NodeInfo struct {
	NodeID int64
	Title  string
	Path   string
}

type TaskPausedEvent struct {
	TaskID         string
	ProcessedNodes int
	TotalNodes     int
}

type TaskResumedEvent struct {
	TaskID         string
	RemainingNodes int
}

type TaskCompletedEvent struct {
	TaskID         string
	ProcessedNodes int
	FailedNodes    int
	SkippedNodes   int
	TotalNodes     int
	DurationMs     int64
}

type TaskFailedEvent struct {
	TaskID string
	Error  string
}

type TaskCancelledEvent struct {
	TaskID         string
	ProcessedNodes int
	TotalNodes     int
}

type NodeQueuedEvent struct {
	TaskID string
	NodeID int64
	Title  string
	Path   string
}

type NodeGeneratingEvent struct {
	TaskID string
	NodeID int64
	Title  string
}

type NodeGeneratedEvent struct {
	TaskID     string
	NodeID     int64
	Title      string
	TokensUsed int
	DurationMs int64
}

type NodePublishingEvent struct {
	TaskID string
	NodeID int64
	Title  string
}

type NodeCompletedEvent struct {
	TaskID    string
	NodeID    int64
	Title     string
	ArticleID int64
	WPPageID  int
	WPPageURL string
}

type NodeFailedEvent struct {
	TaskID string
	NodeID int64
	Title  string
	Error  string
}

type NodeSkippedEvent struct {
	TaskID string
	NodeID int64
	Title  string
	Reason string
}

type EventEmitter struct {
	eventBus *events.EventBus
}

func NewEventEmitter(eventBus *events.EventBus) *EventEmitter {
	return &EventEmitter{eventBus: eventBus}
}

func (e *EventEmitter) EmitTaskStarted(ctx context.Context, taskID string, sitemapID int64, totalNodes int) {
	e.eventBus.Publish(ctx, events.NewEvent(EventTaskStarted, TaskStartedEvent{
		TaskID:     taskID,
		SitemapID:  sitemapID,
		TotalNodes: totalNodes,
	}))
}

func (e *EventEmitter) EmitTaskProgress(ctx context.Context, taskID string, processed, total, failed, skipped int, currentNode *NodeInfo) {
	e.eventBus.Publish(ctx, events.NewEvent(EventTaskProgress, TaskProgressEvent{
		TaskID:         taskID,
		ProcessedNodes: processed,
		TotalNodes:     total,
		FailedNodes:    failed,
		SkippedNodes:   skipped,
		CurrentNode:    currentNode,
	}))
}

func (e *EventEmitter) EmitTaskPaused(ctx context.Context, taskID string, processed, total int) {
	e.eventBus.Publish(ctx, events.NewEvent(EventTaskPaused, TaskPausedEvent{
		TaskID:         taskID,
		ProcessedNodes: processed,
		TotalNodes:     total,
	}))
}

func (e *EventEmitter) EmitTaskResumed(ctx context.Context, taskID string, remaining int) {
	e.eventBus.Publish(ctx, events.NewEvent(EventTaskResumed, TaskResumedEvent{
		TaskID:         taskID,
		RemainingNodes: remaining,
	}))
}

func (e *EventEmitter) EmitTaskCompleted(ctx context.Context, taskID string, processed, failed, skipped, total int, startTime time.Time) {
	e.eventBus.Publish(ctx, events.NewEvent(EventTaskCompleted, TaskCompletedEvent{
		TaskID:         taskID,
		ProcessedNodes: processed,
		FailedNodes:    failed,
		SkippedNodes:   skipped,
		TotalNodes:     total,
		DurationMs:     time.Since(startTime).Milliseconds(),
	}))
}

func (e *EventEmitter) EmitTaskFailed(ctx context.Context, taskID, errMsg string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventTaskFailed, TaskFailedEvent{
		TaskID: taskID,
		Error:  errMsg,
	}))
}

func (e *EventEmitter) EmitTaskCancelled(ctx context.Context, taskID string, processed, total int) {
	e.eventBus.Publish(ctx, events.NewEvent(EventTaskCancelled, TaskCancelledEvent{
		TaskID:         taskID,
		ProcessedNodes: processed,
		TotalNodes:     total,
	}))
}

func (e *EventEmitter) EmitNodeQueued(ctx context.Context, taskID string, nodeID int64, title, path string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventNodeQueued, NodeQueuedEvent{
		TaskID: taskID,
		NodeID: nodeID,
		Title:  title,
		Path:   path,
	}))
}

func (e *EventEmitter) EmitNodeGenerating(ctx context.Context, taskID string, nodeID int64, title string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventNodeGenerating, NodeGeneratingEvent{
		TaskID: taskID,
		NodeID: nodeID,
		Title:  title,
	}))
}

func (e *EventEmitter) EmitNodeGenerated(ctx context.Context, taskID string, nodeID int64, title string, tokensUsed int, startTime time.Time) {
	e.eventBus.Publish(ctx, events.NewEvent(EventNodeGenerated, NodeGeneratedEvent{
		TaskID:     taskID,
		NodeID:     nodeID,
		Title:      title,
		TokensUsed: tokensUsed,
		DurationMs: time.Since(startTime).Milliseconds(),
	}))
}

func (e *EventEmitter) EmitNodePublishing(ctx context.Context, taskID string, nodeID int64, title string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventNodePublishing, NodePublishingEvent{
		TaskID: taskID,
		NodeID: nodeID,
		Title:  title,
	}))
}

func (e *EventEmitter) EmitNodeCompleted(ctx context.Context, taskID string, nodeID int64, title string, articleID int64, wpPageID int, wpURL string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventNodeCompleted, NodeCompletedEvent{
		TaskID:    taskID,
		NodeID:    nodeID,
		Title:     title,
		ArticleID: articleID,
		WPPageID:  wpPageID,
		WPPageURL: wpURL,
	}))
}

func (e *EventEmitter) EmitNodeFailed(ctx context.Context, taskID string, nodeID int64, title, errMsg string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventNodeFailed, NodeFailedEvent{
		TaskID: taskID,
		NodeID: nodeID,
		Title:  title,
		Error:  errMsg,
	}))
}

func (e *EventEmitter) EmitNodeSkipped(ctx context.Context, taskID string, nodeID int64, title, reason string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventNodeSkipped, NodeSkippedEvent{
		TaskID: taskID,
		NodeID: nodeID,
		Title:  title,
		Reason: reason,
	}))
}

// Linking phase events

type LinkingPhaseStartedEvent struct {
	TaskID string
	Phase  string // "suggesting" or "applying"
}

type LinkingPhaseCompletedEvent struct {
	TaskID       string
	Phase        string
	LinksCreated int
	LinksApplied int
	LinksFailed  int
}

func (e *EventEmitter) EmitLinkingPhaseStarted(ctx context.Context, taskID, phase string) {
	e.eventBus.Publish(ctx, events.NewEvent(EventLinkingPhaseStarted, LinkingPhaseStartedEvent{
		TaskID: taskID,
		Phase:  phase,
	}))
}

func (e *EventEmitter) EmitLinkingPhaseCompleted(ctx context.Context, taskID, phase string, created, applied, failed int) {
	e.eventBus.Publish(ctx, events.NewEvent(EventLinkingPhaseCompleted, LinkingPhaseCompletedEvent{
		TaskID:       taskID,
		Phase:        phase,
		LinksCreated: created,
		LinksApplied: applied,
		LinksFailed:  failed,
	}))
}
