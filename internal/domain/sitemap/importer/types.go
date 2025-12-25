package importer

// ImportRow represents a single row from import file
// All formats (JSON, CSV, XLSX) parse into this structure
type ImportRow struct {
	Path     string   // URL path like "/services/web" or "services/web"
	Title    string   // Page title
	Keywords []string // Keywords list
}

// ImportResult contains the result of parsing an import file
type ImportResult struct {
	Rows   []ImportRow
	Errors []ImportError
}

// ImportError represents an error during import
type ImportError struct {
	Row     int    // Row number (1-based, 0 if not applicable)
	Column  string // Column name if applicable
	Message string
}

// Parser interface for all import formats
type Parser interface {
	// Parse reads file content and returns import rows
	Parse(data []byte) (*ImportResult, error)
	// SupportedExtensions returns list of supported file extensions
	SupportedExtensions() []string
}

// ParsedNode represents a node ready for creation
// Built from ImportRow with hierarchy resolved
type ParsedNode struct {
	Path       string   // Full path
	Slug       string   // Just the last segment
	Title      string   // Page title
	Keywords   []string // Keywords
	ParentPath string   // Parent's path (empty for root children)
	Depth      int      // Nesting level
	IsRoot     bool     // True for root node (path = "/" or "")
}

// TreeBuilder builds node hierarchy from flat list of paths
type TreeBuilder struct {
	nodes map[string]*ParsedNode // path -> node
}

// NewTreeBuilder creates a new tree builder
func NewTreeBuilder() *TreeBuilder {
	return &TreeBuilder{
		nodes: make(map[string]*ParsedNode),
	}
}

// BuildTree converts ImportRows into hierarchical ParsedNodes
// Handles:
// - Paths with or without leading slash
// - Auto-creating intermediate nodes
// - Deduplicating paths (same path = same node, merges keywords)
func (tb *TreeBuilder) BuildTree(rows []ImportRow) []*ParsedNode {
	// First pass: normalize and collect all paths
	for _, row := range rows {
		path := normalizePath(row.Path)
		if path == "" || path == "/" {
			continue // Skip root entries
		}

		// Create or update node
		if existing, ok := tb.nodes[path]; ok {
			// Merge keywords
			existing.Keywords = mergeKeywords(existing.Keywords, row.Keywords)
			// Update title if empty
			if existing.Title == "" && row.Title != "" {
				existing.Title = row.Title
			}
		} else {
			slug := extractSlug(path)
			parentPath := extractParentPath(path)
			depth := countDepth(path)

			tb.nodes[path] = &ParsedNode{
				Path:       path,
				Slug:       slug,
				Title:      row.Title,
				Keywords:   row.Keywords,
				ParentPath: parentPath,
				Depth:      depth,
				IsRoot:     false,
			}
		}

		// Ensure all parent paths exist (auto-create intermediate nodes)
		tb.ensureParentNodes(path)
	}

	// Convert map to slice, sorted by depth then path
	result := make([]*ParsedNode, 0, len(tb.nodes))
	for _, node := range tb.nodes {
		result = append(result, node)
	}

	// Sort: by depth first, then by path (for consistent ordering)
	sortNodes(result)

	return result
}

// ensureParentNodes creates any missing intermediate nodes
func (tb *TreeBuilder) ensureParentNodes(path string) {
	parentPath := extractParentPath(path)
	for parentPath != "" {
		if _, ok := tb.nodes[parentPath]; !ok {
			slug := extractSlug(parentPath)
			grandParentPath := extractParentPath(parentPath)
			depth := countDepth(parentPath)

			tb.nodes[parentPath] = &ParsedNode{
				Path:       parentPath,
				Slug:       slug,
				Title:      generateTitleFromSlug(slug), // Auto-generate title
				Keywords:   nil,
				ParentPath: grandParentPath,
				Depth:      depth,
				IsRoot:     false,
			}
		}
		parentPath = extractParentPath(parentPath)
	}
}
