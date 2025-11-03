package wp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

func (c *restyClient) CheckHealth(ctx context.Context, site *entities.Site) (entities.HealthStatus, error) {
	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(site.WPUsername, site.WPPassword).
		Get(c.getAPIURL(site.URL, ""))
	if err != nil {
		return entities.HealthUnknown, errors.WordPress("error while requesting", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return entities.HealthUnhealthy, errors.WordPress(fmt.Sprintf("WordPress API responded with code: %d", resp.StatusCode()), nil)
	}

	return entities.HealthHealthy, nil
}
