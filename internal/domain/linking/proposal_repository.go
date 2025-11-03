package linking

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ ProposalRepository = (*proposalRepository)(nil)

type proposalRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewProposalRepository(db *database.DB, logger *logger.Logger) ProposalRepository {
	return &proposalRepository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("linking_proposals"),
	}
}

func (r *proposalRepository) CreateProposal(ctx context.Context, proposal *Proposal) error {
	query, args := dbx.ST.
		Insert("linking_proposals").
		Columns(
			"task_id", "source_article_id", "target_article_id",
			"anchor_text", "position", "confidence", "status",
		).
		Values(
			proposal.TaskID, proposal.SourceArticleID, proposal.TargetArticleID,
			proposal.AnchorText, proposal.Position, proposal.Confidence, proposal.Status,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid task or article ID")
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	proposal.ID = id
	return nil
}

func (r *proposalRepository) CreateBatch(ctx context.Context, proposals []*Proposal) error {
	if len(proposals) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Database(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, proposal := range proposals {
		query, args := dbx.ST.
			Insert("linking_proposals").
			Columns(
				"task_id", "source_article_id", "target_article_id",
				"anchor_text", "position", "confidence", "status",
			).
			Values(
				proposal.TaskID, proposal.SourceArticleID, proposal.TargetArticleID,
				proposal.AnchorText, proposal.Position, proposal.Confidence, proposal.Status,
			).
			MustSql()

		_, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return errors.Database(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *proposalRepository) GetByID(ctx context.Context, id int64) (*Proposal, error) {
	query, args := dbx.ST.
		Select(
			"id", "task_id", "source_article_id", "target_article_id",
			"anchor_text", "position", "confidence", "status", "created_at",
		).
		From("linking_proposals").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var proposal Proposal
	var confidence sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&proposal.ID,
		&proposal.TaskID,
		&proposal.SourceArticleID,
		&proposal.TargetArticleID,
		&proposal.AnchorText,
		&proposal.Position,
		&confidence,
		&proposal.Status,
		&proposal.CreatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("linking_proposal", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	if confidence.Valid {
		proposal.Confidence = &confidence.Float64
	}

	return &proposal, nil
}

func (r *proposalRepository) GetByTaskID(ctx context.Context, taskID int64) ([]*Proposal, error) {
	query, args := dbx.ST.
		Select(
			"id", "task_id", "source_article_id", "target_article_id",
			"anchor_text", "position", "confidence", "status", "created_at",
		).
		From("linking_proposals").
		Where(squirrel.Eq{"task_id": taskID}).
		OrderBy("position ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var proposals []*Proposal
	for rows.Next() {
		var proposal Proposal
		var confidence sql.NullFloat64

		err = rows.Scan(
			&proposal.ID,
			&proposal.TaskID,
			&proposal.SourceArticleID,
			&proposal.TargetArticleID,
			&proposal.AnchorText,
			&proposal.Position,
			&confidence,
			&proposal.Status,
			&proposal.CreatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		if confidence.Valid {
			proposal.Confidence = &confidence.Float64
		}

		proposals = append(proposals, &proposal)
	}

	switch {
	case dbx.IsNoRows(err) || len(proposals) == 0:
		return proposals, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return proposals, nil
}

func (r *proposalRepository) Update(ctx context.Context, proposal *Proposal) error {
	query, args := dbx.ST.
		Update("linking_proposals").
		Set("task_id", proposal.TaskID).
		Set("source_article_id", proposal.SourceArticleID).
		Set("target_article_id", proposal.TargetArticleID).
		Set("anchor_text", proposal.AnchorText).
		Set("position", proposal.Position).
		Set("confidence", proposal.Confidence).
		Set("status", proposal.Status).
		Where(squirrel.Eq{"id": proposal.ID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid task or article ID")
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("linking_proposal", proposal.ID)
	}

	return nil
}

func (r *proposalRepository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("linking_proposals").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("linking_proposal", id)
	}

	return nil
}

func (r *proposalRepository) DeleteByTaskID(ctx context.Context, taskID int64) error {
	query, args := dbx.ST.
		Delete("linking_proposals").
		Where(squirrel.Eq{"task_id": taskID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *proposalRepository) CountByStatus(ctx context.Context, taskID int64, status ProposalStatus) (int, error) {
	query, args := dbx.ST.
		Select("COUNT(id)").
		From("linking_proposals").
		Where(squirrel.Eq{"task_id": taskID, "status": status}).
		MustSql()

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	switch {
	case dbx.IsNoRows(err):
		return 0, nil
	case err != nil:
		return 0, errors.Database(err)
	}

	return count, nil
}
