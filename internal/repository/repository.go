package repository

import (
	"database/sql"

	"github.com/Masterminds/squirrel"
	br "github.com/lann/builder"
)

var (
	_ SiteRepository       = (*Repository)(nil)
	_ TopicRepository      = (*Repository)(nil)
	_ SiteTopicRepository  = (*Repository)(nil)
	_ TopicUsageRepository = (*Repository)(nil)
	_ PromptRepository     = (*Repository)(nil)
	_ SitePromptRepository = (*Repository)(nil)
)

var builder = squirrel.StatementBuilderType(br.EmptyBuilder).PlaceholderFormat(squirrel.Dollar)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}
