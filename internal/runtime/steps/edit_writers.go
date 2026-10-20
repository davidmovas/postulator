package steps

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/edit"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type editWriters struct {
	client *wp.Client
}

func editorFor(client *wp.Client) edit.Service {
	writers := editWriters{client: client}
	return edit.New(edit.Writers{Core: writers, Store: writers, Plugin: writers})
}

func (w editWriters) ReadCore(ctx context.Context, target edit.Target) (map[field.Key]string, error) {
	item, err := w.client.GetItem(ctx, wp.ItemType(target.Type), target.ID)
	if err != nil {
		return nil, err
	}
	return coreValues(item), nil
}

func (w editWriters) WriteCore(ctx context.Context, target edit.Target, values map[field.Key]string) (map[field.Key]string, error) {
	var in wp.UpdateItem
	for key, value := range values {
		switch key {
		case field.Title:
			in.Title = &value
		case field.Excerpt:
			in.Excerpt = &value
		case field.Body:
			in.Content = &value
		case field.FeaturedImage:
			media, err := mediaID(value)
			if err != nil {
				return nil, err
			}
			in.FeaturedMedia = &media
		default:
			return nil, errors.New(errors.Invalid, "the core writer takes no such field").WithDetail("key", string(key))
		}
	}
	item, err := w.client.UpdateItem(ctx, wp.ItemType(target.Type), target.ID, in)
	if err != nil {
		return nil, err
	}
	return coreValues(item), nil
}

func coreValues(item wp.Item) map[field.Key]string {
	featured := ""
	if item.FeaturedMedia != 0 {
		featured = strconv.FormatInt(item.FeaturedMedia, 10)
	}
	return map[field.Key]string{
		field.Title: item.Title, field.Excerpt: item.Excerpt, field.Body: item.Content, field.FeaturedImage: featured,
	}
}

func mediaID(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 0 {
		return 0, errors.New(errors.Invalid, "a featured image is a media id").WithDetail("value", value)
	}
	return id, nil
}

func (w editWriters) ReadStore(ctx context.Context, target edit.Target) (map[field.Key]string, error) {
	product, err := w.client.GetProduct(ctx, target.ID)
	if err != nil {
		return nil, err
	}
	return storeValues(product), nil
}

func (w editWriters) WriteStore(ctx context.Context, target edit.Target, values map[field.Key]string) (map[field.Key]string, error) {
	var in wp.UpdateProduct
	for key, value := range values {
		switch key {
		case field.ProductShortDescription:
			in.ShortDescription = &value
		case field.FeaturedImage:
			image, err := mediaID(value)
			if err != nil || image == 0 {
				return nil, errors.New(errors.Invalid, "a product's first image is only ever filled with a media id").
					WithDetail("value", value)
			}
			in.Images = &[]int64{image}
		default:
			return nil, errors.New(errors.Invalid, "the store writer takes no such field").WithDetail("key", string(key))
		}
	}
	product, err := w.client.UpdateProduct(ctx, target.ID, in)
	if err != nil {
		return nil, err
	}
	return storeValues(product), nil
}

func storeValues(product wp.Product) map[field.Key]string {
	first := ""
	if len(product.Images) > 0 {
		first = strconv.FormatInt(product.Images[0].ID, 10)
	}
	return map[field.Key]string{field.ProductShortDescription: product.ShortDescription, field.FeaturedImage: first}
}

func (w editWriters) Plugin(ctx context.Context) (bool, error) {
	err := w.client.GuardedWrites(ctx)
	switch {
	case wp.IsPluginMissing(err):
		return false, nil
	case err != nil:
		return true, err
	default:
		return true, nil
	}
}

func (w editWriters) WriteBody(ctx context.Context, target edit.Target, content, expected string) (edit.Stored, *string, error) {
	written, err := w.client.PutRaw(ctx, wp.ItemType(target.Type), target.ID, content, expected)
	if err != nil {
		if errors.IsCode(err, errors.Conflict) {
			current := currentHashOf(err)
			return edit.Stored{}, &current, nil
		}
		return edit.Stored{}, nil, err
	}
	return edit.Stored{Value: &written.Content, Hash: &written.Hash}, nil, nil
}

func (w editWriters) WriteSEO(ctx context.Context, target edit.Target, values, expected map[string]string) (hashes, moved map[string]string, err error) {
	var meta wp.SEOMeta
	fields := make([]string, 0, len(values))
	for name, value := range values {
		switch name {
		case "title":
			meta.Title = value
		case "description":
			meta.Description = value
		case "canonical":
			meta.Canonical = value
		case "ogTitle":
			meta.OGTitle = value
		case "ogDescription":
			meta.OGDescription = value
		default:
			return nil, nil, errors.New(errors.Invalid, "the SEO writer takes no such field").WithDetail("field", name)
		}
		fields = append(fields, name)
	}
	result, err := w.client.ReplaceSEOMeta(ctx, wp.ItemType(target.Type), target.ID, meta, fields, expected)
	if err != nil {
		if shifted, conflict := movedOf(err); conflict {
			return nil, shifted, nil
		}
		return nil, nil, err
	}
	return result.Hashes, nil, nil
}

func (w editWriters) WriteMeta(ctx context.Context, target edit.Target, writes map[string]edit.MetaWrite) (written map[string]edit.Stored,
	moved map[string]*string, err error) {
	wire := make(map[string]wp.MetaWrite, len(writes))
	for key, write := range writes {
		wire[key] = wp.MetaWrite(write)
	}
	stored, err := w.client.PutMeta(ctx, wp.ItemType(target.Type), target.ID, wire)
	if err != nil {
		if shifted, conflict := movedOf(err); conflict {
			out := make(map[string]*string, len(shifted))
			for key, hash := range shifted {
				if hash == "" {
					out[key] = nil
					continue
				}
				out[key] = &hash
			}
			return nil, out, nil
		}
		return nil, nil, err
	}
	out := make(map[string]edit.Stored, len(stored))
	for key, held := range stored {
		out[key] = edit.Stored(held)
	}
	return out, nil, nil
}

func refusedFor(err error, reason string) bool {
	var kerr *errors.Error
	if !errors.IsCode(err, errors.Invalid) || !stderrors.As(err, &kerr) {
		return false
	}
	held, ok := kerr.Details["reason"].(string)
	return ok && held == reason
}

func movedOf(err error) (map[string]string, bool) {
	var kerr *errors.Error
	if !errors.IsCode(err, errors.Conflict) || !stderrors.As(err, &kerr) {
		return nil, false
	}
	moved, ok := kerr.Details["moved"].(map[string]string)
	return moved, ok
}
