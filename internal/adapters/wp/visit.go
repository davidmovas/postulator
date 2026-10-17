package wp

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	acceptPage    = "text/html,application/xhtml+xml"
	nonceArgument = "_"
)

type Visit struct {
	Body   string
	Status int
}

func (c *Client) Visit(ctx context.Context, link string) (Visit, error) {
	target, err := url.Parse(strings.TrimSpace(link))
	if err != nil || !target.IsAbs() || target.Host == "" {
		return Visit{}, errors.New(errors.Invalid, "the address to visit is not an absolute URL").WithDetail("link", link)
	}
	if !strings.EqualFold(target.Host, c.base.Host) {
		return Visit{}, errors.New(errors.Invalid, "the address to visit is not on this site").WithDetail("link", link)
	}
	target.RawQuery = freshQuery(target.RawQuery, time.Now())

	resp, body, err := c.do(ctx, request{
		method: http.MethodGet, path: target.Path, absolute: target.String(), anonymous: true, accept: acceptPage,
		header: http.Header{"Cache-Control": []string{"no-cache"}},
	})
	if err != nil {
		return Visit{}, err
	}
	return Visit{Status: resp.StatusCode, Body: string(body)}, nil
}

func freshQuery(raw string, at time.Time) string {
	nonce := nonceArgument + "=" + strconv.FormatInt(at.UnixNano(), 10)
	if raw == "" {
		return nonce
	}
	return raw + "&" + nonce
}
