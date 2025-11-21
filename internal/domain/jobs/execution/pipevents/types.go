package pipevents

import (
	"time"

	"github.com/davidmovas/postulator/internal/infra/events"
)

type EventType = events.EventType

const (
	EventPipelineStarted   EventType = "pipeline.started"
	EventPipelineCompleted EventType = "pipeline.completed"
	EventPipelineFailed    EventType = "pipeline.failed"
	EventPipelinePaused    EventType = "pipeline.paused"

	EventStepStarted   EventType = "step.started"
	EventStepCompleted EventType = "step.completed"
	EventStepFailed    EventType = "step.failed"
	EventStepRetrying  EventType = "step.retrying"

	EventStateChanged EventType = "state.changed"

	EventValidationStarted   EventType = "validation.started"
	EventValidationCompleted EventType = "validation.completed"
	EventValidationFailed    EventType = "validation.failed"
	EventValidationRequired  EventType = "validation.required"

	EventTopicSelected     EventType = "topic.selected"
	EventCategorySelected  EventType = "category.selected"
	EventNoTopicsAvailable EventType = "no_topics.available"

	EventGenerationStarted   EventType = "generation.started"
	EventGenerationCompleted EventType = "generation.completed"
	EventGenerationFailed    EventType = "generation.failed"

	EventPublishingStarted   EventType = "publishing.started"
	EventPublishingCompleted EventType = "publishing.completed"
	EventPublishingFailed    EventType = "publishing.failed"
	EventArticlePublished    EventType = "article.published"

	EventStatsRecorded  EventType = "stats.recorded"
	EventTokensConsumed EventType = "tokens.consumed"
	EventCostIncurred   EventType = "cost.incurred"
)

type Event struct {
	events.Event
	JobID int64
}

type PipelineStartedEvent struct {
	JobID   int64
	JobName string
}

type PipelineCompletedEvent struct {
	JobID       int64
	JobName     string
	Duration    time.Duration
	ArticleID   int64
	ExecutionID int64
}

type PipelineFailedEvent struct {
	JobID       int64
	JobName     string
	Duration    time.Duration
	ErrorCode   string
	ErrorMsg    string
	FailedStep  string
	FailedState string
}

type PipelinePausedEvent struct {
	JobID    int64
	JobName  string
	Reason   string
	PausedAt string
}

type StepStartedEvent struct {
	JobID    int64
	StepName string
	State    string
}

type StepCompletedEvent struct {
	JobID    int64
	StepName string
	Duration time.Duration
	State    string
}

type StepFailedEvent struct {
	JobID     int64
	StepName  string
	Duration  time.Duration
	ErrorCode string
	ErrorMsg  string
	State     string
}

type StepRetryingEvent struct {
	JobID      int64
	StepName   string
	Attempt    int
	MaxRetries int
	Reason     string
}

type StateChangedEvent struct {
	JobID     int64
	FromState string
	ToState   string
	Reason    string
}

type ValidationRequiredEvent struct {
	JobID        int64
	ExecutionID  int64
	ArticleID    int64
	ArticleTitle string
}

type TopicSelectedEvent struct {
	JobID            int64
	TopicID          int64
	TopicTitle       string
	OriginalTopicID  int64
	VariationTopicID int64
}

type CategorySelectedEvent struct {
	JobID      int64
	Categories []string
	Strategy   string
}

type NoTopicsAvailableEvent struct {
	JobID    int64
	JobName  string
	Strategy string
}

type GenerationCompletedEvent struct {
	JobID          int64
	ExecutionID    int64
	Title          string
	ContentLength  int
	GenerationTime time.Duration
	TokensUsed     int
	CostUSD        float64
}

type ArticlePublishedEvent struct {
	JobID     int64
	ArticleID int64
	SiteID    int64
	Title     string
	WPPostID  int
	WPPostURL string
	Status    string
}

type StatsRecordedEvent struct {
	JobID       int64
	SiteID      int64
	CategoryIDs []int64
	WordCount   int
}

type TokensConsumedEvent struct {
	JobID       int64
	ExecutionID int64
	ProviderID  int64
	TokensUsed  int
	CostUSD     float64
}
