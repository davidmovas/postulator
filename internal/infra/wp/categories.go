package wp

import (
	"context"
	"fmt"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

func (c *restyClient) GetCategories(ctx context.Context, s *entities.Site) ([]*entities.Category, error) {
	var wpCategories []wpCategory

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetQueryParams(map[string]string{
			"per_page": "100",
			"orderby":  "name",
		}).
		SetResult(&wpCategories).
		Get(c.getAPIURL(s.URL, "categories"))
	if err != nil {
		return nil, errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() != 200 {
		return nil, errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	categories := make([]*entities.Category, 0, len(wpCategories))
	for _, wpCat := range wpCategories {
		categories = append(categories, &entities.Category{
			WPCategoryID: wpCat.ID,
			Name:         wpCat.Name,
			Slug:         &wpCat.Slug,
			Description:  &wpCat.Description,
			Count:        wpCat.Count,
		})
	}

	return categories, nil
}

func (c *restyClient) CreateCategory(ctx context.Context, s *entities.Site, category *entities.Category) error {
	wpCategoryData := map[string]interface{}{
		"name": category.Name,
	}

	if category.Slug != nil && *category.Slug != "" {
		wpCategoryData["slug"] = *category.Slug
	}
	if category.Description != nil && *category.Description != "" {
		wpCategoryData["description"] = *category.Description
	}

	var createdCategory wpCategory

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetBody(wpCategoryData).
		SetResult(&createdCategory).
		Post(c.getAPIURL(s.URL, "categories"))
	if err != nil {
		return errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() != 201 {
		return errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	category.WPCategoryID = createdCategory.ID
	if createdCategory.Slug != "" {
		category.Slug = &createdCategory.Slug
	}
	if createdCategory.Description != "" {
		category.Description = &createdCategory.Description
	}
	category.Count = createdCategory.Count

	return nil
}

func (c *restyClient) UpdateCategory(ctx context.Context, s *entities.Site, category *entities.Category) error {
	wpCategoryData := map[string]interface{}{}

	if category.Name != "" {
		wpCategoryData["name"] = category.Name
	}
	if category.Slug != nil {
		wpCategoryData["slug"] = *category.Slug
	}
	if category.Description != nil {
		wpCategoryData["description"] = *category.Description
	}

	var updatedCategory wpCategory

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetBody(wpCategoryData).
		SetResult(&updatedCategory).
		Post(c.getAPIURL(s.URL, fmt.Sprintf("categories/%d", category.WPCategoryID)))
	if err != nil {
		return errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() != 200 {
		return errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	if updatedCategory.Slug != "" {
		category.Slug = &updatedCategory.Slug
	}
	if updatedCategory.Description != "" {
		category.Description = &updatedCategory.Description
	}
	category.Count = updatedCategory.Count

	return nil
}

func (c *restyClient) DeleteCategory(ctx context.Context, s *entities.Site, wpCategoryID int) error {
	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetQueryParam("force", "true").
		Delete(c.getAPIURL(s.URL, fmt.Sprintf("categories/%d", wpCategoryID)))
	if err != nil {
		return errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
		return errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	return nil
}
