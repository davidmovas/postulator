package openai_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/images/openai"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type vault struct {
	key string
	err error
}

func (v vault) Get(context.Context, string) (string, error) {
	if v.err != nil {
		return "", v.err
	}
	return v.key, nil
}

type call struct {
	Keys    map[string]any `json:"-"`
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Size    string         `json:"size"`
	Format  string         `json:"response_format"`
	Quality string         `json:"quality"`
	Count   int            `json:"n"`
}

func provider(t *testing.T, status int, body string, secrets vault, opts ...openai.Option) (*openai.Images, *call, *string) {
	t.Helper()
	return providerOf(t, "gpt-image-test", status, body, secrets, opts...)
}

func providerOf(t *testing.T, model string, status int, body string, secrets vault,
	opts ...openai.Option) (*openai.Images, *call, *string) {
	t.Helper()

	recorded := &call{}
	auth := new(string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Errorf("path = %q", r.URL.Path)
		}
		*auth = r.Header.Get("Authorization")

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read the request: %v", err)
		}
		if err = json.Unmarshal(payload, recorded); err != nil {
			t.Errorf("decode the request: %v", err)
		}
		if err = json.Unmarshal(payload, &recorded.Keys); err != nil {
			t.Errorf("decode the request keys: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if _, err = w.Write([]byte(body)); err != nil {
			t.Logf("write the response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	opts = append([]openai.Option{openai.WithBaseURL(server.URL + "/v1"), openai.WithHTTPClient(server.Client())}, opts...)
	return openai.New(secrets, model, opts...), recorded, auth
}

func TestGenerateAsksForTheQualityAndReadsWhatItCost(t *testing.T) {
	t.Parallel()

	raw := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50})
	body := `{"data":[{"b64_json":"` + raw + `"}],"usage":{"total_tokens":1300,"input_tokens":240,` +
		`"output_tokens":1060,"input_tokens_details":{"text_tokens":240,"cached_tokens":40}}}`

	cases := []struct {
		name string
		opts []openai.Option
		want string
	}{
		{name: "the default quality", want: openai.DefaultQuality},
		{name: "a chosen quality", opts: []openai.Option{openai.WithQuality("low")}, want: "low"},
		{name: "a blank choice keeps the default", opts: []openai.Option{openai.WithQuality("  ")}, want: openai.DefaultQuality},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			generator, recorded, _ := provider(t, http.StatusOK, body, vault{key: "sk"}, tc.opts...)
			image, err := generator.Generate(t.Context(), images.Prompt{Subject: "Kettle"})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if recorded.Quality != tc.want {
				t.Fatalf("quality = %q, want %q", recorded.Quality, tc.want)
			}
			if image.Usage.Input != 240 || image.Usage.Output != 1060 || image.Usage.Total != 1300 ||
				image.Usage.CachedInput != 40 {
				t.Fatalf("usage = %+v, want what the provider billed", image.Usage)
			}
		})
	}
}

func TestGenerateReturnsThePNG(t *testing.T) {
	t.Parallel()

	raw := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d}
	body := `{"data":[{"b64_json":"` + base64.StdEncoding.EncodeToString(raw) + `"}]}`

	generator, recorded, auth := provider(t, http.StatusOK, body, vault{key: "sk-test"})
	image, err := generator.Generate(t.Context(), images.Prompt{
		Subject: "An espresso machine", Context: "on a wooden counter", Alt: "An espresso machine",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !bytes.Equal(image.Bytes, raw) {
		t.Errorf("bytes = %v, want %v", image.Bytes, raw)
	}
	if image.Filename != "an-espresso-machine.png" || image.ContentType != "image/png" {
		t.Errorf("image = %+v", image)
	}
	if image.Alt != "An espresso machine" {
		t.Errorf("alt = %q", image.Alt)
	}
	if *auth != "Bearer sk-test" {
		t.Errorf("authorization = %q", *auth)
	}
	if recorded.Model != "gpt-image-test" || recorded.Count != 1 {
		t.Errorf("request = %+v", recorded)
	}
	if recorded.Size != openai.DefaultSize {
		t.Errorf("size = %q, want %q", recorded.Size, openai.DefaultSize)
	}
	if recorded.Prompt != "An espresso machine. on a wooden counter" {
		t.Errorf("prompt = %q", recorded.Prompt)
	}
}

func TestGenerateSendsOnlyWhatTheModelTakes(t *testing.T) {
	t.Parallel()

	body := `{"data":[{"b64_json":"` + base64.StdEncoding.EncodeToString([]byte{0x89, 0x50}) + `"}]}`

	cases := []struct {
		model   string
		format  string
		quality string
	}{
		{model: "gpt-image-2", quality: openai.DefaultQuality},
		{model: "gpt-image-1", quality: openai.DefaultQuality},
		{model: "gpt-image-1.5", quality: openai.DefaultQuality},
		{model: "dall-e-3", format: "b64_json"},
		{model: "dall-e-2", format: "b64_json"},
		{model: "DALL-E-3", format: "b64_json"},
	}

	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			t.Parallel()

			generator, recorded, _ := providerOf(t, tc.model, http.StatusOK, body, vault{key: "sk"})
			if _, err := generator.Generate(t.Context(), images.Prompt{Subject: "Kettle"}); err != nil {
				t.Fatalf("Generate: %v", err)
			}

			format, sentFormat := recorded.Keys["response_format"]
			if (tc.format != "") != sentFormat || (sentFormat && format != tc.format) {
				t.Errorf("response_format = %v (sent %v), want %q", format, sentFormat, tc.format)
			}
			quality, sentQuality := recorded.Keys["quality"]
			if (tc.quality != "") != sentQuality || (sentQuality && quality != tc.quality) {
				t.Errorf("quality = %v (sent %v), want %q", quality, sentQuality, tc.quality)
			}
			if recorded.Model != tc.model {
				t.Errorf("model = %q, want %q", recorded.Model, tc.model)
			}
		})
	}
}

func TestGenerateMapsTheProviderStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		body   string
		want   errors.Code
	}{
		{name: "bad key", status: http.StatusUnauthorized, body: `{"error":{"message":"bad key"}}`, want: errors.Unauthorized},
		{name: "rate limited", status: http.StatusTooManyRequests, body: `{}`, want: errors.RateLimited},
		{name: "server error", status: http.StatusBadGateway, body: `{}`, want: errors.External},
		{name: "rejected", status: http.StatusBadRequest, body: `{"error":{"message":"content policy"}}`, want: errors.Invalid},
		{name: "no image", status: http.StatusOK, body: `{"data":[]}`, want: errors.External},
		{name: "not base64", status: http.StatusOK, body: `{"data":[{"b64_json":"!!!"}]}`, want: errors.External},
		{name: "not json", status: http.StatusOK, body: `nonsense`, want: errors.External},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			generator, _, _ := provider(t, tc.status, tc.body, vault{key: "sk-test"})
			_, err := generator.Generate(t.Context(), images.Prompt{Subject: "An espresso machine"})
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.want, err)
			}
		})
	}
}

func TestGenerateRefusesWithoutASubjectOrAKey(t *testing.T) {
	t.Parallel()

	generator, _, _ := provider(t, http.StatusOK, `{"data":[]}`, vault{key: "sk-test"})
	if _, err := generator.Generate(t.Context(), images.Prompt{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}

	locked, _, _ := provider(t, http.StatusOK, `{}`, vault{err: errors.New(errors.NotFound, "no key stored")})
	if _, err := locked.Generate(t.Context(), images.Prompt{Subject: "x"}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}

func TestGenerateReportsACancelledCaller(t *testing.T) {
	t.Parallel()

	generator, _, _ := provider(t, http.StatusOK, `{"data":[]}`, vault{key: "sk-test"})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := generator.Generate(ctx, images.Prompt{Subject: "x"}); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), errors.Cancelled, err)
	}
}

func TestTheModelFallsBackToTheDefault(t *testing.T) {
	t.Parallel()

	raw := base64.StdEncoding.EncodeToString([]byte{0x89})
	generator, recorded, _ := provider(t, http.StatusOK, `{"data":[{"b64_json":"`+raw+`"}]}`, vault{key: "sk"})

	bare := openai.New(vault{key: "sk"}, "   ")
	if bare == nil {
		t.Fatal("New returned nothing")
	}

	if _, err := generator.Generate(t.Context(), images.Prompt{Subject: "Kettle", Size: "512x512"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if recorded.Size != "512x512" {
		t.Errorf("size = %q", recorded.Size)
	}
}
