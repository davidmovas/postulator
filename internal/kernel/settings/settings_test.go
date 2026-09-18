package settings_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

func layer(pairs map[string]string) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(pairs))
	for key, value := range pairs {
		out[key] = json.RawMessage(value)
	}
	return out
}

func TestIntSettingDefaultsAndRange(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	workers := registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	if workers.Key() != "runs.workers" {
		t.Fatalf("Key() = %q", workers.Key())
	}
	if workers.Default() != 2 {
		t.Fatalf("Default() = %d, want 2", workers.Default())
	}

	values := registry.NewValues()
	if got := workers.Get(values); got != 2 {
		t.Fatalf("Get() = %d, want the default 2", got)
	}

	cases := []struct {
		name  string
		raw   string
		want  int
		valid bool
	}{
		{name: "below the minimum", raw: "0", valid: false},
		{name: "above the maximum", raw: "17", valid: false},
		{name: "at the minimum", raw: "1", want: 1, valid: true},
		{name: "at the maximum", raw: "16", want: 16, valid: true},
		{name: "inside the range", raw: "8", want: 8, valid: true},
		{name: "wrong type", raw: `"eight"`, valid: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			live := registry.NewValues()
			_, err := registry.Apply(live, layer(map[string]string{"runs.workers": tc.raw}))
			if !tc.valid {
				if !errors.IsCode(err, errors.Invalid) {
					t.Fatalf("Apply() error = %v, want code %s", err, errors.Invalid)
				}
				return
			}
			if err != nil {
				t.Fatalf("Apply() error: %v", err)
			}
			if got := workers.Get(live); got != tc.want {
				t.Fatalf("Get() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestStringBoolDurationSettings(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	name := registry.String("wp.userAgent", "postulator", settings.NonEmpty())
	enabled := registry.Bool("proxy.enabled", false)
	sweep := registry.Duration("runs.sweepInterval", 5*time.Second, settings.DurationRange(time.Second, time.Minute))
	mode := registry.Enum("agent.mode", "confirm", []string{"confirm", "autonomous"})

	live := registry.NewValues()
	unknown, err := registry.Apply(live, layer(map[string]string{
		"wp.userAgent":       `"postulator/2"`,
		"proxy.enabled":      `true`,
		"runs.sweepInterval": `"30s"`,
		"agent.mode":         `"autonomous"`,
	}))
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if len(unknown) != 0 {
		t.Fatalf("unknown = %v, want none", unknown)
	}

	if got := name.Get(live); got != "postulator/2" {
		t.Fatalf("userAgent = %q", got)
	}
	if !enabled.Get(live) {
		t.Fatal("proxy.enabled = false, want true")
	}
	if got := sweep.Get(live); got != 30*time.Second {
		t.Fatalf("sweepInterval = %v", got)
	}
	if got := mode.Get(live); got != "autonomous" {
		t.Fatalf("agent.mode = %q", got)
	}
}

func TestRejections(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	registry.String("wp.userAgent", "postulator", settings.NonEmpty())
	registry.Duration("runs.sweepInterval", 5*time.Second, settings.DurationRange(time.Second, time.Minute))
	registry.Enum("agent.mode", "confirm", []string{"confirm", "autonomous"})

	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "empty string", key: "wp.userAgent", value: `""`},
		{name: "duration below range", key: "runs.sweepInterval", value: `"100ms"`},
		{name: "duration above range", key: "runs.sweepInterval", value: `"5m"`},
		{name: "duration not parseable", key: "runs.sweepInterval", value: `"soon"`},
		{name: "enum outside the set", key: "agent.mode", value: `"chaotic"`},
		{name: "malformed json", key: "agent.mode", value: `{`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := registry.Apply(registry.NewValues(), layer(map[string]string{tc.key: tc.value}))
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Apply() error = %v, want code %s", err, errors.Invalid)
			}
		})
	}
}

func TestUnknownKeysAreReportedNotFatal(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	workers := registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	live := registry.NewValues()
	unknown, err := registry.Apply(live, layer(map[string]string{
		"runs.workers": `4`,
		"runs.retired": `"gone"`,
	}))
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if len(unknown) != 1 || unknown[0] != "runs.retired" {
		t.Fatalf("unknown = %v, want [runs.retired]", unknown)
	}
	if got := workers.Get(live); got != 4 {
		t.Fatalf("Get() = %d, want 4", got)
	}
}

func TestApplyIsAtomic(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	workers := registry.Int("runs.workers", 2, settings.IntRange(1, 16))
	perSite := registry.Int("runs.perSite", 1, settings.IntRange(1, 8))

	live := registry.NewValues()
	if _, err := registry.Apply(live, layer(map[string]string{"runs.workers": `4`})); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	if _, err := registry.Apply(live, layer(map[string]string{"runs.workers": `8`, "runs.perSite": `99`})); err == nil {
		t.Fatal("Apply() must fail on an out-of-range value")
	}
	if got := workers.Get(live); got != 4 {
		t.Fatalf("workers = %d, want the previous 4 after a rejected apply", got)
	}
	if got := perSite.Get(live); got != 1 {
		t.Fatalf("perSite = %d, want the default 1", got)
	}
}

func TestGetOnNilValuesReturnsDefault(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	workers := registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	if got := workers.Get(nil); got != 2 {
		t.Fatalf("Get(nil) = %d, want 2", got)
	}
}

func TestSchemaListsEveryDeclaredSetting(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	registry.Int("runs.workers", 2, settings.IntRange(1, 16))
	registry.Bool("proxy.enabled", false)
	registry.Enum("agent.mode", "confirm", []string{"confirm", "autonomous"})
	registry.Duration("runs.sweepInterval", 5*time.Second, settings.DurationRange(time.Second, time.Minute))
	registry.String("wp.userAgent", "postulator", settings.NonEmpty())

	schema, err := registry.Schema()
	if err != nil {
		t.Fatalf("Schema() error: %v", err)
	}
	if len(schema) != 5 {
		t.Fatalf("Schema() has %d entries, want 5", len(schema))
	}

	for i := 1; i < len(schema); i++ {
		if schema[i-1].Key >= schema[i].Key {
			t.Fatalf("Schema() is not sorted by key: %q then %q", schema[i-1].Key, schema[i].Key)
		}
	}

	byKey := make(map[string]settings.Descriptor, len(schema))
	for _, descriptor := range schema {
		byKey[descriptor.Key] = descriptor
	}

	workers := byKey["runs.workers"]
	if workers.Group != "runs" || workers.Type != "int" {
		t.Fatalf("runs.workers descriptor = %+v", workers)
	}
	if string(workers.Default) != "2" {
		t.Fatalf("runs.workers default = %s", workers.Default)
	}
	if workers.Min != 1 || workers.Max != 16 {
		t.Fatalf("runs.workers bounds = %v..%v", workers.Min, workers.Max)
	}

	mode := byKey["agent.mode"]
	if mode.Type != "enum" || len(mode.Enum) != 2 {
		t.Fatalf("agent.mode descriptor = %+v", mode)
	}

	agent := byKey["wp.userAgent"]
	if !agent.NonEmpty {
		t.Fatal("wp.userAgent must be described as non-empty")
	}

	sweep := byKey["runs.sweepInterval"]
	if sweep.Type != "duration" || string(sweep.Default) != `"5s"` {
		t.Fatalf("runs.sweepInterval descriptor = %+v", sweep)
	}
}

func TestKeys(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	registry.Int("runs.workers", 2, settings.IntRange(1, 16))
	registry.Bool("proxy.enabled", false)

	keys := registry.Keys()
	if len(keys) != 2 || keys[0] != "proxy.enabled" || keys[1] != "runs.workers" {
		t.Fatalf("Keys() = %v", keys)
	}
	if !registry.Has("runs.workers") || registry.Has("runs.missing") {
		t.Fatal("Has() is wrong")
	}
}

func TestRegistrationRejectsBadDeclarations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		declare func(*settings.Registry)
	}{
		{name: "duplicate key", declare: func(r *settings.Registry) {
			r.Int("runs.workers", 2)
			r.Int("runs.workers", 3)
		}},
		{name: "key without a group", declare: func(r *settings.Registry) { r.Int("workers", 2) }},
		{name: "empty key", declare: func(r *settings.Registry) { r.Int("", 2) }},
		{name: "default outside the range", declare: func(r *settings.Registry) {
			r.Int("runs.workers", 99, settings.IntRange(1, 16))
		}},
		{name: "enum default outside the set", declare: func(r *settings.Registry) {
			r.Enum("agent.mode", "chaotic", []string{"confirm"})
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recover() == nil {
					t.Fatal("a bad declaration must panic at registration time")
				}
			}()
			tc.declare(settings.New())
		})
	}
}

func TestPackageLevelConstructorsUseTheDefaultRegistry(t *testing.T) {
	t.Parallel()

	setting := settings.Int("kerneltest.workers", 3, settings.IntRange(1, 4))
	if !settings.Default().Has("kerneltest.workers") {
		t.Fatal("the package-level constructor must register on the default registry")
	}
	if setting.Get(settings.Default().NewValues()) != 3 {
		t.Fatal("the package-level constructor must carry its default")
	}
}

func TestPackageLevelConstructorsCoverEveryKind(t *testing.T) {
	t.Parallel()

	boolean := settings.Bool("kernelkinds.enabled", true)
	text := settings.String("kernelkinds.agent", "postulator", settings.NonEmpty())
	mode := settings.Enum("kernelkinds.mode", "confirm", []string{"confirm", "autonomous"})
	interval := settings.Duration("kernelkinds.interval", time.Minute)

	values := settings.Default().NewValues()
	if !boolean.Get(values) || text.Get(values) != "postulator" || mode.Get(values) != "confirm" || interval.Get(values) != time.Minute {
		t.Fatal("package-level constructors must carry their defaults")
	}

	kinds := map[settings.Kind]string{
		boolean.Kind():  "bool",
		text.Kind():     "string",
		mode.Kind():     "enum",
		interval.Kind(): "duration",
	}
	for kind, want := range kinds {
		if kind.String() != want {
			t.Fatalf("Kind.String() = %q, want %q", kind.String(), want)
		}
	}
	if settings.Kind(200).String() != "unknown" {
		t.Fatalf("an unregistered kind must render as unknown, got %q", settings.Kind(200).String())
	}
}

func TestBoundValidatorsReportTheirOwnSide(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	registry.Int("runs.floor", 5, settings.IntMin(1))
	registry.Int("runs.ceiling", 5, settings.IntMax(9))

	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "below the floor", key: "runs.floor", value: `0`},
		{name: "above the ceiling", key: "runs.ceiling", value: `10`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := registry.Apply(registry.NewValues(), layer(map[string]string{tc.key: tc.value}))
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Apply() error = %v, want code %s", err, errors.Invalid)
			}
		})
	}

	for _, key := range []string{"runs.floor", "runs.ceiling"} {
		if _, err := registry.Apply(registry.NewValues(), layer(map[string]string{key: `5`})); err != nil {
			t.Fatalf("Apply(%s) error: %v", key, err)
		}
	}
}

func TestMalformedValuesPerKind(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	registry.Bool("proxy.enabled", false)
	registry.Int("runs.workers", 2)
	registry.String("wp.userAgent", "postulator")

	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "bool from string", key: "proxy.enabled", value: `"yes"`},
		{name: "int from object", key: "runs.workers", value: `{}`},
		{name: "string from number", key: "wp.userAgent", value: `7`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := registry.Apply(registry.NewValues(), layer(map[string]string{tc.key: tc.value}))
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Apply() error = %v, want code %s", err, errors.Invalid)
			}
		})
	}
}

func TestKeyWithEmptySegmentIsRejected(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("a key with an empty segment must panic at registration time")
		}
	}()
	settings.New().Int("runs..workers", 1)
}

func TestValidateChecksOneDeclaredKey(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	registry.Int("runs.workers", 2, settings.IntRange(1, 16))
	registry.Enum("llm.mode", "live", []string{"live", "replay"})

	cases := []struct {
		name  string
		key   string
		value string
		want  errors.Code
	}{
		{name: "a value in range", key: "runs.workers", value: `4`, want: ""},
		{name: "a declared enum member", key: "llm.mode", value: `"replay"`, want: ""},
		{name: "an undeclared key", key: "runs.nope", value: `4`, want: errors.NotFound},
		{name: "a value of the wrong type", key: "runs.workers", value: `"four"`, want: errors.Invalid},
		{name: "a value out of range", key: "runs.workers", value: `99`, want: errors.Invalid},
		{name: "a value outside the enum", key: "llm.mode", value: `"record"`, want: errors.Invalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := registry.Validate(tc.key, json.RawMessage(tc.value))
			if tc.want == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want no error", err)
				}
				return
			}
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("Validate() = %v, want code %s", err, tc.want)
			}
		})
	}
}

func TestValidateDoesNotChangeTheLiveValues(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	workers := registry.Int("runs.workers", 2, settings.IntRange(1, 16))
	values := registry.NewValues()

	if err := registry.Validate("runs.workers", json.RawMessage(`8`)); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got := workers.Get(values); got != 2 {
		t.Fatalf("Get() = %d, want the default 2 until Apply runs", got)
	}
}
