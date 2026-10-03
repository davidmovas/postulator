package openai

import (
	"encoding/json"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	statusCompleted  = "completed"
	statusIncomplete = "incomplete"
	statusFailed     = "failed"

	itemMessage    = "message"
	partOutputText = "output_text"
	partRefusal    = "refusal"
	emptyArguments = "{}"
)

type wireResponse struct {
	Error             *wireFault      `json:"error"`
	IncompleteDetails *wireIncomplete `json:"incomplete_details"`
	Usage             *wireUsage      `json:"usage"`
	Status            string          `json:"status"`
	ServiceTier       string          `json:"service_tier"`
	Output            []wireOutput    `json:"output"`
}

type wireIncomplete struct {
	Reason string `json:"reason"`
}

type wireOutput struct {
	Type      string        `json:"type"`
	CallID    string        `json:"call_id"`
	Name      string        `json:"name"`
	Arguments string        `json:"arguments"`
	Content   []wireContent `json:"content"`
}

type wireContent struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal"`
}

func responseOf(payload []byte, sentTier string) (port.Response, error) {
	var decoded wireResponse
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return port.Response{}, errors.New(errors.External, "the model provider answered something that is not a response").
			WithInternal(err).WithRetry(0)
	}
	return answerOf(decoded, sentTier)
}

func answerOf(decoded wireResponse, sentTier string) (port.Response, error) {
	switch decoded.Status {
	case statusFailed:
		return port.Response{}, failedOf(decoded)
	case statusCompleted, statusIncomplete, "":
	default:
		return port.Response{}, errors.New(errors.External, "the model provider returned an answer that is not finished").
			WithDetail("status", decoded.Status).WithRetry(0)
	}

	calls, err := callsOf(decoded.Output)
	if err != nil {
		return port.Response{}, err
	}
	return port.Response{
		Text:         textOf(decoded.Output),
		Usage:        usageOf(decoded.Usage),
		FinishReason: finishOf(decoded),
		Calls:        calls,
		Tier:         tierOf(decoded.ServiceTier, sentTier),
	}, nil
}

func failedOf(decoded wireResponse) *refusal {
	failed := &refusal{}
	if decoded.Error != nil {
		failed.fault = *decoded.Error
	}
	return failed
}

func textOf(output []wireOutput) string {
	var builder strings.Builder
	for _, item := range output {
		if item.Type != itemMessage {
			continue
		}
		for _, part := range item.Content {
			if part.Type == partOutputText {
				builder.WriteString(part.Text)
			}
		}
	}
	return builder.String()
}

func refused(output []wireOutput) bool {
	for _, item := range output {
		for _, part := range item.Content {
			if part.Type == partRefusal {
				return true
			}
		}
	}
	return false
}

func callsOf(output []wireOutput) ([]port.ToolCall, error) {
	var calls []port.ToolCall
	for _, item := range output {
		if item.Type != itemFunctionCall {
			continue
		}
		call, err := callOf(item)
		if err != nil {
			return nil, err
		}
		calls = append(calls, call)
	}
	return calls, nil
}

func callOf(item wireOutput) (port.ToolCall, error) {
	args := item.Arguments
	if strings.TrimSpace(args) == "" {
		args = emptyArguments
	}
	if item.CallID == "" || item.Name == "" || !json.Valid([]byte(args)) {
		return port.ToolCall{}, errors.New(errors.External, "the model asked for a tool in a shape it cannot be called with").
			WithDetail("reason", port.ReasonMalformedAnswer).
			WithDetail("tool", item.Name).
			WithRetry(0)
	}
	return port.ToolCall{ID: item.CallID, Name: item.Name, Args: json.RawMessage(args)}, nil
}
