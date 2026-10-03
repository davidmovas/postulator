package openai

import (
	"fmt"
	"net/url"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	leastTimeout = 5 * time.Second
	mostTimeout  = 30 * time.Minute
)

var (
	timeoutSetting = settings.Duration("llm.timeout", DefaultTimeout, settings.DurationRange(leastTimeout, mostTimeout))
	baseURLSetting = settings.String("llm.openai.baseUrl", "", baseURLValidator())
)

func baseURLValidator() settings.Validator[string] {
	return settings.Check(func(value string) error {
		if value == "" {
			return nil
		}

		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" {
			return fmt.Errorf("value %q is not a valid base URL", value)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("scheme %q is not one of http, https", parsed.Scheme)
		}
		return nil
	})
}

func Timeout(values *settings.Values) time.Duration {
	return timeoutSetting.Get(values)
}

func BaseURL(values *settings.Values) string {
	return baseURLSetting.Get(values)
}
