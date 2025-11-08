package wp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

func (c *restyClient) CheckHealth(ctx context.Context, site *entities.Site) (*entities.HealthCheck, error) {
	health := &entities.HealthCheck{
		SiteID: site.ID,
		Status: entities.HealthUnknown,
	}

	start := time.Now()
	defer func() {
		health.ResponseTime = time.Since(start)
	}()

	url := fmt.Sprintf("%s/wp-json", strings.TrimSuffix(site.URL, "/"))

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(site.WPUsername, site.WPPassword).
		Get(url)

	if err != nil {
		health.Error = err.Error()
		health.Status = entities.HealthUnhealthy
		health.Code = 0
		return health, nil
	}

	health.Code = resp.StatusCode()
	health.StatusCode = resp.Status()

	switch {
	case resp.StatusCode() == http.StatusOK:
		var data map[string]any
		if err = json.Unmarshal(resp.Body(), &data); err != nil {
			health.Error = fmt.Sprintf("Invalid JSON in /wp-json response: %v", err)
			health.Status = entities.HealthUnhealthy
			return health, nil
		}

		if _, ok := data["namespaces"]; ok {
			health.Status = entities.HealthHealthy
			return health, nil
		}

		health.Error = "Unexpected /wp-json structure - missing 'namespaces'"
		health.Status = entities.HealthUnhealthy
		return health, nil

	case resp.StatusCode() >= 500:
		health.Error = fmt.Sprintf("WordPress server error: %s", resp.Status())
		health.Status = entities.HealthUnhealthy
		return health, nil

	case resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden:
		health.Error = fmt.Sprintf("Authentication failed: %s", resp.Status())
		health.Status = entities.HealthError
		return health, nil

	case resp.StatusCode() >= 400:
		health.Error = fmt.Sprintf("WordPress API error: %s", resp.Status())
		health.Status = entities.HealthUnhealthy
		return health, nil

	case resp.StatusCode() >= 300:
		location := resp.Header().Get("Location")

		if location == "" {
			health.Error = fmt.Sprintf("Redirect without Location header: %s", resp.Status())
			health.Status = entities.HealthUnhealthy
			return health, nil
		}

		if strings.HasPrefix(location, "https://") {
			httpURL := strings.TrimPrefix(site.URL, "http://")
			httpsURL := strings.TrimPrefix(location, "https://")
			if httpURL == httpsURL || strings.HasPrefix(httpsURL, httpURL) {
				health.Status = entities.HealthHealthy
				health.Error = "Site redirects to HTTPS"
				return health, nil
			}
		}

		if strings.Contains(location, "wp-login.php") || strings.Contains(location, "wp-admin") {
			health.Error = "Redirected to login page - authentication required"
			health.Status = entities.HealthError
			return health, nil
		}

		health.Error = fmt.Sprintf("Unexpected redirect: %s to %s", resp.Status(), location)
		health.Status = entities.HealthUnhealthy
		return health, nil

	default:
		health.Error = fmt.Sprintf("Unexpected status code: %s", resp.Status())
		health.Status = entities.HealthUnknown
		return health, nil
	}
}
