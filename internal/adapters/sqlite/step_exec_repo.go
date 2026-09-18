package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	execColumns = `id, run_id, item_id, step, attempt, status, input_hash, artifact_id, tokens, usd,
		started_at, finished_at, error`
	insertExec = `INSERT INTO step_execs (` + execColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	selectDone = `SELECT ` + execColumns + ` FROM step_execs
		WHERE item_id = ? AND step = ? AND status = 'done' AND input_hash = ? ORDER BY attempt DESC LIMIT 1`
	selectExecsByItem = `SELECT ` + execColumns + ` FROM step_execs WHERE item_id = ? ORDER BY started_at, id`
)

type StepExecRepo struct {
	store *Store
}

func NewStepExecRepo(store *Store) *StepExecRepo {
	return &StepExecRepo{store: store}
}

func (r *StepExecRepo) Insert(ctx context.Context, exec run.StepExec) error {
	if !exec.Status.Valid() {
		return errors.New(errors.Invalid, "step execution status is not recognized").
			WithDetail("status", string(exec.Status))
	}

	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertExec, []any{
		exec.ID, exec.RunID, exec.ItemID, exec.Step, exec.Attempt, string(exec.Status), exec.InputHash,
		nullString(exec.ArtifactID), exec.Tokens, exec.USD, formatTime(exec.StartedAt),
		nullTime(exec.FinishedAt), exec.Error,
	}, errors.New(errors.Conflict, "this attempt of the step is already recorded"), "record the step execution")
	return err
}

func (r *StepExecRepo) Done(ctx context.Context, itemID, step, inputHash string) (run.StepExec, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectDone, []any{itemID, step, inputHash}, scanExec,
		errors.New(errors.NotFound, "the step has no completed attempt with this input"), "read the step execution")
}

func (r *StepExecRepo) ByItem(ctx context.Context, itemID string) ([]run.StepExec, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectExecsByItem, []any{itemID}, scanExec,
		"list the step executions of the run item")
}

func scanExec(rows *sql.Rows) (run.StepExec, error) {
	var (
		exec       run.StepExec
		status     string
		artifactID sql.NullString
		startedAt  string
		finishedAt sql.NullString
	)
	if err := rows.Scan(
		&exec.ID, &exec.RunID, &exec.ItemID, &exec.Step, &exec.Attempt, &status, &exec.InputHash, &artifactID,
		&exec.Tokens, &exec.USD, &startedAt, &finishedAt, &exec.Error,
	); err != nil {
		return run.StepExec{}, err
	}

	exec.Status = run.ExecStatus(status)
	exec.ArtifactID = optString(artifactID)

	var err error
	if exec.StartedAt, err = parseTime(startedAt); err != nil {
		return run.StepExec{}, err
	}
	if exec.FinishedAt, err = parseNullTime(finishedAt); err != nil {
		return run.StepExec{}, err
	}
	return exec, nil
}
