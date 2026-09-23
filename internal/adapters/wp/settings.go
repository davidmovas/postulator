package wp

import (
	"fmt"
	"net/url"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/settings"
)

var (
	timeoutSetting = settings.Duration("wp.timeout", DefaultTimeout, settings.DurationRange(time.Second, 5*time.Minute))
	retriesSetting = settings.Int("wp.retries", DefaultRetries, settings.IntRange(0, 10))
	rateSetting    = settings.Int("wp.rateLimitPerSecond", DefaultRateLimitPerSecond, settings.IntRange(1, 100))
	proxySetting   = settings.String("wp.proxyUrl", "", proxyURLValidator())
)

func proxyURLValidator() settings.Validator[string] {
	return settings.Check(func(value string) error {
		if value == "" {
			return nil
		}

		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" {
			return fmt.Errorf("value %q is not a valid proxy URL", value)
		}
		switch parsed.Scheme {
		case "http", "https", "socks5", "socks5h":
			return nil
		default:
			return fmt.Errorf("scheme %q is not one of http, https, socks5, socks5h", parsed.Scheme)
		}
	})
}

func FromSettings(values *settings.Values) []Option {
	return []Option{
		WithTimeout(timeoutSetting.Get(values)),
		WithRetries(retriesSetting.Get(values)),
		WithRateLimit(float64(rateSetting.Get(values))),
		WithProxy(proxySetting.Get(values)),
	}
}
