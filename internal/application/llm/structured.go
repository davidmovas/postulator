package llm

import (
	"context"
	"encoding/json"

	domain "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ReasonOutputTruncated = "output_truncated"
	ReasonMalformedAnswer = "malformed_answer"
	ReasonContentFilter   = "content_filter"
)

func Structured[T any](ctx context.Context, client Client, req Request) (T, domain.Usage, error) {
	var zero T

	schema, err := SchemaFor[T]()
	if err != nil {
		return zero, domain.Usage{}, err
	}
	req.Schema = schema

	resp, err := client.Complete(ctx, req)
	if err != nil {
		return zero, domain.Usage{}, err
	}
	if refusal := unusable(resp, req); refusal != nil {
		return zero, resp.Usage, refusal
	}

	var decoded T
	if decodeErr := json.Unmarshal([]byte(resp.Text), &decoded); decodeErr != nil {
		return zero, resp.Usage, errors.New(errors.External, "the model did not answer in the shape it was asked for").
			WithDetail("reason", ReasonMalformedAnswer).
			WithDetail("model", req.Ref.String()).
			WithInternal(decodeErr)
	}
	return decoded, resp.Usage, nil
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
