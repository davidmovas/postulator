package dbx

import (
	"context"
	"database/sql"
	stderrors "errors"
	"time"

	"github.com/ncruces/go-sqlite3"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const BusyRetryAfter = 250 * time.Millisecond

func Classify(err error) errors.Code {
	switch {
	case err == nil:
		return ""
	case stderrors.Is(err, sql.ErrNoRows), stderrors.Is(err, sqlite3.NOTFOUND):
		return errors.NotFound
	case stderrors.Is(err, sqlite3.CONSTRAINT_UNIQUE), stderrors.Is(err, sqlite3.CONSTRAINT_PRIMARYKEY):
		return errors.Conflict
	case stderrors.Is(err, sqlite3.CONSTRAINT_FOREIGNKEY), stderrors.Is(err, sqlite3.CONSTRAINT):
		return errors.Invalid
	case stderrors.Is(err, sqlite3.BUSY), stderrors.Is(err, sqlite3.LOCKED):
		return errors.External
	case stderrors.Is(err, context.Canceled), stderrors.Is(err, context.DeadlineExceeded), stderrors.Is(err, sqlite3.INTERRUPT):
		return errors.Cancelled
	default:
		return errors.Internal
	}
}

func Convert(err error, message string) error {
	if err == nil {
		return nil
	}
	if isKernel(err) {
		return err
	}

	code := Classify(err)
	if code == errors.External {
		return errors.New(code, message).WithInternal(err).WithRetry(BusyRetryAfter)
	}
	return errors.Wrap(err, code, message)
}

func IsNotFound(err error) bool {
	return errors.IsCode(err, errors.NotFound) || Classify(err) == errors.NotFound
}

func IsConflict(err error) bool {
	return errors.IsCode(err, errors.Conflict) || Classify(err) == errors.Conflict
}

func isKernel(err error) bool {
	var kernel *errors.Error
	return stderrors.As(err, &kernel)
}
