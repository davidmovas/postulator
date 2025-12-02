package dto

import (
	"github.com/davidmovas/postulator/internal/domain/entities"
)

// =========================================================================
// Sitemap DTO
// =========================================================================

type Sitemap struct {
	ID          int64   `json:"id"`
	SiteID      int64   `json:"siteId"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Source      string  `json:"source"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

func NewSitemap(entity *entities.Sitemap) *Sitemap {
	s := &Sitemap{}
	return s.FromEntity(entity)
}

func (d *Sitemap) ToEntity() *entities.Sitemap {
	createdAt, _ := StringToTime(d.CreatedAt)
	updatedAt, _ := StringToTime(d.UpdatedAt)

	return &entities.Sitemap{
		ID:          d.ID,
		SiteID:      d.SiteID,
		Name:        d.Name,
		Description: d.Description,
		Source:      entities.SitemapSource(d.Source),
		Status:      entities.SitemapStatus(d.Status),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func (d *Sitemap) FromEntity(entity *entities.Sitemap) *Sitemap {
	d.ID = entity.ID
	d.SiteID = entity.SiteID
	d.Name = entity.Name
	d.Description = entity.Description
	d.Source = string(entity.Source)
	d.Status = string(entity.Status)
	d.CreatedAt = TimeToString(entity.CreatedAt)
	d.UpdatedAt = TimeToString(entity.UpdatedAt)
	return d
}

// =========================================================================
// SitemapNode DTO
// =========================================================================

type SitemapNode struct {
	ID            int64          `json:"id"`
	SitemapID     int64          `json:"sitemapId"`
	ParentID      *int64         `json:"parentId,omitempty"`
	Title         string         `json:"title"`
	Slug          string         `json:"slug"`
	Description   *string        `json:"description,omitempty"`
	IsRoot        bool           `json:"isRoot"`
	Depth         int            `json:"depth"`
	Position      int            `json:"position"`
	Path          string         `json:"path"`
	ContentType   string         `json:"contentType"`
	ArticleID     *int64         `json:"articleId,omitempty"`
	WPPageID      *int           `json:"wpPageId,omitempty"`
	WPURL         *string        `json:"wpUrl,omitempty"`
	Source        string         `json:"source"`
	IsSynced      bool           `json:"isSynced"`
	LastSyncedAt  *string        `json:"lastSyncedAt,omitempty"`
	ContentStatus string         `json:"contentStatus"`
	PositionX     *float64       `json:"positionX,omitempty"`
	PositionY     *float64       `json:"positionY,omitempty"`
	Keywords      []string       `json:"keywords,omitempty"`
	Children      []*SitemapNode `json:"children,omitempty"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
}

func NewSitemapNode(entity *entities.SitemapNode) *SitemapNode {
	n := &SitemapNode{}
	return n.FromEntity(entity)
}

func (d *SitemapNode) ToEntity() *entities.SitemapNode {
	createdAt, _ := StringToTime(d.CreatedAt)
	updatedAt, _ := StringToTime(d.UpdatedAt)

	node := &entities.SitemapNode{
		ID:            d.ID,
		SitemapID:     d.SitemapID,
		ParentID:      d.ParentID,
		Title:         d.Title,
		Slug:          d.Slug,
		Description:   d.Description,
		Depth:         d.Depth,
		Position:      d.Position,
		Path:          d.Path,
		ContentType:   entities.NodeContentType(d.ContentType),
		ArticleID:     d.ArticleID,
		WPPageID:      d.WPPageID,
		WPURL:         d.WPURL,
		Source:        entities.NodeSource(d.Source),
		IsSynced:      d.IsSynced,
		ContentStatus: entities.NodeContentStatus(d.ContentStatus),
		PositionX:     d.PositionX,
		PositionY:     d.PositionY,
		Keywords:      d.Keywords,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}

	if d.LastSyncedAt != nil {
		lastSyncedAt, _ := StringToTime(*d.LastSyncedAt)
		node.LastSyncedAt = &lastSyncedAt
	}

	return node
}

func (d *SitemapNode) FromEntity(entity *entities.SitemapNode) *SitemapNode {
	d.ID = entity.ID
	d.SitemapID = entity.SitemapID
	d.ParentID = entity.ParentID
	d.Title = entity.Title
	d.Slug = entity.Slug
	d.Description = entity.Description
	d.IsRoot = entity.IsRoot
	d.Depth = entity.Depth
	d.Position = entity.Position
	d.Path = entity.Path
	d.ContentType = string(entity.ContentType)
	d.ArticleID = entity.ArticleID
	d.WPPageID = entity.WPPageID
	d.WPURL = entity.WPURL
	d.Source = string(entity.Source)
	d.IsSynced = entity.IsSynced
	d.ContentStatus = string(entity.ContentStatus)
	d.PositionX = entity.PositionX
	d.PositionY = entity.PositionY
	d.Keywords = entity.Keywords
	d.CreatedAt = TimeToString(entity.CreatedAt)
	d.UpdatedAt = TimeToString(entity.UpdatedAt)

	if entity.LastSyncedAt != nil {
		lastSyncedAt := TimeToString(*entity.LastSyncedAt)
		d.LastSyncedAt = &lastSyncedAt
	}

	// Convert children recursively
	if len(entity.Children) > 0 {
		d.Children = make([]*SitemapNode, len(entity.Children))
		for i, child := range entity.Children {
			d.Children[i] = NewSitemapNode(child)
		}
	}

	return d
}

// =========================================================================
// Request/Response DTOs
// =========================================================================

type CreateSitemapRequest struct {
	SiteID      int64   `json:"siteId"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Source      string  `json:"source"`
	SiteURL     string  `json:"siteUrl"` // Used to create root node
}

type UpdateSitemapRequest struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Status      string  `json:"status"`
}

type DuplicateSitemapRequest struct {
	ID      int64  `json:"id"`
	NewName string `json:"newName"`
}

type CreateNodeRequest struct {
	SitemapID   int64    `json:"sitemapId"`
	ParentID    *int64   `json:"parentId,omitempty"`
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	Description *string  `json:"description,omitempty"`
	Position    int      `json:"position"`
	Source      string   `json:"source"`
	Keywords    []string `json:"keywords,omitempty"`
}

type UpdateNodeRequest struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	Description *string  `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
}

type MoveNodeRequest struct {
	NodeID      int64  `json:"nodeId"`
	NewParentID *int64 `json:"newParentId,omitempty"`
	Position    int    `json:"position"`
}

type UpdateNodePositionsRequest struct {
	NodeID    int64   `json:"nodeId"`
	PositionX float64 `json:"positionX"`
	PositionY float64 `json:"positionY"`
}

type LinkNodeToArticleRequest struct {
	NodeID    int64 `json:"nodeId"`
	ArticleID int64 `json:"articleId"`
}

type LinkNodeToPageRequest struct {
	NodeID   int64  `json:"nodeId"`
	WPPageID int    `json:"wpPageId"`
	WPURL    string `json:"wpUrl"`
}

type SetNodeKeywordsRequest struct {
	NodeID   int64    `json:"nodeId"`
	Keywords []string `json:"keywords"`
}

type DistributeKeywordsRequest struct {
	SitemapID int64    `json:"sitemapId"`
	Keywords  []string `json:"keywords"`
	Strategy  string   `json:"strategy"`
}

type SitemapWithNodes struct {
	Sitemap *Sitemap       `json:"sitemap"`
	Nodes   []*SitemapNode `json:"nodes"`
}
