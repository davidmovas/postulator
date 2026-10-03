package steps

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const ReasonNoPlugin = "the site has no Postulator companion plugin"

var editableStatuses = []string{"publish", "future", "draft", "pending", "private"}

func clientFor(ctx context.Context, deps Deps, siteID string) (*wp.Client, error) {
	if deps.WordPress == nil {
		return nil, errors.New(errors.Invalid, "no WordPress client is configured for this site").
			WithDetail("siteId", siteID)
	}
	return deps.WordPress.Client(ctx, siteID)
}

func itemTypeOf(page pagemap.Page) (wp.ItemType, error) {
	switch page.WPType {
	case pagemap.WPPage:
		return wp.TypePage, nil
	case pagemap.WPPost:
		return wp.TypePost, nil
	default:
		return "", errors.New(errors.Invalid, "only pages, posts and products are written by the publish step").
			WithDetail("wpType", string(page.WPType)).WithDetail("pageId", page.ID)
	}
}

func onSiteType(page pagemap.Page) wp.ItemType {
	return wp.ItemType(page.WPType)
}

func observedOf(item wp.Item) pagemap.Observed {
	return pagemap.Observed{
		Link: permalinkOf(item), Slug: item.Slug, Status: item.Status, Title: item.Title,
	}
}

func permalinkOf(item wp.Item) string {
	if strings.Contains(item.Link, "?") {
		return ""
	}
	return item.Link
}

func currentHashOf(err error) string {
	return detailOf(err, "currentHash")
}

func detailOf(err error, key string) string {
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel == nil {
		return ""
	}
	value, ok := kernel.Details[key].(string)
	if !ok {
		return ""
	}
	return value
}
