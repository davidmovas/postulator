package wp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

func (c *restyClient) CheckHealth(ctx context.Context, site *entities.Site) (entities.HealthStatus, error) {
	url := fmt.Sprintf("%s/wp-json/", strings.TrimSuffix(site.URL, "/"))

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(site.WPUsername, site.WPPassword).
		Get(url)

	if err != nil {
		return entities.HealthUnknown, errors.WordPress("error while requesting site health", err)
	}

	status := resp.StatusCode()

	switch {
	case status == http.StatusOK:
		var data map[string]any
		if err = json.Unmarshal(resp.Body(), &data); err != nil {
			return entities.HealthUnknown, errors.WordPress("invalid JSON in /wp-json response", err)
		}
		if _, ok := data["namespaces"]; ok {
			return entities.HealthHealthy, nil
		}
		return entities.HealthUnknown, errors.WordPress("unexpected /wp-json structure", nil)

	case status >= 500:
		return entities.HealthUnhealthy, errors.WordPress(
			fmt.Sprintf("WordPress server error: %d", status), nil)

	case status >= 400:
		return entities.HealthUnhealthy, errors.WordPress(
			fmt.Sprintf("WordPress API returned code: %d", status), nil)

	case status >= 300:
		location := resp.Header().Get("Location")
		if location == "" {
			return entities.HealthUnknown, errors.WordPress(
				fmt.Sprintf("redirect (%d) without Location header", status), nil)
		}

		if strings.HasPrefix(location, "https://") &&
			strings.TrimPrefix(site.URL, "http://") == strings.TrimPrefix(location, "https://") {
			return entities.HealthHealthy, nil
		}

		if strings.Contains(location, "wp-login.php") {
			return entities.HealthUnknown, errors.WordPress("redirected to login page", nil)
		}

		return entities.HealthUnhealthy, errors.WordPress(
			fmt.Sprintf("unexpected redirect (%d) to: %s", status, location), nil)

	default:
		return entities.HealthUnknown, errors.WordPress(
			fmt.Sprintf("unexpected status code: %d", status), nil)
	}
}
