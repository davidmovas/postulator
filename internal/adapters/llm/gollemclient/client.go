package gollemclient

import (
	"context"
	stderrors "errors"
	"slices"
	"strings"
	"time"

	"github.com/gollem-dev/gollem"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type ProviderFactory interface {
	New(ctx context.Context, ref llm.ModelRef) (gollem.LLMClient, error)
}

type ModelReader interface {
	Lookup(ctx context.Context, ref llm.ModelRef) (llm.ModelInfo, error)
}

type Client struct {
	factory ProviderFactory
	models  ModelReader
	timeout time.Duration
}

func New(factory ProviderFactory, models ModelReader, timeout time.Duration) *Client {
	return &Client{factory: factory, models: models, timeout: timeout}
}

func (c *Client) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	if err := req.Validate(); err != nil {
		return port.Response{}, err
	}

	call, cancel := c.withTimeout(ctx)
	defer cancel()

	session, err := c.session(call, req)
	if err != nil {
		return port.Response{}, err
	}

	ceiling := c.ceiling(call, req)
	resp, err := session.Generate(call, lastInput(req), generateOptions(req, ceiling)...)
	if err != nil {
		if stderrors.Is(err, gollem.ErrProhibitedContent) {
			return port.Response{FinishReason: port.FinishContentFilter}, nil
		}
		return port.Response{}, classify(ctx, err)
	}

	usage := usageOf(resp.InputToken, resp.CacheReadInputToken, resp.OutputToken)
	return port.Response{
		Text:         strings.Join(resp.Texts, ""),
		Usage:        usage,
		FinishReason: finishReason(ceiling, usage),
	}, nil
}

func (c *Client) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	call, cancel := c.withTimeout(ctx)

	session, err := c.session(call, req)
	if err != nil {
		cancel()
		return nil, err
	}

	chunks, err := session.Stream(call, lastInput(req), generateOptions(req, c.ceiling(call, req))...)
	if err != nil {
		cancel()
		return nil, classify(ctx, err)
	}

	out := make(chan port.Delta)
	go func() {
		defer cancel()
		defer close(out)

		var usage llm.Usage
		for chunk := range chunks {
			if chunk.Error != nil {
				send(call, out, port.Delta{Err: classify(ctx, chunk.Error)})
				return
			}
			if chunk.InputToken > 0 || chunk.OutputToken > 0 {
				usage = usageOf(chunk.InputToken, chunk.CacheReadInputToken, chunk.OutputToken)
			}
			text := strings.Join(chunk.Texts, "")
			if text == "" {
				continue
			}
			if !send(call, out, port.Delta{Text: text}) {
				return
			}
		}
		final := usage
		send(call, out, port.Delta{Done: true, Usage: &final})
	}()
	return out, nil
}

func (c *Client) ceiling(ctx context.Context, req port.Request) int {
	if req.MaxTokens <= 0 || c.models == nil {
		return req.MaxTokens
	}

	info, err := c.models.Lookup(ctx, req.Ref)
	if err != nil {
		return req.MaxTokens
	}
	return budget(req.MaxTokens, info)
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.timeout)
}

func (c *Client) session(ctx context.Context, req port.Request) (gollem.Session, error) {
	provider, err := c.factory.New(ctx, req.Ref)
	if err != nil {
		return nil, err
	}

	options := make([]gollem.SessionOption, 0, 4)
	if req.System != "" {
		options = append(options, gollem.WithSessionSystemPrompt(req.System))
	}
	if req.Schema != nil {
		options = append(options,
			gollem.WithSessionContentType(gollem.ContentTypeJSON),
			gollem.WithSessionResponseSchema(parameterOf(req.Schema)),
		)
	}

	history, err := historyOf(req)
	if err != nil {
		return nil, err
	}
	if history != nil {
		options = append(options, gollem.WithSessionHistory(history))
	}

	session, err := provider.NewSession(ctx, options...)
	if err != nil {
		return nil, classify(ctx, err)
	}
	if session == nil {
		return nil, errors.New(errors.External, "the model provider returned no session").WithDetail("model", req.Ref.String())
	}
	return session, nil
}

func send(ctx context.Context, out chan<- port.Delta, delta port.Delta) bool {
	select {
	case out <- delta:
		return true
	case <-ctx.Done():
		return false
	}
}

func lastInput(req port.Request) []gollem.Input {
	return []gollem.Input{gollem.Text(req.Messages[len(req.Messages)-1].Text)}
}

func generateOptions(req port.Request, ceiling int) []gollem.GenerateOption {
	options := make([]gollem.GenerateOption, 0, 2)
	if ceiling > 0 {
		options = append(options, gollem.WithMaxTokens(ceiling))
	}
	if req.Temperature != nil {
		options = append(options, gollem.WithTemperature(*req.Temperature))
	}
	return options
}

func usageOf(input, cached, output int) llm.Usage {
	return llm.Usage{Input: input, CachedInput: cached, Output: output, Total: input + output}
}

func finishReason(ceiling int, usage llm.Usage) port.FinishReason {
	if ceiling > 0 && usage.Output >= ceiling {
		return port.FinishLength
	}
	return port.FinishStop
}

func historyOf(req port.Request) (*gollem.History, error) {
	prior := req.Messages[:len(req.Messages)-1]
	messages := make([]gollem.Message, 0, len(prior)+1)

	if req.System != "" && req.Ref.Provider == ProviderOpenAI {
		system, err := textMessage(gollem.RoleSystem, req.System)
		if err != nil {
			return nil, err
		}
		messages = append(messages, system)
	}

	for _, message := range prior {
		converted, err := textMessage(roleOf(message.Role), message.Text)
		if err != nil {
			return nil, err
		}
		messages = append(messages, converted)
	}

	if len(messages) == 0 {
		return nil, nil
	}
	return &gollem.History{LLType: llmTypeOf(req.Ref.Provider), Version: gollem.HistoryVersion, Messages: messages}, nil
}

func textMessage(role gollem.MessageRole, text string) (gollem.Message, error) {
	content, err := gollem.NewTextContent(text)
	if err != nil {
		return gollem.Message{}, errors.Wrap(err, errors.Internal, "encode a conversation message")
	}
	return gollem.Message{Role: role, Contents: []gollem.MessageContent{content}}, nil
}

func roleOf(role port.Role) gollem.MessageRole {
	if role == port.RoleAssistant {
		return gollem.RoleAssistant
	}
	return gollem.RoleUser
}

func llmTypeOf(provider string) gollem.LLMType {
	switch provider {
	case ProviderAnthropic:
		return gollem.LLMTypeClaude
	case ProviderGemini:
		return gollem.LLMTypeGemini
	case ProviderGeminiOpenAI:
		return gollem.LLMTypeOpenAI
	default:
		return gollem.LLMTypeOpenAI
	}
}

func parameterOf(schema *port.Schema) *gollem.Parameter {
	if schema == nil {
		return nil
	}

	parameter := &gollem.Parameter{
		Title:       schema.Title,
		Type:        gollem.ParameterType(schema.Type),
		Description: schema.Description,
		Enum:        slices.Clone(schema.Enum),
		Items:       parameterOf(schema.Items),
	}
	if len(schema.Properties) == 0 {
		return parameter
	}

	parameter.Properties = make(map[string]*gollem.Parameter, len(schema.Properties))
	for name, child := range schema.Properties {
		converted := parameterOf(child)
		converted.Required = slices.Contains(schema.Required, name)
		parameter.Properties[name] = converted
	}
	return parameter
}
