// Package actions provides undo/redo action implementations for sitemap operations.
package actions

import (
	"context"
	"fmt"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/pkg/history"
)

// Ensure all actions implement the interface
var (
	_ history.Action = (*CreateNodeAction)(nil)
	_ history.Action = (*DeleteNodeAction)(nil)
	_ history.Action = (*UpdateNodeAction)(nil)
	_ history.Action = (*MoveNodeAction)(nil)
	_ history.Action = (*BatchCreateNodesAction)(nil)
)

// CreateNodeAction represents a single node creation.
type CreateNodeAction struct {
	svc      sitemap.Service
	node     *entities.SitemapNode // Created node (with ID after Do)
	keywords []string              // Separate keywords since they're not part of initial create
}

// NewCreateNodeAction creates an action for node creation.
// The node should have all fields set except ID (which is assigned after creation).
func NewCreateNodeAction(svc sitemap.Service, node *entities.SitemapNode, keywords []string) *CreateNodeAction {
	return &CreateNodeAction{
		svc:      svc,
		node:     node,
		keywords: keywords,
	}
}

func (a *CreateNodeAction) Do(ctx context.Context) error {
	// Reset ID in case this is a redo
	a.node.ID = 0

	if err := a.svc.CreateNode(ctx, a.node); err != nil {
		return err
	}

	// Set keywords if any
	if len(a.keywords) > 0 {
		if err := a.svc.SetNodeKeywords(ctx, a.node.ID, a.keywords); err != nil {
			// Log but don't fail
		}
	}

	return nil
}

func (a *CreateNodeAction) Undo(ctx context.Context) error {
	if a.node.ID == 0 {
		return nil
	}
	return a.svc.DeleteNode(ctx, a.node.ID)
}

func (a *CreateNodeAction) Description() string {
	return fmt.Sprintf("Create node '%s'", a.node.Title)
}

// GetNodeID returns the ID of the created node (available after Do).
func (a *CreateNodeAction) GetNodeID() int64 {
	return a.node.ID
}

// DeleteNodeAction represents a node deletion.
type DeleteNodeAction struct {
	svc      sitemap.Service
	snapshot *entities.SitemapNode // Full node snapshot for restoration
	keywords []string
}

// NewDeleteNodeAction creates an action for node deletion.
// Takes a snapshot of the node for undo.
func NewDeleteNodeAction(svc sitemap.Service, node *entities.SitemapNode, keywords []string) *DeleteNodeAction {
	// Deep copy the node
	snapshot := &entities.SitemapNode{
		ID:            node.ID,
		SitemapID:     node.SitemapID,
		ParentID:      node.ParentID,
		Title:         node.Title,
		Slug:          node.Slug,
		Description:   node.Description,
		IsRoot:        node.IsRoot,
		Depth:         node.Depth,
		Position:      node.Position,
		Path:          node.Path,
		ContentType:   node.ContentType,
		ArticleID:     node.ArticleID,
		WPPageID:      node.WPPageID,
		WPURL:         node.WPURL,
		Source:        node.Source,
		IsSynced:      node.IsSynced,
		LastSyncedAt:  node.LastSyncedAt,
		WPTitle:       node.WPTitle,
		WPSlug:        node.WPSlug,
		ContentStatus: node.ContentStatus,
		PositionX:     node.PositionX,
		PositionY:     node.PositionY,
		CreatedAt:     node.CreatedAt,
		UpdatedAt:     node.UpdatedAt,
	}

	return &DeleteNodeAction{
		svc:      svc,
		snapshot: snapshot,
		keywords: keywords,
	}
}

func (a *DeleteNodeAction) Do(ctx context.Context) error {
	return a.svc.DeleteNode(ctx, a.snapshot.ID)
}

func (a *DeleteNodeAction) Undo(ctx context.Context) error {
	// Recreate the node
	node := &entities.SitemapNode{
		SitemapID:     a.snapshot.SitemapID,
		ParentID:      a.snapshot.ParentID,
		Title:         a.snapshot.Title,
		Slug:          a.snapshot.Slug,
		Description:   a.snapshot.Description,
		IsRoot:        a.snapshot.IsRoot,
		Position:      a.snapshot.Position,
		ContentType:   a.snapshot.ContentType,
		Source:        a.snapshot.Source,
		ContentStatus: a.snapshot.ContentStatus,
		PositionX:     a.snapshot.PositionX,
		PositionY:     a.snapshot.PositionY,
	}

	if err := a.svc.CreateNode(ctx, node); err != nil {
		return err
	}

	// Update snapshot with new ID
	a.snapshot.ID = node.ID

	// Restore keywords
	if len(a.keywords) > 0 {
		if err := a.svc.SetNodeKeywords(ctx, node.ID, a.keywords); err != nil {
			// Log but don't fail
		}
	}

	// Restore position if set
	if a.snapshot.PositionX != nil && a.snapshot.PositionY != nil {
		if err := a.svc.UpdateNodePositions(ctx, node.ID, *a.snapshot.PositionX, *a.snapshot.PositionY); err != nil {
			// Log but don't fail
		}
	}

	return nil
}

func (a *DeleteNodeAction) Description() string {
	return fmt.Sprintf("Delete node '%s'", a.snapshot.Title)
}

// UpdateNodeAction represents a node update.
type UpdateNodeAction struct {
	svc         sitemap.Service
	nodeID      int64
	oldData     NodeUpdateData
	newData     NodeUpdateData
	oldKeywords []string
	newKeywords []string
}

// NodeUpdateData holds updatable fields.
type NodeUpdateData struct {
	Title       string
	Slug        string
	Description *string
}

// NewUpdateNodeAction creates an action for node update.
func NewUpdateNodeAction(
	svc sitemap.Service,
	nodeID int64,
	oldData, newData NodeUpdateData,
	oldKeywords, newKeywords []string,
) *UpdateNodeAction {
	return &UpdateNodeAction{
		svc:         svc,
		nodeID:      nodeID,
		oldData:     oldData,
		newData:     newData,
		oldKeywords: oldKeywords,
		newKeywords: newKeywords,
	}
}

func (a *UpdateNodeAction) Do(ctx context.Context) error {
	return a.applyData(ctx, a.newData, a.newKeywords)
}

func (a *UpdateNodeAction) Undo(ctx context.Context) error {
	return a.applyData(ctx, a.oldData, a.oldKeywords)
}

func (a *UpdateNodeAction) applyData(ctx context.Context, data NodeUpdateData, keywords []string) error {
	node, err := a.svc.GetNode(ctx, a.nodeID)
	if err != nil {
		return err
	}

	node.Title = data.Title
	node.Slug = data.Slug
	node.Description = data.Description

	if err := a.svc.UpdateNode(ctx, node); err != nil {
		return err
	}

	if keywords != nil {
		if err := a.svc.SetNodeKeywords(ctx, a.nodeID, keywords); err != nil {
			// Log but don't fail
		}
	}

	return nil
}

func (a *UpdateNodeAction) Description() string {
	return fmt.Sprintf("Update node '%s'", a.newData.Title)
}

// MoveNodeAction represents changing a node's parent.
type MoveNodeAction struct {
	svc         sitemap.Service
	nodeID      int64
	oldParentID *int64
	newParentID *int64
	oldPosition int
	newPosition int
}

// NewMoveNodeAction creates an action for moving a node.
func NewMoveNodeAction(
	svc sitemap.Service,
	nodeID int64,
	oldParentID, newParentID *int64,
	oldPosition, newPosition int,
) *MoveNodeAction {
	return &MoveNodeAction{
		svc:         svc,
		nodeID:      nodeID,
		oldParentID: oldParentID,
		newParentID: newParentID,
		oldPosition: oldPosition,
		newPosition: newPosition,
	}
}

func (a *MoveNodeAction) Do(ctx context.Context) error {
	return a.svc.MoveNode(ctx, a.nodeID, a.newParentID, a.newPosition)
}

func (a *MoveNodeAction) Undo(ctx context.Context) error {
	return a.svc.MoveNode(ctx, a.nodeID, a.oldParentID, a.oldPosition)
}

func (a *MoveNodeAction) Description() string {
	return "Move node"
}

// BatchCreateNodesAction represents creation of multiple nodes as one atomic operation.
// Used for import, scan, and AI generation.
// NOTE: This action supports undo but NOT redo - after undoing a batch create,
// the nodes cannot be recreated with the same IDs.
type BatchCreateNodesAction struct {
	svc        sitemap.Service
	sitemapID  int64
	createdIDs []int64 // Populated after Do
	label      string
	undone     bool // Marks if this action was undone (redo not supported)
}

// NewBatchCreateNodesAction creates an action for batch node creation.
// The actual node creation happens externally; this action just tracks IDs for undo.
func NewBatchCreateNodesAction(svc sitemap.Service, sitemapID int64, label string) *BatchCreateNodesAction {
	return &BatchCreateNodesAction{
		svc:        svc,
		sitemapID:  sitemapID,
		createdIDs: make([]int64, 0),
		label:      label,
	}
}

// AddCreatedID records a node ID that was created as part of this batch.
func (a *BatchCreateNodesAction) AddCreatedID(id int64) {
	a.createdIDs = append(a.createdIDs, id)
}

// GetCreatedIDs returns all created node IDs.
func (a *BatchCreateNodesAction) GetCreatedIDs() []int64 {
	return a.createdIDs
}

func (a *BatchCreateNodesAction) Do(ctx context.Context) error {
	// Redo is not supported for batch operations - we can't recreate the exact same nodes
	if a.undone {
		return fmt.Errorf("redo not supported for batch operations")
	}
	return nil
}

func (a *BatchCreateNodesAction) Undo(ctx context.Context) error {
	// Delete all created nodes in reverse order
	for i := len(a.createdIDs) - 1; i >= 0; i-- {
		if err := a.svc.DeleteNode(ctx, a.createdIDs[i]); err != nil {
			// Continue deleting other nodes even if one fails
		}
	}
	a.undone = true
	return nil
}

func (a *BatchCreateNodesAction) Description() string {
	return fmt.Sprintf("%s (%d nodes)", a.label, len(a.createdIDs))
}

// UpdatePositionAction represents updating a node's canvas position.
type UpdatePositionAction struct {
	svc     sitemap.Service
	nodeID  int64
	oldX    float64
	oldY    float64
	newX    float64
	newY    float64
}

// NewUpdatePositionAction creates an action for position update.
func NewUpdatePositionAction(
	svc sitemap.Service,
	nodeID int64,
	oldX, oldY, newX, newY float64,
) *UpdatePositionAction {
	return &UpdatePositionAction{
		svc:    svc,
		nodeID: nodeID,
		oldX:   oldX,
		oldY:   oldY,
		newX:   newX,
		newY:   newY,
	}
}

func (a *UpdatePositionAction) Do(ctx context.Context) error {
	return a.svc.UpdateNodePositions(ctx, a.nodeID, a.newX, a.newY)
}

func (a *UpdatePositionAction) Undo(ctx context.Context) error {
	return a.svc.UpdateNodePositions(ctx, a.nodeID, a.oldX, a.oldY)
}

func (a *UpdatePositionAction) Description() string {
	return "Move node position"
}
