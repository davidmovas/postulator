package wails_test

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

func TestMarshalErrorProducesTheTransportShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input error
		want  string
	}{
		{
			name:  "kernel error with details and retry",
			input: errors.New(errors.RateLimited, "provider is throttling").WithDetail("provider", "openai").WithRetry(1500 * time.Millisecond),
			want:  `{"code":"RATE_LIMITED","message":"provider is throttling","details":{"provider":"openai"},"retry":{"afterMs":1500}}`,
		},
		{
			name:  "kernel error without details",
			input: errors.New(errors.NotFound, "site not found"),
			want:  `{"code":"NOT_FOUND","message":"site not found"}`,
		},
		{
			name:  "internal kernel error drops its details",
			input: errors.New(errors.Internal, "handler panicked").WithDetail("panic", "runtime error: index out of range"),
			want:  `{"code":"INTERNAL","message":"handler panicked"}`,
		},
		{
			name:  "wrapped driver error keeps the authored message only",
			input: errors.Wrap(io.ErrUnexpectedEOF, errors.External, "wordpress rejected the request"),
			want:  `{"code":"EXTERNAL","message":"wordpress rejected the request"}`,
		},
		{
			name:  "foreign error becomes internal",
			input: io.ErrUnexpectedEOF,
			want:  `{"code":"INTERNAL","message":"unexpected internal error"}`,
		},
		{
			name:  "unmarshalable detail falls back",
			input: errors.New(errors.Invalid, "bad input").WithDetail("channel", make(chan int)),
			want:  `{"code":"INTERNAL","message":"unexpected internal error"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := string(wails.MarshalError(tc.input)); got != tc.want {
				t.Fatalf("MarshalError() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestConvertStripsTheInternalChain(t *testing.T) {
	t.Parallel()

	original := errors.New(errors.NotFound, "site not found").
		WithDetail("siteId", "s1").
		WithRetry(2 * time.Second).
		WithInternal(io.ErrUnexpectedEOF)

	converted := wails.Convert(original)
	if converted.Error() != "site not found" {
		t.Fatalf("Convert().Error() = %q, want %q", converted.Error(), "site not found")
	}
	if stderrors.Is(converted, io.ErrUnexpectedEOF) {
		t.Fatal("Convert() kept the internal cause, which would leak it into CallError.Message")
	}
	if errors.CodeOf(converted) != errors.NotFound {
		t.Fatalf("CodeOf(Convert()) = %q, want %q", errors.CodeOf(converted), errors.NotFound)
	}

	const want = `{"code":"NOT_FOUND","message":"site not found","details":{"siteId":"s1"},"retry":{"afterMs":2000}}`
	if got := string(wails.MarshalError(converted)); got != want {
		t.Fatalf("MarshalError(Convert()) = %s, want %s", got, want)
	}
}

func TestConvertReturnsNilForNil(t *testing.T) {
	t.Parallel()

	if got := wails.Convert(nil); got != nil {
		t.Fatalf("Convert(nil) = %v, want nil", got)
	}
}

func TestTheFrontendReceivesTheDocumentedBody(t *testing.T) {
	t.Parallel()

	converted := wails.Convert(errors.New(errors.Conflict, "slug already exists").
		WithDetail("slug", "about-us").
		WithInternal(io.ErrUnexpectedEOF))

	body, err := json.Marshal(&application.CallError{
		Message: converted.Error(),
		Cause:   json.RawMessage(wails.MarshalError(converted)),
		Kind:    application.RuntimeError,
	})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	const want = `{"message":"slug already exists","cause":{"code":"CONFLICT","message":"slug already exists","details":{"slug":"about-us"}},"kind":"RuntimeError"}`
	if string(body) != want {
		t.Fatalf("call error body = %s, want %s", body, want)
	}
}
