package middleware_test

import (
	"context"
	stderrors "errors"
	"io"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	kernelctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

func TestRecover(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		panic any
		want  string
	}{
		{name: "string", panic: "boom", want: "boom"},
		{name: "error", panic: io.EOF, want: "EOF"},
		{name: "value", panic: 42, want: "42"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := middleware.Recover(func(context.Context, string) (int, error) {
				panic(tc.panic)
			})

			out, err := handler(context.Background(), "in")
			if out != 0 {
				t.Fatalf("out = %d, want the zero value", out)
			}
			if !errors.IsCode(err, errors.Internal) {
				t.Fatalf("err = %v, want code %s", err, errors.Internal)
			}
			if got := errors.Stack(err); len(got) == 0 {
				t.Fatal("a recovered panic must carry a stack")
			}
			var kernelErr *errors.Error
			if !stderrors.As(err, &kernelErr) {
				t.Fatalf("err = %v, want a kernel error", err)
			}
			if details := kernelErr.Details["panic"]; details != tc.want {
				t.Fatalf("panic detail = %v, want %q", details, tc.want)
			}
		})
	}
}

func TestRecoverPassesThrough(t *testing.T) {
	t.Parallel()

	handler := middleware.Recover(func(_ context.Context, in string) (string, error) {
		return in + "!", nil
	})

	out, err := handler(context.Background(), "hi")
	if err != nil || out != "hi!" {
		t.Fatalf("handler() = (%q, %v)", out, err)
	}
}

func TestRecoverKeepsTheHandlerError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New(errors.NotFound, "gone")
	handler := middleware.Recover(func(context.Context, string) (string, error) {
		return "", sentinel
	})

	if _, err := handler(context.Background(), "x"); !stderrors.Is(err, sentinel) {
		t.Fatalf("err = %v, want the handler error", err)
	}
}

func TestTimeout(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	handler := middleware.Timeout(10*time.Millisecond, func(ctx context.Context, _ string) (string, error) {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return "late", nil
	})

	out, err := handler(context.Background(), "x")
	if out != "" {
		t.Fatalf("out = %q, want the zero value", out)
	}
	if !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("err = %v, want code %s", err, errors.Cancelled)
	}
}

func TestTimeoutPassesThrough(t *testing.T) {
	t.Parallel()

	handler := middleware.Timeout(time.Minute, func(_ context.Context, in int) (int, error) {
		return in * 2, nil
	})

	out, err := handler(context.Background(), 21)
	if err != nil || out != 42 {
		t.Fatalf("handler() = (%d, %v)", out, err)
	}
}

func TestTimeoutPropagatesCancellation(t *testing.T) {
	t.Parallel()

	parent, cancel := context.WithCancel(context.Background())
	cancel()

	handler := middleware.Timeout(time.Minute, func(ctx context.Context, _ int) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})

	if _, err := handler(parent, 1); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("err = %v, want code %s", err, errors.Cancelled)
	}
}

func TestAudit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		err       error
		wantLevel zapcore.Level
		wantCode  string
	}{
		{name: "success", err: nil, wantLevel: zapcore.InfoLevel},
		{name: "kernel failure", err: errors.New(errors.NotFound, "gone"), wantLevel: zapcore.WarnLevel, wantCode: "NOT_FOUND"},
		{name: "foreign failure", err: io.EOF, wantLevel: zapcore.WarnLevel, wantCode: "INTERNAL"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			core, logs := observer.New(zapcore.DebugLevel)
			handler := middleware.Audit(zap.New(core), "pages.Create", func(context.Context, int) (int, error) {
				return 1, tc.err
			})

			_, err := handler(context.Background(), 0)
			if !stderrors.Is(err, tc.err) {
				t.Fatalf("err = %v, want %v", err, tc.err)
			}

			entries := logs.All()
			if len(entries) != 1 {
				t.Fatalf("logged %d entries, want 1", len(entries))
			}
			entry := entries[0]
			if entry.Level != tc.wantLevel {
				t.Fatalf("level = %v, want %v", entry.Level, tc.wantLevel)
			}

			fields := entry.ContextMap()
			if fields["operation"] != "pages.Create" {
				t.Fatalf("operation = %v", fields["operation"])
			}
			if _, ok := fields["durationMs"]; !ok {
				t.Fatalf("durationMs missing: %v", fields)
			}
			if tc.wantCode == "" {
				if _, ok := fields["code"]; ok {
					t.Fatalf("a successful call must not log a code: %v", fields)
				}
				return
			}
			if fields["code"] != tc.wantCode {
				t.Fatalf("code = %v, want %q", fields["code"], tc.wantCode)
			}
		})
	}
}

func TestAuditCarriesContextIdentifiers(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zapcore.DebugLevel)
	handler := middleware.Audit(zap.New(core), "runs.Advance", func(context.Context, int) (int, error) {
		return 0, nil
	})

	carried := kernelctx.WithActor(
		kernelctx.WithConversationID(kernelctx.WithRunID(context.Background(), "run-1"), "conv-1"),
		kernelctx.ActorAgent,
	)
	if _, err := handler(carried, 0); err != nil {
		t.Fatalf("handler() error: %v", err)
	}

	fields := logs.All()[0].ContextMap()
	if fields["runId"] != "run-1" || fields["conversationId"] != "conv-1" || fields["actor"] != "agent" {
		t.Fatalf("context identifiers missing: %v", fields)
	}
}

func TestChain(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zapcore.DebugLevel)
	handler := middleware.Audit(zap.New(core), "pages.Create",
		middleware.Recover(
			middleware.Timeout(time.Minute, func(context.Context, int) (int, error) {
				panic("boom")
			}),
		),
	)

	if _, err := handler(context.Background(), 0); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("err = %v, want code %s", err, errors.Internal)
	}
	if len(logs.All()) != 1 {
		t.Fatalf("logged %d entries, want 1", len(logs.All()))
	}
}
