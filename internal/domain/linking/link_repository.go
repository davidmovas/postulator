package linking

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/davidmovas/postulator/pkg/errors"
)

type linkRepository struct {
	db *sql.DB
}

// linkColumns defines the column list for planned_links table
var linkColumns = []string{
	"id", "plan_id", "source_node_id", "target_node_id", "anchor_text", "anchor_context",
	"status", "source", "position", "confidence", "error", "applied_at", "created_at", "updated_at",
}

// linkInsertColumns defines the columns for insert operations (excludes id)
var linkInsertColumns = []string{
	"plan_id", "source_node_id", "target_node_id", "anchor_text", "anchor_context",
	"status", "source", "position", "confidence", "error", "applied_at", "created_at", "updated_at",
}

func NewLinkRepository(db *sql.DB) LinkRepository {
	return &linkRepository{db: db}
}

func (r *linkRepository) Create(ctx context.Context, link *PlannedLink) error {
	now := time.Now()
	link.CreatedAt = now
	link.UpdatedAt = now

	query, args, err := sq.Insert("planned_links").
		Columns(linkInsertColumns...).
		Values(
			link.PlanID, link.SourceNodeID, link.TargetNodeID, link.AnchorText, link.AnchorContext,
			link.Status, link.Source, link.Position, link.Confidence, link.Error, link.AppliedAt, link.CreatedAt, link.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return errors.Database(err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}
	link.ID = id

	return nil
}

func (r *linkRepository) CreateBatch(ctx context.Context, links []*PlannedLink) error {
	if len(links) == 0 {
		return nil
	}

	now := time.Now()
	builder := sq.Insert("planned_links").
		Columns(linkInsertColumns...)

	for _, link := range links {
		link.CreatedAt = now
		link.UpdatedAt = now
		builder = builder.Values(
			link.PlanID, link.SourceNodeID, link.TargetNodeID, link.AnchorText, link.AnchorContext,
			link.Status, link.Source, link.Position, link.Confidence, link.Error, link.AppliedAt, link.CreatedAt, link.UpdatedAt,
		)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return errors.Database(err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *linkRepository) GetByID(ctx context.Context, id int64) (*PlannedLink, error) {
	query, args, err := sq.Select(linkColumns...).
		From("planned_links").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, errors.Database(err)
	}

	link := &PlannedLink{}
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&link.ID, &link.PlanID, &link.SourceNodeID, &link.TargetNodeID, &link.AnchorText, &link.AnchorContext,
		&link.Status, &link.Source, &link.Position, &link.Confidence, &link.Error, &link.AppliedAt, &link.CreatedAt, &link.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound("planned_link", id)
	}
	if err != nil {
		return nil, errors.Database(err)
	}

	return link, nil
}

func (r *linkRepository) GetByPlanID(ctx context.Context, planID int64) ([]*PlannedLink, error) {
	return r.queryLinks(ctx, sq.Eq{"plan_id": planID})
}

func (r *linkRepository) GetBySourceNodeID(ctx context.Context, planID int64, nodeID int64) ([]*PlannedLink, error) {
	return r.queryLinks(ctx, sq.Eq{"plan_id": planID, "source_node_id": nodeID})
}

func (r *linkRepository) GetByTargetNodeID(ctx context.Context, planID int64, nodeID int64) ([]*PlannedLink, error) {
	return r.queryLinks(ctx, sq.Eq{"plan_id": planID, "target_node_id": nodeID})
}

func (r *linkRepository) GetByNodePair(ctx context.Context, planID int64, sourceID int64, targetID int64) (*PlannedLink, error) {
	query, args, err := sq.Select(linkColumns...).
		From("planned_links").
		Where(sq.Eq{"plan_id": planID, "source_node_id": sourceID, "target_node_id": targetID}).
		ToSql()
	if err != nil {
		return nil, errors.Database(err)
	}

	link := &PlannedLink{}
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&link.ID, &link.PlanID, &link.SourceNodeID, &link.TargetNodeID, &link.AnchorText, &link.AnchorContext,
		&link.Status, &link.Source, &link.Position, &link.Confidence, &link.Error, &link.AppliedAt, &link.CreatedAt, &link.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Database(err)
	}

	return link, nil
}

func (r *linkRepository) queryLinks(ctx context.Context, where interface{}) ([]*PlannedLink, error) {
	query, args, err := sq.Select(linkColumns...).
		From("planned_links").
		Where(where).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, errors.Database(err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer rows.Close()

	var links []*PlannedLink
	for rows.Next() {
		link := &PlannedLink{}
		if err := rows.Scan(
			&link.ID, &link.PlanID, &link.SourceNodeID, &link.TargetNodeID, &link.AnchorText, &link.AnchorContext,
			&link.Status, &link.Source, &link.Position, &link.Confidence, &link.Error, &link.AppliedAt, &link.CreatedAt, &link.UpdatedAt,
		); err != nil {
			return nil, errors.Database(err)
		}
		links = append(links, link)
	}

	return links, nil
}

func (r *linkRepository) Update(ctx context.Context, link *PlannedLink) error {
	link.UpdatedAt = time.Now()

	query, args, err := sq.Update("planned_links").
		Set("anchor_text", link.AnchorText).
		Set("anchor_context", link.AnchorContext).
		Set("status", link.Status).
		Set("position", link.Position).
		Set("confidence", link.Confidence).
		Set("error", link.Error).
		Set("applied_at", link.AppliedAt).
		Set("updated_at", link.UpdatedAt).
		Where(sq.Eq{"id": link.ID}).
		ToSql()
	if err != nil {
		return errors.Database(err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errors.NotFound("planned_link", link.ID)
	}

	return nil
}

func (r *linkRepository) UpdateStatus(ctx context.Context, id int64, status LinkStatus, errorMsg *string) error {
	query, args, err := sq.Update("planned_links").
		Set("status", status).
		Set("error", errorMsg).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return errors.Database(err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *linkRepository) Delete(ctx context.Context, id int64) error {
	query, args, err := sq.Delete("planned_links").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return errors.Database(err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NotFound("planned_link", id)
	}

	return nil
}

func (r *linkRepository) DeleteByPlanID(ctx context.Context, planID int64) error {
	query, args, err := sq.Delete("planned_links").
		Where(sq.Eq{"plan_id": planID}).
		ToSql()
	if err != nil {
		return errors.Database(err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *linkRepository) CountByStatus(ctx context.Context, planID int64, status LinkStatus) (int, error) {
	query, args, err := sq.Select("COUNT(*)").
		From("planned_links").
		Where(sq.Eq{"plan_id": planID, "status": status}).
		ToSql()
	if err != nil {
		return 0, errors.Database(err)
	}

	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, errors.Database(err)
	}

	return count, nil
}
