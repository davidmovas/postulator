package dbx

import (
	"context"
	"database/sql"
	stderrors "errors"
	"io"
	"testing"

	"github.com/ncruces/go-sqlite3"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want errors.Code
	}{
		{name: "no error", err: nil, want: ""},
		{name: "no rows", err: sql.ErrNoRows, want: errors.NotFound},
		{name: "not found opcode", err: sqlite3.NOTFOUND, want: errors.NotFound},
		{name: "unique", err: sqlite3.CONSTRAINT_UNIQUE, want: errors.Conflict},
		{name: "primary key", err: sqlite3.CONSTRAINT_PRIMARYKEY, want: errors.Conflict},
		{name: "foreign key", err: sqlite3.CONSTRAINT_FOREIGNKEY, want: errors.Invalid},
		{name: "not null", err: sqlite3.CONSTRAINT_NOTNULL, want: errors.Invalid},
		{name: "check", err: sqlite3.CONSTRAINT_CHECK, want: errors.Invalid},
		{name: "busy", err: sqlite3.BUSY, want: errors.External},
		{name: "busy timeout", err: sqlite3.BUSY_TIMEOUT, want: errors.External},
		{name: "locked", err: sqlite3.LOCKED, want: errors.External},
		{name: "interrupt", err: sqlite3.INTERRUPT, want: errors.Cancelled},
		{name: "context cancelled", err: context.Canceled, want: errors.Cancelled},
		{name: "context deadline exceeded", err: context.DeadlineExceeded, want: errors.Cancelled},
		{name: "read only", err: sqlite3.READONLY, want: errors.Internal},
		{name: "foreign error", err: io.EOF, want: errors.Internal},
		{name: "wrapped unique", err: stderrors.Join(io.EOF, sqlite3.CONSTRAINT_UNIQUE), want: errors.Conflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := Classify(tc.err); got != tc.want {
				t.Errorf("Classify(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestConvert(t *testing.T) {
	t.Parallel()

	if Convert(nil, "read row") != nil {
		t.Fatal("Convert(nil) must be nil")
	}

	already := errors.New(errors.NeedsHuman, "left alone")
	if got := Convert(already, "read row"); !stderrors.Is(got, already) {
		t.Errorf("Convert must not rewrap a kernel error, got %v", got)
	}

	converted := Convert(sqlite3.CONSTRAINT_UNIQUE, "insert secret")
	if !errors.IsCode(converted, errors.Conflict) {
		t.Errorf("code = %q, want %q", errors.CodeOf(converted), errors.Conflict)
	}
	if !stderrors.Is(converted, sqlite3.CONSTRAINT_UNIQUE) {
		t.Error("the driver error must stay reachable through Unwrap")
	}

	busy := Convert(sqlite3.BUSY, "begin transaction")
	if !errors.IsCode(busy, errors.External) {
		t.Errorf("code = %q, want %q", errors.CodeOf(busy), errors.External)
	}
	if !stderrors.Is(busy, sqlite3.BUSY) {
		t.Error("the driver error must stay reachable through Unwrap")
	}

	var kernel *errors.Error
	if !stderrors.As(busy, &kernel) {
		t.Fatalf("Convert must produce a kernel error, got %v", busy)
	}
	if kernel.Retry == nil || kernel.Retry.After != BusyRetryAfter {
		t.Errorf("Retry = %+v, want an After of %v", kernel.Retry, BusyRetryAfter)
	}
}

func TestPredicates(t *testing.T) {
	t.Parallel()

	if !IsNotFound(sql.ErrNoRows) || !IsNotFound(errors.New(errors.NotFound, "gone")) {
		t.Error("IsNotFound must accept the driver error and the kernel error")
	}
	if IsNotFound(io.EOF) {
		t.Error("IsNotFound must reject a foreign error")
	}
	if !IsConflict(sqlite3.CONSTRAINT_PRIMARYKEY) || !IsConflict(errors.New(errors.Conflict, "taken")) {
		t.Error("IsConflict must accept the driver error and the kernel error")
	}
	if IsConflict(io.EOF) {
		t.Error("IsConflict must reject a foreign error")
	}
}
