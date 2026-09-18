package fake

import (
	"context"

	port "github.com/davidmovas/postulator/internal/application/llm"
)

type Scripted struct {
	client  *Client
	replies map[string]string
}

func NewScripted(replies map[string]string) *Scripted {
	return &Scripted{client: New(), replies: replies}
}

func (s *Scripted) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	return s.client.Complete(ctx, s.directed(req))
}

func (s *Scripted) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	return s.client.Stream(ctx, s.directed(req))
}

func (s *Scripted) Requests() []port.Request {
	return s.client.Requests()
}

func (s *Scripted) CallsTo(step string) int {
	requests := s.client.Requests()

	total := 0
	for i := range requests {
		if requests[i].Meta.Step == step {
			total++
		}
	}
	return total
}

func (s *Scripted) directed(req port.Request) port.Request {
	reply, ok := s.replies[req.Meta.Step]
	if !ok {
		return req
	}

	req.Messages = append(append([]port.Message(nil), req.Messages...),
		port.Message{Role: port.RoleUser, Text: JSONDirective + reply})
	return req
}
