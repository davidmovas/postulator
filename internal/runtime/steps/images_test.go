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
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const imageBody = "<h1>Espresso</h1><h2>About</h2><p>One.</p><h2>Brewing</h2><p>Two.</p>"

type drawing struct {
	err     error
	prompts []images.Prompt
	calls   int
}

func (d *drawing) Generate(_ context.Context, prompt images.Prompt) (images.Image, error) {
	if d.err != nil {
		return images.Image{}, d.err
	}
	d.calls++
	d.prompts = append(d.prompts, prompt)
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

func TestGenerateImagesHonoursItsProducesWhenTheTemplateAsksForNone(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	manifest, result := runImages(t, deps, imageContext(t, template.Images{Source: template.ImagesAI}))

	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactImages {
		t.Fatalf("the step produced %+v, want the empty manifest its Produces promises", result.Artifacts)
	}
	if len(manifest.Images) != 0 || len(manifest.Findings) != 0 {
		t.Fatalf("manifest = %+v, want an empty one", manifest)
	}
	if len(server.Uploads()) != 0 {
		t.Fatalf("the step uploaded %+v", server.Uploads())
	}
}

func TestGenerateImagesDrawsUploadsAndPlaces(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	drawn := &drawing{}
	deps.ImageProvider = drawn
	sc := imageContext(t, template.Images{Featured: true, Inline: 1, Source: template.ImagesAI})
	manifest, result := runImages(t, deps, sc)

	if len(server.Uploads()) != 2 {
		t.Fatalf("the step uploaded %d files, want 2", len(server.Uploads()))
	}
	for _, prompt := range drawn.prompts {
		if prompt.RunID != sc.Run.ID || prompt.ItemID != sc.Item.ID || prompt.Step != steps.NameGenerateImages {
			t.Fatalf("the prompt = %+v, want it to name the run, the item and the step it is spent on", prompt)
		}
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

func TestGenerateImagesSaysWhenALibraryPicksTooFew(t *testing.T) {
	t.Parallel()

	picked := []images.Image{
		{WPID: 41, URL: "https://shop.example.com/a.png", Alt: "a cup"},
		{WPID: 42, URL: "https://shop.example.com/b.png", Alt: "a mug"},
	}

	cases := []struct {
		name   string
		items  []images.Image
		placed int
		short  string
	}{
		{name: "nothing matched", items: nil, placed: 0, short: "0 of 2"},
		{name: "one of two matched", items: picked[:1], placed: 1, short: "1 of 2"},
		{name: "both matched", items: picked, placed: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, _ := imageDeps(t)
			deps.ImageSources = map[template.ImageSource]steps.ImageSource{
				template.ImagesWPMedia: library{items: tc.items},
			}

			manifest, result := runImages(t, deps, imageContext(t,
				template.Images{Featured: true, Inline: 1, Source: template.ImagesWPMedia}))
			if manifest.Wanted != 2 || len(manifest.Images) != tc.placed {
				t.Fatalf("manifest = %+v, want %d of 2 placed", manifest, tc.placed)
			}
			if tc.short == "" {
				if len(manifest.Findings) != 0 {
					t.Fatalf("findings = %+v, want none", manifest.Findings)
				}
				return
			}
			if len(manifest.Findings) != 1 || manifest.Findings[0].Code != steps.CodeImagesShort ||
				manifest.Findings[0].Severity != content.SeverityWarn {
				t.Fatalf("findings = %+v, want the shortfall named", manifest.Findings)
			}
			message := manifest.Findings[0].Message
			for _, want := range []string{tc.short, "wpmedia", "Espresso", "/coffee/espresso/"} {
				if !strings.Contains(message, want) {
					t.Errorf("message = %q, want it to carry %q", message, want)
				}
			}
			if !strings.Contains(result.Message, message) {
				t.Errorf("the step says %q, want the reason in it", result.Message)
			}
		})
	}
}

func TestGenerateImagesSaysHowManyOfTheImagesItPlaced(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		with func(steps.Deps) steps.Deps
		spec template.Images
		want string
	}{
		{
			name: "every image placed",
			spec: template.Images{Featured: true, Inline: 1, Source: template.ImagesAI},
			want: "placed 2 of 2 images on /coffee/espresso/",
		},
		{
			name: "none asked for",
			spec: template.Images{Source: template.ImagesAI},
			want: "the template asks for no image",
		},
		{
			name: "the model failed",
			with: func(d steps.Deps) steps.Deps {
				d.ImageProvider = &drawing{err: errors.New(errors.Invalid, "Unknown parameter: 'response_format'")}
				return d
			},
			spec: template.Images{Featured: true, Inline: 1, Source: template.ImagesAI},
			want: "placed 0 of 2 images on /coffee/espresso/; the image model stopped after 0 of 2 images",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, _ := imageDeps(t)
			if tc.with != nil {
				deps = tc.with(deps)
			}
			_, result := runImages(t, deps, imageContext(t, tc.spec))
			if !strings.HasPrefix(result.Message, tc.want) {
				t.Fatalf("message = %q, want it to open with %q", result.Message, tc.want)
			}
		})
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
			if manifest.FeaturedID != 0 {
				t.Fatalf("manifest = %+v, want no featured image", manifest)
			}
			if len(manifest.Findings) != 1 {
				t.Fatalf("findings = %+v, want the reason to reach the report", manifest.Findings)
			}
			finding := manifest.Findings[0]
			if finding.Code != tc.want || finding.Severity != content.SeverityWarn {
				t.Errorf("finding = %+v", finding)
			}
			if !strings.Contains(finding.Message, "/coffee/espresso/") {
				t.Errorf("message = %q, want the page named", finding.Message)
			}
			if finding.Details["pageId"] != "page-child" {
				t.Errorf("details = %+v, want the page named", finding.Details)
			}
		})
	}
}

func TestGenerateImagesKeepsWhatItPlacedWhenTimeRunsOut(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)
	deps.ImageProvider = &drawing{err: errors.New(errors.Cancelled, "the image model ran out of time")}
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()

	result, err := steps.GenerateImages(deps).Run(ctx, imageContext(t, template.Images{Featured: true, Source: template.ImagesAI}))
	if err != nil {
		t.Fatalf("an image step that ran out of its own time failed the item: %v", err)
	}
	var manifest steps.ImagesResult
	if err = json.Unmarshal(result.Artifacts[0].Blob, &manifest); err != nil {
		t.Fatalf("decode the manifest: %v", err)
	}
	if len(manifest.Findings) != 1 || manifest.Findings[0].Code != steps.CodeImagesFailed {
		t.Fatalf("findings = %+v, want the images_failed warning", manifest.Findings)
	}
	if steps.GenerateImages(deps).Timeout < 10*time.Minute {
		t.Errorf("the image step runs under %s, want room for every image", steps.GenerateImages(deps).Timeout)
	}
}

func TestGenerateImagesStopsForACancelledRun(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)
	deps.ImageProvider = &drawing{err: errors.New(errors.Cancelled, "the run was stopped")}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := steps.GenerateImages(deps).Run(ctx, imageContext(t, template.Images{Featured: true, Source: template.ImagesAI})); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("a stopped run must hand the step back, got %v", err)
	}
}

func TestGenerateImagesReportsAnImageItCouldNotPlace(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		picked images.Image
		spec   template.Images
	}{
		{
			name:   "an inline image the site gave no address",
			picked: images.Image{WPID: 42, Alt: "a cup"},
			spec:   template.Images{Inline: 1, Source: template.ImagesWPMedia},
		},
		{
			name:   "a featured image with no media id behind it",
			picked: images.Image{URL: "https://shop.example.com/a.png", Alt: "a cup"},
			spec:   template.Images{Featured: true, Source: template.ImagesWPMedia},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, _ := imageDeps(t)
			deps.ImageSources = map[template.ImageSource]steps.ImageSource{
				template.ImagesWPMedia: library{items: []images.Image{tc.picked}},
			}

			manifest, result := runImages(t, deps, imageContext(t, tc.spec))
			if len(manifest.Images) != 0 || manifest.FeaturedID != 0 {
				t.Fatalf("manifest = %+v, want nothing reported as placed", manifest)
			}
			for i := range result.Artifacts {
				if result.Artifacts[i].Kind == run.ArtifactBodyHTML {
					t.Fatal("the step rewrote the body although it placed nothing")
				}
			}
			if len(manifest.Findings) != 1 || manifest.Findings[0].Code != steps.CodeImageNotPlaced {
				t.Fatalf("findings = %+v, want the drop named", manifest.Findings)
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
