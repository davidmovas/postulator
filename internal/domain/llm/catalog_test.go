package llm_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func validInfo() llm.ModelInfo {
	return llm.ModelInfo{
		Ref:             llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
		ContextTokens:   1050000,
		MaxOutputTokens: 128000,
		InputUSDPerM:    2,
		OutputUSDPerM:   12,
		RPM:             60,
		TPM:             120000,
	}
}

func TestModelInfoValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		mutate  func(*llm.ModelInfo)
		wantErr bool
	}{
		{name: "a catalog entry is valid"},
		{name: "the reference is required", mutate: func(i *llm.ModelInfo) { i.Ref.Model = "" }, wantErr: true},
		{name: "the context window is positive", mutate: func(i *llm.ModelInfo) { i.ContextTokens = 0 }, wantErr: true},
		{name: "the output ceiling is positive", mutate: func(i *llm.ModelInfo) { i.MaxOutputTokens = 0 }, wantErr: true},
		{name: "the output fits the context", mutate: func(i *llm.ModelInfo) { i.MaxOutputTokens = i.ContextTokens + 1 }, wantErr: true},
		{name: "a price is not negative", mutate: func(i *llm.ModelInfo) { i.InputUSDPerM = -1 }, wantErr: true},
		{name: "the output price is not negative", mutate: func(i *llm.ModelInfo) { i.OutputUSDPerM = -1 }, wantErr: true},
		{name: "the request rate is positive", mutate: func(i *llm.ModelInfo) { i.RPM = 0 }, wantErr: true},
		{name: "the token rate is positive", mutate: func(i *llm.ModelInfo) { i.TPM = 0 }, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			info := validInfo()
			if tc.mutate != nil {
				tc.mutate(&info)
			}

			err := info.Validate()
			if tc.wantErr != (err != nil) {
				t.Fatalf("Validate() = %v, want error %t", err, tc.wantErr)
			}
			if tc.wantErr && !errors.IsCode(err, errors.Invalid) {
				t.Errorf("Validate() code = %s, want %s", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestCallValidate(t *testing.T) {
	t.Parallel()

	valid := llm.Call{
		ID:        "6fb0b1d6-1ad5-4a26-8f26-2f2b6d9e4b23",
		Ref:       llm.ModelRef{Provider: "openai", Model: "gpt-5.6-luna"},
		Status:    llm.CallOK,
		CreatedAt: time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC),
	}

	cases := []struct {
		name    string
		mutate  func(*llm.Call)
		wantErr bool
	}{
		{name: "a recorded call is valid"},
		{name: "the identifier is required", mutate: func(c *llm.Call) { c.ID = "" }, wantErr: true},
		{name: "the reference is required", mutate: func(c *llm.Call) { c.Ref = llm.ModelRef{} }, wantErr: true},
		{name: "the status is known", mutate: func(c *llm.Call) { c.Status = "maybe" }, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			call := valid
			if tc.mutate != nil {
				tc.mutate(&call)
			}

			err := call.Validate()
			if tc.wantErr != (err != nil) {
				t.Fatalf("Validate() = %v, want error %t", err, tc.wantErr)
			}
		})
	}
}

func TestCallStatusValid(t *testing.T) {
	t.Parallel()

	if !llm.CallOK.Valid() || !llm.CallError.Valid() || llm.CallStatus("queued").Valid() {
		t.Fatal("CallStatus.Valid does not accept exactly ok and error")
	}
}
