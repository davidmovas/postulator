package entities

import "time"

// =========================================================================
// Sitemap (Site Structure Tree)
// =========================================================================

type SitemapSource string

const (
	SitemapSourceManual    SitemapSource = "manual"
	SitemapSourceImported  SitemapSource = "imported"
	SitemapSourceGenerated SitemapSource = "generated"
	SitemapSourceScanned   SitemapSource = "scanned"
)

type SitemapStatus string

const (
	SitemapStatusDraft    SitemapStatus = "draft"
	SitemapStatusActive   SitemapStatus = "active"
	SitemapStatusArchived SitemapStatus = "archived"
)

type Sitemap struct {
	ID          int64
	SiteID      int64
	Name        string
	Description *string
	Source      SitemapSource
	Status      SitemapStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// =========================================================================
// SitemapNode (Tree Node)
// =========================================================================

type NodeSource string

const (
	NodeSourceManual    NodeSource = "manual"
	NodeSourceImported  NodeSource = "imported"
	NodeSourceGenerated NodeSource = "generated"
	NodeSourceScanned   NodeSource = "scanned"
)

type NodeContentType string

const (
	NodeContentTypePage NodeContentType = "page"
	NodeContentTypePost NodeContentType = "post"
	NodeContentTypeNone NodeContentType = "none"
)

type NodeContentStatus string

const (
	NodeContentStatusPending   NodeContentStatus = "pending"
	NodeContentStatusDraft     NodeContentStatus = "draft"
	NodeContentStatusPublished NodeContentStatus = "published"
)

type SitemapNode struct {
	ID        int64
	SitemapID int64
	ParentID  *int64

	// Basic fields
	Title       string
	Slug        string
	Description *string
	IsRoot      bool

	// Hierarchy
	Depth    int
	Position int
	Path     string

	// Content association
	ContentType NodeContentType
	ArticleID   *int64
	WPPageID    *int
	WPURL       *string

	// Source and sync
	Source       NodeSource
	IsSynced     bool
	LastSyncedAt *time.Time

	// Content status
	ContentStatus NodeContentStatus

	// React Flow positions
	PositionX *float64
	PositionY *float64

	CreatedAt time.Time
	UpdatedAt time.Time

	// Loaded relations (not stored in DB directly)
	Keywords []string
	Children []*SitemapNode
}

// =========================================================================
// SitemapNodeKeyword
// =========================================================================

type SitemapNodeKeyword struct {
	ID        int64
	NodeID    int64
	Keyword   string
	Position  int
	CreatedAt time.Time
}
