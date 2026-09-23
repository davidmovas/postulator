package images_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

func TestContentTypeIsDerivedFromTheExtension(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		file string
		want string
	}{
		{name: "png", file: "espresso.PNG", want: "image/png"},
		{name: "jpeg", file: "espresso.jpeg", want: "image/jpeg"},
		{name: "jpg", file: "a/b/espresso.jpg", want: "image/jpeg"},
		{name: "webp", file: "espresso.webp", want: "image/webp"},
		{name: "not an image", file: "espresso.txt", want: ""},
		{name: "no extension", file: "espresso", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := images.ContentType(tc.file); got != tc.want {
				t.Errorf("ContentType(%q) = %q, want %q", tc.file, got, tc.want)
			}
			if got := images.Supported(tc.file); got != (tc.want != "") {
				t.Errorf("Supported(%q) = %t", tc.file, got)
			}
		})
	}
}

func TestTheSettingsCarryTheirDefaults(t *testing.T) {
	t.Parallel()

	values := settings.Default().NewValues()
	if got := images.LocalDir(values); got != "" {
		t.Errorf("LocalDir = %q, want the empty default", got)
	}
	if got := images.OpenAIModel(values); got != images.DefaultOpenAIModel {
		t.Errorf("OpenAIModel = %q, want %q", got, images.DefaultOpenAIModel)
	}
	if got := images.OpenAIQuality(values); got != "medium" {
		t.Errorf("OpenAIQuality = %q, want medium: auto lets the provider pick its dearest quality", got)
	}
	if err := settings.Default().Validate("images.openaiQuality", json.RawMessage(`"ultra"`)); err == nil {
		t.Error("an image quality the provider does not offer was accepted")
	}

	stored := map[string]json.RawMessage{
		"images.localDir":      json.RawMessage(`"C:\\pictures"`),
		"images.openaiModel":   json.RawMessage(`"gpt-image-2"`),
		"images.openaiQuality": json.RawMessage(`"low"`),
	}
	if _, err := settings.Default().Apply(values, stored); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := images.LocalDir(values); got != `C:\pictures` {
		t.Errorf("LocalDir = %q", got)
	}
	if got := images.OpenAIModel(values); got != "gpt-image-2" {
		t.Errorf("OpenAIModel = %q", got)
	}
	if got := images.OpenAIQuality(values); got != "low" {
		t.Errorf("OpenAIQuality = %q", got)
	}
}
