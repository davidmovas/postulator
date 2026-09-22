package llm

import (
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type ModelOverride struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Info      ModelInfo
	Enabled   bool
}

func (i ModelInfo) Validate() error {
	if !i.Ref.Valid() {
		return invalid("a model entry must name a provider and a model", "ref")
	}
	if i.ContextTokens <= 0 {
		return invalid("the context window must be positive", "contextTokens")
	}
	if i.MaxOutputTokens <= 0 {
		return invalid("the output ceiling must be positive", "maxOutputTokens")
	}
	if i.MaxOutputTokens > i.ContextTokens {
		return invalid("the output ceiling must fit inside the context window", "maxOutputTokens")
	}
	if i.InputUSDPerM < 0 || i.OutputUSDPerM < 0 {
		return invalid("a price must not be negative", "inputUsdPerM")
	}
	if i.CachedInputUSDPerM < 0 {
		return invalid("a price must not be negative", "cachedInputUsdPerM")
	}
	if i.CachedInputUSDPerM > i.InputUSDPerM {
		return invalid("a cached input token must not cost more than a fresh one", "cachedInputUsdPerM")
	}
	if i.RPM <= 0 {
		return invalid("the request rate limit must be positive", "rpm")
	}
	if i.TPM <= 0 {
		return invalid("the token rate limit must be positive", "tpm")
	}
	if i.ReasoningEffort != "" && !i.ReasoningEffort.Valid() {
		return invalid("the reasoning effort must be none, low, medium, high or xhigh", "reasoningEffort")
	}
	return nil
}

func invalid(message, field string) error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
