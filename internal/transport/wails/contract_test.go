package wails_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type failure int

const (
	missing failure = iota
	panicking
)

const (
	missingBody = `{"code":"NOT_FOUND","message":"no record carries that id","details":{"id":"nope"}}`
	panicBody   = `{"code":"INTERNAL","message":"handler panicked"}`
	lockedBody  = `{"code":"LOCKED","message":"the application is locked"}`
)

func answer[T any](mode failure) (T, error) {
	var zero T
	if mode == panicking {
		panic("postgres://user:hunter2@host/db")
	}
	return zero, errors.New(errors.NotFound, "no record carries that id").WithDetail("id", "nope")
}

func assertEveryMethodConverts(t *testing.T, service any, want string, skip ...string) {
	t.Helper()

	value := reflect.ValueOf(service)
	if value.NumMethod() == 0 {
		t.Fatalf("%T exposes no methods", service)
	}

	for index := range value.NumMethod() {
		method := value.Type().Method(index)
		if slices.Contains(skip, method.Name) {
			continue
		}
		t.Run(method.Name, func(t *testing.T) {
			signature := method.Type
			if signature.NumIn() != 3 || signature.NumOut() != 2 {
				t.Fatalf("%s takes %d arguments and returns %d values, want (ctx, request) (response, error)",
					method.Name, signature.NumIn()-1, signature.NumOut())
			}

			results := value.Method(index).Call([]reflect.Value{
				reflect.ValueOf(context.Background()),
				reflect.New(signature.In(2)).Elem(),
			})

			if !results[0].IsZero() {
				t.Errorf("%s returned %+v beside its error, want the zero response", method.Name, results[0].Interface())
			}

			err, failed := results[1].Interface().(error)
			if !failed || err == nil {
				t.Fatalf("%s returned no error", method.Name)
			}
			if got := string(wails.MarshalError(err)); got != want {
				t.Fatalf("%s converted to %s, want %s", method.Name, got, want)
			}
		})
	}
}

func assertMethodNames(t *testing.T, service any, want []string) {
	t.Helper()

	value := reflect.ValueOf(service)
	got := make([]string, 0, value.NumMethod())
	for index := range value.NumMethod() {
		got = append(got, value.Type().Method(index).Name)
	}
	if len(got) != len(want) {
		t.Fatalf("%T exposes %v, want %v", service, got, want)
	}
	for index, name := range want {
		if got[index] != name {
			t.Fatalf("%T exposes %v, want %v", service, got, want)
		}
	}
}

func ready[T any](useCase T) wails.Source[T] {
	return func() (T, error) {
		return useCase, nil
	}
}

func refused[T any]() wails.Source[T] {
	return func() (T, error) {
		var zero T
		return zero, errors.New(errors.Locked, "the application is locked")
	}
}

func TestEveryServiceRefusesWhileTheApplicationIsLocked(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	services := []any{
		wails.NewSitesService(logger, refused[wails.SitesUseCase]()),
		wails.NewGraphService(logger, refused[wails.GraphUseCase]()),
		wails.NewPagesService(logger, refused[wails.PagesUseCase]()),
		wails.NewTemplatesService(logger, refused[wails.TemplatesUseCase]()),
		wails.NewRunsService(logger, refused[wails.RunsUseCase]()),
		wails.NewSyncService(logger, refused[wails.SyncUseCase]()),
		wails.NewReportsService(logger, refused[wails.ReportsUseCase](), refused[wails.JudgeUseCase]()),
		wails.NewImportService(logger, refused[wails.ImportsUseCase]()),
		wails.NewModelsService(logger, refused[wails.ModelsUseCase]()),
		wails.NewAgentService(logger, refused[wails.AgentUseCase]()),
		wails.NewSchedulesService(logger, refused[wails.SchedulesUseCase]()),
		wails.NewToolsService(logger, refused[wails.ToolCatalog]()),
		wails.NewBrowserService(logger, refused[wails.BrowserUseCase]()),
		wails.NewSettingsService(logger, wails.SettingsDeps{
			Access: refused[wails.SettingsAccess](),
			Backup: refused[wails.BackupControl](),
			Lock:   &lockFake{locked: true, protected: true},
		}),
	}

	for _, service := range services {
		t.Run(reflect.TypeOf(service).String(), func(t *testing.T) {
			assertEveryMethodConverts(t, service, lockedBody, "Lock", "LockState", "SetMasterPassword", "Unlock")
		})
	}
}
