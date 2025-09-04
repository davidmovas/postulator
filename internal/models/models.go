package models

import "time"

type Site struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`             // Название для удобства
	URL       string    `json:"url" db:"url"`               // https://example.com
	Username  string    `json:"username" db:"username"`     // WP логин
	Password  string    `json:"password" db:"password"`     // Зашифрованный пароль/app password
	APIKey    string    `json:"api_key" db:"api_key"`       // Если используется JWT или другой токен
	IsActive  bool      `json:"is_active" db:"is_active"`   // Включен ли сайт
	LastCheck time.Time `json:"last_check" db:"last_check"` // Последняя проверка доступности
	Status    string    `json:"status" db:"status"`         // "connected", "error", "pending"
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Topic struct {
	ID          int64     `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`             // Название темы
	Description string    `json:"description" db:"description"` // Описание темы
	Keywords    string    `json:"keywords" db:"keywords"`       // Ключевые слова через запятую
	Prompt      string    `json:"prompt" db:"prompt"`           // Промпт для GPT
	Category    string    `json:"category" db:"category"`       // Категория на WP
	Tags        string    `json:"tags" db:"tags"`               // Теги через запятую
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type SiteTopic struct {
	ID       int64 `json:"id" db:"id"`
	SiteID   int64 `json:"site_id" db:"site_id"`
	TopicID  int64 `json:"topic_id" db:"topic_id"`
	Priority int   `json:"priority" db:"priority"` // Приоритет темы для сайта (1-10)
	IsActive bool  `json:"is_active" db:"is_active"`
}

type Schedule struct {
	ID          int64     `json:"id" db:"id"`
	SiteID      int64     `json:"site_id" db:"site_id"`
	CronExpr    string    `json:"cron_expr" db:"cron_expr"`         // "0 */6 * * *" - каждые 6 часов
	PostsPerDay int       `json:"posts_per_day" db:"posts_per_day"` // Сколько постов в день
	IsActive    bool      `json:"is_active" db:"is_active"`
	LastRun     time.Time `json:"last_run" db:"last_run"`
	NextRun     time.Time `json:"next_run" db:"next_run"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Article struct {
	ID          int64     `json:"id" db:"id"`
	SiteID      int64     `json:"site_id" db:"site_id"`
	TopicID     int64     `json:"topic_id" db:"topic_id"`
	Title       string    `json:"title" db:"title"`
	Content     string    `json:"content" db:"content"` // HTML контент
	Excerpt     string    `json:"excerpt" db:"excerpt"` // Краткое описание
	Keywords    string    `json:"keywords" db:"keywords"`
	Tags        string    `json:"tags" db:"tags"`
	Category    string    `json:"category" db:"category"`
	Status      string    `json:"status" db:"status"`             // "generated", "published", "failed"
	WordPressID int64     `json:"wordpress_id" db:"wordpress_id"` // ID в WordPress после публикации
	GPTModel    string    `json:"gpt_model" db:"gpt_model"`       // Какая модель использовалась
	Tokens      int       `json:"tokens" db:"tokens"`             // Потрачено токенов
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
