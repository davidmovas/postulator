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
	"github.com/davidmovas/postulator/internal/domain/keyword"
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
	CodeImagesShort    = "images_short"

	CodeImagesWaitForLive = "images_wait_for_live"
	CodeFeaturedImageKept = "featured_image_kept"

	imageStepTimeout  = 15 * time.Minute
	imageOutputTokens = 1056
)

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
	Wanted     int               `json:"wanted"`
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
			Calls: func(target run.Target, _ map[string]any) int {
				if target.Spec.Images.Source != template.ImagesAI {
					return 0
				}
				return target.Spec.Images.Wanted()
			},
		},
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			result := ImagesResult{
				Images:   make([]PlacedImage, 0),
				Findings: make([]content.Finding, 0),
			}

			spec := sc.Spec.Images
			result.Wanted = spec.Wanted()
			if result.Wanted == 0 {
				return manifest(result, nil, "the template asks for no image")
			}
			if reviewing(sc) && sc.Page.WPID != nil {
				result.skip(sc.Page, CodeImagesWaitForLive,
					sc.Page.Path+" is on the site and this is a draft run, so no image was drawn or uploaded: an upload "+
						"writes to the site, which a draft run does not do to a page that is already there", "draft_run")
				return manifest(result, nil, "no image for a page already on the site in a draft run")
			}
			if spec.Featured && sc.Page.WPID != nil {
				kept, keptErr := keepsItsImage(ctx, deps, sc)
				if keptErr != nil {
					return run.Result{}, keptErr
				}
				if kept {
					spec.Featured = false
					result.Wanted = spec.Wanted()
					result.Findings = append(result.Findings, content.Finding{
						Severity: content.SeverityInfo,
						Code:     CodeFeaturedImageKept,
						Message:  sc.Page.Path + " keeps the image it has on the site, so no featured image was drawn or uploaded for it",
						Details:  map[string]any{"pageId": sc.Page.ID, "path": sc.Page.Path},
					})
				}
				if result.Wanted == 0 {
					return manifest(result, nil, "kept the image "+sc.Page.Path+" has on the site")
				}
			}

			doc, err := bodyOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}

			acquired, err := acquire(ctx, deps, sc, entity, result.Wanted, &result)
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
			if place(doc, spec, uploaded, &result, sc.Page) {
				rendered, renderErr := doc.Render()
				if renderErr != nil {
					return run.Result{}, renderErr
				}
				body = []byte(rendered)
			}

			return manifest(result, body, placedMessage(result, sc.Page))
		},
	}
}

func keepsItsImage(ctx context.Context, deps Deps, sc *run.StepContext) (bool, error) {
	if deps.WordPress == nil {
		return false, errors.New(errors.Invalid, "no WordPress client is configured for this site").
			WithDetail("siteId", sc.Run.SiteID)
	}
	client, err := deps.WordPress.Client(ctx, sc.Run.SiteID)
	if err != nil {
		return false, err
	}
	if sc.Page.WPType == pagemap.WPProduct {
		product, productErr := client.GetProduct(ctx, *sc.Page.WPID)
		return len(product.Images) > 0, productErr
	}
	itemType, err := itemTypeOf(sc.Page)
	if err != nil {
		return false, err
	}
	item, err := client.GetItem(ctx, itemType, *sc.Page.WPID)
	return item.FeaturedMedia != 0, err
}

func placedMessage(result ImagesResult, page pagemap.Page) string {
	message := "placed " + strconv.Itoa(len(result.Images)) + " of " + strconv.Itoa(result.Wanted) +
		" images on " + page.Path
	if len(result.Images) < result.Wanted && len(result.Findings) > 0 {
		message += "; " + result.Findings[0].Message
	}
	return message
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

	term := subjectOf(sc, entity)
	picked, err := source.Pick(ctx, images.Query{SiteID: sc.Run.SiteID, Term: term, Limit: wanted})
	if err != nil {
		if stopped(ctx) {
			return nil, err
		}
		result.skip(sc.Page, CodeImagesFailed,
			"the image library answered nothing for "+sc.Page.Path+": "+err.Error(), err.Error())
		return nil, nil
	}
	if len(picked) < wanted {
		result.skip(sc.Page, CodeImagesShort,
			"the "+string(sc.Spec.Images.Source)+" library found "+strconv.Itoa(len(picked))+" of "+
				strconv.Itoa(wanted)+" images matching "+strconv.Quote(term)+" for "+sc.Page.Path+
				", so the page carries fewer images than its template asks for", string(sc.Spec.Images.Source))
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
			Alt:     altOf(pagemap.Keywords(sc.Page, entity), subject),
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

func altOf(keywords keyword.List, subject string) string {
	if main := keywords.Main(); main != "" {
		return main
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
