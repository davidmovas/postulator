package importer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

// NodeCreator interface for creating nodes (implemented by sitemap.Service)
type NodeCreator interface {
	CreateNode(ctx context.Context, node *entities.SitemapNode) error
	GetNodes(ctx context.Context, sitemapID int64) ([]*entities.SitemapNode, error)
	FindNodeBySlugAndParent(ctx context.Context, sitemapID int64, slug string, parentID *int64) (*entities.SitemapNode, error)
}

// Importer handles file imports for sitemaps
type Importer struct {
	parsers map[string]Parser
	logger  *logger.Logger
}

// NewImporter creates a new importer with all supported parsers
func NewImporter(logger *logger.Logger) *Importer {
	imp := &Importer{
		parsers: make(map[string]Parser),
		logger:  logger.WithScope("importer"),
	}

	// Register parsers
	imp.registerParser(NewCSVParser())
	imp.registerParser(NewJSONParser())
	imp.registerParser(NewXLSXParser())

	return imp
}

func (i *Importer) registerParser(parser Parser) {
	for _, ext := range parser.SupportedExtensions() {
		i.parsers[strings.ToLower(ext)] = parser
	}
}

// SupportedFormats returns list of supported file extensions
func (i *Importer) SupportedFormats() []string {
	formats := make([]string, 0, len(i.parsers))
	seen := make(map[string]bool)
	for ext := range i.parsers {
		if !seen[ext] {
			formats = append(formats, ext)
			seen[ext] = true
		}
	}
	return formats
}

// ImportOptions configures import behavior
type ImportOptions struct {
	// ParentNodeID - if set, imported nodes will be children of this node
	// If nil, nodes are added as children of root
	ParentNodeID *int64
}

// ImportStats contains statistics about the import
type ImportStats struct {
	TotalRows     int
	NodesCreated  int
	NodesSkipped  int // Already existed (same path)
	Errors        []ImportError
	ProcessingTime time.Duration
}

// Import parses file and creates nodes in the sitemap
func (i *Importer) Import(
	ctx context.Context,
	filename string,
	data []byte,
	sitemapID int64,
	nodeCreator NodeCreator,
	opts *ImportOptions,
) (*ImportStats, error) {
	startTime := time.Now()

	stats := &ImportStats{
		Errors: make([]ImportError, 0),
	}

	// Get parser for file type
	ext := strings.ToLower(filepath.Ext(filename))
	parser, ok := i.parsers[ext]
	if !ok {
		return nil, fmt.Errorf("unsupported file format: %s (supported: %v)", ext, i.SupportedFormats())
	}

	// Parse file
	i.logger.Infof("Parsing file %s (%d bytes)", filename, len(data))
	result, err := parser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	stats.TotalRows = len(result.Rows)
	stats.Errors = append(stats.Errors, result.Errors...)

	if len(result.Rows) == 0 {
		return stats, nil
	}

	// Build tree from flat rows
	builder := NewTreeBuilder()
	parsedNodes := builder.BuildTree(result.Rows)

	i.logger.Infof("Built tree with %d nodes from %d rows", len(parsedNodes), len(result.Rows))

	// Get existing nodes to find root and avoid duplicates
	existingNodes, err := nodeCreator.GetNodes(ctx, sitemapID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing nodes: %w", err)
	}

	// Build path -> node map for existing nodes
	existingByPath := make(map[string]*entities.SitemapNode)
	var rootNode *entities.SitemapNode
	for _, node := range existingNodes {
		existingByPath[node.Path] = node
		if node.IsRoot {
			rootNode = node
		}
	}

	if rootNode == nil {
		return nil, fmt.Errorf("sitemap has no root node")
	}

	// Determine parent for top-level imports
	var baseParentID *int64
	if opts != nil && opts.ParentNodeID != nil {
		baseParentID = opts.ParentNodeID
	} else {
		baseParentID = &rootNode.ID
	}

	// Map of created nodes: path -> node ID (for parent lookups)
	createdNodes := make(map[string]int64)

	// Create nodes in order (sorted by depth)
	for _, parsed := range parsedNodes {
		// Check if node already exists
		if existing, ok := existingByPath[parsed.Path]; ok {
			stats.NodesSkipped++
			createdNodes[parsed.Path] = existing.ID
			continue
		}

		// Determine parent ID
		var parentID *int64
		if parsed.ParentPath == "" {
			// Direct child of import target (root or specified parent)
			parentID = baseParentID
		} else {
			// Look for parent in created nodes or existing nodes
			if parentNodeID, ok := createdNodes[parsed.ParentPath]; ok {
				parentID = &parentNodeID
			} else if existing, ok := existingByPath[parsed.ParentPath]; ok {
				parentID = &existing.ID
			} else {
				// Parent should have been created already (nodes are sorted by depth)
				stats.Errors = append(stats.Errors, ImportError{
					Message: fmt.Sprintf("parent not found for path %s (parent: %s)", parsed.Path, parsed.ParentPath),
				})
				continue
			}
		}

		// Create node
		node := &entities.SitemapNode{
			SitemapID:     sitemapID,
			ParentID:      parentID,
			Title:         parsed.Title,
			Slug:          parsed.Slug,
			IsRoot:        false,
			Source:        entities.NodeSourceImported,
			ContentType:   entities.NodeContentTypeNone,
			ContentStatus: entities.NodeContentStatusDraft,
			Keywords:      parsed.Keywords,
		}

		if err := nodeCreator.CreateNode(ctx, node); err != nil {
			// Check if node already exists - if so, find it and use its ID for children
			if errors.IsAlreadyExists(err) {
				existingNode, findErr := nodeCreator.FindNodeBySlugAndParent(ctx, sitemapID, parsed.Slug, parentID)
				if findErr == nil && existingNode != nil {
					// Node exists, use it as parent for children
					createdNodes[parsed.Path] = existingNode.ID
					stats.NodesSkipped++
					continue
				}
			}
			stats.Errors = append(stats.Errors, ImportError{
				Message: fmt.Sprintf("failed to create node %s: %v", parsed.Path, err),
			})
			continue
		}

		createdNodes[parsed.Path] = node.ID
		stats.NodesCreated++
	}

	stats.ProcessingTime = time.Since(startTime)

	i.logger.Infof("Import complete: %d created, %d skipped, %d errors in %v",
		stats.NodesCreated, stats.NodesSkipped, len(stats.Errors), stats.ProcessingTime)

	return stats, nil
}
