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
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameGenerateImages = string(run.StepGenerateImages)

	RoleFeatured = "featured"
	RoleInline   = "inline"

	CodeNoImageSource  = "no_image_source"
	CodeImagesFailed   = "images_failed"
	CodeUploadFailed   = "image_upload_failed"
	CodeImageNotPlaced = "image_not_placed"

	imageStepTimeout  = 15 * time.Minute
	imageOutputTokens = 1056
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
	Images     []PlacedImage     `json:"images"`
	Findings   []content.Finding `json:"findings"`
	FeaturedID int64             `json:"featuredId"`
}

func (r *ImagesResult) skip(page pagemap.Page, code, message, reason string) {
	r.Findings = append(r.Findings, content.Finding{
		Severity: content.SeverityWarn,
		Code:     code,
		Message:  message,
		Details:  map[string]any{"pageId": page.ID, "path": page.Path, "reason": reason},
	})
}

func GenerateImages(deps Deps) run.StepDef {
	return run.StepDef{
		Name:      NameGenerateImages,
		Preflight: imagesPreflight(deps),
		Requires:  []run.ArtifactKind{run.ArtifactBodyHTML},
		Produces:  []run.ArtifactKind{run.ArtifactImages, run.ArtifactBodyHTML},
		Retry:     run.RetryPolicy{Max: 2},
		Timeout:   imageStepTimeout,
		Price: run.Price{
			Ref:          deps.ImageModel,
			OutputTokens: imageOutputTokens,
			Unpriced:     deps.ImageModel == nil,
			Calls: func(spec template.TemplateSpec, _ map[string]any) int {
				if spec.Images.Source != template.ImagesAI {
					return 0
				}
				return spec.Images.Wanted()
			},
		},
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			result := ImagesResult{
				Images:   make([]PlacedImage, 0),
				Findings: make([]content.Finding, 0),
			}

			wanted := sc.Spec.Images.Wanted()
			if wanted == 0 {
				return manifest(result, nil, "the template asks for no image")
			}

			doc, err := bodyOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}

			acquired, err := acquire(ctx, deps, sc, entity, wanted, &result)
			if err != nil {
				return run.Result{}, err
			}

			uploaded := make([]images.Image, 0, len(acquired))
			for i := range acquired {
				stored, uploadErr := store(ctx, deps, sc, acquired[i])
				if uploadErr != nil {
					if stopped(ctx) {
						return run.Result{}, uploadErr
					}
					result.skip(sc.Page, CodeUploadFailed,
						"an image for "+sc.Page.Path+" reached no media library, so it was not placed: "+
							uploadErr.Error(), uploadErr.Error())
					continue
				}
				uploaded = append(uploaded, stored)
			}

			var body []byte
			if place(doc, sc.Spec.Images, uploaded, &result, sc.Page) {
				rendered, renderErr := doc.Render()
				if renderErr != nil {
					return run.Result{}, renderErr
				}
				body = []byte(rendered)
			}

			return manifest(result, body,
				"placed "+strconv.Itoa(len(result.Images))+" images on "+sc.Page.Path)
		},
	}
}

func manifest(result ImagesResult, body []byte, message string) (run.Result, error) {
	artifacts := make([]run.Artifact, 0, 2)
	if body != nil {
		artifacts = append(artifacts, run.Artifact{Kind: run.ArtifactBodyHTML, Blob: body})
	}

	blob, err := encode(result, "image manifest")
	if err != nil {
		return run.Result{}, err
	}
	artifacts = append(artifacts, run.Artifact{Kind: run.ArtifactImages, Blob: blob})
	return run.Result{Artifacts: artifacts, Message: message}, nil
}

func acquire(ctx context.Context, deps Deps, sc *run.StepContext, entity graph.Entity, wanted int,
	result *ImagesResult) ([]images.Image, error) {
	if sc.Spec.Images.Source == template.ImagesAI {
		return generated(ctx, deps, sc, entity, wanted, result)
	}

	source, ok := deps.ImageSources[sc.Spec.Images.Source]
	if !ok || source == nil {
		result.skip(sc.Page, CodeNoImageSource,
			"no image library named "+string(sc.Spec.Images.Source)+" is configured, so "+sc.Page.Path+
				" was written without the images its template asks for", string(sc.Spec.Images.Source))
		return nil, nil
	}

	picked, err := source.Pick(ctx, images.Query{
		SiteID: sc.Run.SiteID, Term: subjectOf(sc, entity), Limit: wanted,
	})
	if err != nil {
		if stopped(ctx) {
			return nil, err
		}
		result.skip(sc.Page, CodeImagesFailed,
			"the image library answered nothing for "+sc.Page.Path+": "+err.Error(), err.Error())
		return nil, nil
	}
	return picked, nil
}

func generated(ctx context.Context, deps Deps, sc *run.StepContext, entity graph.Entity, wanted int,
	result *ImagesResult) ([]images.Image, error) {
	if deps.ImageProvider == nil {
		result.skip(sc.Page, CodeNoImageSource,
			"no model is configured to draw images, so "+sc.Page.Path+
				" was written without the images its template asks for", string(template.ImagesAI))
		return nil, nil
	}

	subject := subjectOf(sc, entity)
	out := make([]images.Image, 0, wanted)
	for index := range wanted {
		drawn, err := deps.ImageProvider.Generate(ctx, images.Prompt{
			SiteID:  sc.Run.SiteID,
			RunID:   sc.Run.ID,
			ItemID:  sc.Item.ID,
			Step:    NameGenerateImages,
			Subject: subject,
			Context: sceneOf(sc.Spec, index),
			Alt:     altOf(entity, subject),
		})
		if err != nil {
			if stopped(ctx) {
				return nil, err
			}
			result.skip(sc.Page, CodeImagesFailed,
				"the image model stopped after "+strconv.Itoa(len(out))+" of "+strconv.Itoa(wanted)+
					" images for "+sc.Page.Path+": "+err.Error(), err.Error())
			break
		}
		out = append(out, drawn)
	}
	return out, nil
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

func place(doc *content.Document, spec template.Images, uploaded []images.Image, result *ImagesResult,
	page pagemap.Page) bool {
	remaining := uploaded
	if spec.Featured && len(remaining) > 0 {
		featured := remaining[0]
		remaining = remaining[1:]
		if featured.WPID == 0 {
			result.skip(page, CodeImageNotPlaced,
				"the featured image of "+page.Path+" carries no media id, so the site has nothing to set",
				featured.URL)
		} else {
			result.FeaturedID = featured.WPID
			result.Images = append(result.Images, PlacedImage{
				Role: RoleFeatured, URL: featured.URL, Alt: featured.Alt, WPID: featured.WPID,
			})
		}
	}

	changed := false
	for index := range min(spec.Inline, len(remaining)) {
		inline := remaining[index]
		if inline.URL == "" {
			result.skip(page, CodeImageNotPlaced,
				"an image for "+page.Path+" carries no address, so it was left out of the body",
				strconv.FormatInt(inline.WPID, 10))
			continue
		}
		if err := doc.InsertAfterSection(index, figureOf(inline)); err != nil {
			result.skip(page, CodeImageNotPlaced,
				"an image for "+page.Path+" found no place in the body: "+err.Error(), err.Error())
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
