package importer

import (
	"sort"
	"strings"
	"unicode"
)

// normalizePath ensures consistent path format
// - Adds leading slash if missing
// - Removes trailing slash
// - Converts to lowercase
// - Trims whitespace
func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	// Lowercase
	path = strings.ToLower(path)

	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Add leading slash if missing
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Handle root case
	if path == "/" {
		return "/"
	}

	return path
}

// extractSlug returns the last segment of the path
// "/services/web" -> "web"
// "/about" -> "about"
func extractSlug(path string) string {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// extractParentPath returns the parent's path
// "/services/web" -> "/services"
// "/about" -> ""
// "/" -> ""
func extractParentPath(path string) string {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return ""
	}

	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return "" // Direct child of root
	}

	return "/" + strings.Join(parts[:len(parts)-1], "/")
}

// countDepth returns the nesting level
// "/" -> 0
// "/about" -> 1
// "/services/web" -> 2
func countDepth(path string) int {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return 0
	}

	return len(strings.Split(path, "/"))
}

// mergeKeywords combines two keyword lists, removing duplicates
func mergeKeywords(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(a)+len(b))

	for _, kw := range a {
		kw = strings.TrimSpace(kw)
		if kw != "" && !seen[kw] {
			seen[kw] = true
			result = append(result, kw)
		}
	}

	for _, kw := range b {
		kw = strings.TrimSpace(kw)
		if kw != "" && !seen[kw] {
			seen[kw] = true
			result = append(result, kw)
		}
	}

	return result
}

// generateTitleFromSlug creates a readable title from slug
// "web-development" -> "Web Development"
func generateTitleFromSlug(slug string) string {
	if slug == "" {
		return ""
	}

	// Replace hyphens and underscores with spaces
	title := strings.ReplaceAll(slug, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Title case
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}

	return strings.Join(words, " ")
}

// sortNodes sorts by depth first, then by path
func sortNodes(nodes []*ParsedNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Depth != nodes[j].Depth {
			return nodes[i].Depth < nodes[j].Depth
		}
		return nodes[i].Path < nodes[j].Path
	})
}

// parseKeywordsString parses comma-separated keywords string
func parseKeywordsString(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		kw := strings.TrimSpace(part)
		if kw != "" {
			result = append(result, kw)
		}
	}

	return result
}

// findColumnIndex finds column index by possible names (case-insensitive)
func findColumnIndex(headers []string, possibleNames ...string) int {
	for i, header := range headers {
		headerLower := strings.ToLower(strings.TrimSpace(header))
		for _, name := range possibleNames {
			if headerLower == strings.ToLower(name) {
				return i
			}
		}
	}
	return -1
}
