package wp

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

var _ Client = (*restyClient)(nil)

const (
	requestTimeout = time.Second * 30
	apiPath        = "/wp-json/wp/v2/"
)

type restyClient struct {
	resty *resty.Client
}

func NewRestyClient() Client {
	return &restyClient{
		resty: resty.New().
			SetTimeout(requestTimeout),
	}
}

func (c *restyClient) WithProxy(proxyURL string) Client {
	c.resty.SetProxy(proxyURL)
	return c
}

func (c *restyClient) WithoutProxy() Client {
	c.resty.RemoveProxy()
	return c
}

func (c *restyClient) WithTimeout(timeout int) Client {
	c.resty.SetTimeout(time.Duration(timeout) * time.Second)
	return c
}

func (c *restyClient) getAPIURL(siteURL, endpoint string) string {
	baseURL := strings.TrimSuffix(siteURL, "/")

	if strings.HasPrefix(endpoint, "/") {
		return baseURL + endpoint
	}

	return baseURL + apiPath + endpoint
}

func (c *restyClient) setAppPasswordAuth(req *http.Request, username, appPassword string) {
	auth := username + ":" + appPassword
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", basicAuth)
}
