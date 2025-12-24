package generation

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/linking"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/infra/events"
	"github.com/davidmovas/postulator/pkg/logger"
	"github.com/google/uuid"
)

type Executor struct {
	tasks       sync.Map
	sitemapSvc  sitemap.Service
	linkingSvc  linking.Service
	generator   *Generator
	publisher   *Publisher
	eventBus    *events.EventBus
	emitter     *EventEmitter
	logger      *logger.Logger
	cancelFuncs sync.Map
	pauseChs    sync.Map
}

func NewExecutor(
	sitemapSvc sitemap.Service,
	linkingSvc linking.Service,
	generator *Generator,
	publisher *Publisher,
	eventBus *events.EventBus,
	logger *logger.Logger,
) *Executor {
	return &Executor{
		sitemapSvc: sitemapSvc,
		linkingSvc: linkingSvc,
		generator:  generator,
		publisher:  publisher,
		eventBus:   eventBus,
		emitter:    NewEventEmitter(eventBus),
		logger:     logger.WithScope("page_executor"),
	}
}

func (e *Executor) Start(ctx context.Context, config GenerationConfig) (*Task, error) {
	nodes, err := e.prepareNodes(ctx, config)
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes to process")
	}

	taskID := uuid.New().String()
	task := &Task{
		ID:         taskID,
		SitemapID:  config.SitemapID,
		SiteID:     config.SiteID,
		ProviderID: config.ProviderID,
		PromptID:   config.PromptID,
		PublishAs:  config.PublishAs,
		Nodes:      nodes,
		TotalNodes: len(nodes),
		Status:     TaskStatusRunning,
		StartedAt:  time.Now(),
	}

	e.tasks.Store(taskID, task)

	// Use a background context with long timeout for the goroutine
	// This ensures the task continues even if the original request context is cancelled
	taskCtx, cancel := context.WithTimeout(context.Background(), DefaultTaskTimeout)
	e.cancelFuncs.Store(taskID, cancel)

	pauseCh := make(chan struct{})
	e.pauseChs.Store(taskID, pauseCh)

	e.emitter.EmitTaskStarted(ctx, taskID, config.SitemapID, len(nodes))

	go e.runTask(taskCtx, task, config, pauseCh)

	return task, nil
}

func (e *Executor) prepareNodes(ctx context.Context, config GenerationConfig) ([]*TaskNode, error) {
	var sitemapNodes []*entities.SitemapNode
	var err error

	if len(config.NodeIDs) > 0 {
		// Collect selected nodes and their ungenerated ancestors
		nodeMap := make(map[int64]*entities.SitemapNode)

		for _, nodeID := range config.NodeIDs {
			node, err := e.sitemapSvc.GetNode(ctx, nodeID)
			if err != nil {
				e.logger.ErrorWithErr(err, fmt.Sprintf("Failed to get node %d", nodeID))
				continue
			}
			nodeMap[node.ID] = node

			// Add all ungenerated ancestors
			ancestors, _ := e.getAncestors(ctx, node)
			for _, ancestor := range ancestors {
				if ancestor.IsRoot {
					continue
				}
				// Only add if not already generated
				if ancestor.GenerationStatus != entities.GenStatusGenerated || ancestor.WPPageID == nil {
					nodeMap[ancestor.ID] = ancestor
				}
			}
		}

		for _, node := range nodeMap {
			sitemapNodes = append(sitemapNodes, node)
		}
	} else {
		sitemapNodes, err = e.sitemapSvc.GetNodes(ctx, config.SitemapID)
		if err != nil {
			return nil, fmt.Errorf("failed to get sitemap nodes: %w", err)
		}
	}

	var taskNodes []*TaskNode
	for _, node := range sitemapNodes {
		if node.IsRoot {
			continue
		}
		if node.GenerationStatus == entities.GenStatusGenerated && node.WPPageID != nil {
			continue
		}

		taskNodes = append(taskNodes, &TaskNode{
			NodeID:       node.ID,
			Title:        node.Title,
			Slug:         node.Slug,
			Path:         node.Path,
			Keywords:     node.Keywords,
			Depth:        node.Depth,
			ParentNodeID: node.ParentID,
			Status:       NodeStatusPending,
		})
	}

	// Sort by depth first, then by ID for consistent order within same depth
	sort.Slice(taskNodes, func(i, j int) bool {
		if taskNodes[i].Depth != taskNodes[j].Depth {
			return taskNodes[i].Depth < taskNodes[j].Depth
		}
		return taskNodes[i].NodeID < taskNodes[j].NodeID
	})

	// Log prepared nodes for debugging
	for _, n := range taskNodes {
		e.logger.Infof("Prepared node: id=%d title=%q depth=%d parentId=%v",
			n.NodeID, n.Title, n.Depth, n.ParentNodeID)
	}

	return taskNodes, nil
}

func (e *Executor) runTask(ctx context.Context, task *Task, config GenerationConfig, pauseCh chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			e.logger.Errorf("Panic in task %s: %v", task.ID, r)
			task.SetStatus(TaskStatusFailed)
			task.SetError(fmt.Sprintf("panic: %v", r))
			e.emitter.EmitTaskFailed(ctx, task.ID, fmt.Sprintf("panic: %v", r))
		}
	}()

	nodesByDepth := make(map[int][]*TaskNode)
	for _, node := range task.Nodes {
		nodesByDepth[node.Depth] = append(nodesByDepth[node.Depth], node)
	}

	depths := make([]int, 0, len(nodesByDepth))
	for d := range nodesByDepth {
		depths = append(depths, d)
	}
	sort.Ints(depths)

	var wpPageIDMap sync.Map // thread-safe map for parallel access

	// Determine concurrency limit
	maxConcurrency := config.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 3 // default
	}

	for _, depth := range depths {
		nodes := nodesByDepth[depth]

		// Log depth level processing for debugging
		nodeIDs := make([]int64, len(nodes))
		for i, n := range nodes {
			nodeIDs[i] = n.NodeID
		}
		e.logger.Infof("Processing depth %d with %d nodes: %v", depth, len(nodes), nodeIDs)

		// Check for cancellation before processing depth level
		select {
		case <-ctx.Done():
			task.SetStatus(TaskStatusCancelled)
			e.emitter.EmitTaskCancelled(ctx, task.ID, task.ProcessedNodes, task.TotalNodes)
			return
		default:
		}

		// Check for pause
		select {
		case <-pauseCh:
			task.SetStatus(TaskStatusPaused)
			e.emitter.EmitTaskPaused(ctx, task.ID, task.ProcessedNodes, task.TotalNodes)
			<-pauseCh
			task.SetStatus(TaskStatusRunning)
			remaining := task.TotalNodes - task.ProcessedNodes
			e.emitter.EmitTaskResumed(ctx, task.ID, remaining)
		default:
		}

		// Process nodes at this depth level in parallel
		e.processNodesParallel(ctx, task, nodes, config, &wpPageIDMap, maxConcurrency)
		e.logger.Infof("Completed processing depth %d", depth)
	}

	task.Complete()
	e.emitter.EmitTaskCompleted(ctx, task.ID,
		task.ProcessedNodes, task.FailedNodes, task.SkippedNodes, task.TotalNodes, task.StartedAt)

	e.logger.Infof("Task %s completed: %d/%d nodes processed, %d failed",
		task.ID, task.ProcessedNodes, task.TotalNodes, task.FailedNodes)
}

func (e *Executor) processNodesParallel(ctx context.Context, task *Task, nodes []*TaskNode, config GenerationConfig, wpPageIDMap *sync.Map, maxConcurrency int) {
	if len(nodes) == 0 {
		return
	}

	// Semaphore to limit concurrency
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, node := range nodes {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		wg.Add(1)
		sem <- struct{}{} // acquire semaphore

		go func(n *TaskNode) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore

			// Set parent WP page ID if available
			if n.ParentNodeID != nil {
				if wpID, ok := wpPageIDMap.Load(*n.ParentNodeID); ok {
					id := wpID.(int)
					n.ParentWPPageID = &id
				} else {
					parentNode, err := e.sitemapSvc.GetNode(ctx, *n.ParentNodeID)
					if err == nil && parentNode.WPPageID != nil {
						n.ParentWPPageID = parentNode.WPPageID
					}
				}
			}

			err := e.processNode(ctx, task, n, config)
			if err != nil {
				errMsg := err.Error()
				n.MarkFailed(errMsg)
				task.IncrementFailed()
				e.logger.Errorf("Node %d (%s) failed: %s", n.NodeID, n.Title, errMsg)
				e.emitter.EmitNodeFailed(ctx, task.ID, n.NodeID, n.Title, errMsg)
			} else if n.WPPageID != nil {
				wpPageIDMap.Store(n.NodeID, *n.WPPageID)
			}

			task.IncrementProcessed()
			e.emitter.EmitTaskProgress(ctx, task.ID,
				task.ProcessedNodes, task.TotalNodes, task.FailedNodes, task.SkippedNodes,
				&NodeInfo{NodeID: n.NodeID, Title: n.Title, Path: n.Path})
		}(node)
	}

	wg.Wait()
}

func (e *Executor) processNode(ctx context.Context, task *Task, taskNode *TaskNode, config GenerationConfig) error {
	node, err := e.sitemapSvc.GetNode(ctx, taskNode.NodeID)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	if node.GenerationStatus == entities.GenStatusGenerated && node.WPPageID != nil {
		taskNode.Status = NodeStatusSkipped
		task.IncrementSkipped()
		e.emitter.EmitNodeSkipped(ctx, task.ID, taskNode.NodeID, taskNode.Title, "already generated")
		return nil
	}

	if err := e.sitemapSvc.UpdateNodeGenerationStatus(ctx, node.ID, entities.GenStatusGenerating, nil); err != nil {
		e.logger.ErrorWithErr(err, "Failed to update node status to generating")
	}

	taskNode.MarkStarted()
	taskNode.SetStatus(NodeStatusGenerating)
	e.emitter.EmitNodeGenerating(ctx, task.ID, taskNode.NodeID, taskNode.Title)

	ancestors, err := e.getAncestors(ctx, node)
	if err != nil {
		e.logger.ErrorWithErr(err, "Failed to get ancestors")
	}

	// Fetch link targets if IncludeLinks is enabled
	var linkTargets []LinkTarget
	if config.ContentSettings != nil && config.ContentSettings.IncludeLinks {
		linkTargets = e.getApprovedLinkTargets(ctx, task.SitemapID, task.SiteID, node.ID)
		if len(linkTargets) > 0 {
			e.logger.Infof("Node %d: including %d approved link targets in generation", node.ID, len(linkTargets))
		}
	}

	genStartTime := time.Now()
	genResult, err := e.generator.Generate(ctx, GenerateRequest{
		Node:            node,
		Ancestors:       ancestors,
		SiteID:          task.SiteID,
		ProviderID:      config.ProviderID,
		PromptID:        config.PromptID,
		Placeholders:    config.Placeholders,
		ContentSettings: config.ContentSettings,
		LinkTargets:     linkTargets,
	})
	if err != nil {
		errStr := err.Error()
		_ = e.sitemapSvc.UpdateNodeGenerationStatus(ctx, node.ID, entities.GenStatusFailed, &errStr)
		return fmt.Errorf("generation failed: %w", err)
	}

	tokensUsed := genResult.Content.InputTokens + genResult.Content.OutputTokens
	e.emitter.EmitNodeGenerated(ctx, task.ID, taskNode.NodeID, taskNode.Title, tokensUsed, genStartTime)

	taskNode.SetStatus(NodeStatusPublishing)
	e.emitter.EmitNodePublishing(ctx, task.ID, taskNode.NodeID, taskNode.Title)

	if err := e.sitemapSvc.UpdateNodePublishStatus(ctx, node.ID, entities.PubStatusPublishing, nil); err != nil {
		e.logger.ErrorWithErr(err, "Failed to update node status to publishing")
	}

	pubResult, err := e.publisher.Publish(ctx, PublishRequest{
		Node:           node,
		Content:        genResult.Content,
		SiteID:         task.SiteID,
		PublishAs:      task.PublishAs,
		ParentWPPageID: taskNode.ParentWPPageID,
	})
	if err != nil {
		errStr := err.Error()
		_ = e.sitemapSvc.UpdateNodePublishStatus(ctx, node.ID, entities.PubStatusFailed, &errStr)
		return fmt.Errorf("publish failed: %w", err)
	}

	// Mark links as applied if IncludeLinks was used
	if len(linkTargets) > 0 {
		e.markLinksAsApplied(ctx, linkTargets)
	}

	taskNode.MarkCompleted(pubResult.ArticleID, pubResult.WPPageID, pubResult.WPURL)
	e.emitter.EmitNodeCompleted(ctx, task.ID, taskNode.NodeID, taskNode.Title,
		pubResult.ArticleID, pubResult.WPPageID, pubResult.WPURL)

	return nil
}

func (e *Executor) getAncestors(ctx context.Context, node *entities.SitemapNode) ([]*entities.SitemapNode, error) {
	var ancestors []*entities.SitemapNode
	currentID := node.ParentID

	for currentID != nil {
		parent, err := e.sitemapSvc.GetNode(ctx, *currentID)
		if err != nil {
			return ancestors, err
		}
		ancestors = append([]*entities.SitemapNode{parent}, ancestors...)
		currentID = parent.ParentID
	}

	return ancestors, nil
}

func (e *Executor) Pause(taskID string) error {
	task, ok := e.tasks.Load(taskID)
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	t := task.(*Task)
	if t.GetStatus() != TaskStatusRunning {
		return fmt.Errorf("task is not running")
	}

	if pauseCh, ok := e.pauseChs.Load(taskID); ok {
		pauseCh.(chan struct{}) <- struct{}{}
	}

	return nil
}

func (e *Executor) Resume(taskID string) error {
	task, ok := e.tasks.Load(taskID)
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	t := task.(*Task)
	if t.GetStatus() != TaskStatusPaused {
		return fmt.Errorf("task is not paused")
	}

	if pauseCh, ok := e.pauseChs.Load(taskID); ok {
		pauseCh.(chan struct{}) <- struct{}{}
	}

	return nil
}

func (e *Executor) Cancel(taskID string) error {
	task, ok := e.tasks.Load(taskID)
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	t := task.(*Task)

	if cancelFunc, ok := e.cancelFuncs.Load(taskID); ok {
		cancelFunc.(context.CancelFunc)()
	}

	e.resetPendingNodes(t)
	return nil
}

func (e *Executor) resetPendingNodes(task *Task) {
	ctx := context.Background()
	for _, node := range task.Nodes {
		if node.Status == NodeStatusPending || node.Status == NodeStatusGenerating {
			if err := e.sitemapSvc.UpdateNodeGenerationStatus(ctx, node.NodeID, entities.GenStatusNone, nil); err != nil {
				e.logger.ErrorWithErr(err, fmt.Sprintf("Failed to reset node %d status", node.NodeID))
			}
		}
	}
}

func (e *Executor) GetTask(taskID string) *Task {
	if task, ok := e.tasks.Load(taskID); ok {
		return task.(*Task)
	}
	return nil
}

func (e *Executor) ListActiveTasks() []*Task {
	var activeTasks []*Task
	e.tasks.Range(func(key, value interface{}) bool {
		task := value.(*Task)
		if task.Status == TaskStatusRunning || task.Status == TaskStatusPaused {
			activeTasks = append(activeTasks, task)
		}
		return true
	})
	return activeTasks
}

func (e *Executor) CleanupCompletedTasks(olderThan time.Duration) {
	cutoff := time.Now().Add(-olderThan)
	e.tasks.Range(func(key, value interface{}) bool {
		task := value.(*Task)
		if task.CompletedAt != nil && task.CompletedAt.Before(cutoff) {
			e.tasks.Delete(key)
			e.cancelFuncs.Delete(key)
			e.pauseChs.Delete(key)
		}
		return true
	})
}

// getApprovedLinkTargets fetches approved outgoing links for a node from the linking plan
func (e *Executor) getApprovedLinkTargets(ctx context.Context, sitemapID, siteID, nodeID int64) []LinkTarget {
	if e.linkingSvc == nil {
		return nil
	}

	// Get active link plan for this sitemap
	plan, err := e.linkingSvc.GetActivePlan(ctx, sitemapID)
	if err != nil || plan == nil {
		return nil
	}

	// Get all links for this plan
	links, err := e.linkingSvc.GetLinks(ctx, plan.ID)
	if err != nil {
		return nil
	}

	// Filter for approved outgoing links from this node
	var targets []LinkTarget
	for _, link := range links {
		// Only outgoing links from this node that are approved
		if link.SourceNodeID != nodeID || link.Status != linking.LinkStatusApproved {
			continue
		}

		// Get target node info
		targetNode, err := e.sitemapSvc.GetNode(ctx, link.TargetNodeID)
		if err != nil {
			continue
		}

		targets = append(targets, LinkTarget{
			LinkID:       link.ID,
			TargetNodeID: targetNode.ID,
			TargetTitle:  targetNode.Title,
			TargetPath:   targetNode.Path,
			AnchorText:   link.AnchorText,
		})
	}

	if len(targets) > 0 {
		e.logger.Infof("Node %d: found %d approved link targets", nodeID, len(targets))
	}
	return targets
}

// markLinksAsApplied marks the given links as applied in the linking plan
func (e *Executor) markLinksAsApplied(ctx context.Context, linkTargets []LinkTarget) {
	if e.linkingSvc == nil || len(linkTargets) == 0 {
		return
	}

	for _, target := range linkTargets {
		if target.LinkID > 0 {
			if err := e.linkingSvc.ApproveAndApplyLink(ctx, target.LinkID); err != nil {
				e.logger.ErrorWithErr(err, fmt.Sprintf("Failed to mark link %d as applied", target.LinkID))
			}
		}
	}

	e.logger.Infof("Marked %d links as applied", len(linkTargets))
}
