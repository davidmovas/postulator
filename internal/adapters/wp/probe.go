package wp

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type ProbeStatus string

const (
	ProbeOK              ProbeStatus = "ok"
	ProbeUpgradeRequired ProbeStatus = "upgrade_required"
)

type ProbeResult struct {
	Status           ProbeStatus
	Namespaces       []string
	Warnings         []string
	SiteName         string
	Description      string
	HomeURL          string
	SuggestedBaseURL string
	HasPlugin        bool
	HasWoo           bool
}

type restRoot struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Home        string   `json:"home"`
	Namespaces  []string `json:"namespaces"`
}

func (c *Client) Probe(ctx context.Context) (ProbeResult, error) {
	resp, body, err := c.do(ctx, request{
		method:       http.MethodGet,
		namespace:    rootPath,
		client:       c.probe,
		keepRedirect: true,
	})
	if err != nil {
		if errors.IsCode(err, errors.NotFound) {
			return ProbeResult{}, errors.New(errors.External, "the site has no WordPress REST API at /wp-json").WithInternal(err)
		}
		return ProbeResult{}, err
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		return c.classifyRedirect(resp)
	}

	var root restRoot
	if err = decodeJSON(body, &root); err != nil {
		return ProbeResult{}, errors.New(errors.External, "the REST root is not a WordPress REST index").WithInternal(err)
	}
	if len(root.Namespaces) == 0 {
		return ProbeResult{}, errors.New(errors.External, "the REST root reported no namespaces, so this is not a WordPress site")
	}

	return ProbeResult{
		Status:      ProbeOK,
		Namespaces:  root.Namespaces,
		SiteName:    root.Name,
		Description: root.Description,
		HomeURL:     root.Home,
		HasPlugin:   slices.Contains(root.Namespaces, "postulator/v1"),
		HasWoo:      slices.Contains(root.Namespaces, "wc/v3"),
	}, nil
}

func (c *Client) classifyRedirect(resp *http.Response) (ProbeResult, error) {
	location := resp.Header.Get("Location")
	if location == "" {
		return ProbeResult{}, errors.New(errors.External, "the REST root redirected without a location").
			WithDetail("status", resp.StatusCode)
	}

	parsed, err := url.Parse(location)
	if err != nil {
		return ProbeResult{}, errors.New(errors.External, "the REST root redirected to an unreadable location").WithInternal(err)
	}
	target := resp.Request.URL.ResolveReference(parsed)

	if loginRedirect(target.Path) {
		return ProbeResult{}, errors.New(errors.Unauthorized, "the site redirected the REST root to the WordPress login page").
			WithDetail("location", target.Path)
	}

	if target.Scheme == "https" && c.base.Scheme == "http" && strings.EqualFold(target.Host, c.base.Host) {
		return ProbeResult{
			Status:           ProbeUpgradeRequired,
			SuggestedBaseURL: "https://" + c.base.Host + c.base.Path,
			Warnings:         []string{"the site redirects http to https; store the site base URL with the https scheme"},
		}, nil
	}

	return ProbeResult{}, errors.New(errors.External, "the REST root redirected somewhere unexpected").
		WithDetail("location", target.String())
}

func loginRedirect(path string) bool {
	return strings.Contains(path, "wp-login.php") || strings.Contains(path, "/wp-admin")
}
