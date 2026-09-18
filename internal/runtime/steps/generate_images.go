package steps

import (
	"context"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameGenerateImages = "generate_images"

	RoleFeatured = "featured"
	RoleInline   = "inline"

	CodeNoImageSource = "no_image_source"
	CodeImagesFailed  = "images_failed"
	CodeUploadFailed  = "image_upload_failed"

	imageStepTimeout = 5 * time.Minute
)

type ImageProvider interface {
	Generate(ctx context.Context, prompt images.Prompt) (images.Image, error)
}

type ImageSource interface {
	Pick(ctx context.Context, query images.Query) ([]images.Image, error)
}

type PlacedImage struct {
	Role string `json:"role"`
	URL  string `json:"url"`
	Alt  string `json:"alt"`
	WPID int64  `json:"wpId"`
}

type ImagesResult struct {
	Images     []PlacedImage `json:"images"`
	Skipped    []string      `json:"skipped"`
	FeaturedID int64         `json:"featuredId"`
}

func GenerateImages(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameGenerateImages,
		Requires: []run.ArtifactKind{run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactImages, run.ArtifactBodyHTML},
		Retry:    run.RetryPolicy{Max: 2},
		Timeout:  imageStepTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			wanted := wantedImages(sc.Spec.Images)
			if wanted == 0 {
				return run.Result{Message: "the template asks for no image"}, nil
			}

			doc, err := bodyOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}

			acquired, skipped, err := acquire(ctx, deps, sc, entity, wanted)
			if err != nil {
				return run.Result{}, err
			}

			result := ImagesResult{Images: make([]PlacedImage, 0, len(acquired)), Skipped: skipped}
			uploaded := make([]images.Image, 0, len(acquired))
			for i := range acquired {
				stored, uploadErr := store(ctx, deps, sc, acquired[i])
				if uploadErr != nil {
					if ctx.Err() != nil {
						return run.Result{}, uploadErr
					}
					result.Skipped = append(result.Skipped, CodeUploadFailed+": "+uploadErr.Error())
					continue
				}
				uploaded = append(uploaded, stored)
			}

			artifacts := make([]run.Artifact, 0, 2)
			if place(doc, sc.Spec.Images, uploaded, &result) {
				artifacts = append(artifacts, run.Artifact{Kind: run.ArtifactBodyHTML, Blob: []byte(doc.HTML())})
			}

			blob, err := encode(result, "image manifest")
			if err != nil {
				return run.Result{}, err
			}
			artifacts = append(artifacts, run.Artifact{Kind: run.ArtifactImages, Blob: blob})

			return run.Result{
				Artifacts: artifacts,
				Message:   "placed " + strconv.Itoa(len(result.Images)) + " images on " + sc.Page.Path,
			}, nil
		},
	}
}

func wantedImages(spec template.Images) int {
	count := max(spec.Inline, 0)
	if spec.Featured {
		count++
	}
	return count
}

func acquire(ctx context.Context, deps Deps, sc *run.StepContext, entity graph.Entity, wanted int) ([]images.Image, []string, error) {
	if sc.Spec.Images.Source == template.ImagesAI {
		return generated(ctx, deps, sc, entity, wanted)
	}

	source, ok := deps.ImageSources[sc.Spec.Images.Source]
	if !ok || source == nil {
		return nil, []string{CodeNoImageSource + ": " + string(sc.Spec.Images.Source)}, nil
	}

	picked, err := source.Pick(ctx, images.Query{
		SiteID: sc.Run.SiteID, Term: subjectOf(sc, entity), Limit: wanted,
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, err
		}
		return nil, []string{CodeImagesFailed + ": " + err.Error()}, nil
	}
	return picked, nil, nil
}

func generated(ctx context.Context, deps Deps, sc *run.StepContext, entity graph.Entity, wanted int) ([]images.Image, []string, error) {
	if deps.ImageProvider == nil {
		return nil, []string{CodeNoImageSource + ": " + string(template.ImagesAI)}, nil
	}

	subject := subjectOf(sc, entity)
	out := make([]images.Image, 0, wanted)
	skipped := make([]string, 0)
	for index := range wanted {
		drawn, err := deps.ImageProvider.Generate(ctx, images.Prompt{
			SiteID:  sc.Run.SiteID,
			Subject: subject,
			Context: sceneOf(sc.Spec, index),
			Alt:     altOf(entity, subject),
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, nil, err
			}
			skipped = append(skipped, CodeImagesFailed+": "+err.Error())
			break
		}
		out = append(out, drawn)
	}
	return out, skipped, nil
}

func subjectOf(sc *run.StepContext, entity graph.Entity) string {
	if entity.Name != "" {
		return entity.Name
	}
	if sc.Page.Title != "" {
		return sc.Page.Title
	}
	return sc.Page.Path
}

func altOf(entity graph.Entity, subject string) string {
	if entity.PrimaryKeyword != "" {
		return entity.PrimaryKeyword
	}
	return subject
}

func sceneOf(spec template.TemplateSpec, index int) string {
	if index < len(spec.Sections) {
		return spec.Sections[index].Heading
	}
	return spec.Tone
}

func store(ctx context.Context, deps Deps, sc *run.StepContext, image images.Image) (images.Image, error) {
	if image.WPID != 0 || len(image.Bytes) == 0 {
		return image, nil
	}
	if deps.WordPress == nil {
		return images.Image{}, errors.New(errors.Invalid, "no WordPress client is configured for this site").
			WithDetail("siteId", sc.Run.SiteID)
	}

	client, err := deps.WordPress.Client(ctx, sc.Run.SiteID)
	if err != nil {
		return images.Image{}, err
	}

	uploaded, err := client.UploadMedia(ctx, wp.Media{
		Filename:    image.Filename,
		ContentType: image.ContentType,
		Bytes:       image.Bytes,
		Alt:         image.Alt,
		Title:       image.Alt,
	})
	if err != nil {
		return images.Image{}, err
	}

	image.WPID = uploaded.ID
	image.URL = uploaded.SourceURL
	image.Bytes = nil
	return image, nil
}

func place(doc *content.Document, spec template.Images, uploaded []images.Image, result *ImagesResult) bool {
	remaining := uploaded
	if spec.Featured && len(remaining) > 0 {
		featured := remaining[0]
		remaining = remaining[1:]
		result.FeaturedID = featured.WPID
		result.Images = append(result.Images, PlacedImage{
			Role: RoleFeatured, URL: featured.URL, Alt: featured.Alt, WPID: featured.WPID,
		})
	}

	changed := false
	for index := range min(spec.Inline, len(remaining)) {
		inline := remaining[index]
		if inline.URL == "" {
			continue
		}
		if err := doc.InsertAfterSection(index, figureOf(inline)); err != nil {
			continue
		}
		changed = true
		result.Images = append(result.Images, PlacedImage{
			Role: RoleInline, URL: inline.URL, Alt: inline.Alt, WPID: inline.WPID,
		})
	}
	return changed
}

func figureOf(image images.Image) string {
	var builder strings.Builder
	builder.WriteString(`<figure><img src="`)
	builder.WriteString(html.EscapeString(image.URL))
	builder.WriteString(`" alt="`)
	builder.WriteString(html.EscapeString(image.Alt))
	builder.WriteString(`"></figure>`)
	return builder.String()
}
