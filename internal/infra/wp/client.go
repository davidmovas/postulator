package wp

import (
	"Postulator/internal/domain/entities"
	"Postulator/pkg/errors"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	requestTimeout = time.Second * 30
	userAgent      = "WordPress-Go-Client/1.0"
	apiPath        = "/wp-json/wp/v2/"
)

type Client struct {
	httpClient *http.Client
	userAgent  string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		userAgent: userAgent,
	}
}

func (c *Client) CheckHealth(ctx context.Context, site *entities.Site) error {
	endpoint := c.getAPIURL(site.URL, "")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return errors.WordPress("ошибка при создании запрос", err)
	}

	c.setAppPasswordAuth(req, site.WPUsername, site.WPPassword)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.WordPress("ошибка при запросе", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return errors.WordPress(fmt.Sprintf("WordPress API вернул код: %d", resp.StatusCode), nil)
	}

	return nil
}

func (c *Client) GetCategories(ctx context.Context, s *entities.Site) ([]*entities.Category, error) {
	endpoint := c.getAPIURL(s.URL, "categories")

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("per_page", "100")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAppPasswordAuth(req, s.WPUsername, s.WPPassword)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("wordpress API returned status %d: %s", resp.StatusCode, string(body))
	}

	var wpCategories []wpCategory
	if err = json.NewDecoder(resp.Body).Decode(&wpCategories); err != nil {
		return nil, fmt.Errorf("failed to decode categories: %w", err)
	}

	categories := make([]*entities.Category, 0, len(wpCategories))
	for _, wpCat := range wpCategories {
		slug := wpCat.Slug
		categories = append(categories, &entities.Category{
			WPCategoryID: wpCat.ID,
			Name:         wpCat.Name,
			Slug:         &slug,
			Count:        wpCat.Count,
		})
	}

	return categories, nil
}

func (c *Client) PublishPost(ctx context.Context, site *entities.Site, title, content string, categoryID int) (postID int, postURL string, err error) {
	endpoint := c.getAPIURL(site.URL, "posts")

	postData := map[string]any{
		"title":      title,
		"content":    content,
		"status":     "publish",
		"categories": []int{categoryID},
	}

	jsonData, err := json.Marshal(postData)
	if err != nil {
		return 0, "", fmt.Errorf("failed to marshal post data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	c.setAppPasswordAuth(req, site.WPUsername, site.WPPassword)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", fmt.Errorf("wordpress API returned status %d: %s", resp.StatusCode, string(body))
	}

	var createdPost wpPost
	if err = json.NewDecoder(resp.Body).Decode(&createdPost); err != nil {
		return 0, "", fmt.Errorf("failed to decode post response: %w", err)
	}

	return createdPost.ID, createdPost.Link, nil
}

func (c *Client) setAppPasswordAuth(req *http.Request, username, appPassword string) {
	auth := username + ":" + appPassword
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", basicAuth)
	req.Header.Set("User-Agent", c.userAgent)
}

func (c *Client) getAPIURL(siteURL, endpoint string) string {
	return strings.TrimSuffix(siteURL, "/") + path.Join(apiPath, endpoint)
}

func (c *Client) CreateApplicationPassword(ctx context.Context, site *entities.Site, appName string) (string, error) {
	endpoint := c.getAPIURL(site.URL, "users/me/application-passwords")

	passwordData := map[string]string{
		"name": appName,
	}

	jsonData, err := json.Marshal(passwordData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal password data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.setAppPasswordAuth(req, site.WPUsername, site.WPPassword)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create application password: status %d", resp.StatusCode)
	}

	var result struct {
		Password string `json:"password"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Password, nil
}
