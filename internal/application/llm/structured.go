package llm

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	domain "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ReasonOutputTruncated = "output_truncated"
	ReasonMalformedAnswer = "malformed_answer"
	ReasonContentFilter   = "content_filter"

	jsonInstruction   = "Respond with a single JSON object that matches the requested schema. Emit no prose, no explanation and no code fence."
	repairInstruction = "That answer could not be decoded as JSON matching the schema. Reply again with the corrected JSON object only. The decoder reported: "
)

func Structured[T any](ctx context.Context, client Client, req Request) (T, domain.Usage, error) {
	var zero T

	schema, err := SchemaFor[T]()
	if err != nil {
		return zero, domain.Usage{}, err
	}
	req.Schema = schema
	req.System = withInstruction(req.System)
	messages := slices.Clone(req.Messages)

	var usage domain.Usage
	for round := range 2 {
		req.Messages = messages
		resp, callErr := client.Complete(ctx, req)
		if callErr != nil {
			return zero, usage, callErr
		}
		usage = usage.Add(resp.Usage)

		if refusal := unusable(resp, req); refusal != nil {
			return zero, usage, refusal
		}

		var decoded T
		decodeErr := json.Unmarshal([]byte(resp.Text), &decoded)
		if decodeErr == nil {
			return decoded, usage, nil
		}
		if round == 0 {
			messages = append(slices.Clone(messages),
				Message{Role: RoleAssistant, Text: resp.Text},
				Message{Role: RoleUser, Text: repairInstruction + decodeErr.Error()},
			)
			continue
		}
		return zero, usage, errors.New(errors.External, "the model did not answer in the shape it was asked for, twice").
			WithDetail("reason", ReasonMalformedAnswer).
			WithDetail("model", req.Ref.String()).
			WithInternal(decodeErr)
	}
	return zero, usage, nil
}

func unusable(resp Response, req Request) error {
	switch resp.FinishReason {
	case FinishContentFilter:
		return errors.New(errors.NeedsHuman, "the model provider refused to write this content; change the brief or the template before trying again").
			WithDetail("reason", ReasonContentFilter).
			WithDetail("model", req.Ref.String())
	case FinishLength:
		return errors.New(errors.External, "the model stopped before it finished the answer, so more room is needed").
			WithDetail("reason", ReasonOutputTruncated).
			WithDetail("model", req.Ref.String()).
			WithDetail("maxTokens", req.MaxTokens)
	default:
		return nil
	}
}

func withInstruction(system string) string {
	trimmed := strings.TrimSpace(system)
	if trimmed == "" {
		return jsonInstruction
	}
	return trimmed + "\n\n" + jsonInstruction
}
