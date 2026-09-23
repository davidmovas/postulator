package fake

import (
	"context"
	"strings"
	"sync"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	AnswerDirective   = "ANSWER:"
	JSONDirective     = "JSON:"
	ErrorDirective    = "ERROR:"
	LengthDirective   = "LENGTH"
	FilteredDirective = "FILTERED"

	charactersPerToken = 4
	defaultAnswer      = "ok"
)

type Client struct {
	requests []port.Request
	mu       sync.Mutex
}

func New() *Client {
	return &Client{}
}

func (c *Client) Requests() []port.Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]port.Request(nil), c.requests...)
}

func (c *Client) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	if err := c.accept(ctx, req); err != nil {
		return port.Response{}, err
	}

	text, reason, err := script(req)
	if err != nil {
		return port.Response{}, err
	}
	return port.Response{Text: text, Usage: usageOf(req, text), FinishReason: reason}, nil
}

func (c *Client) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	if err := c.accept(ctx, req); err != nil {
		return nil, err
	}

	text, _, err := script(req)
	if err != nil {
		return nil, err
	}

	usage := usageOf(req, text)
	out := make(chan port.Delta, len(text)+1)
	for _, word := range chunks(text) {
		out <- port.Delta{Text: word}
	}
	out <- port.Delta{Done: true, Usage: &usage}
	close(out)
	return out, nil
}

func (c *Client) accept(ctx context.Context, req port.Request) error {
	if err := ctx.Err(); err != nil {
		return errors.New(errors.Cancelled, "the scripted model call was cancelled").WithInternal(err)
	}
	if err := req.Validate(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, req)
	return nil
}

func script(req port.Request) (text string, reason port.FinishReason, err error) {
	prompt := req.Messages[len(req.Messages)-1].Text
	reason = port.FinishStop
	text = defaultAnswer

	for line := range strings.SplitSeq(prompt, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, ErrorDirective):
			code := errors.Code(strings.TrimSpace(strings.TrimPrefix(trimmed, ErrorDirective)))
			return "", "", errors.New(code, "the scripted model failed on purpose").WithDetail("model", req.Ref.String())
		case strings.HasPrefix(trimmed, AnswerDirective):
			text = strings.TrimSpace(strings.TrimPrefix(trimmed, AnswerDirective))
		case strings.HasPrefix(trimmed, JSONDirective):
			text = strings.TrimSpace(strings.TrimPrefix(trimmed, JSONDirective))
		case trimmed == LengthDirective:
			reason = port.FinishLength
		case trimmed == FilteredDirective:
			return "", port.FinishContentFilter, nil
		}
	}
	return text, reason, nil
}

func usageOf(req port.Request, text string) llm.Usage {
	input := tokens(req.System)
	for _, message := range req.Messages {
		input += tokens(message.Text)
	}
	output := tokens(text)
	return llm.Usage{Input: input, Output: output, Total: input + output}
}

func tokens(text string) int {
	if text == "" {
		return 0
	}
	return len(text)/charactersPerToken + 1
}

func chunks(text string) []string {
	if text == "" {
		return nil
	}

	words := strings.Fields(text)
	out := make([]string, 0, len(words))
	for i, word := range words {
		if i > 0 {
			word = " " + word
		}
		out = append(out, word)
	}
	return out
}
