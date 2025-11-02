package wp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

func (c *restyClient) PublishPost(ctx context.Context, site *entities.Site, title, content, status string, categoryID int) (postID int, postURL string, err error) {
	endpoint := c.getAPIURL(site.URL, "posts")

	postData := map[string]any{
		"title":      title,
		"content":    content,
		"status":     status,
		"categories": []int{categoryID},
	}

	jsonData, err := json.Marshal(postData)
	if err != nil {
		return 0, "", errors.WordPress("failed to marshal post data", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return 0, "", errors.WordPress("failed to create request", err)
	}

	c.setAppPasswordAuth(req, site.WPUsername, site.WPPassword)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", errors.WordPress("failed to make request", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode, string(body)), nil)
	}

	var createdPost wpPost
	if err = json.NewDecoder(resp.Body).Decode(&createdPost); err != nil {
		return 0, "", errors.WordPress("failed to decode post response", err)
	}

	return createdPost.ID, createdPost.Link, nil
}
