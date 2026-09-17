package errors_test

import (
	stderrors "errors"
	"io"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestErrorMessage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "plain", err: errors.New(errors.NotFound, "x"), want: "x"},
		{name: "wrapped", err: errors.Wrap(io.EOF, errors.External, "wp"), want: "wp: EOF"},
		{name: "with internal", err: errors.New(errors.Internal, "boom").WithInternal(io.ErrUnexpectedEOF), want: "boom: unexpected EOF"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCodeOf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want errors.Code
	}{
		{name: "nil", err: nil, want: ""},
		{name: "foreign", err: io.EOF, want: errors.Internal},
		{name: "wrapped foreign", err: errors.Wrap(io.EOF, errors.External, "wp"), want: errors.External},
		{name: "direct", err: errors.New(errors.Conflict, "dup"), want: errors.Conflict},
		{name: "nested in fmt", err: stderrors.Join(errors.New(errors.Locked, "db"), io.EOF), want: errors.Locked},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := errors.CodeOf(tc.err); got != tc.want {
				t.Fatalf("CodeOf() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		code errors.Code
		want bool
	}{
		{name: "match", err: errors.New(errors.NeedsHuman, "review"), code: errors.NeedsHuman, want: true},
		{name: "mismatch", err: errors.New(errors.NeedsHuman, "review"), code: errors.Cancelled, want: false},
		{name: "foreign is internal", err: io.EOF, code: errors.Internal, want: true},
		{name: "nil", err: nil, code: errors.Internal, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := errors.IsCode(tc.err, tc.code); got != tc.want {
				t.Fatalf("IsCode() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsComparesByCode(t *testing.T) {
	t.Parallel()

	if !stderrors.Is(errors.New(errors.NotFound, "a"), errors.New(errors.NotFound, "b")) {
		t.Fatal("errors with the same code must compare equal")
	}
	if stderrors.Is(errors.New(errors.NotFound, "a"), errors.New(errors.Conflict, "a")) {
		t.Fatal("errors with different codes must not compare equal")
	}
}

func TestUnwrapReachesInternal(t *testing.T) {
	t.Parallel()

	wrapped := errors.Wrap(io.EOF, errors.External, "wp")
	if !stderrors.Is(wrapped, io.EOF) {
		t.Fatal("wrapped error must unwrap to the original")
	}
	if got := stderrors.Unwrap(wrapped); got != io.EOF {
		t.Fatalf("Unwrap() = %v, want io.EOF", got)
	}
	if stderrors.Unwrap(errors.New(errors.Invalid, "x")) != nil {
		t.Fatal("an error without an internal cause must unwrap to nil")
	}
}

func TestWrapNilReturnsNil(t *testing.T) {
	t.Parallel()

	if got := errors.Wrap(nil, errors.External, "wp"); got != nil {
		t.Fatalf("Wrap(nil) = %v, want nil", got)
	}
}

func TestWithDetailClones(t *testing.T) {
	t.Parallel()

	original := errors.New(errors.Invalid, "bad").WithDetail("field", "path")
	derived := original.WithDetail("field", "slug").WithDetail("limit", 10)

	if got := original.Details["field"]; got != "path" {
		t.Fatalf("original detail mutated: %v", got)
	}
	if _, ok := original.Details["limit"]; ok {
		t.Fatal("original gained a detail from the clone")
	}
	if got := derived.Details["field"]; got != "slug" {
		t.Fatalf("derived detail = %v, want slug", got)
	}
	if got := derived.Details["limit"]; got != 10 {
		t.Fatalf("derived limit = %v, want 10", got)
	}
	if original.Code != derived.Code || original.Message != derived.Message {
		t.Fatal("clone must preserve code and message")
	}
}

func TestWithRetryClones(t *testing.T) {
	t.Parallel()

	original := errors.New(errors.RateLimited, "slow down")
	derived := original.WithRetry(2 * time.Second)

	if original.Retry != nil {
		t.Fatal("original must not gain retry information")
	}
	if derived.Retry == nil {
		t.Fatal("clone must carry retry information")
	}
	if derived.Retry.After != 2*time.Second {
		t.Fatalf("Retry.After = %v, want 2s", derived.Retry.After)
	}

	again := derived.WithRetry(5 * time.Second)
	if derived.Retry.After != 2*time.Second {
		t.Fatalf("original retry mutated to %v", derived.Retry.After)
	}
	if again.Retry.After != 5*time.Second {
		t.Fatalf("Retry.After = %v, want 5s", again.Retry.After)
	}
}

func TestWithInternalClones(t *testing.T) {
	t.Parallel()

	original := errors.New(errors.External, "wp")
	derived := original.WithInternal(io.EOF)

	if stderrors.Unwrap(original) != nil {
		t.Fatal("original must not gain an internal cause")
	}
	if !stderrors.Is(derived, io.EOF) {
		t.Fatal("clone must carry the internal cause")
	}
}

func TestRetryAfter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		err   error
		want  time.Duration
		known bool
	}{
		{name: "nil", err: nil, want: 0, known: false},
		{name: "no retry", err: errors.New(errors.External, "wp"), want: 0, known: false},
		{name: "foreign", err: io.EOF, want: 0, known: false},
		{name: "with retry", err: errors.New(errors.RateLimited, "slow").WithRetry(time.Minute), want: time.Minute, known: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := errors.RetryAfter(tc.err)
			if ok != tc.known || got != tc.want {
				t.Fatalf("RetryAfter() = (%v, %v), want (%v, %v)", got, ok, tc.want, tc.known)
			}
		})
	}
}

func TestStack(t *testing.T) {
	t.Parallel()

	frames := errors.Stack(errors.New(errors.Internal, "boom"))
	if len(frames) == 0 {
		t.Fatal("Stack() must not be empty")
	}
	if frames[0].Function == "" || frames[0].File == "" || frames[0].Line == 0 {
		t.Fatalf("first frame is incomplete: %+v", frames[0])
	}

	if got := errors.Stack(io.EOF); got != nil {
		t.Fatalf("Stack(foreign) = %v, want nil", got)
	}
	if got := errors.Stack(nil); got != nil {
		t.Fatalf("Stack(nil) = %v, want nil", got)
	}
}

func TestAs(t *testing.T) {
	t.Parallel()

	var target *errors.Error
	if !stderrors.As(errors.Wrap(io.EOF, errors.BudgetExceeded, "cap"), &target) {
		t.Fatal("As must find the kernel error")
	}
	if target.Code != errors.BudgetExceeded {
		t.Fatalf("Code = %q, want %q", target.Code, errors.BudgetExceeded)
	}
}

func TestCodesAreFrozen(t *testing.T) {
	t.Parallel()

	want := map[errors.Code]string{
		errors.NotFound:       "NOT_FOUND",
		errors.Conflict:       "CONFLICT",
		errors.Invalid:        "INVALID",
		errors.Unauthorized:   "UNAUTHORIZED",
		errors.RateLimited:    "RATE_LIMITED",
		errors.BudgetExceeded: "BUDGET_EXCEEDED",
		errors.External:       "EXTERNAL",
		errors.Internal:       "INTERNAL",
		errors.Cancelled:      "CANCELLED",
		errors.NeedsHuman:     "NEEDS_HUMAN",
		errors.Locked:         "LOCKED",
	}

	for code, text := range want {
		if code.String() != text {
			t.Fatalf("code %q renders as %q", text, code.String())
		}
	}
	if len(errors.Codes()) != len(want) {
		t.Fatalf("Codes() has %d entries, want %d", len(errors.Codes()), len(want))
	}
	for _, code := range errors.Codes() {
		if _, ok := want[code]; !ok {
			t.Fatalf("Codes() contains unexpected %q", code)
		}
	}
}
