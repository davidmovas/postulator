package wp

import (
	"math"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	DefaultTimeout            = 30 * time.Second
	DefaultRetries            = 4
	DefaultRateLimitPerSecond = 4

	backoffBase     = 500 * time.Millisecond
	backoffCap      = 8 * time.Second
	backoffMaxShift = 5
)

type Logger interface {
	Debug(message string, fields ...zap.Field)
	Warn(message string, fields ...zap.Field)
}

type options struct {
	backoff  func(int) time.Duration
	logger   Logger
	proxyURL string
	timeout  time.Duration
	rate     float64
	retries  int
}

type Option func(*options)

func WithProxy(rawURL string) Option {
	return func(o *options) { o.proxyURL = rawURL }
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *options) { o.timeout = timeout }
}

func WithRetries(retries int) Option {
	return func(o *options) { o.retries = retries }
}

func WithBackoff(backoff func(attempt int) time.Duration) Option {
	return func(o *options) {
		if backoff != nil {
			o.backoff = backoff
		}
	}
}

func WithRateLimit(perSecond float64) Option {
	return func(o *options) { o.rate = perSecond }
}

func WithLogger(logger Logger) Option {
	return func(o *options) {
		if logger != nil {
			o.logger = logger
		}
	}
}

func defaultBackoff(attempt int) time.Duration {
	if attempt < 0 {
		return backoffBase
	}
	if attempt > backoffMaxShift {
		return backoffCap
	}
	delay := backoffBase << attempt
	if delay > backoffCap {
		return backoffCap
	}
	return delay
}

func newLimiter(perSecond float64) *rate.Limiter {
	if perSecond <= 0 {
		return rate.NewLimiter(rate.Inf, 1)
	}
	return rate.NewLimiter(rate.Limit(perSecond), int(math.Ceil(perSecond)))
}
