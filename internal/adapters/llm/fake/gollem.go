package fake

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync"

	"github.com/gollem-dev/gollem"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ToolDirective  = "TOOL:"
	FinalDirective = "FAKE:"
	FailDirective  = "FAIL:"

	callPrefix     = "fake-call-"
	toolCallTokens = 8
)

var toolPattern = regexp.MustCompile(`TOOL:([A-Za-z0-9_.-]+)(\{.*\})`)

type Turn struct {
	Text string
	Tool string
	Args json.RawMessage
}

type Script func(prompt string) Turn

type GollemOption func(*Gollem)

func WithScript(script Script) GollemOption {
	return func(g *Gollem) { g.script = script }
}

type Gollem struct {
	script   Script
	mu       sync.Mutex
	sessions []*GollemSession
}

func NewGollem(options ...GollemOption) *Gollem {
	client := &Gollem{}
	for _, option := range options {
		option(client)
	}
	return client
}

func (g *Gollem) New(_ context.Context, _ llm.ModelRef) (gollem.LLMClient, error) {
	return g, nil
}

func (g *Gollem) NewSession(_ context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
	cfg := gollem.NewSessionConfig(options...)

	history := &gollem.History{LLType: gollem.LLMTypeOpenAI, Version: gollem.HistoryVersion}
	if loaded := cfg.History(); loaded != nil {
		history = loaded.Clone()
	}

	session := &GollemSession{
		history: history,
		blocks:  cfg.ContentBlockMiddlewares(),
		streams: cfg.ContentStreamMiddlewares(),
		system:  cfg.SystemPrompt(),
		written: g.script,
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	g.sessions = append(g.sessions, session)
	return session, nil
}

func (g *Gollem) GenerateEmbedding(context.Context, int, []string) ([][]float64, error) {
	return nil, errors.New(errors.Internal, "the scripted gollem client does not embed")
}

func (g *Gollem) Sessions() []*GollemSession {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]*GollemSession(nil), g.sessions...)
}

type scripted struct {
	name string
	args string
}

type GollemSession struct {
	mu      sync.Mutex
	history *gollem.History
	blocks  []gollem.ContentBlockMiddleware
	streams []gollem.ContentStreamMiddleware
	system  string
	written Script
	script  []scripted
	answer  string
	primed  bool
	turn    int
}

func (s *GollemSession) System() string {
	return s.system
}

func (s *GollemSession) Generate(ctx context.Context, input []gollem.Input, _ ...gollem.GenerateOption) (*gollem.Response, error) {
	handler := gollem.BuildContentBlockChain(s.blocks, s.generate)

	resp, err := handler(ctx, &gollem.ContentRequest{Inputs: input, SystemPrompt: s.system})
	if err != nil {
		return nil, err
	}
	return &gollem.Response{
		Texts:         resp.Texts,
		FunctionCalls: resp.FunctionCalls,
		InputToken:    resp.InputToken,
		OutputToken:   resp.OutputToken,
	}, nil
}

func (s *GollemSession) Stream(ctx context.Context, input []gollem.Input, _ ...gollem.GenerateOption) (<-chan *gollem.Response, error) {
	handler := gollem.BuildContentStreamChain(s.streams, s.stream)

	chunks, err := handler(ctx, &gollem.ContentRequest{Inputs: input, SystemPrompt: s.system})
	if err != nil {
		return nil, err
	}

	out := make(chan *gollem.Response)
	go func() {
		defer close(out)
		for chunk := range chunks {
			out <- &gollem.Response{
				Texts:         chunk.Texts,
				FunctionCalls: chunk.FunctionCalls,
				InputToken:    chunk.InputToken,
				OutputToken:   chunk.OutputToken,
				Error:         chunk.Error,
			}
		}
	}()
	return out, nil
}

func (s *GollemSession) stream(ctx context.Context, req *gollem.ContentRequest) (<-chan *gollem.ContentResponse, error) {
	resp, err := s.generate(ctx, req)
	if err != nil {
		return nil, err
	}

	out := make(chan *gollem.ContentResponse, len(resp.Texts)+1)
	for _, text := range resp.Texts {
		out <- &gollem.ContentResponse{Texts: []string{text}}
	}
	out <- &gollem.ContentResponse{
		FunctionCalls: resp.FunctionCalls,
		InputToken:    resp.InputToken,
		OutputToken:   resp.OutputToken,
	}
	close(out)
	return out, nil
}

func (s *GollemSession) generate(_ context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	characters := 0
	results := make([]gollem.FunctionResponse, 0, len(req.Inputs))
	for _, in := range req.Inputs {
		switch value := in.(type) {
		case gollem.Text:
			characters += len(value)
			s.prime(string(value))
			s.appendText(gollem.RoleUser, string(value))
		case gollem.FunctionResponse:
			characters += len(value.Name)
			results = append(results, value)
			s.appendResult(value)
		}
	}

	if code, failing := scriptedFailure(req.Inputs); failing {
		return nil, errors.New(code, "the scripted model failed on purpose")
	}

	if s.turn < len(s.script) {
		call := s.script[s.turn]
		s.turn++
		return s.emit(call, characters), nil
	}

	answer := s.finalAnswer(results)
	s.appendText(gollem.RoleAssistant, answer)
	return &gollem.ContentResponse{
		Texts:       []string{answer},
		InputToken:  scriptedTokens(characters),
		OutputToken: scriptedTokens(len(answer)),
	}, nil
}

func (s *GollemSession) prime(text string) {
	if s.primed {
		return
	}
	s.primed = true

	for _, match := range toolPattern.FindAllStringSubmatch(text, -1) {
		s.script = append(s.script, scripted{name: match[1], args: match[2]})
	}
	for line := range strings.SplitSeq(text, "\n") {
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, FinalDirective) {
			s.answer = strings.TrimSpace(strings.TrimPrefix(trimmed, FinalDirective))
		}
	}
	if len(s.script) > 0 || s.answer != "" || s.written == nil {
		return
	}

	turn := s.written(text)
	if turn.Tool != "" {
		arguments := string(turn.Args)
		if arguments == "" {
			arguments = "{}"
		}
		s.script = append(s.script, scripted{name: turn.Tool, args: arguments})
	}
	s.answer = turn.Text
}

func (s *GollemSession) emit(call scripted, characters int) *gollem.ContentResponse {
	args := map[string]any{}
	if err := json.Unmarshal([]byte(call.args), &args); err != nil {
		args = map[string]any{}
	}

	callID := callPrefix + call.name
	if content, err := gollem.NewToolCallContent(callID, call.name, args); err == nil {
		s.history.Messages = append(s.history.Messages, gollem.Message{
			Role: gollem.RoleAssistant, Contents: []gollem.MessageContent{content},
		})
	}

	return &gollem.ContentResponse{
		FunctionCalls: []*gollem.FunctionCall{{ID: callID, Name: call.name, Arguments: args}},
		InputToken:    scriptedTokens(characters),
		OutputToken:   toolCallTokens,
	}
}

func (s *GollemSession) finalAnswer(results []gollem.FunctionResponse) string {
	if s.answer != "" {
		return s.answer
	}
	if len(results) == 0 {
		return defaultAnswer
	}

	names := make([]string, 0, len(results))
	for _, result := range results {
		suffix := ""
		if result.Error != nil {
			suffix = " (error)"
		}
		names = append(names, result.Name+suffix)
	}
	return "ran " + strings.Join(names, ", ")
}

func (s *GollemSession) appendText(role gollem.MessageRole, text string) {
	content, err := gollem.NewTextContent(text)
	if err != nil {
		return
	}
	s.history.Messages = append(s.history.Messages, gollem.Message{Role: role, Contents: []gollem.MessageContent{content}})
}

func (s *GollemSession) appendResult(result gollem.FunctionResponse) {
	payload := result.Data
	failed := false
	if result.Error != nil {
		payload = map[string]any{"error": result.Error.Error()}
		failed = true
	}

	content, err := gollem.NewToolResponseContent(result.ID, result.Name, payload, failed)
	if err != nil {
		return
	}
	s.history.Messages = append(s.history.Messages, gollem.Message{
		Role: gollem.RoleTool, Contents: []gollem.MessageContent{content},
	})
}

func (s *GollemSession) GenerateContent(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
	return s.Generate(ctx, input)
}

func (s *GollemSession) GenerateStream(ctx context.Context, input ...gollem.Input) (<-chan *gollem.Response, error) {
	return s.Stream(ctx, input)
}

func (s *GollemSession) History() (*gollem.History, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.history.Clone(), nil
}

func (s *GollemSession) AppendHistory(history *gollem.History) error {
	if history == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.history.Messages = append(s.history.Messages, history.Messages...)
	return nil
}

func (s *GollemSession) CountToken(_ context.Context, input ...gollem.Input) (int, error) {
	characters := 0
	for _, in := range input {
		characters += len(in.String())
	}
	return scriptedTokens(characters), nil
}

func scriptedFailure(inputs []gollem.Input) (errors.Code, bool) {
	for _, in := range inputs {
		text, ok := in.(gollem.Text)
		if !ok {
			continue
		}
		for line := range strings.SplitSeq(string(text), "\n") {
			trimmed := strings.TrimSpace(line)
			for _, directive := range []string{ErrorDirective, FailDirective} {
				if strings.HasPrefix(trimmed, directive) {
					return errors.Code(strings.TrimSpace(strings.TrimPrefix(trimmed, directive))), true
				}
			}
		}
	}
	return "", false
}

func scriptedTokens(characters int) int {
	return max(1, characters/charactersPerToken)
}
