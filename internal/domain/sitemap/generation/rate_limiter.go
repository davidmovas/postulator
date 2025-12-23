package generation

import (
	"context"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"golang.org/x/time/rate"
)

type Limits struct {
	RequestsPerMin int
	BurstSize      int
}

// Default fallback limits for providers (used when model not found)
var defaultProviderLimits = map[string]Limits{
	"openai":    {RequestsPerMin: 60, BurstSize: 10},
	"anthropic": {RequestsPerMin: 50, BurstSize: 5},
	"google":    {RequestsPerMin: 60, BurstSize: 10},
	"default":   {RequestsPerMin: 30, BurstSize: 3},
}

type RateLimiter struct {
	limiters sync.Map // key: "provider:model" -> *rate.Limiter
	mu       sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{}
}

// getLimiterKey creates a unique key for provider+model combination
func getLimiterKey(provider, model string) string {
	return provider + ":" + model
}

// getModelLimits retrieves rate limits for a specific model from the ai package
func getModelLimits(providerName, modelID string) Limits {
	// Map provider name to entity type
	var providerType entities.Type
	switch providerName {
	case "openai":
		providerType = entities.TypeOpenAI
	case "anthropic":
		providerType = entities.TypeAnthropic
	case "google":
		providerType = entities.TypeGoogle
	default:
		return defaultProviderLimits["default"]
	}

	// Get model info from the ai package
	modelInfo := ai.GetModelInfo(providerType, modelID)
	if modelInfo == nil {
		// Fallback to provider defaults if model not found
		if limits, ok := defaultProviderLimits[providerName]; ok {
			return limits
		}
		return defaultProviderLimits["default"]
	}

	// Calculate burst size as 1/6 of RPM (allows 10-second bursts)
	burstSize := modelInfo.RPM / 6
	if burstSize < 1 {
		burstSize = 1
	}
	if burstSize > 20 {
		burstSize = 20 // Cap burst to prevent overwhelming the API
	}

	return Limits{
		RequestsPerMin: modelInfo.RPM,
		BurstSize:      burstSize,
	}
}

func (r *RateLimiter) getLimiter(provider, model string) *rate.Limiter {
	key := getLimiterKey(provider, model)

	if limiter, ok := r.limiters.Load(key); ok {
		return limiter.(*rate.Limiter)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring lock
	if limiter, ok := r.limiters.Load(key); ok {
		return limiter.(*rate.Limiter)
	}

	limits := getModelLimits(provider, model)
	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(limits.RequestsPerMin)), limits.BurstSize)
	r.limiters.Store(key, limiter)
	return limiter
}

// Acquire waits for rate limit permission for a specific provider and model
func (r *RateLimiter) Acquire(ctx context.Context, provider, model string) error {
	limiter := r.getLimiter(provider, model)
	return limiter.Wait(ctx)
}

// SetLimits manually sets rate limits for a provider+model combination
func (r *RateLimiter) SetLimits(provider, model string, limits Limits) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := getLimiterKey(provider, model)
	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(limits.RequestsPerMin)), limits.BurstSize)
	r.limiters.Store(key, limiter)
}

// TryAcquire attempts to acquire rate limit permission without blocking
func (r *RateLimiter) TryAcquire(provider, model string) bool {
	limiter := r.getLimiter(provider, model)
	return limiter.Allow()
}
