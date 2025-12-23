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

type NodeDesignStatus string

const (
	DesignStatusDraft    NodeDesignStatus = "draft"
	DesignStatusReady    NodeDesignStatus = "ready"
	DesignStatusApproved NodeDesignStatus = "approved"
)

type NodeGenerationStatus string

const (
	GenStatusNone       NodeGenerationStatus = "none"
	GenStatusQueued     NodeGenerationStatus = "queued"
	GenStatusGenerating NodeGenerationStatus = "generating"
	GenStatusGenerated  NodeGenerationStatus = "generated"
	GenStatusFailed     NodeGenerationStatus = "failed"
)

type NodePublishStatus string

const (
	PubStatusNone       NodePublishStatus = "none"
	PubStatusPublishing NodePublishStatus = "publishing"
	PubStatusDraft      NodePublishStatus = "draft"
	PubStatusPending    NodePublishStatus = "pending"
	PubStatusPublished  NodePublishStatus = "published"
	PubStatusFailed     NodePublishStatus = "failed"
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

	// Original WP data (for tracking local modifications)
	WPTitle *string
	WPSlug  *string

	// Status groups (3 separate concerns)
	DesignStatus     NodeDesignStatus
	GenerationStatus NodeGenerationStatus
	PublishStatus    NodePublishStatus

	// Local modifications tracking
	IsModifiedLocally bool
	LastError         *string

	// React Flow positions
	PositionX *float64
	PositionY *float64

	CreatedAt time.Time
	UpdatedAt time.Time

	// Loaded relations (not stored in DB directly)
	Keywords []string
	Children []*SitemapNode
}

// IsModified returns true if local data differs from WP data
func (n *SitemapNode) IsModified() bool {
	// Only scanned nodes can be modified
	if n.WPPageID == nil {
		return false
	}
	// Check if we have original data to compare
	if n.WPTitle == nil && n.WPSlug == nil {
		return false
	}
	// Compare title
	if n.WPTitle != nil && n.Title != *n.WPTitle {
		return true
	}
	// Compare slug
	if n.WPSlug != nil && n.Slug != *n.WPSlug {
		return true
	}
	return false
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
