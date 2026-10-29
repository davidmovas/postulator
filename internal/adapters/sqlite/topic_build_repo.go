package sqlite

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	recordTopicBuild     = `INSERT INTO topic_builds (site_id, subject, built_at) VALUES (?, ?, ?) ON CONFLICT (site_id, subject) DO NOTHING`
	selectTopicBuildSite = `SELECT subject FROM topic_builds WHERE site_id = ? ORDER BY subject`
)

type TopicBuildRepo struct {
	store *Store
}

func NewTopicBuildRepo(store *Store) *TopicBuildRepo {
	return &TopicBuildRepo{store: store}
}

func (r *TopicBuildRepo) Record(ctx context.Context, siteID, subject string, at time.Time) error {
	if strings.TrimSpace(subject) == "" {
		return errors.New(errors.Invalid, "a built subject must not be empty").WithDetail("field", "subject")
	}
	_, err := execWrite(ctx, r.store.writeFrom(ctx), recordTopicBuild, []any{siteID, subject, formatTime(at)}, nil,
		"remember what the site's map built")
	return err
}

func (r *TopicBuildRepo) ListBySite(ctx context.Context, siteID string) ([]string, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectTopicBuildSite, []any{siteID}, scanText, "list what the site's map built")
}
