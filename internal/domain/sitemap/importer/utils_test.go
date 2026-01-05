package importer

import "testing"

func TestStripSitePrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "https with path",
			input:    "https://example.com/blog/article",
			expected: "/blog/article",
		},
		{
			name:     "http with path",
			input:    "http://site.com/page",
			expected: "/page",
		},
		{
			name:     "domain only with path",
			input:    "example.com/about",
			expected: "/about",
		},
		{
			name:     "domain only without path",
			input:    "example.com",
			expected: "/",
		},
		{
			name:     "already clean path",
			input:    "/about",
			expected: "/about",
		},
		{
			name:     "https with subdomain",
			input:    "https://blog.example.com/articles/test",
			expected: "/articles/test",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "root only",
			input:    "/",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripSitePrefix(tt.input)
			if result != tt.expected {
				t.Errorf("stripSitePrefix(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "https url with path",
			input:    "https://example.com/Blog/Article",
			expected: "/blog/article",
		},
		{
			name:     "url with trailing slash",
			input:    "https://example.com/about/",
			expected: "/about",
		},
		{
			name:     "path without leading slash",
			input:    "services/web",
			expected: "/services/web",
		},
		{
			name:     "clean path with uppercase",
			input:    "/About/Contact",
			expected: "/about/contact",
		},
		{
			name:     "path with whitespace",
			input:    "  /blog/post  ",
			expected: "/blog/post",
		},
		{
			name:     "domain only",
			input:    "example.com",
			expected: "/",
		},
		{
			name:     "complex url",
			input:    "HTTPS://My-Site.COM/Blog/2024/Article-Name/",
			expected: "/blog/2024/article-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
