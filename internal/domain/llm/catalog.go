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
	if !i.Ref.Supported() {
		return invalid("only OpenAI models can be added; Postulator no longer works with other providers", "provider")
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
	if i.CacheWriteUSDPerM < 0 {
		return invalid("a price must not be negative", "cacheWriteUsdPerM")
	}
	if err := i.validateFlex(); err != nil {
		return err
	}
	if i.RPM <= 0 {
		return invalid("the request rate limit must be positive", "rpm")
	}
	if i.TPM <= 0 {
		return invalid("the token rate limit must be positive", "tpm")
	}
	return nil
}

func (i ModelInfo) validateFlex() error {
	if i.FlexInputUSDPerM < 0 {
		return invalid("a price must not be negative", "flexInputUsdPerM")
	}
	if i.FlexCachedInputUSDPerM < 0 {
		return invalid("a price must not be negative", "flexCachedInputUsdPerM")
	}
	if i.FlexCacheWriteUSDPerM < 0 {
		return invalid("a price must not be negative", "flexCacheWriteUsdPerM")
	}
	if i.FlexOutputUSDPerM < 0 {
		return invalid("a price must not be negative", "flexOutputUsdPerM")
	}
	if i.FlexCachedInputUSDPerM > i.FlexInputUSDPerM {
		return invalid("a cached input token must not cost more than a fresh one", "flexCachedInputUsdPerM")
	}
	if i.FlexInputUSDPerM > 0 && i.FlexOutputUSDPerM == 0 {
		return invalid("a flex input price needs a flex output price beside it", "flexOutputUsdPerM")
	}
	if i.FlexOutputUSDPerM > 0 && i.FlexInputUSDPerM == 0 {
		return invalid("a flex output price needs a flex input price beside it", "flexInputUsdPerM")
	}
	if i.FlexCacheWriteUSDPerM > 0 && i.FlexInputUSDPerM == 0 {
		return invalid("a flex cache write price needs a flex input price beside it", "flexInputUsdPerM")
	}
	return nil
}

func invalid(message, field string) error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
