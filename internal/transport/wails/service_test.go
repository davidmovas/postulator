package wails_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"go.uber.org/zap"

	kernelctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
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

func serviceDeps() wails.Deps {
	registry := settings.New()

	return wails.Deps{
		Sites:     ready[wails.SitesUseCase](sitesFake{}),
		Graph:     ready[wails.GraphUseCase](graphFake{}),
		Pages:     ready[wails.PagesUseCase](pagesFake{}),
		Templates: ready[wails.TemplatesUseCase](templatesFake{}),
		Runs:      ready[wails.RunsUseCase](runsFake{}),
		Sync:      ready[wails.SyncUseCase](syncFake{}),
		Reports:   ready[wails.ReportsUseCase](reportsFake{}),
		Imports:   ready[wails.ImportsUseCase](importsFake{}),
		Models:    ready[wails.ModelsUseCase](modelsFake{}),
		Agent:     ready[wails.AgentUseCase](agentFake{}),
		Schedules: ready[wails.SchedulesUseCase](schedulesFake{}),
		Tools:     ready[wails.ToolCatalog](catalogFake{}),
		Settings: wails.SettingsDeps{
			Access: ready(wails.SettingsAccess{
				Declarations: declarationsFake{registry: registry},
				Values:       registry.NewValues(),
				Store:        &storeFake{stored: map[string]json.RawMessage{}},
				Models:       &providerKeyFake{},
			}),
			Backup: ready[wails.BackupControl](&backupFake{}),
			Lock:   &lockFake{},
		},
	}
}

func TestServicesBindsEveryBoundedContext(t *testing.T) {
	t.Parallel()

	build := wails.BuildInfo{Version: "2.0.0", Commit: "abc1234", BuildDate: "2026-09-17T10:30:00Z"}
	services := wails.Services(zap.NewNop(), build, serviceDeps())

	want := []string{
		"*wails.HealthService",
		"*wails.SitesService",
		"*wails.GraphService",
		"*wails.PagesService",
		"*wails.TemplatesService",
		"*wails.RunsService",
		"*wails.SyncService",
		"*wails.ReportsService",
		"*wails.ImportService",
		"*wails.ModelsService",
		"*wails.AgentService",
		"*wails.SchedulesService",
		"*wails.ToolsService",
		"*wails.SettingsService",
	}
	if len(services) != len(want) {
		t.Fatalf("Services() returned %d services, want %d", len(services), len(want))
	}
	for index, name := range want {
		if got := fmt.Sprintf("%T", services[index].Instance()); got != name {
			t.Errorf("Services()[%d] is %s, want %s", index, got, name)
		}
	}

	health, ok := services[0].Instance().(*wails.HealthService)
	if !ok {
		t.Fatalf("Services()[0] is %T, want *wails.HealthService", services[0].Instance())
	}

	stamped, err := health.Ping(context.Background(), wails.PingRequest{})
	if err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
	if stamped != build {
		t.Fatalf("Ping() = %+v, want the injected build stamps", stamped)
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
