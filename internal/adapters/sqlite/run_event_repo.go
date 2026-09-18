package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	eventColumns = `run_id, seq, type, at, payload`
	insertEvent  = `INSERT INTO run_events (id, run_id, seq, type, at, payload) VALUES (?, ?, ?, ?, ?, ?)`
	nextEventSeq = `SELECT coalesce(max(seq), 0) + 1 FROM run_events WHERE run_id = ?`
	selectEvents = `SELECT ` + eventColumns + ` FROM run_events WHERE run_id = ? AND seq > ? ORDER BY seq LIMIT ?`
)

type RunEventRepo struct {
	store *Store
}

func NewRunEventRepo(store *Store) *RunEventRepo {
	return &RunEventRepo{store: store}
}

func (r *RunEventRepo) Append(ctx context.Context, runID, eventType string, at time.Time, payload []byte) (run.Event, error) {
	var seq int64
	row := r.store.writeFrom(ctx).QueryRowContext(ctx, nextEventSeq, runID)
	if err := row.Scan(&seq); err != nil {
		return run.Event{}, errors.Wrap(err, errors.Internal, "read the next run event sequence")
	}

	event, err := run.NewEvent(run.Event{RunID: runID, Seq: seq, Type: eventType, At: at, Payload: payload})
	if err != nil {
		return run.Event{}, err
	}

	if _, err = execWrite(ctx, r.store.writeFrom(ctx), insertEvent,
		[]any{id.New(), event.RunID, event.Seq, event.Type, formatTime(event.At), string(event.Payload)},
		errors.New(errors.Conflict, "this run event sequence is already recorded"), "append the run event"); err != nil {
		return run.Event{}, err
	}
	return event, nil
}

func (r *RunEventRepo) List(ctx context.Context, runID string, sinceSeq int64, limit int) ([]run.Event, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectEvents, []any{runID, sinceSeq, limit}, scanEvent,
		"list the run events")
}

func scanEvent(rows *sql.Rows) (run.Event, error) {
	var (
		event   run.Event
		at      string
		payload string
	)
	if err := rows.Scan(&event.RunID, &event.Seq, &event.Type, &at, &payload); err != nil {
		return run.Event{}, err
	}

	event.Payload = []byte(payload)

	var err error
	if event.At, err = parseTime(at); err != nil {
		return run.Event{}, err
	}
	return event, nil
}
