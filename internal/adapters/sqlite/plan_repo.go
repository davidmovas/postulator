package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	planColumns = `id, conversation_id, site_id, title, status, steps, created_at, updated_at`
	insertPlan  = `INSERT INTO agent_plans (` + planColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	selectPlan  = `SELECT ` + planColumns + ` FROM agent_plans WHERE id = ?`
	updatePlan  = `UPDATE agent_plans SET status = ?, steps = ?, updated_at = ? WHERE id = ?`
	movePlan    = `UPDATE agent_plans SET status = ?, steps = ?, updated_at = ? WHERE id = ? AND status = ?`
	plansOf     = `SELECT ` + planColumns + ` FROM agent_plans WHERE conversation_id = ? ORDER BY created_at, id`
	plansAt     = `SELECT ` + planColumns + ` FROM agent_plans WHERE status = 'running' ORDER BY updated_at, id`
)

type PlanRepo struct {
	store *Store
}

func NewPlanRepo(store *Store) *PlanRepo {
	return &PlanRepo{store: store}
}

func planNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "plan not found").WithDetail("planId", id)
}

func (r *PlanRepo) Insert(ctx context.Context, p agent.Plan) error {
	steps, err := encodeJSON(p.Steps)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertPlan, []any{
		p.ID, p.ConversationID, p.SiteID, p.Title, string(p.Status), steps, formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	}, errors.New(errors.Conflict, "a plan with this id already exists"), "insert the plan")
	return err
}

func (r *PlanRepo) Get(ctx context.Context, id string) (agent.Plan, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectPlan, []any{id}, scanPlan, planNotFound(id), "read the plan")
}

func (r *PlanRepo) Update(ctx context.Context, p agent.Plan) error {
	steps, err := encodeJSON(p.Steps)
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updatePlan, []any{
		string(p.Status), steps, formatTime(p.UpdatedAt), p.ID,
	}, nil, "update the plan")
	return requireAffected(affected, err, planNotFound(p.ID))
}

func (r *PlanRepo) Transition(ctx context.Context, p agent.Plan, from agent.PlanStatus) (bool, error) {
	steps, err := encodeJSON(p.Steps)
	if err != nil {
		return false, err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), movePlan, []any{
		string(p.Status), steps, formatTime(p.UpdatedAt), p.ID, string(from),
	}, nil, "move the plan")
	return affected > 0, err
}

func (r *PlanRepo) ListByConversation(ctx context.Context, conversationID string) ([]agent.Plan, error) {
	return selectAll(ctx, r.store.execFrom(ctx), plansOf, []any{conversationID}, scanPlan, "list the conversation's plans")
}

func (r *PlanRepo) ListRunning(ctx context.Context) ([]agent.Plan, error) {
	return selectAll(ctx, r.store.execFrom(ctx), plansAt, nil, scanPlan, "list the plans being carried out")
}

func scanPlan(rows *sql.Rows) (agent.Plan, error) {
	var (
		plan                 agent.Plan
		status, steps        string
		createdAt, updatedAt string
	)
	if err := rows.Scan(&plan.ID, &plan.ConversationID, &plan.SiteID, &plan.Title, &status, &steps, &createdAt, &updatedAt); err != nil {
		return agent.Plan{}, err
	}
	plan.Status = agent.PlanStatus(status)
	if err := decodeJSON(steps, &plan.Steps, "stored plan steps are not readable"); err != nil {
		return agent.Plan{}, err
	}
	var err error
	if plan.CreatedAt, err = parseTime(createdAt); err != nil {
		return agent.Plan{}, err
	}
	if plan.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return agent.Plan{}, err
	}
	return plan, nil
}
