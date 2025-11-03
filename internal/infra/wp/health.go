package wp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

func (c *restyClient) CheckHealth(ctx context.Context, site *entities.Site) (entities.HealthStatus, error) {
	endpoint := c.getAPIURL(site.URL, "")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return entities.HealthUnknown, errors.WordPress("error while requesting", err)
	}

	c.setAppPasswordAuth(req, site.WPUsername, site.WPPassword)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return entities.HealthUnknown, errors.WordPress("error while requesting", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return entities.HealthUnhealthy, errors.WordPress(fmt.Sprintf("WordPress API respond with code: %d", resp.StatusCode), nil)
	}

	return entities.HealthHealthy, nil
}
