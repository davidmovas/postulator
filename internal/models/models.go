package models

import "time"

type Site struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`             // Название для удобства
	URL       string    `json:"url" db:"url"`               // https://example.com
	Username  string    `json:"username" db:"username"`     // WP логин
	Password  string    `json:"password" db:"password"`     // Зашифрованный пароль/app password
	IsActive  bool      `json:"is_active" db:"is_active"`   // Включен ли сайт
	LastCheck time.Time `json:"last_check" db:"last_check"` // Последняя проверка доступности
	Status    string    `json:"status" db:"status"`         // "connected", "error", "pending"
	Strategy  string    `json:"strategy" db:"strategy"`     // Topic selection strategy: "unique", "round_robin", "random"
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Topic struct {
	ID        int64     `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`       // Название темы
	Keywords  string    `json:"keywords" db:"keywords"` // Ключевые слова через запятую
	Category  string    `json:"category" db:"category"` // Категория на WP
	Tags      string    `json:"tags" db:"tags"`         // Теги через запятую
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type SiteTopic struct {
	ID            int64      `json:"id" db:"id"`
	SiteID        int64      `json:"site_id" db:"site_id"`
	TopicID       int64      `json:"topic_id" db:"topic_id"`
	Priority      int        `json:"priority" db:"priority"`               // Priority for topic selection (1-10)
	LastUsedAt    *time.Time `json:"last_used_at" db:"last_used_at"`       // When this topic was last used for this site
	UsageCount    int        `json:"usage_count" db:"usage_count"`         // How many times this topic was used
	RoundRobinPos int        `json:"round_robin_pos" db:"round_robin_pos"` // Position in round-robin cycle
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type Schedule struct {
	ID        int64     `json:"id" db:"id"`
	SiteID    int64     `json:"site_id" db:"site_id"`
	CronExpr  string    `json:"cron_expr" db:"cron_expr"` // "0 */6 * * *" - каждые 6 часов
	IsActive  bool      `json:"is_active" db:"is_active"`
	LastRun   time.Time `json:"last_run" db:"last_run"`
	NextRun   time.Time `json:"next_run" db:"next_run"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Article struct {
	ID          int64     `json:"id" db:"id"`
	SiteID      int64     `json:"site_id" db:"site_id"`
	TopicID     int64     `json:"topic_id" db:"topic_id"`
	Title       string    `json:"title" db:"title"`
	Content     string    `json:"content" db:"content"`
	Excerpt     string    `json:"excerpt" db:"excerpt"` // Краткое описание статьи
	Keywords    string    `json:"keywords" db:"keywords"`
	Tags        string    `json:"tags" db:"tags"`
	Category    string    `json:"category" db:"category"`
	Status      string    `json:"status" db:"status"`             // "generated", "published", "failed"
	WordPressID int64     `json:"wordpress_id" db:"wordpress_id"` // ID в WordPress после публикации
	GPTModel    string    `json:"gpt_model" db:"gpt_model"`       // Какая модель использовалась
	Tokens      int       `json:"tokens" db:"tokens"`             // Потрачено токенов
	Slug        string    `json:"slug" db:"slug"`                 // URL-friendly slug, unique per site
	Outline     string    `json:"outline" db:"outline"`           // JSON outline structure
	ErrorMsg    string    `json:"error_msg" db:"error_msg"`       // Last error during generation/publishing
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	PublishedAt time.Time `json:"published_at" db:"published_at"`
}

type PostingJob struct {
	ID          int64     `json:"id" db:"id"`
	Type        string    `json:"type" db:"type"` // "scheduled", "manual"
	SiteID      int64     `json:"site_id" db:"site_id"`
	ArticleID   int64     `json:"article_id" db:"article_id"`
	Status      string    `json:"status" db:"status"`     // "pending", "running", "completed", "failed"
	Progress    int       `json:"progress" db:"progress"` // 0-100%
	ErrorMsg    string    `json:"error_msg" db:"error_msg"`
	StartedAt   time.Time `json:"started_at" db:"started_at"`
	CompletedAt time.Time `json:"completed_at" db:"completed_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Prompt struct {
	ID           int64     `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`                   // User-friendly name for the prompt
	SystemPrompt string    `json:"system_prompt" db:"system_prompt"` // System prompt content with placeholders
	UserPrompt   string    `json:"user_prompt" db:"user_prompt"`     // User prompt content with placeholders
	IsDefault    bool      `json:"is_default" db:"is_default"`       // Whether this is the default prompt
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type SitePrompt struct {
	ID        int64     `json:"id" db:"id"`
	SiteID    int64     `json:"site_id" db:"site_id"`
	PromptID  int64     `json:"prompt_id" db:"prompt_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Setting struct {
	ID        int64     `json:"id" db:"id"`
	Key       string    `json:"key" db:"key"`           // Setting key (unique)
	Value     string    `json:"value" db:"value"`       // Setting value
	Type      string    `json:"type" db:"type"`         // "string", "int", "bool", "json"
	Category  string    `json:"category" db:"category"` // Category for grouping settings
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TopicUsage tracks when and how topics are used for article generation
type TopicUsage struct {
	ID        int64     `json:"id" db:"id"`
	SiteID    int64     `json:"site_id" db:"site_id"`
	TopicID   int64     `json:"topic_id" db:"topic_id"`
	ArticleID int64     `json:"article_id" db:"article_id"`
	Strategy  string    `json:"strategy" db:"strategy"` // Strategy used when selecting this topic
	UsedAt    time.Time `json:"used_at" db:"used_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TopicStats provides statistics about topic usage for a site
type TopicStats struct {
	SiteID             int64      `json:"site_id"`
	TotalTopics        int        `json:"total_topics"`         // Total topics assigned to site
	ActiveTopics       int        `json:"active_topics"`        // Active topics assigned to site
	UsedTopics         int        `json:"used_topics"`          // Topics that have been used at least once
	UnusedTopics       int        `json:"unused_topics"`        // Topics never used
	UniqueTopicsLeft   int        `json:"unique_topics_left"`   // For unique strategy: topics not yet used
	RoundRobinPosition int        `json:"round_robin_position"` // Current position in round-robin cycle
	MostUsedTopicID    int64      `json:"most_used_topic_id"`
	MostUsedTopicCount int        `json:"most_used_topic_count"`
	LastUsedTopicID    int64      `json:"last_used_topic_id"`
	LastUsedAt         *time.Time `json:"last_used_at"`
}

// TopicSelectionStrategy defines strategy constants
type TopicSelectionStrategy string

const (
	StrategyUnique     TopicSelectionStrategy = "unique"
	StrategyRoundRobin TopicSelectionStrategy = "round_robin"
	StrategyRandom     TopicSelectionStrategy = "random"
	StrategyRandomAll  TopicSelectionStrategy = "random_all"
)

// TopicSelectionRequest represents a request to select a topic for article generation
type TopicSelectionRequest struct {
	SiteID   int64                  `json:"site_id"`
	Strategy TopicSelectionStrategy `json:"strategy,omitempty"` // If empty, use site default or global default
}

// TopicSelectionResult represents the result of topic selection
type TopicSelectionResult struct {
	Topic          *Topic     `json:"topic"`
	SiteTopic      *SiteTopic `json:"site_topic"`
	Strategy       string     `json:"strategy"`
	CanContinue    bool       `json:"can_continue"`    // For unique strategy: are there more unused topics?
	RemainingCount int        `json:"remaining_count"` // For unique strategy: how many unused topics remain
}

type PaginationResult[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type PaginationRequest struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}
