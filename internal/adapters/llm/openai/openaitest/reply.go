package openaitest

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"time"
)

const (
	invalidRequest = "invalid_request_error"

	contentTypeJSON = "application/json"
	contentTypeSSE  = "text/event-stream; charset=utf-8"
)

type Event struct {
	Name  string
	Data  string
	Pause time.Duration
}

type Reply struct {
	Header http.Header
	Body   string
	Events []Event
	Delay  time.Duration
	Status int
}

type Fault struct {
	Type    string
	Code    string
	Message string
	Param   string
}

func Quota() Fault {
	return Fault{
		Type: "insufficient_quota", Code: "credit_balance_exhausted",
		Message: "You have no credits remaining. Add credits to continue using the API at https://platform.openai.com/settings/organization/billing/.",
	}
}

func Capacity() Fault {
	return Fault{Type: invalidRequest, Code: "resource_unavailable", Message: "Resource Unavailable"}
}

func RateLimit() Fault {
	return Fault{
		Type: "rate_limit_error", Code: "rate_limit_exceeded",
		Message: "Rate limit reached for gpt-5.6-terra in organization org-test on tokens per min (TPM): " +
			"Limit 30000, Used 29998, Requested 50. Please try again in 1.982s.",
	}
}

func ServerError() Fault {
	return Fault{Type: "server_error", Message: "The server had an error while processing your request. Sorry about that!"}
}

func Failure(status int, fault Fault) Reply {
	return Reply{Status: status, Body: encode(map[string]any{"error": fault.envelope()})}
}

func QuotaExhausted() Reply {
	return Failure(http.StatusTooManyRequests, Quota())
}

func FlexCapacity() Reply {
	return Failure(http.StatusTooManyRequests, Capacity())
}

func Stream(events ...Event) Reply {
	return Reply{Status: http.StatusOK, Events: events}
}

func StreamFailure(fault Fault) Reply {
	var builder events
	pending := response(DefaultModel, "in_progress", "auto")
	builder.add("response.created", map[string]any{"response": pending})
	builder.add("response.in_progress", map[string]any{"response": pending})
	builder.fail(fault, DefaultModel)
	return Stream(builder.list...)
}

func (r Reply) After(delay time.Duration) Reply {
	r.Delay = delay
	return r
}

func (r Reply) WithHeader(key, value string) Reply {
	header := http.Header{}
	if r.Header != nil {
		header = r.Header.Clone()
	}
	header.Add(key, value)
	r.Header = header
	return r
}

func (f Fault) envelope() map[string]any {
	return map[string]any{
		"message": f.Message,
		"type":    f.Type,
		"param":   nullable(f.Param),
		"code":    nullable(f.Code),
	}
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r Reply) serve(w http.ResponseWriter, req *http.Request) {
	if !wait(req.Context(), r.Delay) {
		return
	}

	header := w.Header()
	maps.Copy(header, r.Header)
	status := r.Status
	if status == 0 {
		status = http.StatusOK
	}

	if r.Events != nil {
		header.Set("Content-Type", contentTypeSSE)
		w.WriteHeader(status)
		r.stream(w, req)
		return
	}

	header.Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	deliver(w, r.Body)
}

func (r Reply) stream(w http.ResponseWriter, req *http.Request) {
	controller := http.NewResponseController(w)
	for _, event := range r.Events {
		if !wait(req.Context(), event.Pause) {
			return
		}
		if !deliver(w, "event: "+event.Name+"\ndata: "+event.Data+"\n\n") {
			return
		}
		if controller.Flush() != nil {
			return
		}
	}
}

func deliver(w io.Writer, chunk string) bool {
	_, err := io.WriteString(w, chunk)
	return err == nil
}

func wait(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return ctx.Err() == nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func encode(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `{"error":{"message":"the fake could not encode its reply","type":"server_error","param":null,"code":null}}`
	}
	return string(encoded)
}

type events struct {
	list []Event
}

func (e *events) add(name string, payload map[string]any) {
	data := maps.Clone(payload)
	data["type"] = name
	data["sequence_number"] = len(e.list)
	e.list = append(e.list, Event{Name: name, Data: encode(data)})
}

func (e *events) fail(fault Fault, model string) {
	nested := fault.envelope()
	e.add("error", map[string]any{"error": map[string]any{
		"type": nested["type"], "code": nested["code"], "message": nested["message"], "param": nested["param"],
	}})

	failed := response(model, "failed", "auto")
	failed["error"] = map[string]any{"code": nullable(fault.Code), "message": fault.Message}
	e.add("response.failed", map[string]any{"response": failed})
}

func response(model, status, tier string) map[string]any {
	return map[string]any{
		"id":                  "resp_test",
		"object":              "response",
		"created_at":          1791049547,
		"status":              status,
		"background":          false,
		"error":               nil,
		"incomplete_details":  nil,
		"instructions":        nil,
		"model":               model,
		"output":              []any{},
		"parallel_tool_calls": true,
		"service_tier":        tier,
		"store":               false,
		"usage":               nil,
		"metadata":            map[string]any{},
	}
}
