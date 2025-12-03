package scanner

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/infra/wp"
	apperrors "github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

// Scanner scans WordPress sites and creates sitemaps from pages/posts
type Scanner struct {
	wpClient    wp.Client
	siteService sites.Service
	sitemapSvc  sitemap.Service
	logger      *logger.Logger
}

// NewScanner creates a new scanner instance
func NewScanner(
	wpClient wp.Client,
	siteService sites.Service,
	sitemapSvc sitemap.Service,
	logger *logger.Logger,
) *Scanner {
	return &Scanner{
		wpClient:    wpClient,
		siteService: siteService,
		sitemapSvc:  sitemapSvc,
		logger: logger.
			WithScope("scanner").
			WithScope("sitemap"),
	}
}

// ScanAndCreateSitemap scans a WordPress site and creates a sitemap from its structure
func (s *Scanner) ScanAndCreateSitemap(
	ctx context.Context,
	siteID int64,
	sitemapName string,
	opts *ScanOptions,
) (*ScanResult, error) {
	startTime := time.Now()

	if opts == nil {
		opts = DefaultScanOptions()
	}

	result := &ScanResult{
		Errors: make([]ScanError, 0),
	}

	// Get site with credentials
	site, err := s.siteService.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	s.logger.Infof("Starting scan for site %s (%s)", site.Name, site.URL)

	// Scan pages
	var scannedPages []*ScannedPage
	if opts.ContentFilter == ContentFilterAll || opts.ContentFilter == ContentFilterPages {
		pages, scanErrors := s.scanPages(ctx, site, opts)
		scannedPages = append(scannedPages, pages...)
		result.Errors = append(result.Errors, scanErrors...)
		result.PagesScanned = len(pages)
		s.logger.Infof("Scanned %d pages", len(pages))
	}

	// Scan posts (if needed)
	var scannedPosts []*ScannedPage
	if opts.ContentFilter == ContentFilterAll || opts.ContentFilter == ContentFilterPosts {
		posts, scanErrors := s.scanPosts(ctx, site, opts)
		scannedPosts = append(scannedPosts, posts...)
		result.Errors = append(result.Errors, scanErrors...)
		result.PostsScanned = len(posts)
		s.logger.Infof("Scanned %d posts", len(posts))
	}

	// Build hierarchy for pages (posts are flat, pages have parent relationships)
	pageTree := s.buildPageTree(scannedPages, site.URL)

	// Create sitemap
	sitemapEntity := &entities.Sitemap{
		SiteID: siteID,
		Name:   sitemapName,
		Source: entities.SitemapSourceScanned,
		Status: entities.SitemapStatusDraft,
	}

	if err = s.sitemapSvc.CreateSitemapWithRoot(ctx, sitemapEntity, site.URL); err != nil {
		return nil, fmt.Errorf("failed to create sitemap: %w", err)
	}

	result.SitemapID = sitemapEntity.ID
	s.logger.Infof("Created sitemap %d: %s", sitemapEntity.ID, sitemapName)

	// Get the root node
	nodes, err := s.sitemapSvc.GetNodes(ctx, sitemapEntity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sitemap nodes: %w", err)
	}

	var rootNode *entities.SitemapNode
	for _, n := range nodes {
		if n.IsRoot {
			rootNode = n
			break
		}
	}

	if rootNode == nil {
		return nil, fmt.Errorf("root node not found")
	}

	// Create nodes from scanned pages
	nodesCreated, nodesSkipped, createErrors := s.createNodesFromPages(
		ctx, sitemapEntity.ID, rootNode.ID, pageTree, opts,
	)
	result.NodesCreated += nodesCreated
	result.NodesSkipped += nodesSkipped
	result.Errors = append(result.Errors, createErrors...)

	// Create nodes from posts (as children of root, flat structure)
	if len(scannedPosts) > 0 {
		created, skipped, postErrors := s.createNodesFromPosts(
			ctx, sitemapEntity.ID, rootNode.ID, scannedPosts, opts,
		)
		result.NodesCreated += created
		result.NodesSkipped += skipped
		result.Errors = append(result.Errors, postErrors...)
	}

	result.TotalDuration = time.Since(startTime)

	s.logger.Infof(
		"Scan complete: %d pages, %d posts, %d nodes created, %d skipped, %d errors in %v",
		result.PagesScanned, result.PostsScanned,
		result.NodesCreated, result.NodesSkipped,
		len(result.Errors), result.TotalDuration,
	)

	return result, nil
}

// scanPages fetches all pages from WordPress
func (s *Scanner) scanPages(ctx context.Context, site *entities.Site, opts *ScanOptions) ([]*ScannedPage, []ScanError) {
	var scanned []*ScannedPage
	var errors []ScanError

	// Fetch all pages
	pages, err := s.wpClient.GetAllPages(ctx, site)
	if err != nil {
		errors = append(errors, ScanError{
			Type:    "page",
			Message: fmt.Sprintf("failed to fetch pages: %v", err),
		})
		return scanned, errors
	}

	for _, page := range pages {
		// Filter by status
		if page.Status != "publish" && !opts.IncludeDrafts {
			continue
		}

		slug := page.Slug
		// Generate temporary slug for drafts without slug
		if slug == "" {
			slug = fmt.Sprintf("draft-%d", page.ID)
		}

		scannedPage := &ScannedPage{
			WPID:     page.ID,
			WPType:   entities.NodeContentTypePage,
			ParentID: page.ParentID,
			Title:    page.Title,
			H1:       ExtractH1(page.Content),
			Slug:     slug,
			URL:      page.Link,
			Status:   page.Status,
			Content:  page.Content,
		}

		scanned = append(scanned, scannedPage)
	}

	return scanned, errors
}

// scanPosts fetches all posts from WordPress
func (s *Scanner) scanPosts(ctx context.Context, site *entities.Site, opts *ScanOptions) ([]*ScannedPage, []ScanError) {
	var scanned []*ScannedPage
	var errors []ScanError

	// Fetch all posts
	posts, err := s.wpClient.GetPosts(ctx, site)
	if err != nil {
		errors = append(errors, ScanError{
			Type:    "post",
			Message: fmt.Sprintf("failed to fetch posts: %v", err),
		})
		return scanned, errors
	}

	for _, post := range posts {
		// Filter by status
		wpStatus := mapArticleStatusToWPStatus(post.Status)
		if wpStatus != "publish" && !opts.IncludeDrafts {
			continue
		}

		slug := ""
		if post.Slug != nil {
			slug = *post.Slug
		}
		if slug == "" {
			slug = extractSlugFromLink(post.WPPostURL, site.URL)
		}
		// Generate temporary slug for drafts without slug
		if slug == "" {
			slug = fmt.Sprintf("draft-%d", post.WPPostID)
		}

		scannedPost := &ScannedPage{
			WPID:     post.WPPostID,
			WPType:   entities.NodeContentTypePost,
			ParentID: 0, // Posts don't have parents
			Title:    post.Title,
			H1:       ExtractH1(post.Content),
			Slug:     slug,
			URL:      post.WPPostURL,
			Status:   wpStatus,
			Content:  post.Content,
		}

		scanned = append(scanned, scannedPost)
	}

	return scanned, errors
}

// mapArticleStatusToWPStatus converts ArticleStatus to WP status string
func mapArticleStatusToWPStatus(status entities.ArticleStatus) string {
	switch status {
	case entities.StatusPublished:
		return "publish"
	case entities.StatusDraft:
		return "draft"
	case entities.StatusPending:
		return "pending"
	case entities.StatusPrivate:
		return "private"
	default:
		return "draft"
	}
}

// buildPageTree builds a tree structure from flat pages list
func (s *Scanner) buildPageTree(pages []*ScannedPage, siteURL string) []*ScannedPage {
	// Create a map for quick lookup by WordPress ID
	pageByWPID := make(map[int]*ScannedPage)
	for _, page := range pages {
		pageByWPID[page.WPID] = page
	}

	var roots []*ScannedPage

	// Build tree relationships
	for _, page := range pages {
		if page.ParentID == 0 {
			// Top-level page
			page.Depth = 0
			page.Path = "/" + page.Slug
			roots = append(roots, page)
		} else if parent, ok := pageByWPID[page.ParentID]; ok {
			// Child page
			parent.Children = append(parent.Children, page)
		} else {
			// Parent not found, treat as root
			page.Depth = 0
			page.Path = "/" + page.Slug
			roots = append(roots, page)
		}
	}

	// Calculate depths and paths recursively
	for _, root := range roots {
		s.calculateHierarchy(root, 0, "")
	}

	// Sort roots by slug for consistent ordering
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Slug < roots[j].Slug
	})

	return roots
}

// calculateHierarchy calculates depth and path for a page and its children
func (s *Scanner) calculateHierarchy(page *ScannedPage, depth int, parentPath string) {
	page.Depth = depth
	if parentPath == "" {
		page.Path = "/" + page.Slug
	} else {
		page.Path = parentPath + "/" + page.Slug
	}

	// Sort children by slug
	sort.Slice(page.Children, func(i, j int) bool {
		return page.Children[i].Slug < page.Children[j].Slug
	})

	for _, child := range page.Children {
		s.calculateHierarchy(child, depth+1, page.Path)
	}
}

// createNodesFromPages creates sitemap nodes from scanned pages
func (s *Scanner) createNodesFromPages(
	ctx context.Context,
	sitemapID int64,
	rootNodeID int64,
	pages []*ScannedPage,
	opts *ScanOptions,
) (int, int, []ScanError) {
	var created, skipped int
	var errors []ScanError

	// Map of WordPress page ID to created node ID
	wpIDToNodeID := make(map[int]int64)

	// Process pages in tree order (level by level)
	var processPage func(page *ScannedPage, parentNodeID int64)
	processPage = func(page *ScannedPage, parentNodeID int64) {
		// Check max depth
		if opts.MaxDepth > 0 && page.Depth >= opts.MaxDepth {
			skipped++
			return
		}

		// Determine title based on options
		title := page.GetDisplayTitle(opts.TitleSource)
		if title == "" {
			title = page.Slug // Fallback to slug if no title
		}

		// Check if node with this WP ID already exists - if so, update it
		existingNode, err := s.sitemapSvc.FindNodeByWPID(ctx, sitemapID, page.WPID, entities.NodeContentTypePage)
		if err == nil && existingNode != nil {
			// Update existing node with new data from WP
			existingNode.Title = title
			existingNode.Slug = page.Slug
			existingNode.WPURL = &page.URL
			existingNode.WPTitle = &title     // Update original WP title
			existingNode.WPSlug = &page.Slug  // Update original WP slug
			existingNode.ContentStatus = s.mapWPStatusToNodeStatus(page.Status)

			if updateErr := s.sitemapSvc.UpdateNode(ctx, existingNode); updateErr != nil {
				errors = append(errors, ScanError{
					WPID:    page.WPID,
					Type:    "page",
					Title:   page.Title,
					Message: fmt.Sprintf("failed to update node: %v", updateErr),
				})
			}
			// Node was updated, not created - track for children processing
			wpIDToNodeID[page.WPID] = existingNode.ID
			skipped++ // Count as skipped (not newly created)

			// Process children with existing node as parent
			for _, child := range page.Children {
				processPage(child, existingNode.ID)
			}
			return
		}

		// Create new node
		node := &entities.SitemapNode{
			SitemapID:     sitemapID,
			ParentID:      &parentNodeID,
			Title:         title,
			Slug:          page.Slug,
			Source:        entities.NodeSourceScanned,
			ContentType:   entities.NodeContentTypePage,
			WPPageID:      &page.WPID,
			WPURL:         &page.URL,
			WPTitle:       &title,     // Store original WP title
			WPSlug:        &page.Slug, // Store original WP slug
			ContentStatus: s.mapWPStatusToNodeStatus(page.Status),
		}

		if err := s.sitemapSvc.CreateNode(ctx, node); err != nil {
			// If node already exists by slug, just skip it silently
			if apperrors.IsAlreadyExists(err) {
				skipped++
				return
			}
			errors = append(errors, ScanError{
				WPID:    page.WPID,
				Type:    "page",
				Title:   page.Title,
				Message: fmt.Sprintf("failed to create node: %v", err),
			})
			skipped++
			return
		}

		created++
		wpIDToNodeID[page.WPID] = node.ID

		// Process children
		for _, child := range page.Children {
			processPage(child, node.ID)
		}
	}

	// Process all root pages
	for _, page := range pages {
		processPage(page, rootNodeID)
	}

	return created, skipped, errors
}

// createNodesFromPosts creates sitemap nodes from scanned posts
func (s *Scanner) createNodesFromPosts(
	ctx context.Context,
	sitemapID int64,
	parentNodeID int64,
	posts []*ScannedPage,
	opts *ScanOptions,
) (int, int, []ScanError) {
	var created, skipped int
	var errors []ScanError

	for _, post := range posts {
		// Determine title based on options
		title := post.GetDisplayTitle(opts.TitleSource)
		if title == "" {
			title = post.Slug
		}

		wpID := post.WPID

		// Check if node with this WP ID already exists - if so, update it
		existingNode, err := s.sitemapSvc.FindNodeByWPID(ctx, sitemapID, wpID, entities.NodeContentTypePost)
		if err == nil && existingNode != nil {
			// Update existing node with new data from WP
			existingNode.Title = title
			existingNode.Slug = post.Slug
			existingNode.WPURL = &post.URL
			existingNode.WPTitle = &title     // Update original WP title
			existingNode.WPSlug = &post.Slug  // Update original WP slug
			existingNode.ContentStatus = s.mapWPStatusToNodeStatus(post.Status)

			if updateErr := s.sitemapSvc.UpdateNode(ctx, existingNode); updateErr != nil {
				errors = append(errors, ScanError{
					WPID:    post.WPID,
					Type:    "post",
					Title:   post.Title,
					Message: fmt.Sprintf("failed to update node: %v", updateErr),
				})
			}
			skipped++ // Count as skipped (not newly created)
			continue
		}

		// Create new node
		node := &entities.SitemapNode{
			SitemapID:     sitemapID,
			ParentID:      &parentNodeID,
			Title:         title,
			Slug:          post.Slug,
			Source:        entities.NodeSourceScanned,
			ContentType:   entities.NodeContentTypePost,
			ArticleID:     nil, // We don't have internal article ID yet
			WPPageID:      &wpID,
			WPURL:         &post.URL,
			WPTitle:       &title,     // Store original WP title
			WPSlug:        &post.Slug, // Store original WP slug
			ContentStatus: s.mapWPStatusToNodeStatus(post.Status),
		}

		if err := s.sitemapSvc.CreateNode(ctx, node); err != nil {
			// If node already exists by slug, just skip it silently
			if apperrors.IsAlreadyExists(err) {
				skipped++
				continue
			}
			errors = append(errors, ScanError{
				WPID:    post.WPID,
				Type:    "post",
				Title:   post.Title,
				Message: fmt.Sprintf("failed to create node: %v", err),
			})
			skipped++
			continue
		}

		created++
	}

	return created, skipped, errors
}

// mapWPStatusToNodeStatus maps WordPress status to node content status
func (s *Scanner) mapWPStatusToNodeStatus(wpStatus string) entities.NodeContentStatus {
	switch wpStatus {
	case "publish":
		return entities.NodeContentStatusPublished
	case "draft":
		return entities.NodeContentStatusDraft
	default:
		return entities.NodeContentStatusPending
	}
}

// ScanIntoSitemap scans a WordPress site and adds nodes to an existing sitemap
func (s *Scanner) ScanIntoSitemap(
	ctx context.Context,
	sitemapID int64,
	parentNodeID *int64,
	opts *ScanOptions,
) (*ScanResult, error) {
	startTime := time.Now()

	if opts == nil {
		opts = DefaultScanOptions()
	}

	result := &ScanResult{
		SitemapID: sitemapID,
		Errors:    make([]ScanError, 0),
	}

	// Get the sitemap to find site ID
	existingSitemap, err := s.sitemapSvc.GetSitemap(ctx, sitemapID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sitemap: %w", err)
	}

	// Get site with credentials
	site, err := s.siteService.GetSiteWithPassword(ctx, existingSitemap.SiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	s.logger.Infof("Starting scan into existing sitemap %d for site %s (%s)", sitemapID, site.Name, site.URL)

	// Scan pages
	var scannedPages []*ScannedPage
	if opts.ContentFilter == ContentFilterAll || opts.ContentFilter == ContentFilterPages {
		pages, scanErrors := s.scanPages(ctx, site, opts)
		scannedPages = append(scannedPages, pages...)
		result.Errors = append(result.Errors, scanErrors...)
		result.PagesScanned = len(pages)
		s.logger.Infof("Scanned %d pages", len(pages))
	}

	// Scan posts
	var scannedPosts []*ScannedPage
	if opts.ContentFilter == ContentFilterAll || opts.ContentFilter == ContentFilterPosts {
		posts, scanErrors := s.scanPosts(ctx, site, opts)
		scannedPosts = append(scannedPosts, posts...)
		result.Errors = append(result.Errors, scanErrors...)
		result.PostsScanned = len(posts)
		s.logger.Infof("Scanned %d posts", len(posts))
	}

	// Build hierarchy for pages
	pageTree := s.buildPageTree(scannedPages, site.URL)

	// Determine parent node
	var targetParentID int64
	if parentNodeID != nil {
		targetParentID = *parentNodeID
	} else {
		// Find root node
		nodes, err := s.sitemapSvc.GetNodes(ctx, sitemapID)
		if err != nil {
			return nil, fmt.Errorf("failed to get sitemap nodes: %w", err)
		}

		for _, n := range nodes {
			if n.IsRoot {
				targetParentID = n.ID
				break
			}
		}

		if targetParentID == 0 {
			return nil, fmt.Errorf("root node not found")
		}
	}

	// Create nodes from scanned pages
	nodesCreated, nodesSkipped, createErrors := s.createNodesFromPages(
		ctx, sitemapID, targetParentID, pageTree, opts,
	)
	result.NodesCreated += nodesCreated
	result.NodesSkipped += nodesSkipped
	result.Errors = append(result.Errors, createErrors...)

	// Create nodes from posts
	if len(scannedPosts) > 0 {
		created, skipped, postErrors := s.createNodesFromPosts(
			ctx, sitemapID, targetParentID, scannedPosts, opts,
		)
		result.NodesCreated += created
		result.NodesSkipped += skipped
		result.Errors = append(result.Errors, postErrors...)
	}

	result.TotalDuration = time.Since(startTime)

	s.logger.Infof(
		"Scan into sitemap complete: %d pages, %d posts, %d nodes created, %d skipped, %d errors in %v",
		result.PagesScanned, result.PostsScanned,
		result.NodesCreated, result.NodesSkipped,
		len(result.Errors), result.TotalDuration,
	)

	return result, nil
}

// extractSlugFromLink extracts slug from a WordPress link
func extractSlugFromLink(link, siteURL string) string {
	// Parse the link
	parsedLink, err := url.Parse(link)
	if err != nil {
		return ""
	}

	// Get the path
	path := parsedLink.Path
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	// Split and get the last segment
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}
