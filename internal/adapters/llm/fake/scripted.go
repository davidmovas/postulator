package fake

import (
	"context"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
)

type Reply struct {
	Step  string
	Match string
	Text  string
}

type Scripted struct {
	client  *Client
	replies []Reply
}

func NewScripted(replies ...Reply) *Scripted {
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
	reply, found := s.reply(req)
	if !found {
		return req
	}

	req.Messages = append(append([]port.Message(nil), req.Messages...),
		port.Message{Role: port.RoleUser, Text: JSONDirective + reply})
	return req
}

func (s *Scripted) reply(req port.Request) (string, bool) {
	prompt := ""
	if len(req.Messages) > 0 {
		prompt = req.Messages[len(req.Messages)-1].Text
	}

	for _, reply := range s.replies {
		if reply.Step != req.Meta.Step {
			continue
		}
		if reply.Match != "" && !strings.Contains(prompt, reply.Match) {
			continue
		}
		return reply.Text, true
	}
	return "", false
}
