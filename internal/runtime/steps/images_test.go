package steps_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const imageBody = "<h1>Espresso</h1><h2>About</h2><p>One.</p><h2>Brewing</h2><p>Two.</p>"

type drawing struct {
	calls int
	err   error
}

func (d *drawing) Generate(_ context.Context, prompt images.Prompt) (images.Image, error) {
	if d.err != nil {
		return images.Image{}, d.err
	}
	d.calls++
	return images.Image{
		Filename:    "drawn.png",
		ContentType: "image/png",
		Alt:         prompt.Alt,
		Bytes:       []byte{0x89, 0x50},
	}, nil
}

type library struct {
	items []images.Image
	err   error
}

func (l library) Pick(context.Context, images.Query) ([]images.Image, error) {
	return l.items, l.err
}

type oneClient struct {
	client *wp.Client
	err    error
}

func (o oneClient) Client(context.Context, string) (*wp.Client, error) {
	return o.client, o.err
}

func imageDeps(t *testing.T) (steps.Deps, *wptest.Server) {
	t.Helper()
	return imageDepsWith(t)
}

func imageDepsWith(t *testing.T, opts ...wptest.Option) (steps.Deps, *wptest.Server) {
	t.Helper()

	server := wptest.New(t, opts...)
	client, err := wp.New(wp.Config{
		BaseURL:       server.URL(),
		Username:      wptest.DefaultUser,
		AppPassword:   wptest.DefaultPassword,
		AllowInsecure: true,
	}, wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	deps := unitDeps()
	deps.WordPress = oneClient{client: client}
	deps.ImageProvider = &drawing{}
	return deps, server
}

func imageContext(t *testing.T, spec template.Images) *run.StepContext {
	t.Helper()

	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactBodyHTML: []byte(imageBody)})
	sc.Spec.Images = spec
	return sc
}

func runImages(t *testing.T, deps steps.Deps, sc *run.StepContext) (steps.ImagesResult, run.Result) {
	t.Helper()

	result, err := steps.GenerateImages(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("GenerateImages: %v", err)
	}

	var manifest steps.ImagesResult
	for i := range result.Artifacts {
		if result.Artifacts[i].Kind == run.ArtifactImages {
			if err = json.Unmarshal(result.Artifacts[i].Blob, &manifest); err != nil {
				t.Fatalf("decode the image manifest: %v", err)
			}
		}
	}
	return manifest, result
}

func TestGenerateImagesSkipsWhenTheTemplateAsksForNone(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	_, result := runImages(t, deps, imageContext(t, template.Images{Source: template.ImagesAI}))

	if len(result.Artifacts) != 0 {
		t.Fatalf("the step produced %+v", result.Artifacts)
	}
	if len(server.Uploads()) != 0 {
		t.Fatalf("the step uploaded %+v", server.Uploads())
	}
}

func TestGenerateImagesDrawsUploadsAndPlaces(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	manifest, result := runImages(t, deps, imageContext(t,
		template.Images{Featured: true, Inline: 1, Source: template.ImagesAI}))

	if len(server.Uploads()) != 2 {
		t.Fatalf("the step uploaded %d files, want 2", len(server.Uploads()))
	}
	if manifest.FeaturedID == 0 || len(manifest.Images) != 2 {
		t.Fatalf("manifest = %+v", manifest)
	}
	if manifest.Images[0].Role != steps.RoleFeatured || manifest.Images[1].Role != steps.RoleInline {
		t.Fatalf("roles = %+v", manifest.Images)
	}
	if manifest.Images[0].Alt != "espresso" {
		t.Errorf("alt = %q, want the primary keyword", manifest.Images[0].Alt)
	}
	for _, uploaded := range server.Uploads() {
		if uploaded.Alt != "espresso" {
			t.Errorf("upload = %+v, want the primary keyword as alternative text", uploaded)
		}
	}

	body := ""
	for i := range result.Artifacts {
		if result.Artifacts[i].Kind == run.ArtifactBodyHTML {
			body = string(result.Artifacts[i].Blob)
		}
	}
	if !strings.Contains(body, "<figure><img src=") || !strings.Contains(body, `alt="espresso"`) {
		t.Fatalf("body = %s", body)
	}
	if strings.Index(body, "<figure>") < strings.Index(body, "<h2>About</h2>") {
		t.Fatalf("the figure landed before the first section: %s", body)
	}
	if strings.Index(body, "<figure>") > strings.Index(body, "<h2>Brewing</h2>") {
		t.Fatalf("the figure landed after the second section: %s", body)
	}
}

func TestGenerateImagesPicksFromASource(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	deps.ImageSources = map[template.ImageSource]steps.ImageSource{
		template.ImagesWPMedia: library{items: []images.Image{
			{WPID: 42, URL: "https://shop.example.com/a.png", Alt: "a cup"},
		}},
	}

	manifest, _ := runImages(t, deps, imageContext(t,
		template.Images{Featured: true, Source: template.ImagesWPMedia}))

	if manifest.FeaturedID != 42 {
		t.Fatalf("manifest = %+v", manifest)
	}
	if len(server.Uploads()) != 0 {
		t.Fatalf("a library pick must not be uploaded again: %+v", server.Uploads())
	}
}

func TestGenerateImagesRecordsWhatItCouldNotDo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		with func(steps.Deps) steps.Deps
		spec template.Images
		want string
	}{
		{
			name: "no provider is configured",
			with: func(d steps.Deps) steps.Deps { d.ImageProvider = nil; return d },
			spec: template.Images{Featured: true, Source: template.ImagesAI},
			want: steps.CodeNoImageSource,
		},
		{
			name: "no source is configured",
			spec: template.Images{Featured: true, Source: template.ImagesLocal},
			want: steps.CodeNoImageSource,
		},
		{
			name: "the provider fails",
			with: func(d steps.Deps) steps.Deps {
				d.ImageProvider = &drawing{err: errors.New(errors.External, "the provider is down")}
				return d
			},
			spec: template.Images{Featured: true, Source: template.ImagesAI},
			want: steps.CodeImagesFailed,
		},
		{
			name: "the source fails",
			with: func(d steps.Deps) steps.Deps {
				d.ImageSources = map[template.ImageSource]steps.ImageSource{
					template.ImagesLocal: library{err: errors.New(errors.External, "the folder is gone")},
				}
				return d
			},
			spec: template.Images{Featured: true, Source: template.ImagesLocal},
			want: steps.CodeImagesFailed,
		},
		{
			name: "the upload fails",
			with: func(d steps.Deps) steps.Deps {
				d.WordPress = oneClient{err: errors.New(errors.NotFound, "no such site")}
				return d
			},
			spec: template.Images{Featured: true, Source: template.ImagesAI},
			want: steps.CodeUploadFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, _ := imageDeps(t)
			if tc.with != nil {
				deps = tc.with(deps)
			}

			manifest, _ := runImages(t, deps, imageContext(t, tc.spec))
			if len(manifest.Skipped) != 1 || !strings.HasPrefix(manifest.Skipped[0], tc.want) {
				t.Fatalf("skipped = %v, want one entry starting with %q", manifest.Skipped, tc.want)
			}
			if manifest.FeaturedID != 0 {
				t.Fatalf("manifest = %+v, want no featured image", manifest)
			}
		})
	}
}

func TestGenerateImagesReportsWhatItCannotRead(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)
	sc := imageContext(t, template.Images{Featured: true, Source: template.ImagesAI})
	sc.Artifacts = map[run.ArtifactKind]run.Artifact{}

	if _, err := steps.GenerateImages(deps).Run(t.Context(), sc); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), errors.Invalid, err)
	}
}
