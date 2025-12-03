package scanner

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

// TitleSource determines which field to use as the node title
type TitleSource string

const (
	TitleSourceTitle TitleSource = "title" // Use page/post title
	TitleSourceH1    TitleSource = "h1"    // Use first H1 from content
)

// ContentFilter determines which content types to scan
type ContentFilter string

const (
	ContentFilterAll   ContentFilter = "all"   // Scan both pages and posts
	ContentFilterPages ContentFilter = "pages" // Scan only pages
	ContentFilterPosts ContentFilter = "posts" // Scan only posts
)

// ScanOptions configures scanner behavior
type ScanOptions struct {
	// TitleSource determines what to use as node title (title or h1)
	TitleSource TitleSource

	// ContentFilter determines which content types to scan
	ContentFilter ContentFilter

	// IncludeDrafts includes draft pages/posts in scan
	IncludeDrafts bool

	// MaxDepth limits how deep to scan in the page hierarchy (0 = unlimited)
	MaxDepth int
}

// DefaultScanOptions returns default scan options
func DefaultScanOptions() *ScanOptions {
	return &ScanOptions{
		TitleSource:   TitleSourceTitle,
		ContentFilter: ContentFilterPages,
		IncludeDrafts: false,
		MaxDepth:      0,
	}
}

// ScannedPage represents a page/post discovered during scan
type ScannedPage struct {
	// WordPress data
	WPID     int
	WPType   entities.NodeContentType // "page" or "post"
	ParentID int                      // WordPress parent ID (0 if top-level)

	// Content
	Title   string // Page/post title
	H1      string // First H1 extracted from content
	Slug    string
	URL     string
	Status  string // publish, draft, etc.
	Content string // Raw HTML content

	// Hierarchy (calculated)
	Depth int
	Path  string

	// Children (for tree building)
	Children []*ScannedPage
}

// GetDisplayTitle returns title based on TitleSource preference
func (p *ScannedPage) GetDisplayTitle(source TitleSource) string {
	if source == TitleSourceH1 && p.H1 != "" {
		return p.H1
	}
	return p.Title
}

// ScanProgress represents current scan progress
type ScanProgress struct {
	Phase          string // "fetching_pages", "fetching_posts", "building_tree", "creating_nodes"
	CurrentItem    int
	TotalItems     int
	PagesFound     int
	PostsFound     int
	NodesCreated   int
	NodesSkipped   int
	Errors         []ScanError
	StartedAt      time.Time
	EstimatedTotal int
}

// ScanError represents an error during scanning
type ScanError struct {
	WPID    int
	Type    string // "page" or "post"
	Title   string
	Message string
}

// ScanResult contains the final scan results
type ScanResult struct {
	// Statistics
	PagesScanned  int
	PostsScanned  int
	NodesCreated  int
	NodesSkipped  int
	TotalDuration time.Duration

	// Errors encountered
	Errors []ScanError

	// The created sitemap
	SitemapID int64
}
