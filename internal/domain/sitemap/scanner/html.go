package scanner

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// ExtractH1 extracts the first H1 tag content from HTML
func ExtractH1(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// Fallback to regex if HTML parsing fails
		return extractH1Regex(htmlContent)
	}

	var h1Text string
	var findH1 func(*html.Node)
	findH1 = func(n *html.Node) {
		if h1Text != "" {
			return // Already found
		}

		if n.Type == html.ElementNode && n.Data == "h1" {
			h1Text = extractTextContent(n)
			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findH1(c)
		}
	}

	findH1(doc)
	return strings.TrimSpace(h1Text)
}

// extractTextContent extracts text from an HTML node and its children
func extractTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var result strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result.WriteString(extractTextContent(c))
	}
	return result.String()
}

// extractH1Regex is a fallback regex-based H1 extraction
func extractH1Regex(htmlContent string) string {
	// Match <h1>...</h1> including with attributes
	re := regexp.MustCompile(`(?i)<h1[^>]*>(.*?)</h1>`)
	match := re.FindStringSubmatch(htmlContent)
	if len(match) > 1 {
		// Strip inner HTML tags
		tagRe := regexp.MustCompile(`<[^>]*>`)
		text := tagRe.ReplaceAllString(match[1], "")
		return strings.TrimSpace(html.UnescapeString(text))
	}
	return ""
}

// ExtractSlugFromURL extracts the slug from a WordPress URL
func ExtractSlugFromURL(wpURL, siteURL string) string {
	// Remove site URL prefix
	path := strings.TrimPrefix(wpURL, siteURL)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	// Get the last segment as slug
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}
