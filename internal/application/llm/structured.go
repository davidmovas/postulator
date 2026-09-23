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
	structuredAttempts = 2
	jsonInstruction    = "Respond with a single JSON object that matches the requested schema. Emit no prose, no explanation and no code fence."
	repairInstruction  = "That response could not be decoded as JSON matching the schema. Reply again with the corrected JSON object only. The decoder reported: "
)

func Structured[T any](ctx context.Context, client Client, req Request) (T, domain.Usage, error) {
	var zero T

	schema, err := SchemaFor[T]()
	if err != nil {
		return zero, domain.Usage{}, err
	}
	req.Schema = schema
	req.System = withInstruction(req.System)

	var (
		usage    domain.Usage
		lastErr  error
		messages = slices.Clone(req.Messages)
	)
	for attempt := range structuredAttempts {
		if attempt > 0 {
			messages = append(slices.Clone(messages), Message{Role: RoleUser, Text: repairInstruction + lastErr.Error()})
		}
		req.Messages = messages

		resp, callErr := client.Complete(ctx, req)
		if callErr != nil {
			return zero, usage, callErr
		}
		usage = usage.Add(resp.Usage)

		var decoded T
		if lastErr = json.Unmarshal([]byte(resp.Text), &decoded); lastErr == nil {
			return decoded, usage, nil
		}
	}

	return zero, usage, errors.New(errors.Invalid, "the model did not return json matching the schema").
		WithDetail("attempts", structuredAttempts).
		WithDetail("model", req.Ref.String()).
		WithInternal(lastErr)
}

func withInstruction(system string) string {
	trimmed := strings.TrimSpace(system)
	if trimmed == "" {
		return jsonInstruction
	}
	return trimmed + "\n\n" + jsonInstruction
}
