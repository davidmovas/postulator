package sitemap

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

// =========================================================================
// Sitemap Repository
// =========================================================================

type Repository interface {
	Create(ctx context.Context, sitemap *entities.Sitemap) error
	GetByID(ctx context.Context, id int64) (*entities.Sitemap, error)
	GetBySiteID(ctx context.Context, siteID int64) ([]*entities.Sitemap, error)
	GetAll(ctx context.Context) ([]*entities.Sitemap, error)
	Update(ctx context.Context, sitemap *entities.Sitemap) error
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status entities.SitemapStatus) error
	TouchUpdatedAt(ctx context.Context, id int64) error
}

// =========================================================================
// Node Repository
// =========================================================================

type NodeRepository interface {
	Create(ctx context.Context, node *entities.SitemapNode) error
	CreateBatch(ctx context.Context, nodes []*entities.SitemapNode) error
	GetByID(ctx context.Context, id int64) (*entities.SitemapNode, error)
	GetBySitemapID(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error)
	GetByParentID(ctx context.Context, sitemapID int64, parentID *int64) ([]*entities.SitemapNode, error)
	GetBySlugAndParent(ctx context.Context, sitemapID int64, slug string, parentID *int64) (*entities.SitemapNode, error)
	GetByWPID(ctx context.Context, sitemapID int64, wpID int, contentType entities.NodeContentType) (*entities.SitemapNode, error)
	GetRootNodes(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error)
	Update(ctx context.Context, node *entities.SitemapNode) error
	Delete(ctx context.Context, id int64) error
	DeleteBySitemapID(ctx context.Context, sitemapID int64) error

	UpdateParent(ctx context.Context, nodeID int64, newParentID *int64) error
	UpdatePosition(ctx context.Context, nodeID int64, position int) error
	UpdatePositions(ctx context.Context, nodeID int64, positionX, positionY float64) error
	GetDescendants(ctx context.Context, nodeID int64) ([]*entities.SitemapNode, error)
	GetAncestors(ctx context.Context, nodeID int64) ([]*entities.SitemapNode, error)

	UpdateContentStatus(ctx context.Context, nodeID int64, status entities.NodeContentStatus) error
	UpdateContentLink(ctx context.Context, nodeID int64, contentType entities.NodeContentType, articleID *int64, wpPageID *int, wpURL *string) error

	UpdateSyncStatus(ctx context.Context, nodeID int64, isSynced bool) error
}

// =========================================================================
// Keyword Repository
// =========================================================================

type KeywordRepository interface {
	Create(ctx context.Context, keyword *entities.SitemapNodeKeyword) error
	CreateBatch(ctx context.Context, nodeID int64, keywords []string) error
	GetByNodeID(ctx context.Context, nodeID int64) ([]*entities.SitemapNodeKeyword, error)
	GetKeywordsByNodeIDs(ctx context.Context, nodeIDs []int64) (map[int64][]string, error)
	Delete(ctx context.Context, id int64) error
	DeleteByNodeID(ctx context.Context, nodeID int64) error
	ReplaceKeywords(ctx context.Context, nodeID int64, keywords []string) error

	DistributeKeywords(ctx context.Context, sitemapID int64, keywords []string, strategy KeywordDistributionStrategy) error
}

type KeywordDistributionStrategy string

const (
	KeywordDistributionEven   KeywordDistributionStrategy = "even"   // Равномерно по всем узлам
	KeywordDistributionByPath KeywordDistributionStrategy = "bypath" // По релевантности пути
)

// =========================================================================
// Service
// =========================================================================

type Service interface {
	CreateSitemap(ctx context.Context, sitemap *entities.Sitemap) error
	CreateSitemapWithRoot(ctx context.Context, sitemap *entities.Sitemap, siteURL string) error
	GetSitemap(ctx context.Context, id int64) (*entities.Sitemap, error)
	GetSitemapWithNodes(ctx context.Context, id int64) (*entities.Sitemap, []*entities.SitemapNode, error)
	ListSitemaps(ctx context.Context, siteID int64) ([]*entities.Sitemap, error)
	UpdateSitemap(ctx context.Context, sitemap *entities.Sitemap) error
	DeleteSitemap(ctx context.Context, id int64) error
	DuplicateSitemap(ctx context.Context, id int64, newName string) (*entities.Sitemap, error)
	SetSitemapStatus(ctx context.Context, id int64, status entities.SitemapStatus) error

	CreateNode(ctx context.Context, node *entities.SitemapNode) error
	CreateNodes(ctx context.Context, nodes []*entities.SitemapNode) error
	GetNode(ctx context.Context, id int64) (*entities.SitemapNode, error)
	GetNodeWithKeywords(ctx context.Context, id int64) (*entities.SitemapNode, error)
	GetNodes(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error)
	GetNodesTree(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error)
	FindNodeBySlugAndParent(ctx context.Context, sitemapID int64, slug string, parentID *int64) (*entities.SitemapNode, error)
	FindNodeByWPID(ctx context.Context, sitemapID int64, wpID int, contentType entities.NodeContentType) (*entities.SitemapNode, error)
	UpdateNode(ctx context.Context, node *entities.SitemapNode) error
	DeleteNode(ctx context.Context, id int64) error
	MoveNode(ctx context.Context, nodeID int64, newParentID *int64, position int) error
	UpdateNodePositions(ctx context.Context, nodeID int64, positionX, positionY float64) error

	SetNodeKeywords(ctx context.Context, nodeID int64, keywords []string) error
	AddNodeKeyword(ctx context.Context, nodeID int64, keyword string) error
	RemoveNodeKeyword(ctx context.Context, nodeID int64, keyword string) error
	DistributeKeywords(ctx context.Context, sitemapID int64, keywords []string, strategy KeywordDistributionStrategy) error

	LinkNodeToArticle(ctx context.Context, nodeID int64, articleID int64) error
	LinkNodeToPage(ctx context.Context, nodeID int64, wpPageID int, wpURL string) error
	UnlinkNodeContent(ctx context.Context, nodeID int64) error
	UpdateNodeContentStatus(ctx context.Context, nodeID int64, status entities.NodeContentStatus) error

	// Import/Export (for future phases)
	// ImportFromJSON(ctx context.Context, siteID int64, data []byte) (*entities.Sitemap, error)
	// ExportToJSON(ctx context.Context, sitemapID int64) ([]byte, error)
}
