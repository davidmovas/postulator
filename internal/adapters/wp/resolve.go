package wp

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const maxRedirects = 10

type Resolution struct {
	URL    string
	Status int
}

func (c *Client) Resolve(ctx context.Context, rawURL string) (Resolution, error) {
	current, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !current.IsAbs() || current.Host == "" {
		return Resolution{}, errors.New(errors.Invalid, "the address to resolve is not an absolute URL").WithDetail("link", rawURL)
	}
	if !c.onSite(current) {
		return Resolution{}, errors.New(errors.Invalid, "the address to resolve is not on this site").WithDetail("link", rawURL)
	}

	for redirects := 0; ; redirects++ {
		status, location, err := c.hop(ctx, current)
		if err != nil {
			return Resolution{}, err
		}
		if !redirected(status) || location == "" {
			return Resolution{URL: current.String(), Status: status}, nil
		}
		if redirects == maxRedirects {
			return Resolution{}, errors.New(errors.External, "the address redirects more than ten times").WithDetail("link", rawURL)
		}
		next, err := current.Parse(location)
		if err != nil {
			return Resolution{}, errors.New(errors.External, "the site redirected to an address that cannot be read").
				WithDetail("link", rawURL).WithInternal(err)
		}
		if !c.onSite(next) {
			return Resolution{URL: next.String(), Status: status}, nil
		}
		current = next
	}
}

func (c *Client) hop(ctx context.Context, target *url.URL) (status int, location string, err error) {
	status, location, err = c.peek(ctx, http.MethodHead, target)
	if err != nil || (status != http.StatusMethodNotAllowed && status != http.StatusNotImplemented) {
		return status, location, err
	}
	return c.peek(ctx, http.MethodGet, target)
}

func (c *Client) peek(ctx context.Context, method string, target *url.URL) (status int, location string, err error) {
	if waitErr := c.limiter.Wait(ctx); waitErr != nil {
		return 0, "", errors.New(errors.Cancelled, "the WordPress request was cancelled while waiting for the site rate limit").WithInternal(waitErr)
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), http.NoBody)
	if err != nil {
		return 0, "", errors.New(errors.Invalid, "the WordPress request could not be built").WithInternal(err)
	}
	req.Header.Set("Accept", acceptPage)
	req.Header.Set("User-Agent", c.userAgent)

	started := time.Now()
	resp, err := c.probe.Do(req)
	logged := request{method: method, path: target.EscapedPath()}
	if err != nil {
		failure := transportError(ctx, err)
		c.logAttempt(logged, attempt{err: failure}, time.Since(started))
		return 0, "", failure
	}
	_ = resp.Body.Close()
	c.logAttempt(logged, attempt{status: resp.StatusCode}, time.Since(started))
	return resp.StatusCode, resp.Header.Get("Location"), nil
}

func (c *Client) onSite(target *url.URL) bool {
	return hostPort(target) == hostPort(c.base)
}

func hostPort(target *url.URL) string {
	port := target.Port()
	if port == "" {
		switch strings.ToLower(target.Scheme) {
		case "https":
			port = "443"
		default:
			port = "80"
		}
	}
	return strings.ToLower(target.Hostname()) + ":" + port
}

func redirected(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}
