package generation

import (
	"sync"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type NodeStatus string

const (
	NodeStatusPending    NodeStatus = "pending"
	NodeStatusGenerating NodeStatus = "generating"
	NodeStatusPublishing NodeStatus = "publishing"
	NodeStatusCompleted  NodeStatus = "completed"
	NodeStatusFailed     NodeStatus = "failed"
	NodeStatusSkipped    NodeStatus = "skipped"
)

type PublishAs string

const (
	PublishAsDraft   PublishAs = "draft"
	PublishAsPending PublishAs = "pending"
	PublishAsPublish PublishAs = "publish"
)

type Task struct {
	ID             string
	SitemapID      int64
	SiteID         int64
	ProviderID     int64
	PromptID       *int64
	PublishAs      PublishAs
	Nodes          []*TaskNode
	TotalNodes     int
	ProcessedNodes int
	FailedNodes    int
	SkippedNodes   int
	Status         TaskStatus
	StartedAt      time.Time
	CompletedAt    *time.Time
	Error          *string
	mu             sync.RWMutex
}

type TaskNode struct {
	NodeID         int64
	Title          string
	Slug           string
	Path           string
	Keywords       []string
	Depth          int
	ParentNodeID   *int64
	ParentWPPageID *int
	Status         NodeStatus
	ArticleID      *int64
	WPPageID       *int
	WPURL          *string
	Error          *string
	StartedAt      *time.Time
	CompletedAt    *time.Time
}

type PageContent struct {
	Title           string
	Content         string
	Excerpt         string
	MetaDescription string
	InputTokens     int
	OutputTokens    int
	CostUSD         float64
}

type WritingStyle string

const (
	WritingStyleProfessional WritingStyle = "professional"
	WritingStyleCasual       WritingStyle = "casual"
	WritingStyleFormal       WritingStyle = "formal"
	WritingStyleFriendly     WritingStyle = "friendly"
	WritingStyleTechnical    WritingStyle = "technical"
)

type ContentTone string

const (
	ContentToneInformative ContentTone = "informative"
	ContentTonePersuasive  ContentTone = "persuasive"
	ContentToneEducational ContentTone = "educational"
	ContentToneEngaging    ContentTone = "engaging"
	ContentToneAuthoritative ContentTone = "authoritative"
)

type ContentSettings struct {
	WordCount    string       `json:"wordCount"`    // "1000" or "800-1200"
	WritingStyle WritingStyle `json:"writingStyle"` // professional, casual, etc.
	ContentTone  ContentTone  `json:"contentTone"`  // informative, persuasive, etc.
	CustomInstructions string  `json:"customInstructions"` // Additional instructions
}

type GenerationConfig struct {
	SitemapID       int64
	SiteID          int64
	NodeIDs         []int64
	ProviderID      int64
	PromptID        *int64
	PublishAs       PublishAs
	Placeholders    map[string]string
	MaxConcurrency  int
	ContentSettings *ContentSettings
}

func (t *Task) IncrementProcessed() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ProcessedNodes++
}

func (t *Task) IncrementFailed() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.FailedNodes++
}

func (t *Task) IncrementSkipped() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.SkippedNodes++
}

func (t *Task) SetStatus(status TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status
}

func (t *Task) GetStatus() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

func (t *Task) SetError(err string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Error = &err
}

func (t *Task) GetProgress() (processed, failed, total int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ProcessedNodes, t.FailedNodes, t.TotalNodes
}

func (t *Task) Complete() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	t.CompletedAt = &now
	if t.FailedNodes == t.TotalNodes {
		t.Status = TaskStatusFailed
	} else {
		t.Status = TaskStatusCompleted
	}
}

func (tn *TaskNode) SetStatus(status NodeStatus) {
	tn.Status = status
}

func (tn *TaskNode) SetError(err string) {
	tn.Error = &err
}

func (tn *TaskNode) MarkStarted() {
	now := time.Now()
	tn.StartedAt = &now
}

func (tn *TaskNode) MarkCompleted(articleID int64, wpPageID int, wpURL string) {
	now := time.Now()
	tn.CompletedAt = &now
	tn.ArticleID = &articleID
	tn.WPPageID = &wpPageID
	tn.WPURL = &wpURL
	tn.Status = NodeStatusCompleted
}

func (tn *TaskNode) MarkFailed(err string) {
	now := time.Now()
	tn.CompletedAt = &now
	tn.Error = &err
	tn.Status = NodeStatusFailed
}
