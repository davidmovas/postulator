package wordpress

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"Postulator/internal/models"
)

// Service handles WordPress REST API interactions
type Service struct {
	httpClient *http.Client
	timeout    time.Duration
}

// Config holds WordPress service configuration
type Config struct {
	Timeout time.Duration
}

// NewService creates a new WordPress service instance
func NewService(config Config) *Service {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Service{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		timeout: config.Timeout,
	}
}

// PostRequest represents a WordPress post creation request
type PostRequest struct {
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Excerpt    string   `json:"excerpt"`
	Status     string   `json:"status"`
	Categories []int64  `json:"categories,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Featured   bool     `json:"featured,omitempty"`
}

// PostResponse represents a WordPress post response
type PostResponse struct {
	ID          int64       `json:"id"`
	Title       PostTitle   `json:"title"`
	Content     PostContent `json:"content"`
	Excerpt     PostExcerpt `json:"excerpt"`
	Status      string      `json:"status"`
	Link        string      `json:"link"`
	Date        time.Time   `json:"date"`
	DateGMT     time.Time   `json:"date_gmt"`
	Modified    time.Time   `json:"modified"`
	ModifiedGMT time.Time   `json:"modified_gmt"`
	Categories  []int64     `json:"categories"`
	Tags        []int64     `json:"tags"`
}

// PostTitle represents WordPress post title
type PostTitle struct {
	Rendered string `json:"rendered"`
}

// PostContent represents WordPress post content
type PostContent struct {
	Rendered string `json:"rendered"`
}

// PostExcerpt represents WordPress post excerpt
type PostExcerpt struct {
	Rendered string `json:"rendered"`
}

// CategoryResponse represents a WordPress category
type CategoryResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// TagResponse represents a WordPress tag
type TagResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CreatePostRequest contains parameters for creating a post
type CreatePostRequest struct {
	Site    *models.Site    `json:"site"`
	Article *models.Article `json:"article"`
	Publish bool            `json:"publish,omitempty"`
}

// CreatePostResponse contains the result of post creation
type CreatePostResponse struct {
	WordPressID int64     `json:"wordpress_id"`
	URL         string    `json:"url"`
	Status      string    `json:"status"`
	PublishedAt time.Time `json:"published_at"`
}

// CreatePost creates a new post on WordPress site
func (s *Service) CreatePost(ctx context.Context, req CreatePostRequest) (*CreatePostResponse, error) {
	if req.Site == nil {
		return nil, fmt.Errorf("site is required")
	}
	if req.Article == nil {
		return nil, fmt.Errorf("article is required")
	}

	// Get or create categories and tags
	categoryIDs, err := s.getOrCreateCategories(ctx, req.Site, []string{req.Article.Category})
	if err != nil {
		return nil, fmt.Errorf("failed to handle categories: %w", err)
	}

	tagNames := strings.Split(req.Article.Tags, ",")
	for i := range tagNames {
		tagNames[i] = strings.TrimSpace(tagNames[i])
	}

	// Prepare post data
	postData := PostRequest{
		Title:      req.Article.Title,
		Content:    req.Article.Content,
		Excerpt:    req.Article.Excerpt,
		Status:     "draft", // Default to draft
		Categories: categoryIDs,
		Tags:       tagNames,
	}

	if req.Publish {
		postData.Status = "publish"
	}

	// Make API call
	response, err := s.makePostRequest(ctx, req.Site, postData)
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	return &CreatePostResponse{
		WordPressID: response.ID,
		URL:         response.Link,
		Status:      response.Status,
		PublishedAt: response.Date,
	}, nil
}

// makePostRequest performs the HTTP request to create a post
func (s *Service) makePostRequest(ctx context.Context, site *models.Site, postData PostRequest) (*PostResponse, error) {
	// Build API URL
	apiURL := strings.TrimRight(site.URL, "/") + "/wp-json/wp/v2/posts"

	// Marshal request data
	jsonData, err := json.Marshal(postData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal post data: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Set authentication
	if site.APIKey != "" {
		// JWT or API key authentication
		req.Header.Set("Authorization", "Bearer "+site.APIKey)
	} else {
		// Basic authentication
		auth := site.Username + ":" + site.Password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", "Basic "+encodedAuth)
	}

	// Make the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var postResponse PostResponse
	if err := json.Unmarshal(body, &postResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &postResponse, nil
}

// getOrCreateCategories ensures categories exist and returns their IDs
func (s *Service) getOrCreateCategories(ctx context.Context, site *models.Site, categoryNames []string) ([]int64, error) {
	var categoryIDs []int64

	for _, name := range categoryNames {
		if name == "" {
			continue
		}

		// First, try to find existing category
		categoryID, err := s.findCategoryByName(ctx, site, name)
		if err != nil {
			return nil, fmt.Errorf("failed to find category %s: %w", name, err)
		}

		if categoryID > 0 {
			categoryIDs = append(categoryIDs, categoryID)
			continue
		}

		// Create new category if not found
		categoryID, err = s.createCategory(ctx, site, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create category %s: %w", name, err)
		}

		categoryIDs = append(categoryIDs, categoryID)
	}

	return categoryIDs, nil
}

// findCategoryByName searches for a category by name
func (s *Service) findCategoryByName(ctx context.Context, site *models.Site, name string) (int64, error) {
	// Build API URL with search parameter
	apiURL := fmt.Sprintf("%s/wp-json/wp/v2/categories?search=%s",
		strings.TrimRight(site.URL, "/"), url.QueryEscape(name))

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication
	s.setAuth(req, site)

	// Make request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, nil // Category not found
	}

	// Parse response
	var categories []CategoryResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &categories); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Find exact match
	for _, category := range categories {
		if strings.EqualFold(category.Name, name) {
			return category.ID, nil
		}
	}

	return 0, nil // Not found
}

// createCategory creates a new category
func (s *Service) createCategory(ctx context.Context, site *models.Site, name string) (int64, error) {
	// Build API URL
	apiURL := strings.TrimRight(site.URL, "/") + "/wp-json/wp/v2/categories"

	// Prepare category data
	categoryData := map[string]interface{}{
		"name": name,
		"slug": strings.ToLower(strings.ReplaceAll(name, " ", "-")),
	}

	// Marshal data
	jsonData, err := json.Marshal(categoryData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal category data: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	s.setAuth(req, site)

	// Make request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("failed to create category, status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var category CategoryResponse
	if err := json.Unmarshal(body, &category); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return category.ID, nil
}

// setAuth sets authentication headers
func (s *Service) setAuth(req *http.Request, site *models.Site) {
	if site.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+site.APIKey)
	} else {
		auth := site.Username + ":" + site.Password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", "Basic "+encodedAuth)
	}
}

// TestConnection tests the connection to WordPress site
func (s *Service) TestConnection(ctx context.Context, site *models.Site) error {
	// Try to get site info
	apiURL := strings.TrimRight(site.URL, "/") + "/wp-json/wp/v2/users/me"

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	s.setAuth(req, site)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication failed")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetPostByID retrieves a post by its WordPress ID
func (s *Service) GetPostByID(ctx context.Context, site *models.Site, postID int64) (*PostResponse, error) {
	apiURL := fmt.Sprintf("%s/wp-json/wp/v2/posts/%d", strings.TrimRight(site.URL, "/"), postID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setAuth(req, site)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("post not found")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var post PostResponse
	if err := json.Unmarshal(body, &post); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &post, nil
}

// UpdatePostStatus updates the status of a WordPress post
func (s *Service) UpdatePostStatus(ctx context.Context, site *models.Site, postID int64, status string) error {
	apiURL := fmt.Sprintf("%s/wp-json/wp/v2/posts/%d", strings.TrimRight(site.URL, "/"), postID)

	updateData := map[string]interface{}{
		"status": status,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	s.setAuth(req, site)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
