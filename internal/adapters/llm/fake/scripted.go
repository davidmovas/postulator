package fake

import (
	"context"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
)

type Reply struct {
	Make    func(req port.Request) string
	Step    string
	Match   string
	Text    string
	Sources []port.Source
}

type Scripted struct {
	client  *Client
	replies []Reply
}

func NewScripted(replies ...Reply) *Scripted {
	return NewScriptedOver(New(), replies...)
}

func NewScriptedOver(client *Client, replies ...Reply) *Scripted {
	return &Scripted{client: client, replies: replies}
}

func (s *Scripted) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	answer, err := s.client.Complete(ctx, s.directed(req))
	if err != nil {
		return answer, err
	}
	if reply, found := s.match(req); found && len(reply.Sources) > 0 {
		answer.Sources = append([]port.Source(nil), reply.Sources...)
	}
	return answer, nil
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
	reply, found := s.match(req)
	if !found {
		return "", false
	}
	if reply.Text == "" && reply.Make != nil {
		return reply.Make(req), true
	}
	return reply.Text, true
}

func (s *Scripted) match(req port.Request) (Reply, bool) {
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
		return reply, true
	}
	return Reply{}, false
}
