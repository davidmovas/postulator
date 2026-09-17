package wails_test

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"

	kernelctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

func TestWrapPutsTheUserActorInTheContext(t *testing.T) {
	t.Parallel()

	handler := wails.Wrap(zap.NewNop(), "test.actor", func(c context.Context, _ struct{}) (kernelctx.Actor, error) {
		actor, ok := kernelctx.ActorFrom(c)
		if !ok {
			t.Error("the wrapper did not put an actor in the context")
		}
		return actor, nil
	})

	actor, err := handler(context.Background(), struct{}{})
	if err != nil {
		t.Fatalf("handler() error: %v", err)
	}
	if actor != kernelctx.ActorUser {
		t.Fatalf("actor = %q, want %q", actor, kernelctx.ActorUser)
	}
}

func TestWrapConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		handler func(context.Context, struct{}) (int, error)
		want    string
	}{
		{
			name: "kernel error keeps its code and loses its cause",
			handler: func(context.Context, struct{}) (int, error) {
				return 0, errors.New(errors.Invalid, "limit is out of range").WithDetail("limit", 9000)
			},
			want: `{"code":"INVALID","message":"limit is out of range","details":{"limit":9000}}`,
		},
		{
			name: "foreign error becomes internal",
			handler: func(context.Context, struct{}) (int, error) {
				return 0, context.DeadlineExceeded
			},
			want: `{"code":"INTERNAL","message":"unexpected internal error"}`,
		},
		{
			name: "panic becomes internal without the panic text",
			handler: func(context.Context, struct{}) (int, error) {
				panic("secret connection string")
			},
			want: `{"code":"INTERNAL","message":"handler panicked"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := wails.Wrap(zap.NewNop(), "test.failure", tc.handler)(context.Background(), struct{}{})
			if err == nil {
				t.Fatal("handler() returned no error")
			}
			if result != 0 {
				t.Fatalf("handler() result = %d, want the zero value", result)
			}
			if got := string(wails.MarshalError(err)); got != tc.want {
				t.Fatalf("MarshalError() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestServicesBindsTheHealthService(t *testing.T) {
	t.Parallel()

	services := wails.Services(zap.NewNop(), wails.BuildInfo{Version: "2.0.0", Commit: "abc1234", BuildDate: "2026-09-17T10:30:00Z"})
	if len(services) != 1 {
		t.Fatalf("Services() returned %d services, want 1", len(services))
	}

	health, ok := services[0].Instance().(*wails.HealthService)
	if !ok {
		t.Fatalf("Services()[0] is %T, want *wails.HealthService", services[0].Instance())
	}

	build, err := health.Ping(context.Background(), wails.PingRequest{})
	if err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
	if build.Version != "2.0.0" || build.Commit != "abc1234" || build.BuildDate != "2026-09-17T10:30:00Z" {
		t.Fatalf("Ping() = %+v, want the injected build stamps", build)
	}
}

func TestBuildInfoMarshalsCamelCase(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(wails.BuildInfo{Version: "2.0.0", Commit: "abc1234", BuildDate: "2026-09-17T10:30:00Z"})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	const want = `{"version":"2.0.0","commit":"abc1234","buildDate":"2026-09-17T10:30:00Z"}`
	if string(encoded) != want {
		t.Fatalf("Marshal() = %s, want %s", encoded, want)
	}
}
