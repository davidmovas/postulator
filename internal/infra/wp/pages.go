package wp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

func (c *restyClient) GetPage(ctx context.Context, s *entities.Site, pageID int) (*WPPage, error) {
	var page wpPage

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetResult(&page).
		Get(c.getAPIURL(s.URL, fmt.Sprintf("pages/%d", pageID)))
	if err != nil {
		return nil, errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() == 404 {
		return nil, errors.NotFound("page", pageID)
	}

	if resp.StatusCode() != 200 {
		return nil, errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	return c.convertWPPageToPage(page), nil
}

func (c *restyClient) GetPages(ctx context.Context, s *entities.Site, opts *PageListOptions) ([]*WPPage, error) {
	var wpPages []wpPage

	params := make(map[string]string)

	// Default values
	perPage := 25
	page := 1

	if opts != nil {
		if opts.PerPage > 0 && opts.PerPage <= 100 {
			perPage = opts.PerPage
		}
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.Status != "" {
			params["status"] = opts.Status
		}
		if opts.Parent != nil {
			params["parent"] = strconv.Itoa(*opts.Parent)
		}
		if opts.OrderBy != "" {
			params["orderby"] = opts.OrderBy
		}
		if opts.Order != "" {
			params["order"] = opts.Order
		}
		if opts.Search != "" {
			params["search"] = opts.Search
		}
		if len(opts.Exclude) > 0 {
			params["exclude"] = intsToString(opts.Exclude)
		}
		if len(opts.Include) > 0 {
			params["include"] = intsToString(opts.Include)
		}
	}

	params["per_page"] = strconv.Itoa(perPage)
	params["page"] = strconv.Itoa(page)

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetQueryParams(params).
		SetResult(&wpPages).
		Get(c.getAPIURL(s.URL, "pages"))
	if err != nil {
		return nil, errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() == 400 {
		respBody := resp.String()
		if strings.Contains(strings.ToLower(respBody), "page number") ||
		   strings.Contains(strings.ToLower(respBody), "larger than") {
			return []*WPPage{}, nil
		}
	}

	if resp.StatusCode() != 200 {
		return nil, errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	pages := make([]*WPPage, 0, len(wpPages))
	for _, p := range wpPages {
		pages = append(pages, c.convertWPPageToPage(p))
	}

	return pages, nil
}

// GetAllPages fetches all pages from WordPress using pagination
func (c *restyClient) GetAllPages(ctx context.Context, s *entities.Site) ([]*WPPage, error) {
	var allPages []*WPPage
	page := 1
	perPage := 25

	for {
		opts := &PageListOptions{
			Page:    page,
			PerPage: perPage,
			Status:  "any", // Get all statuses
			OrderBy: "id",
			Order:   "asc",
		}

		pages, err := c.GetPages(ctx, s, opts)
		if err != nil {
			return nil, err
		}

		if len(pages) == 0 {
			break
		}

		allPages = append(allPages, pages...)

		// If we got fewer than perPage, we've reached the end
		if len(pages) < perPage {
			break
		}

		page++
	}

	return allPages, nil
}

func (c *restyClient) CreatePage(ctx context.Context, s *entities.Site, page *WPPage, opts *PageCreateOptions) (int, error) {
	status := "publish"
	if opts != nil && opts.Status != "" {
		status = opts.Status
	}

	pageData := map[string]interface{}{
		"title":   page.Title,
		"content": page.Content,
		"status":  status,
	}

	if page.Slug != "" {
		pageData["slug"] = page.Slug
	}

	if page.ParentID > 0 {
		pageData["parent"] = page.ParentID
	}

	if page.Excerpt != "" {
		pageData["excerpt"] = page.Excerpt
	}

	if page.Author > 0 {
		pageData["author"] = page.Author
	}

	if page.FeaturedMedia > 0 {
		pageData["featured_media"] = page.FeaturedMedia
	}

	if page.MenuOrder > 0 {
		pageData["menu_order"] = page.MenuOrder
	}

	if page.Template != "" {
		pageData["template"] = page.Template
	}

	var createdPage wpPage

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetBody(pageData).
		SetResult(&createdPage).
		Post(c.getAPIURL(s.URL, "pages"))
	if err != nil {
		return 0, errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() != 201 {
		return 0, errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	// Update the page with response data
	page.ID = createdPage.ID
	page.Link = createdPage.Link
	page.Slug = createdPage.Slug

	return createdPage.ID, nil
}

func (c *restyClient) UpdatePage(ctx context.Context, s *entities.Site, page *WPPage) error {
	if page.ID == 0 {
		return errors.Validation("page ID is required for update")
	}

	pageData := map[string]interface{}{}

	if page.Title != "" {
		pageData["title"] = page.Title
	}

	if page.Content != "" {
		pageData["content"] = page.Content
	}

	if page.Slug != "" {
		pageData["slug"] = page.Slug
	}

	if page.Status != "" {
		pageData["status"] = page.Status
	}

	// Only set parent if explicitly provided (non-zero value)
	// ParentID = 0 means "don't change parent" (not "remove parent")
	// To explicitly move to top-level, set ParentID = -1 which we'll convert to 0
	if page.ParentID > 0 {
		pageData["parent"] = page.ParentID
	} else if page.ParentID == -1 {
		// Special case: explicitly move to top-level
		pageData["parent"] = 0
	}

	if page.Excerpt != "" {
		pageData["excerpt"] = page.Excerpt
	}

	if page.Author > 0 {
		pageData["author"] = page.Author
	}

	pageData["featured_media"] = page.FeaturedMedia
	pageData["menu_order"] = page.MenuOrder

	if page.Template != "" {
		pageData["template"] = page.Template
	}

	var updatedPage wpPage

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetBody(pageData).
		SetResult(&updatedPage).
		Post(c.getAPIURL(s.URL, fmt.Sprintf("pages/%d", page.ID)))
	if err != nil {
		return errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() == 404 {
		return errors.NotFound("page", page.ID)
	}

	if resp.StatusCode() != 200 {
		return errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	// Update the page with response data
	page.Link = updatedPage.Link
	page.Slug = updatedPage.Slug

	return nil
}

func (c *restyClient) DeletePage(ctx context.Context, s *entities.Site, pageID int, force bool) error {
	req := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword)

	if force {
		req.SetQueryParam("force", "true")
	}

	resp, err := req.Delete(c.getAPIURL(s.URL, fmt.Sprintf("pages/%d", pageID)))
	if err != nil {
		return errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() == 404 {
		return errors.NotFound("page", pageID)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
		return errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	return nil
}

func (c *restyClient) convertWPPageToPage(p wpPage) *WPPage {
	return &WPPage{
		ID:            p.ID,
		ParentID:      p.Parent,
		Title:         p.Title.Rendered,
		Slug:          p.Slug,
		Status:        p.Status,
		Link:          p.Link,
		Content:       p.Content.Rendered,
		Excerpt:       p.Excerpt.Rendered,
		Author:        p.Author,
		FeaturedMedia: p.FeaturedMedia,
		MenuOrder:     p.MenuOrder,
		Template:      p.Template,
		Date:          p.Date.Time,
		Modified:      p.Modified.Time,
	}
}

// intsToString converts a slice of ints to a comma-separated string
func intsToString(ids []int) string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = strconv.Itoa(id)
	}
	return strings.Join(strs, ",")
}
