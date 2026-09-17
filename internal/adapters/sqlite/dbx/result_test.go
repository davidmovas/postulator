package dbx

import (
	"database/sql"
	stderrors "errors"
	"io"
	"testing"

	"github.com/ncruces/go-sqlite3"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestResultUnwrap(t *testing.T) {
	t.Parallel()

	value, err := Ok("secret").Unwrap()
	if value != "secret" || err != nil {
		t.Fatalf("Ok = %q, %v", value, err)
	}

	value, err = Err[string](io.EOF).Unwrap()
	if value != "" || !stderrors.Is(err, io.EOF) {
		t.Fatalf("Err = %q, %v", value, err)
	}

	value, err = From("row", sql.ErrNoRows).Unwrap()
	if value != "row" || !stderrors.Is(err, sql.ErrNoRows) {
		t.Fatalf("From = %q, %v", value, err)
	}
}

func TestResultReplacements(t *testing.T) {
	t.Parallel()

	missing := errors.New(errors.NotFound, "secret is not stored")
	taken := errors.New(errors.Conflict, "secret already exists")

	cases := []struct {
		name   string
		result Result[int]
		want   errors.Code
	}{
		{
			name:   "not found is replaced",
			result: From(0, sql.ErrNoRows).NotFound(missing),
			want:   errors.NotFound,
		},
		{
			name:   "conflict is replaced",
			result: From(0, sqlite3.CONSTRAINT_UNIQUE).Conflict(taken),
			want:   errors.Conflict,
		},
		{
			name:   "unrelated error survives NotFound",
			result: From(0, sqlite3.BUSY).NotFound(missing).WrapErr(wrap),
			want:   errors.External,
		},
		{
			name:   "unrelated error survives Conflict",
			result: From(0, sqlite3.BUSY).Conflict(taken).WrapErr(wrap),
			want:   errors.External,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := tc.result.Unwrap()
			if errors.CodeOf(err) != tc.want {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), tc.want)
			}
		})
	}
}

func TestResultWrapErr(t *testing.T) {
	t.Parallel()

	_, err := Ok(1).WrapErr(wrap).Unwrap()
	if err != nil {
		t.Fatalf("WrapErr on a value must not produce an error, got %v", err)
	}

	already := errors.New(errors.NeedsHuman, "left alone")
	_, err = From(0, already).WrapErr(wrap).Unwrap()
	if !stderrors.Is(err, already) {
		t.Fatalf("WrapErr must leave a kernel error alone, got %v", err)
	}

	_, err = From(0, io.EOF).WrapErr(wrap).Unwrap()
	if errors.CodeOf(err) != errors.Internal || !stderrors.Is(err, io.EOF) {
		t.Fatalf("WrapErr must convert a foreign error, got %v", err)
	}
}

func wrap(cause error) error {
	return Convert(cause, "read row")
}
