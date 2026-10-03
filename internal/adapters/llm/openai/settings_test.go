package openai_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

func TestTheConnectionSettingsCarryTheirDefaults(t *testing.T) {
	t.Parallel()

	values := settings.Default().NewValues()
	if got := openai.Timeout(values); got != openai.DefaultTimeout {
		t.Fatalf("Timeout = %s, want %s", got, openai.DefaultTimeout)
	}
	if got := openai.BaseURL(values); got != "" {
		t.Fatalf("BaseURL = %q, want none so the client keeps its own", got)
	}
}

func TestTheConnectionSettingsTakeWhatIsStored(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		stored   map[string]json.RawMessage
		wantBase string
		wantTime time.Duration
		refused  bool
	}{
		{
			name:     "a proxy and a longer wait",
			stored:   map[string]json.RawMessage{"llm.openai.baseUrl": json.RawMessage(`"https://proxy.example/v1"`), "llm.timeout": json.RawMessage(`"5m"`)},
			wantBase: "https://proxy.example/v1",
			wantTime: 5 * time.Minute,
		},
		{
			name:     "a local endpoint over http",
			stored:   map[string]json.RawMessage{"llm.openai.baseUrl": json.RawMessage(`"http://127.0.0.1:8080/v1"`)},
			wantBase: "http://127.0.0.1:8080/v1",
			wantTime: openai.DefaultTimeout,
		},
		{name: "an address without a host", stored: map[string]json.RawMessage{"llm.openai.baseUrl": json.RawMessage(`"/v1"`)}, refused: true},
		{name: "an address in another scheme", stored: map[string]json.RawMessage{"llm.openai.baseUrl": json.RawMessage(`"ftp://proxy.example"`)}, refused: true},
		{name: "a wait too short to answer", stored: map[string]json.RawMessage{"llm.timeout": json.RawMessage(`"1s"`)}, refused: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			values := settings.Default().NewValues()
			_, err := settings.Default().Apply(values, tc.stored)
			if tc.refused {
				if err == nil {
					t.Fatal("Apply accepted a value the setting refuses")
				}
				return
			}
			if err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if got := openai.BaseURL(values); got != tc.wantBase {
				t.Fatalf("BaseURL = %q, want %q", got, tc.wantBase)
			}
			if got := openai.Timeout(values); got != tc.wantTime {
				t.Fatalf("Timeout = %s, want %s", got, tc.wantTime)
			}
		})
	}
}
