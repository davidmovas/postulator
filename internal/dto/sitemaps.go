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
	ID                int64          `json:"id"`
	SitemapID         int64          `json:"sitemapId"`
	ParentID          *int64         `json:"parentId,omitempty"`
	Title             string         `json:"title"`
	Slug              string         `json:"slug"`
	Description       *string        `json:"description,omitempty"`
	IsRoot            bool           `json:"isRoot"`
	Depth             int            `json:"depth"`
	Position          int            `json:"position"`
	Path              string         `json:"path"`
	ContentType       string         `json:"contentType"`
	ArticleID         *int64         `json:"articleId,omitempty"`
	WPPageID          *int           `json:"wpPageId,omitempty"`
	WPURL             *string        `json:"wpUrl,omitempty"`
	Source            string         `json:"source"`
	IsSynced          bool           `json:"isSynced"`
	LastSyncedAt      *string        `json:"lastSyncedAt,omitempty"`
	WPTitle           *string        `json:"wpTitle,omitempty"`
	WPSlug            *string        `json:"wpSlug,omitempty"`
	IsModified        bool           `json:"isModified"`
	DesignStatus      string         `json:"designStatus"`
	GenerationStatus  string         `json:"generationStatus"`
	PublishStatus     string         `json:"publishStatus"`
	IsModifiedLocally bool           `json:"isModifiedLocally"`
	LastError         *string        `json:"lastError,omitempty"`
	PositionX         *float64       `json:"positionX,omitempty"`
	PositionY         *float64       `json:"positionY,omitempty"`
	Keywords          []string       `json:"keywords,omitempty"`
	Children          []*SitemapNode `json:"children,omitempty"`
	CreatedAt         string         `json:"createdAt"`
	UpdatedAt         string         `json:"updatedAt"`
}

func NewSitemapNode(entity *entities.SitemapNode) *SitemapNode {
	n := &SitemapNode{}
	return n.FromEntity(entity)
}

func (d *SitemapNode) ToEntity() *entities.SitemapNode {
	createdAt, _ := StringToTime(d.CreatedAt)
	updatedAt, _ := StringToTime(d.UpdatedAt)

	node := &entities.SitemapNode{
		ID:                d.ID,
		SitemapID:         d.SitemapID,
		ParentID:          d.ParentID,
		Title:             d.Title,
		Slug:              d.Slug,
		Description:       d.Description,
		Depth:             d.Depth,
		Position:          d.Position,
		Path:              d.Path,
		ContentType:       entities.NodeContentType(d.ContentType),
		ArticleID:         d.ArticleID,
		WPPageID:          d.WPPageID,
		WPURL:             d.WPURL,
		Source:            entities.NodeSource(d.Source),
		IsSynced:          d.IsSynced,
		DesignStatus:      entities.NodeDesignStatus(d.DesignStatus),
		GenerationStatus:  entities.NodeGenerationStatus(d.GenerationStatus),
		PublishStatus:     entities.NodePublishStatus(d.PublishStatus),
		IsModifiedLocally: d.IsModifiedLocally,
		LastError:         d.LastError,
		PositionX:         d.PositionX,
		PositionY:         d.PositionY,
		Keywords:          d.Keywords,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
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
	d.WPTitle = entity.WPTitle
	d.WPSlug = entity.WPSlug
	d.IsModified = entity.IsModified()
	d.DesignStatus = string(entity.DesignStatus)
	d.GenerationStatus = string(entity.GenerationStatus)
	d.PublishStatus = string(entity.PublishStatus)
	d.IsModifiedLocally = entity.IsModifiedLocally
	d.LastError = entity.LastError
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

// =========================================================================
// Import DTOs
// =========================================================================

type ImportNodesRequest struct {
	SitemapID      int64  `json:"sitemapId"`
	ParentNodeID   *int64 `json:"parentNodeId,omitempty"` // If set, import as children of this node
	Filename       string `json:"filename"`
	FileDataBase64 string `json:"fileDataBase64"` // Base64 encoded file content
}

type ImportNodesResponse struct {
	TotalRows      int           `json:"totalRows"`
	NodesCreated   int           `json:"nodesCreated"`
	NodesSkipped   int           `json:"nodesSkipped"`
	Errors         []ImportError `json:"errors,omitempty"`
	ProcessingTime string        `json:"processingTime"`
}

type ImportError struct {
	Row     int    `json:"row,omitempty"`
	Column  string `json:"column,omitempty"`
	Message string `json:"message"`
}

type SupportedFormatsResponse struct {
	Formats []string `json:"formats"`
}

// =========================================================================
// Scanner DTOs
// =========================================================================

// ScanSiteRequest contains parameters for scanning a WordPress site
type ScanSiteRequest struct {
	SiteID        int64  `json:"siteId"`
	SitemapName   string `json:"sitemapName"`
	TitleSource   string `json:"titleSource"`   // "title" or "h1"
	ContentFilter string `json:"contentFilter"` // "all", "pages", or "posts"
	IncludeDrafts bool   `json:"includeDrafts"`
	MaxDepth      int    `json:"maxDepth"` // 0 = unlimited
}

// ScanSiteResponse contains the result of a site scan
type ScanSiteResponse struct {
	SitemapID     int64       `json:"sitemapId"`
	PagesScanned  int         `json:"pagesScanned"`
	PostsScanned  int         `json:"postsScanned"`
	NodesCreated  int         `json:"nodesCreated"`
	NodesSkipped  int         `json:"nodesSkipped"`
	TotalDuration string      `json:"totalDuration"`
	Errors        []ScanError `json:"errors,omitempty"`
}

// ScanError represents an error during scanning
type ScanError struct {
	WPID    int    `json:"wpId,omitempty"`
	Type    string `json:"type,omitempty"` // "page" or "post"
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}

// ScanIntoSitemapRequest contains parameters for scanning into an existing sitemap
type ScanIntoSitemapRequest struct {
	SitemapID     int64  `json:"sitemapId"`
	ParentNodeID  *int64 `json:"parentNodeId,omitempty"` // Optional: if nil, uses root node
	TitleSource   string `json:"titleSource"`            // "title" or "h1"
	ContentFilter string `json:"contentFilter"`          // "all", "pages", or "posts"
	IncludeDrafts bool   `json:"includeDrafts"`
	MaxDepth      int    `json:"maxDepth"` // 0 = unlimited
}

// =========================================================================
// Sync DTOs
// =========================================================================

// SyncNodesRequest contains parameters for syncing nodes from WordPress
type SyncNodesRequest struct {
	SiteID  int64   `json:"siteId"`
	NodeIDs []int64 `json:"nodeIds"`
}

// SyncNodesResponse contains the result of syncing nodes
type SyncNodesResponse struct {
	Results []SyncNodeResult `json:"results"`
}

// SyncNodeResult contains the result of syncing a single node
type SyncNodeResult struct {
	NodeID  int64  `json:"nodeId"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// UpdateNodesToWPRequest contains parameters for updating WordPress from local nodes
type UpdateNodesToWPRequest struct {
	SiteID  int64   `json:"siteId"`
	NodeIDs []int64 `json:"nodeIds"`
}

// ChangePublishStatusRequest contains parameters for changing publish status
type ChangePublishStatusRequest struct {
	SiteID    int64  `json:"siteId"`
	NodeID    int64  `json:"nodeId"`
	NewStatus string `json:"newStatus"` // "published", "draft", "pending"
}

// =========================================================================
// AI Generation DTOs
// =========================================================================

// TitleInput represents a single title with optional keywords for AI generation
type TitleInput struct {
	Title    string   `json:"title"`
	Keywords []string `json:"keywords,omitempty"`
}

// GenerateSitemapStructureRequest contains parameters for AI sitemap generation
type GenerateSitemapStructureRequest struct {
	// SitemapID is set when adding to existing sitemap, nil for new sitemap
	SitemapID *int64 `json:"sitemapId,omitempty"`
	// SiteID is required when creating new sitemap
	SiteID *int64 `json:"siteId,omitempty"`
	// Name for new sitemap (required when SitemapID is nil)
	Name string `json:"name,omitempty"`
	// PromptID is the ID of the prompt template to use
	PromptID int64 `json:"promptId"`
	// Placeholders for the prompt template
	Placeholders map[string]string `json:"placeholders,omitempty"`
	// Titles with optional keywords to generate structure for
	Titles []TitleInput `json:"titles"`
	// ParentNodeIDs - nodes to use as roots for new nodes (empty = use root node)
	ParentNodeIDs []int64 `json:"parentNodeIds,omitempty"`
	// MaxDepth limits the depth of generated structure (0 = no limit)
	MaxDepth int `json:"maxDepth,omitempty"`
	// IncludeExistingTree sends the current tree structure to AI for context
	IncludeExistingTree bool `json:"includeExistingTree,omitempty"`
	// ProviderID is the AI provider to use
	ProviderID int64 `json:"providerId"`
}

// GeneratedNode represents a single node in AI-generated structure
type GeneratedNode struct {
	Title    string          `json:"title"`
	Slug     string          `json:"slug"`
	Keywords []string        `json:"keywords,omitempty"`
	Children []GeneratedNode `json:"children,omitempty"`
}

// GenerateSitemapStructureResponse contains the AI-generated sitemap structure
type GenerateSitemapStructureResponse struct {
	SitemapID    int64 `json:"sitemapId"`
	NodesCreated int   `json:"nodesCreated"`
	DurationMs   int64 `json:"durationMs"`
}

// ExistingNodeInfo represents simplified node info sent to AI for context
type ExistingNodeInfo struct {
	Title    string             `json:"title"`
	Slug     string             `json:"slug"`
	Path     string             `json:"path"`
	Keywords []string           `json:"keywords,omitempty"`
	Children []ExistingNodeInfo `json:"children,omitempty"`
}

// =========================================================================
// History DTOs
// =========================================================================

// HistoryState represents the current state of the history stack
type HistoryState struct {
	CanUndo       bool   `json:"canUndo"`
	CanRedo       bool   `json:"canRedo"`
	UndoCount     int    `json:"undoCount"`
	RedoCount     int    `json:"redoCount"`
	LastAction    string `json:"lastAction,omitempty"`
	ActionApplied string `json:"actionApplied,omitempty"` // Description of the action that was just applied (undo/redo)
}

// =========================================================================
// Page Generation DTOs
// =========================================================================

type ContentSettingsDTO struct {
	WordCount               string `json:"wordCount"`                         // e.g. "1000" or "800-1200"
	WritingStyle            string `json:"writingStyle"`                      // professional, casual, formal, friendly, technical
	ContentTone             string `json:"contentTone"`                       // informative, persuasive, educational, engaging, authoritative
	CustomInstructions      string `json:"customInstructions"`                // Additional instructions
	UseWebSearch            bool   `json:"useWebSearch"`                      // Enable web search for AI generation
	IncludeLinks            bool   `json:"includeLinks"`                      // Include approved links from linking plan
	AutoLinkMode            string `json:"autoLinkMode,omitempty"`            // "none", "before", or "after"
	AutoLinkProviderID      *int64 `json:"autoLinkProviderId,omitempty"`      // Provider for link suggestion
	AutoLinkSuggestPromptID *int64 `json:"autoLinkSuggestPromptId,omitempty"` // Prompt for link suggestion (link_suggest)
	AutoLinkApplyPromptID   *int64 `json:"autoLinkApplyPromptId,omitempty"`   // Prompt for link insertion (link_apply)
	MaxIncomingLinks        int    `json:"maxIncomingLinks,omitempty"`        // Max incoming links per page
	MaxOutgoingLinks        int    `json:"maxOutgoingLinks,omitempty"`        // Max outgoing links per page
}

// LinkTargetDTO represents a target page for internal linking during generation
type LinkTargetDTO struct {
	TargetNodeID int64   `json:"targetNodeId"`
	TargetTitle  string  `json:"targetTitle"`
	TargetPath   string  `json:"targetPath"`
	AnchorText   *string `json:"anchorText,omitempty"` // Suggested anchor text (optional)
}

type StartPageGenerationRequest struct {
	SitemapID       int64               `json:"sitemapId"`
	NodeIDs         []int64             `json:"nodeIds,omitempty"`
	ProviderID      int64               `json:"providerId"`
	PromptID        *int64              `json:"promptId,omitempty"`
	PublishAs       string              `json:"publishAs"`
	Placeholders    map[string]string   `json:"placeholders,omitempty"`
	MaxConcurrency  int                 `json:"maxConcurrency,omitempty"`
	ContentSettings *ContentSettingsDTO `json:"contentSettings,omitempty"`
}

type GenerationTaskResponse struct {
	ID             string               `json:"id"`
	SitemapID      int64                `json:"sitemapId"`
	SiteID         int64                `json:"siteId"`
	TotalNodes     int                  `json:"totalNodes"`
	ProcessedNodes int                  `json:"processedNodes"`
	FailedNodes    int                  `json:"failedNodes"`
	SkippedNodes   int                  `json:"skippedNodes"`
	Status         string               `json:"status"`
	StartedAt      string               `json:"startedAt"`
	CompletedAt    *string              `json:"completedAt,omitempty"`
	Error          *string              `json:"error,omitempty"`
	Nodes          []GenerationNodeInfo `json:"nodes,omitempty"`
	// Linking phase tracking
	LinkingPhase string `json:"linkingPhase,omitempty"` // "none", "suggesting", "applying", "completed"
	LinksCreated int    `json:"linksCreated,omitempty"` // Number of links suggested
	LinksApplied int    `json:"linksApplied,omitempty"` // Number of links applied
	LinksFailed  int    `json:"linksFailed,omitempty"`  // Number of links that failed to apply
}

type GenerationNodeInfo struct {
	NodeID      int64   `json:"nodeId"`
	Title       string  `json:"title"`
	Path        string  `json:"path"`
	Status      string  `json:"status"`
	ArticleID   *int64  `json:"articleId,omitempty"`
	WPPageID    *int    `json:"wpPageId,omitempty"`
	WPURL       *string `json:"wpUrl,omitempty"`
	Error       *string `json:"error,omitempty"`
	StartedAt   *string `json:"startedAt,omitempty"`
	CompletedAt *string `json:"completedAt,omitempty"`
}

type DefaultPromptResponse struct {
	Name         string   `json:"name"`
	SystemPrompt string   `json:"systemPrompt"`
	UserPrompt   string   `json:"userPrompt"`
	Placeholders []string `json:"placeholders"`
}
