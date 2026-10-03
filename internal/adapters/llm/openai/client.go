package openai

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"strings"
	"time"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	Provider       = "openai"
	DefaultBaseURL = "https://api.openai.com/v1"
	DefaultTimeout = 2 * time.Minute

	responsesPath   = "/responses"
	contentTypeJSON = "application/json"
	contentTypeSSE  = "text/event-stream"
	maxBodyBytes    = 32 << 20
)

type secretReader interface {
	Get(ctx context.Context, ref string) (string, error)
}

type modelReader interface {
	Lookup(ctx context.Context, ref llm.ModelRef) (llm.ModelInfo, error)
}

type Option func(*Client)

func WithBaseURL(base string) Option {
	return func(c *Client) {
		if trimmed := strings.TrimRight(strings.TrimSpace(base), "/"); trimmed != "" {
			c.baseURL = trimmed
		}
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) { c.timeout = max(timeout, 0) }
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.http = client
		}
	}
}

func WithClock(clk clock.Clock) Option {
	return func(c *Client) {
		if clk != nil {
			c.clock = clk
		}
	}
}

type Client struct {
	secrets  secretReader
	models   modelReader
	http     *http.Client
	clock    clock.Clock
	arm      func(after time.Duration, ring func()) alarm
	baseURL  string
	timeout  time.Duration
	patience time.Duration
}

func New(secrets secretReader, models modelReader, opts ...Option) *Client {
	client := &Client{
		secrets: secrets,
		models:  models,
		http:    &http.Client{},
		clock:   clock.System{},
		arm:     afterPatience,
		baseURL: DefaultBaseURL,
		timeout: DefaultTimeout,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

type exchange struct {
	key  string
	ref  llm.ModelRef
	body wireRequest
}

func (c *Client) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	if err := req.Validate(); err != nil {
		return port.Response{}, err
	}

	call, cancel := c.withTimeout(ctx)
	defer cancel()

	ex, err := c.prepare(call, req, false)
	if err != nil {
		return port.Response{}, err
	}

	resp, err := c.complete(call, ex)
	if err == nil {
		return resp, nil
	}
	if refused := refusalIn(err); refused != nil && refused.exhaustedOutput() && ctx.Err() == nil {
		return port.Response{FinishReason: port.FinishLength}, nil
	}
	return port.Response{}, c.classify(ctx, ex, err)
}

func (c *Client) completeOnce(ctx context.Context, ex exchange) (port.Response, error) {
	resp, err := c.post(ctx, ex)
	if err != nil {
		return port.Response{}, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return port.Response{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return port.Response{}, refusalOf(resp.StatusCode, resp.Header, payload)
	}
	return responseOf(payload, ex.body.ServiceTier)
}

func (c *Client) prepare(ctx context.Context, req port.Request, stream bool) (exchange, error) {
	if req.Ref.Provider != Provider {
		return exchange{}, errors.New(errors.Invalid, "this provider is not supported").WithDetail("provider", req.Ref.Provider)
	}

	body, err := requestOf(req, c.row(ctx, req.Ref), stream)
	if err != nil {
		return exchange{}, err
	}
	key, err := c.key(ctx)
	if err != nil {
		return exchange{}, err
	}
	return exchange{key: key, ref: req.Ref, body: body}, nil
}

func (c *Client) row(ctx context.Context, ref llm.ModelRef) catalogRow {
	if c.models == nil {
		return catalogRow{}
	}
	info, err := c.models.Lookup(ctx, ref)
	if err != nil {
		return catalogRow{}
	}
	return listed(info)
}

func (c *Client) key(ctx context.Context) (string, error) {
	key, err := c.secrets.Get(ctx, llm.SecretRef(Provider))
	if err != nil {
		if errors.IsCode(err, errors.NotFound) {
			return "", errors.New(errors.Unauthorized, "no api key is stored for this provider").WithDetail("provider", Provider)
		}
		return "", err
	}
	if key = strings.TrimSpace(key); key == "" {
		return "", errors.New(errors.Unauthorized, "the stored api key for this provider is empty").WithDetail("provider", Provider)
	}
	return key, nil
}

func (c *Client) post(ctx context.Context, ex exchange) (*http.Response, error) {
	var payload bytes.Buffer
	encoder := json.NewEncoder(&payload)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(ex.body); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "encode the model request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+responsesPath, &payload)
	if err != nil {
		return nil, errors.New(errors.Invalid, "the model provider's address is not a valid URL").WithDetail("baseUrl", c.baseURL)
	}
	accept := contentTypeJSON
	if ex.body.Stream {
		accept = contentTypeSSE
	}
	req.Header.Set("Authorization", "Bearer "+ex.key)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", accept)
	return c.http.Do(req)
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, bounded := ctx.Deadline(); bounded || c.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.timeout)
}

func (c *Client) classify(caller context.Context, ex exchange, err error) error {
	if caller.Err() != nil {
		return errors.New(errors.Cancelled, "the model call was cancelled").WithInternal(caller.Err())
	}
	if refused := refusalIn(err); refused != nil {
		return refused.kernel(ex.ref, c.clock.Now())
	}

	var kernel *errors.Error
	if stderrors.As(err, &kernel) {
		return err
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return errors.New(errors.External, "the model did not answer before the timeout").WithInternal(err).WithRetry(0)
	}
	return errors.New(errors.External, "the model provider could not be reached").WithInternal(err).WithRetry(0)
}
