package wp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
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
		err = errors.SiteUnreachable(site.URL, fmt.Errorf("request failed: %w", err))
		health.Error = err.Error()
		health.Status = entities.HealthError
		return health, err
	}

	health.Code = resp.StatusCode()
	health.StatusCode = resp.Status()

	switch {
	case resp.StatusCode() == http.StatusOK:
		var data map[string]any
		if err = json.Unmarshal(resp.Body(), &data); err != nil {
			err = errors.WordPress("invalid JSON in /wp-json response", err)
			health.Error = err.Error()
			health.Status = entities.HealthError
			return health, err
		}

		if _, ok := data["namespaces"]; ok {
			health.Status = entities.HealthHealthy
			return health, nil
		}

		err = errors.WordPress("unexpected /wp-json structure", nil)
		health.Error = err.Error()
		health.Status = entities.HealthUnknown
		return health, err

	case resp.StatusCode() >= 500:
		err = errors.WordPress(fmt.Sprintf("WordPress server error: %d", resp.StatusCode()), nil)
		health.Error = err.Error()
		health.Status = entities.HealthUnhealthy
		return health, err

	case resp.StatusCode() >= 400:
		err = errors.WordPress(fmt.Sprintf("WordPress API returned code: %d", resp.StatusCode()), nil)
		health.Error = err.Error()
		health.Status = entities.HealthError
		return health, err

	case resp.StatusCode() >= 300:
		location := resp.Header().Get("Location")
		if location == "" {
			err = errors.WordPress(fmt.Sprintf("redirect (%d) without Location header", resp.StatusCode()), nil)
			health.Error = err.Error()
			health.Status = entities.HealthUnknown
			return health, err
		}

		if strings.HasPrefix(location, "https://") &&
			strings.TrimPrefix(site.URL, "http://") == strings.TrimPrefix(location, "https://") {
			health.Status = entities.HealthHealthy
			return health, nil
		}

		if strings.Contains(location, "wp-login.php") {
			err = errors.WordPress("redirected to login page", nil)
			health.Error = err.Error()
			health.Status = entities.HealthHealthy
			return health, err
		}

		err = errors.WordPress(fmt.Sprintf("unexpected redirect (%d) to: %s", resp.StatusCode(), location), nil)
		health.Error = err.Error()
		health.Status = entities.HealthUnknown
		return health, err

	default:
		err = errors.WordPress(fmt.Sprintf("unexpected status code: %d", resp.StatusCode()), nil)
		health.Error = err.Error()
		health.Status = entities.HealthUnknown
		return health, err
	}
}
