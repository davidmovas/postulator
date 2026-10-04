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

type options struct {
	script Script
}

type Option func(*options)

func WithScript(script Script) Option {
	return func(o *options) { o.script = script }
}

func settle(opts []Option) options {
	var chosen options
	for _, opt := range opts {
		opt(&chosen)
	}
	return chosen
}

type Client struct {
	script   Script
	requests []port.Request
	mu       sync.Mutex
}

func New(opts ...Option) *Client {
	return &Client{script: settle(opts).script}
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
	return c.answer(req)
}

func (c *Client) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	if err := c.accept(ctx, req); err != nil {
		return nil, err
	}

	resp, err := c.answer(req)
	if err != nil {
		return nil, err
	}

	pieces := chunks(resp.Text)
	out := make(chan port.Delta, len(pieces)+len(resp.Calls)+1)
	for _, piece := range pieces {
		out <- port.Delta{Text: piece}
	}
	for i := range resp.Calls {
		out <- port.Delta{Call: &resp.Calls[i]}
	}
	out <- port.Delta{Done: true, Usage: &resp.Usage, Finish: resp.FinishReason, Tier: resp.Tier}
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

func (c *Client) answer(req port.Request) (port.Response, error) {
	if len(req.Tools) > 0 {
		return c.converse(req)
	}

	text, reason, err := script(req)
	if err != nil {
		return port.Response{}, err
	}
	return port.Response{Text: text, Usage: usageOf(req, tokens(text)), FinishReason: reason, Tier: served(req.Tier)}, nil
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

func served(asked llm.ServiceTier) llm.ServiceTier {
	if asked == llm.TierFlex {
		return llm.TierFlex
	}
	return llm.TierDefault
}

func usageOf(req port.Request, output int) llm.Usage {
	input := tokens(req.System)
	for _, message := range req.Messages {
		switch {
		case message.Call != nil:
			input += tokens(message.Call.Name + string(message.Call.Args))
		case message.Result != nil:
			input += tokens(string(message.Result.Output))
		default:
			input += tokens(message.Text)
		}
	}
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
