package gollemclient

import (
	"fmt"
	"net/url"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const DefaultTimeout = 2 * time.Minute

var (
	timeoutSetting   = settings.Duration("llm.timeout", DefaultTimeout, settings.DurationRange(5*time.Second, 30*time.Minute))
	openaiBaseURL    = settings.String("llm.openai.baseUrl", "", baseURLValidator())
	anthropicBaseURL = settings.String("llm.anthropic.baseUrl", "", baseURLValidator())
	geminiProject    = settings.String("llm.gemini.projectId", "")
	geminiLocation   = settings.String("llm.gemini.location", "")
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
