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

	taskCtx, cancel := context.WithTimeout(context.Background(), DefaultTaskTimeout)
	e.cancelFuncs.Store(taskID, cancel)

	e.emitter.EmitTaskStarted(ctx, taskID, config.SitemapID, len(nodes))

	go e.runTask(taskCtx, task, config)

	return task, nil
}

func (e *Executor) prepareNodes(ctx context.Context, config GenerationConfig) ([]*TaskNode, error) {
	var sitemapNodes []*entities.SitemapNode
	var err error

	if len(config.NodeIDs) > 0 {
		// Collect selected nodes and their ungenerated ancestors
		nodeMap := make(map[int64]*entities.SitemapNode)

		for _, nodeID := range config.NodeIDs {
			node, err := e.sitemapSvc.GetNodeWithKeywords(ctx, nodeID)
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

func (e *Executor) runTask(ctx context.Context, task *Task, config GenerationConfig) {
	defer func() {
		if r := recover(); r != nil {
			e.logger.Errorf("Panic in task %s: %v", task.ID, r)
			task.SetStatus(TaskStatusFailed)
			task.SetError(fmt.Sprintf("panic: %v", r))
			e.emitter.EmitTaskFailed(ctx, task.ID, fmt.Sprintf("panic: %v", r))
		}
	}()

	// Handle "before" auto-link mode - suggest and approve links before generation
	if config.ContentSettings != nil && config.ContentSettings.AutoLinkMode == AutoLinkModeBefore {
		if err := e.autoSuggestAndApproveLinksBefore(ctx, task, config); err != nil {
			e.logger.ErrorWithErr(err, "Failed to auto-suggest links before generation")
			// Continue without links - don't fail the entire task
		}
	}

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

		select {
		case <-ctx.Done():
			task.SetStatus(TaskStatusCancelled)
			e.emitter.EmitTaskCancelled(ctx, task.ID, task.ProcessedNodes, task.TotalNodes)
			return
		default:
		}

		e.processNodesParallel(ctx, task, nodes, config, &wpPageIDMap, maxConcurrency)
		e.logger.Infof("Completed processing depth %d", depth)
	}

	// Handle "after" auto-link mode - suggest and apply links after all content is generated
	// This must happen BEFORE task.Complete() so the task stays in "running" status during linking
	if config.ContentSettings != nil && config.ContentSettings.AutoLinkMode == AutoLinkModeAfter {
		if err := e.autoSuggestAndApplyLinksAfter(ctx, task, config); err != nil {
			e.logger.ErrorWithErr(err, "Failed to auto-suggest/apply links after generation")
			// Continue to complete the task, but linking failed
		}
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
	node, err := e.sitemapSvc.GetNodeWithKeywords(ctx, taskNode.NodeID)
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

	// Fetch link targets if IncludeLinks is enabled OR if AutoLinkMode is "before"
	// (in "before" mode, links were auto-approved before generation started)
	var linkTargets []LinkTarget
	shouldIncludeLinks := config.ContentSettings != nil &&
		(config.ContentSettings.IncludeLinks || config.ContentSettings.AutoLinkMode == AutoLinkModeBefore)
	if shouldIncludeLinks {
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

func (e *Executor) Cancel(taskID string) error {
	taskVal, ok := e.tasks.Load(taskID)
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task := taskVal.(*Task)

	if cancelFunc, ok := e.cancelFuncs.Load(taskID); ok {
		cancelFunc.(context.CancelFunc)()
	}

	go func() {
		time.Sleep(60 * time.Second)

		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()

		e.resetPendingNodesWithContext(cleanupCtx, task)
		task.SetStatus(TaskStatusCancelled)
	}()

	return nil
}

func (e *Executor) resetPendingNodesWithContext(ctx context.Context, task *Task) {
	for _, node := range task.Nodes {
		if node.Status != NodeStatusCompleted && node.Status != NodeStatusSkipped {
			select {
			case <-ctx.Done():
				e.logger.Warn("Cleanup timeout reached, some nodes may not be reset")
				return
			default:
			}

			if err := e.sitemapSvc.UpdateNodeGenerationStatus(ctx, node.NodeID, entities.GenStatusNone, nil); err != nil {
				if !isContextCanceled(err) {
					e.logger.ErrorWithErr(err, fmt.Sprintf("Failed to reset node %d status", node.NodeID))
				}
			}
			if err := e.sitemapSvc.UpdateNodePublishStatus(ctx, node.NodeID, entities.PubStatusNone, nil); err != nil {
				if !isContextCanceled(err) {
					e.logger.ErrorWithErr(err, fmt.Sprintf("Failed to reset node %d publish status", node.NodeID))
				}
			}
		}
	}
}

func isContextCanceled(err error) bool {
	return err != nil && (err == context.Canceled || err.Error() == "context canceled")
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
		if task.Status == TaskStatusRunning {
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

// autoSuggestAndApproveLinksBefore suggests links via AI and auto-approves them for embedding during generation
func (e *Executor) autoSuggestAndApproveLinksBefore(ctx context.Context, task *Task, config GenerationConfig) error {
	if e.linkingSvc == nil {
		return fmt.Errorf("linking service not available")
	}

	task.SetLinkingPhase(LinkingPhaseSuggesting)
	e.emitter.EmitLinkingPhaseStarted(ctx, task.ID, string(LinkingPhaseSuggesting))

	// Get or create a link plan for this sitemap
	plan, err := e.linkingSvc.GetOrCreateActivePlan(ctx, task.SitemapID, task.SiteID)
	if err != nil {
		task.SetLinkingPhase(LinkingPhaseNone)
		return fmt.Errorf("failed to get/create link plan: %w", err)
	}

	// Collect all node IDs to be generated
	nodeIDs := make([]int64, len(task.Nodes))
	for i, node := range task.Nodes {
		nodeIDs[i] = node.NodeID
	}

	// Determine which provider to use for link suggestion
	providerID := config.ProviderID
	if config.ContentSettings != nil && config.ContentSettings.AutoLinkProviderID != nil {
		providerID = *config.ContentSettings.AutoLinkProviderID
	}

	// Get link limits
	maxIncoming := 5 // default
	maxOutgoing := 3 // default
	if config.ContentSettings != nil {
		if config.ContentSettings.MaxIncomingLinks > 0 {
			maxIncoming = config.ContentSettings.MaxIncomingLinks
		}
		if config.ContentSettings.MaxOutgoingLinks > 0 {
			maxOutgoing = config.ContentSettings.MaxOutgoingLinks
		}
	}

	e.logger.Infof("Auto-suggesting links for %d nodes using provider %d", len(nodeIDs), providerID)

	// Suggest links
	err = e.linkingSvc.SuggestLinks(ctx, linking.SuggestLinksConfig{
		PlanID:      plan.ID,
		ProviderID:  providerID,
		PromptID:    config.ContentSettings.AutoLinkSuggestPromptID,
		NodeIDs:     nodeIDs,
		MaxIncoming: maxIncoming,
		MaxOutgoing: maxOutgoing,
	})
	if err != nil {
		task.SetLinkingPhase(LinkingPhaseNone)
		return fmt.Errorf("failed to suggest links: %w", err)
	}

	// Auto-approve all newly created (planned) links
	links, err := e.linkingSvc.GetLinks(ctx, plan.ID)
	if err != nil {
		task.SetLinkingPhase(LinkingPhaseNone)
		return fmt.Errorf("failed to get links: %w", err)
	}

	linksApproved := 0
	for _, link := range links {
		if link.Status == linking.LinkStatusPlanned {
			if err := e.linkingSvc.ApproveLink(ctx, link.ID); err != nil {
				e.logger.ErrorWithErr(err, fmt.Sprintf("Failed to auto-approve link %d", link.ID))
			} else {
				linksApproved++
			}
		}
	}

	task.SetLinkingResults(linksApproved, 0, 0)
	task.SetLinkingPhase(LinkingPhaseCompleted)
	e.emitter.EmitLinkingPhaseCompleted(ctx, task.ID, string(LinkingPhaseSuggesting), linksApproved, 0, 0)

	e.logger.Infof("Auto-approved %d links before generation", linksApproved)
	return nil
}

// autoSuggestAndApplyLinksAfter suggests links via AI and applies them to already-published WordPress content
func (e *Executor) autoSuggestAndApplyLinksAfter(ctx context.Context, task *Task, config GenerationConfig) error {
	if e.linkingSvc == nil {
		return fmt.Errorf("linking service not available")
	}

	// Phase 1: Suggesting
	task.SetLinkingPhase(LinkingPhaseSuggesting)
	e.emitter.EmitLinkingPhaseStarted(ctx, task.ID, string(LinkingPhaseSuggesting))

	plan, err := e.linkingSvc.GetOrCreateActivePlan(ctx, task.SitemapID, task.SiteID)
	if err != nil {
		task.SetLinkingPhase(LinkingPhaseNone)
		return fmt.Errorf("failed to get/create link plan: %w", err)
	}

	// Get successfully generated node IDs (those with WPPageID)
	var nodeIDs []int64
	for _, node := range task.Nodes {
		if node.Status == NodeStatusCompleted && node.WPPageID != nil {
			nodeIDs = append(nodeIDs, node.NodeID)
		}
	}

	if len(nodeIDs) == 0 {
		e.logger.Info("No successfully generated nodes for auto-linking")
		task.SetLinkingPhase(LinkingPhaseCompleted)
		return nil
	}

	providerID := config.ProviderID
	if config.ContentSettings != nil && config.ContentSettings.AutoLinkProviderID != nil {
		providerID = *config.ContentSettings.AutoLinkProviderID
	}

	maxIncoming := 5
	maxOutgoing := 3
	if config.ContentSettings != nil {
		if config.ContentSettings.MaxIncomingLinks > 0 {
			maxIncoming = config.ContentSettings.MaxIncomingLinks
		}
		if config.ContentSettings.MaxOutgoingLinks > 0 {
			maxOutgoing = config.ContentSettings.MaxOutgoingLinks
		}
	}

	e.logger.Infof("Auto-suggesting links for %d generated nodes", len(nodeIDs))

	// Get prompt IDs
	var suggestPromptID *int64
	var applyPromptID int64
	if config.ContentSettings != nil {
		suggestPromptID = config.ContentSettings.AutoLinkSuggestPromptID
		if config.ContentSettings.AutoLinkApplyPromptID != nil {
			applyPromptID = *config.ContentSettings.AutoLinkApplyPromptID
		}
	}

	// Suggest links
	err = e.linkingSvc.SuggestLinks(ctx, linking.SuggestLinksConfig{
		PlanID:      plan.ID,
		ProviderID:  providerID,
		PromptID:    suggestPromptID,
		NodeIDs:     nodeIDs,
		MaxIncoming: maxIncoming,
		MaxOutgoing: maxOutgoing,
	})
	if err != nil {
		task.SetLinkingPhase(LinkingPhaseNone)
		return fmt.Errorf("failed to suggest links: %w", err)
	}

	// Phase 2: Approving and Applying
	task.SetLinkingPhase(LinkingPhaseApplying)
	e.emitter.EmitLinkingPhaseStarted(ctx, task.ID, string(LinkingPhaseApplying))

	// Get all planned links and auto-approve them
	links, err := e.linkingSvc.GetLinks(ctx, plan.ID)
	if err != nil {
		task.SetLinkingPhase(LinkingPhaseNone)
		return fmt.Errorf("failed to get links: %w", err)
	}

	var linkIDsToApply []int64
	for _, link := range links {
		if link.Status == linking.LinkStatusPlanned {
			if err := e.linkingSvc.ApproveLink(ctx, link.ID); err != nil {
				e.logger.ErrorWithErr(err, fmt.Sprintf("Failed to auto-approve link %d", link.ID))
			} else {
				linkIDsToApply = append(linkIDsToApply, link.ID)
			}
		}
	}

	if len(linkIDsToApply) == 0 {
		e.logger.Info("No links to apply after suggestion")
		task.SetLinkingResults(0, 0, 0)
		task.SetLinkingPhase(LinkingPhaseCompleted)
		e.emitter.EmitLinkingPhaseCompleted(ctx, task.ID, string(LinkingPhaseApplying), 0, 0, 0)
		return nil
	}

	e.logger.Infof("Applying %d auto-approved links to WordPress content", len(linkIDsToApply))

	// Apply links to WordPress content
	result, err := e.linkingSvc.ApplyLinks(ctx, plan.ID, linkIDsToApply, providerID, applyPromptID)
	if err != nil {
		task.SetLinkingResults(len(linkIDsToApply), 0, len(linkIDsToApply))
		task.SetLinkingPhase(LinkingPhaseCompleted)
		e.emitter.EmitLinkingPhaseCompleted(ctx, task.ID, string(LinkingPhaseApplying), len(linkIDsToApply), 0, len(linkIDsToApply))
		return fmt.Errorf("failed to apply links: %w", err)
	}

	task.SetLinkingResults(len(linkIDsToApply), result.AppliedLinks, result.FailedLinks)
	task.SetLinkingPhase(LinkingPhaseCompleted)
	e.emitter.EmitLinkingPhaseCompleted(ctx, task.ID, string(LinkingPhaseApplying), len(linkIDsToApply), result.AppliedLinks, result.FailedLinks)

	e.logger.Infof("Applied %d/%d links after generation", result.AppliedLinks, len(linkIDsToApply))
	return nil
}
