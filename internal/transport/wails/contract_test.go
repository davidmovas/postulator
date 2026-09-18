package wails_test

import (
	"context"
	"reflect"
	"testing"

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
)

func answer[T any](mode failure) (T, error) {
	var zero T
	if mode == panicking {
		panic("postgres://user:hunter2@host/db")
	}
	return zero, errors.New(errors.NotFound, "no record carries that id").WithDetail("id", "nope")
}

func assertEveryMethodConverts(t *testing.T, service any, want string) {
	t.Helper()

	value := reflect.ValueOf(service)
	if value.NumMethod() == 0 {
		t.Fatalf("%T exposes no methods", service)
	}

	for index := range value.NumMethod() {
		method := value.Type().Method(index)
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

			err, _ := results[1].Interface().(error)
			if err == nil {
				t.Fatalf("%s returned no error", method.Name)
			}
			if got := string(wails.MarshalError(err)); got != want {
				t.Fatalf("%s marshalled %s, want %s", method.Name, got, want)
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
