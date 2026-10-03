package openai

import (
	"encoding/json"
	stderrors "errors"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	rejectedRequest = "the model provider rejected the request"
	exhaustedOutput = "the model used its whole output budget before it answered"
	quotaMessage    = "the OpenAI account is out of credit or over its spending limit; " +
		"add credits or raise the limit in the OpenAI billing settings, then try again"

	codeModelNotFound   = "model_not_found"
	codeContextTooLong  = "context_length_exceeded"
	codeRateLimit       = "rate_limit_exceeded"
	codeInvalidKey      = "invalid_api_key"
	typeRateLimit       = "rate_limit_error"
	typeInvalidRequest  = "invalid_request_error"
	typeAuthentication  = "authentication_error"
	maxStatedDelay      = 2 * time.Minute
	maskedKeyPrefix     = "sk-"
	maskedKeyEllipsis   = "…"
	maskedKeyVisibleEnd = 4
)

var (
	quotaCodes = []string{
		"credit_balance_exhausted", "insufficient_quota", "organization_spend_limit_exceeded",
		"project_spend_limit_exceeded", "organization_usage_limit_exceeded",
	}
	ceilingCodes       = []string{"integer_below_min_value", "integer_above_max_value"}
	rateLimitBuckets   = []string{"requests", "tokens"}
	keyPattern         = regexp.MustCompile(`sk-[A-Za-z0-9_*-]{8,}`)
	statedDelayPattern = regexp.MustCompile(`(?i)try again in ((?:\d+(?:\.\d+)?(?:ms|s|m|h))+)`)
)

type wireFault struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

type refusal struct {
	header http.Header
	fault  wireFault
	status int
}

func (r *refusal) Error() string {
	return rejectedRequest
}

func refusalOf(status int, header http.Header, payload []byte) *refusal {
	refused := &refusal{status: status, header: header}

	var envelope struct {
		Error *wireFault `json:"error"`
	}
	if json.Unmarshal(payload, &envelope) == nil && envelope.Error != nil {
		refused.fault = *envelope.Error
	}
	return refused
}

func refusalIn(err error) *refusal {
	var refused *refusal
	if stderrors.As(err, &refused) {
		return refused
	}
	return nil
}

func (r *refusal) effectiveStatus() int {
	if r.status != 0 {
		return r.status
	}
	return inferredStatus(r.fault)
}

func inferredStatus(fault wireFault) int {
	switch {
	case exhausted(fault), fault.Code == codeRateLimit, fault.Type == typeRateLimit:
		return http.StatusTooManyRequests
	case fault.Code == codeModelNotFound:
		return http.StatusNotFound
	case fault.Code == codeInvalidKey, fault.Type == typeAuthentication:
		return http.StatusUnauthorized
	case fault.Type == typeInvalidRequest:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func exhausted(fault wireFault) bool {
	return slices.Contains(quotaCodes, fault.Code) || slices.Contains(quotaCodes, fault.Type)
}

func (r *refusal) exhaustedOutput() bool {
	if r.effectiveStatus() != http.StatusBadRequest || slices.Contains(ceilingCodes, r.fault.Code) {
		return false
	}
	told := strings.ToLower(r.fault.Message)
	if !strings.Contains(told, "max_tokens") && !strings.Contains(told, "max_output_tokens") {
		return false
	}
	return strings.Contains(told, "limit") || strings.Contains(told, "reached")
}

type verdict struct {
	message   string
	code      errors.Code
	retryable bool
}

func (r *refusal) verdict() verdict {
	status := r.effectiveStatus()
	switch {
	case exhausted(r.fault):
		return verdict{code: errors.NeedsHuman, message: quotaMessage}
	case status == http.StatusUnauthorized:
		return verdict{code: errors.Unauthorized, message: "the model provider rejected the api key"}
	case status == http.StatusForbidden:
		return verdict{code: errors.Unauthorized, message: "the key has no access to this model"}
	case status == http.StatusNotFound && r.fault.Code == codeModelNotFound:
		return verdict{code: errors.NotFound, message: "the model provider has no such model"}
	case status == http.StatusNotFound:
		return verdict{code: errors.NotFound, message: "the model provider could not find what the request named"}
	case status == http.StatusRequestTimeout, status == http.StatusConflict:
		return verdict{code: errors.External, message: "the model provider could not take the request just now", retryable: true}
	case status == http.StatusTooManyRequests:
		return verdict{code: errors.RateLimited, message: "the model provider is rate limiting this key", retryable: true}
	case status >= http.StatusInternalServerError:
		return verdict{code: errors.External, message: "the model provider returned a server error", retryable: true}
	case status >= http.StatusBadRequest && r.fault.Code == codeContextTooLong:
		return verdict{code: errors.Invalid, message: "the prompt does not fit the model context window"}
	case status >= http.StatusBadRequest && r.exhaustedOutput():
		return verdict{code: errors.Invalid, message: exhaustedOutput}
	case status >= http.StatusBadRequest:
		return verdict{code: errors.Invalid, message: rejectedRequest}
	default:
		return verdict{code: errors.External, message: "the model provider returned an unexpected status"}
	}
}

func (r *refusal) kernel(ref llm.ModelRef, now time.Time) error {
	judged := r.verdict()
	built := errors.New(judged.code, judged.message).WithDetail("model", ref.String())
	if r.status != 0 {
		built = built.WithDetail("status", r.status)
	}
	if told := maskKeys(strings.TrimSpace(r.fault.Message)); told != "" {
		built = built.WithDetail("providerMessage", told)
	}
	if r.fault.Code != "" {
		built = built.WithDetail("code", r.fault.Code)
	}
	if r.fault.Param != "" {
		built = built.WithDetail("param", r.fault.Param)
	}
	if judged.retryable {
		built = built.WithRetry(retryDelay(r.header, r.fault.Message, now))
	}
	return built
}

func retryDelay(header http.Header, message string, now time.Time) time.Duration {
	for _, delay := range []time.Duration{
		scaled(header.Get("retry-after-ms"), time.Millisecond),
		retryAfterOf(header.Get("Retry-After"), now),
		resetOf(header),
		statedDelay(message),
	} {
		if delay > 0 && delay <= maxStatedDelay {
			return delay
		}
	}
	return 0
}

func scaled(raw string, unit time.Duration) time.Duration {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || value <= 0 || value > float64(maxStatedDelay)/float64(unit) {
		return 0
	}
	return time.Duration(value * float64(unit))
}

func retryAfterOf(raw string, now time.Time) time.Duration {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	if seconds := scaled(trimmed, time.Second); seconds > 0 {
		return seconds
	}
	when, err := http.ParseTime(trimmed)
	if err != nil {
		return 0
	}
	return max(when.Sub(now), 0)
}

func resetOf(header http.Header) time.Duration {
	var longest time.Duration
	for _, bucket := range rateLimitBuckets {
		if strings.TrimSpace(header.Get("x-ratelimit-remaining-"+bucket)) != "0" {
			continue
		}
		reset, err := time.ParseDuration(strings.TrimSpace(header.Get("x-ratelimit-reset-" + bucket)))
		if err == nil && reset > longest {
			longest = reset
		}
	}
	return longest
}

func statedDelay(text string) time.Duration {
	match := statedDelayPattern.FindStringSubmatch(text)
	if match == nil {
		return 0
	}
	delay, err := time.ParseDuration(strings.ToLower(match[1]))
	if err != nil {
		return 0
	}
	return delay
}

func maskKeys(text string) string {
	return keyPattern.ReplaceAllStringFunc(text, func(token string) string {
		return maskedKeyPrefix + maskedKeyEllipsis + token[len(token)-maskedKeyVisibleEnd:]
	})
}
