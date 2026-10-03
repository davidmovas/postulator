package openai

import (
	"net/http"
	"testing"
)

func TestAFlexCapacityRefusalIsToldApartFromEveryOther429(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		fault wireFault
		want  bool
	}{
		{name: "the code the docs imply", fault: wireFault{Type: "invalid_request_error", Code: "resource_unavailable", Message: "Resource Unavailable"}, want: true},
		{name: "the code as the type", fault: wireFault{Type: "resource_unavailable"}, want: true},
		{name: "only the documented sentence", fault: wireFault{Message: "Resource Unavailable"}, want: true},
		{name: "a sentence about flex capacity", fault: wireFault{Type: "rate_limit_error", Message: "Flex processing is at capacity right now. Try again later or use the default tier."}, want: true},
		{name: "a capacity code", fault: wireFault{Code: "flex_capacity_exceeded"}, want: true},
		{name: "the probed credit refusal", fault: wireFault{Type: "insufficient_quota", Code: "credit_balance_exhausted", Message: "You have no credits remaining."}},
		{name: "a spend limit that mentions capacity", fault: wireFault{Type: "insufficient_quota", Code: "project_spend_limit_exceeded", Message: "capacity"}},
		{name: "an ordinary rate limit", fault: wireFault{Type: "rate_limit_error", Code: "rate_limit_exceeded", Message: "Rate limit reached for gpt-5.6-terra. Please try again in 1.982s."}},
		{name: "a request limit", fault: wireFault{Type: "requests", Code: "rate_limit_exceeded", Message: "Rate limit reached for requests"}},
		{name: "a body that says nothing", fault: wireFault{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := outOfCapacity(tc.fault); got != tc.want {
				t.Errorf("outOfCapacity = %t, want %t", got, tc.want)
			}
			over429 := &refusal{status: http.StatusTooManyRequests, fault: tc.fault}
			if got := capacityRefusal(over429); got != tc.want {
				t.Errorf("capacityRefusal over 429 = %t, want %t", got, tc.want)
			}
			if capacityRefusal(&refusal{status: http.StatusServiceUnavailable, fault: tc.fault}) {
				t.Error("a 503 was taken for a capacity refusal")
			}
		})
	}
}

func TestAnInStreamCapacityRefusalCountsAsA429(t *testing.T) {
	t.Parallel()

	inStream := &refusal{fault: wireFault{Type: "invalid_request_error", Code: "resource_unavailable", Message: "Resource Unavailable"}}
	if !capacityRefusal(inStream) || inStream.effectiveStatus() != http.StatusTooManyRequests {
		t.Fatalf("an in-stream capacity refusal was read as %d", inStream.effectiveStatus())
	}
	if capacityRefusal(nil) {
		t.Error("no failure was taken for a capacity refusal")
	}
}
