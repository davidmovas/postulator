package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/images"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	Provider = "openai"

	DefaultBaseURL = "https://api.openai.com/v1"
	DefaultTimeout = 4 * time.Minute
	DefaultSize    = "1024x1024"
	DefaultQuality = images.DefaultOpenAIQuality

	generationsPath = "/images/generations"
	maxBodyBytes    = 32 << 20
	contentTypeJSON = "application/json"
	dallEPrefix     = "dall-e"
	dallEFormat     = "b64_json"
)

type secretStore interface {
	Get(ctx context.Context, ref string) (string, error)
}

type Option func(*Images)

func WithBaseURL(baseURL string) Option {
	return func(i *Images) {
		if trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/"); trimmed != "" {
			i.baseURL = trimmed
		}
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(i *Images) {
		if client != nil {
			i.http = client
		}
	}
}

func WithQuality(quality string) Option {
	return func(i *Images) {
		if trimmed := strings.TrimSpace(quality); trimmed != "" {
			i.quality = trimmed
		}
	}
}

type Images struct {
	secrets secretStore
	http    *http.Client
	baseURL string
	model   string
	quality string
}

func New(secrets secretStore, model string, opts ...Option) *Images {
	generator := &Images{
		secrets: secrets,
		http:    &http.Client{Timeout: DefaultTimeout},
		baseURL: DefaultBaseURL,
		model:   strings.TrimSpace(model),
		quality: DefaultQuality,
	}
	if generator.model == "" {
		generator.model = images.DefaultOpenAIModel
	}
	for _, opt := range opts {
		opt(generator)
	}
	return generator
}

type generationRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Size    string `json:"size"`
	Format  string `json:"response_format,omitempty"`
	Quality string `json:"quality,omitempty"`
	Count   int    `json:"n"`
}

type generationResponse struct {
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
	Data []struct {
		B64JSON string `json:"b64_json"`
	} `json:"data"`
	Usage struct {
		Details struct {
			Cached int `json:"cached_tokens"`
		} `json:"input_tokens_details"`
		Total  int `json:"total_tokens"`
		Input  int `json:"input_tokens"`
		Output int `json:"output_tokens"`
	} `json:"usage"`
}

func (i *Images) Generate(ctx context.Context, prompt images.Prompt) (images.Image, error) {
	text := strings.TrimSpace(prompt.Subject)
	if text == "" {
		return images.Image{}, errors.New(errors.Invalid, "an image needs a subject to draw")
	}
	if extra := strings.TrimSpace(prompt.Context); extra != "" {
		text += ". " + extra
	}

	key, err := i.secrets.Get(ctx, domainllm.SecretRef(Provider))
	if err != nil {
		return images.Image{}, err
	}

	size := strings.TrimSpace(prompt.Size)
	if size == "" {
		size = DefaultSize
	}

	request := generationRequest{Model: i.model, Prompt: text, Size: size, Count: 1}
	if dallE(i.model) {
		request.Format = dallEFormat
	} else {
		request.Quality = i.quality
	}
	body, err := json.Marshal(request)
	if err != nil {
		return images.Image{}, errors.Wrap(err, errors.Internal, "encode the image request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.baseURL+generationsPath, bytes.NewReader(body))
	if err != nil {
		return images.Image{}, errors.Wrap(err, errors.Internal, "build the image request")
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := i.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return images.Image{}, errors.New(errors.Cancelled, "the image request was cancelled").WithInternal(ctx.Err())
		}
		return images.Image{}, errors.Wrap(err, errors.External, "reach the image provider")
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return images.Image{}, errors.Wrap(err, errors.External, "read the image response")
	}

	var decoded generationResponse
	if unmarshalErr := json.Unmarshal(payload, &decoded); unmarshalErr != nil && resp.StatusCode == http.StatusOK {
		return images.Image{}, errors.Wrap(unmarshalErr, errors.External, "decode the image response")
	}
	if resp.StatusCode != http.StatusOK {
		return images.Image{}, classify(resp.StatusCode, decoded)
	}
	if len(decoded.Data) == 0 || decoded.Data[0].B64JSON == "" {
		return images.Image{}, errors.New(errors.External, "the image provider returned no image")
	}

	raw, err := base64.StdEncoding.DecodeString(decoded.Data[0].B64JSON)
	if err != nil {
		return images.Image{}, errors.Wrap(err, errors.External, "decode the generated image")
	}

	return images.Image{
		Filename:    filename(prompt),
		ContentType: "image/png",
		Alt:         alt(prompt),
		Bytes:       raw,
		Usage: domainllm.Usage{
			Input: decoded.Usage.Input, CachedInput: decoded.Usage.Details.Cached,
			Output: decoded.Usage.Output, Total: decoded.Usage.Total,
		},
	}, nil
}

func dallE(model string) bool {
	return strings.HasPrefix(strings.ToLower(model), dallEPrefix)
}

func classify(status int, decoded generationResponse) error {
	message := "the image provider rejected the request"
	if decoded.Error != nil && decoded.Error.Message != "" {
		message = decoded.Error.Message
	}

	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return errors.New(errors.Unauthorized, "the image provider rejected the api key").
			WithDetail("status", status)
	case status == http.StatusTooManyRequests:
		return errors.New(errors.RateLimited, "the image provider is rate limiting this key").
			WithDetail("status", status)
	case status >= http.StatusInternalServerError:
		return errors.New(errors.External, "the image provider returned a server error").
			WithDetail("status", status)
	default:
		return errors.New(errors.Invalid, message).WithDetail("status", status)
	}
}

func filename(prompt images.Prompt) string {
	base := slug(prompt.Subject)
	if base == "" {
		base = "image"
	}
	return base + ".png"
}

func alt(prompt images.Prompt) string {
	if trimmed := strings.TrimSpace(prompt.Alt); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(prompt.Subject)
}

func slug(text string) string {
	var builder strings.Builder
	dashed := false
	for _, symbol := range strings.ToLower(text) {
		if (symbol >= 'a' && symbol <= 'z') || (symbol >= '0' && symbol <= '9') {
			builder.WriteRune(symbol)
			dashed = false
			continue
		}
		if !dashed && builder.Len() > 0 {
			builder.WriteByte('-')
			dashed = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
