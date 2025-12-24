import { useCallback, useState, useEffect, useRef } from "react";
import { sitemapService } from "@/services/sitemaps";
import { SitemapNode, GenerationTask, GenerationNodeInfo } from "@/models/sitemaps";
import { EventsOn } from "@/wailsjs/wailsjs/runtime/runtime";

// Event payload types from backend
interface TaskProgressPayload {
    TaskID: string;
    ProcessedNodes: number;
    TotalNodes: number;
    FailedNodes: number;
    SkippedNodes: number;
    CurrentNode?: { NodeID: number; Title: string; Path: string };
}

interface TaskStatusPayload {
    TaskID: string;
    ProcessedNodes?: number;
    TotalNodes?: number;
}

interface NodeStatusPayload {
    TaskID: string;
    NodeID: number;
    Title: string;
    Error?: string;
    ArticleID?: number;
    WPPageID?: number;
    WPPageURL?: string;
}

interface UseSitemapWPOperationsProps {
    siteId: number;
    sitemapId: number;
    execute: <T>(fn: () => Promise<T>, options?: {
        errorTitle?: string;
        successTitle?: string;
        successDescription?: string;
    }) => Promise<T | null>;
    loadData: (preservePositions?: boolean) => Promise<void>;
}

export function useSitemapWPOperations({
    siteId,
    sitemapId,
    execute,
    loadData,
}: UseSitemapWPOperationsProps) {
    const [activeGenerationTask, setActiveGenerationTask] = useState<GenerationTask | null>(null);
    const [contextMenuSelectedNodes, setContextMenuSelectedNodes] = useState<SitemapNode[]>([]);
    const [pageGenerateDialogOpen, setPageGenerateDialogOpen] = useState(false);
    const pendingRefreshRef = useRef<NodeJS.Timeout | null>(null);

    // Helper to update a node's status in the task
    const updateNodeStatus = useCallback((
        nodeId: number,
        status: GenerationNodeInfo["status"],
        extra?: Partial<GenerationNodeInfo>
    ) => {
        setActiveGenerationTask((prev) => {
            if (!prev || !prev.nodes) return prev;
            const updatedNodes = prev.nodes.map((n) =>
                n.nodeId === nodeId ? { ...n, status, ...extra } : n
            );
            return { ...prev, nodes: updatedNodes };
        });
    }, []);

    // Debounced data refresh to avoid too many refreshes during parallel generation
    const scheduleDataRefresh = useCallback(() => {
        if (pendingRefreshRef.current) {
            clearTimeout(pendingRefreshRef.current);
        }
        pendingRefreshRef.current = setTimeout(() => {
            loadData(true);
            pendingRefreshRef.current = null;
        }, 500);
    }, [loadData]);

    // Check for active task on mount and subscribe to events
    useEffect(() => {
        // Initial check for active task
        const checkActiveTask = async () => {
            try {
                const tasks = await sitemapService.listActivePageGenerationTasks();
                const activeTask = tasks.find((t) => t.sitemapId === sitemapId);
                setActiveGenerationTask(activeTask || null);
            } catch {
                // Error handled silently - active task will remain null
            }
        };
        checkActiveTask();

        // Subscribe to page generation events
        const cleanupFns: (() => void)[] = [];

        // Task started event - fetch full task data
        cleanupFns.push(EventsOn("pagegeneration.task.started", async (data: { TaskID: string; SitemapID: number; TotalNodes: number }) => {
            // Only handle if it's for our sitemap
            if (data.SitemapID !== sitemapId) return;

            try {
                const task = await sitemapService.getPageGenerationTask(data.TaskID);
                setActiveGenerationTask(task);
            } catch {
                // Fallback: create minimal task from event data
                setActiveGenerationTask({
                    id: data.TaskID,
                    sitemapId: data.SitemapID,
                    siteId: siteId,
                    status: "running",
                    totalNodes: data.TotalNodes,
                    processedNodes: 0,
                    failedNodes: 0,
                    skippedNodes: 0,
                    startedAt: new Date().toISOString(),
                    nodes: [],
                });
            }
        }));

        // Task progress event
        cleanupFns.push(EventsOn("pagegeneration.task.progress", (data: TaskProgressPayload) => {
            setActiveGenerationTask((prev) => {
                if (!prev) return prev;
                return {
                    ...prev,
                    processedNodes: data.ProcessedNodes,
                    totalNodes: data.TotalNodes,
                    failedNodes: data.FailedNodes,
                    skippedNodes: data.SkippedNodes,
                };
            });
        }));

        // Task paused
        cleanupFns.push(EventsOn("pagegeneration.task.paused", () => {
            setActiveGenerationTask((prev) => prev ? { ...prev, status: "paused" } : null);
        }));

        // Task resumed
        cleanupFns.push(EventsOn("pagegeneration.task.resumed", () => {
            setActiveGenerationTask((prev) => prev ? { ...prev, status: "running" } : null);
        }));

        // Task completed
        cleanupFns.push(EventsOn("pagegeneration.task.completed", () => {
            setActiveGenerationTask((prev) => prev ? { ...prev, status: "completed" } : null);
            loadData(true);
        }));

        // Task failed
        cleanupFns.push(EventsOn("pagegeneration.task.failed", (data: { Error: string }) => {
            setActiveGenerationTask((prev) => prev ? { ...prev, status: "failed", error: data.Error } : null);
            loadData(true);
        }));

        // Task cancelled
        cleanupFns.push(EventsOn("pagegeneration.task.cancelled", () => {
            setActiveGenerationTask((prev) => prev ? { ...prev, status: "cancelled" } : null);
            loadData(true);
        }));

        // Node generating - update status and refresh canvas
        cleanupFns.push(EventsOn("pagegeneration.node.generating", (data: NodeStatusPayload) => {
            updateNodeStatus(data.NodeID, "generating");
            scheduleDataRefresh(); // Refresh canvas to show generating status
        }));

        // Node publishing - update status and refresh canvas
        cleanupFns.push(EventsOn("pagegeneration.node.publishing", (data: NodeStatusPayload) => {
            updateNodeStatus(data.NodeID, "publishing");
            scheduleDataRefresh(); // Refresh canvas to show publishing status
        }));

        // Node completed - triggers canvas refresh
        cleanupFns.push(EventsOn("pagegeneration.node.completed", (data: NodeStatusPayload) => {
            updateNodeStatus(data.NodeID, "completed", {
                articleId: data.ArticleID,
                wpPageId: data.WPPageID,
                wpUrl: data.WPPageURL,
            });
            scheduleDataRefresh();
        }));

        // Node failed
        cleanupFns.push(EventsOn("pagegeneration.node.failed", (data: NodeStatusPayload) => {
            updateNodeStatus(data.NodeID, "failed", { error: data.Error });
            scheduleDataRefresh();
        }));

        // Node skipped
        cleanupFns.push(EventsOn("pagegeneration.node.skipped", (data: NodeStatusPayload) => {
            updateNodeStatus(data.NodeID, "skipped");
        }));

        return () => {
            cleanupFns.forEach((fn) => fn());
            if (pendingRefreshRef.current) {
                clearTimeout(pendingRefreshRef.current);
            }
        };
    }, [sitemapId, loadData, updateNodeStatus, scheduleDataRefresh]);

    const handleSyncFromWP = useCallback(async (nodeIds: number[]) => {
        if (!siteId) return;
        await execute(
            () => sitemapService.syncNodesFromWP({ siteId, nodeIds }),
            {
                successTitle: "Synced from WordPress",
                successDescription: `${nodeIds.length} node(s) updated`,
                errorTitle: "Failed to sync from WordPress",
            }
        );
        await loadData(true);
    }, [siteId, execute, loadData]);

    const handleUpdateToWP = useCallback(async (nodeIds: number[]) => {
        if (!siteId) return;
        await execute(
            () => sitemapService.updateNodesToWP({ siteId, nodeIds }),
            {
                successTitle: "Updated to WordPress",
                successDescription: `${nodeIds.length} node(s) updated`,
                errorTitle: "Failed to update to WordPress",
            }
        );
        await loadData(true);
    }, [siteId, execute, loadData]);

    const handlePublish = useCallback(async (nodeId: number) => {
        if (!siteId) return;
        await execute(
            () => sitemapService.changePublishStatus({ siteId, nodeId, newStatus: "published" }),
            {
                successTitle: "Published",
                successDescription: "Page published successfully",
                errorTitle: "Failed to publish",
            }
        );
        await loadData(true);
    }, [siteId, execute, loadData]);

    const handleUnpublish = useCallback(async (nodeId: number) => {
        if (!siteId) return;
        await execute(
            () => sitemapService.changePublishStatus({ siteId, nodeId, newStatus: "draft" }),
            {
                successTitle: "Unpublished",
                successDescription: "Page changed to draft",
                errorTitle: "Failed to unpublish",
            }
        );
        await loadData(true);
    }, [siteId, execute, loadData]);

    const handleGenerateContent = useCallback((nodes: SitemapNode[]) => {
        setContextMenuSelectedNodes(nodes);
        setPageGenerateDialogOpen(true);
    }, []);

    const handlePauseGeneration = useCallback(async () => {
        if (!activeGenerationTask) return;
        try {
            await sitemapService.pausePageGeneration(activeGenerationTask.id);
            setActiveGenerationTask((prev) => prev ? { ...prev, status: "paused" } : null);
        } catch {
            // Error handled silently
        }
    }, [activeGenerationTask]);

    const handleResumeGeneration = useCallback(async () => {
        if (!activeGenerationTask) return;
        try {
            await sitemapService.resumePageGeneration(activeGenerationTask.id);
            setActiveGenerationTask((prev) => prev ? { ...prev, status: "running" } : null);
        } catch {
            // Error handled silently
        }
    }, [activeGenerationTask]);

    const handleCancelGeneration = useCallback(async () => {
        if (!activeGenerationTask) return;
        try {
            await sitemapService.cancelPageGeneration(activeGenerationTask.id);
            setActiveGenerationTask((prev) => prev ? { ...prev, status: "cancelled" } : null);
        } catch {
            // Error handled silently
        }
    }, [activeGenerationTask]);

    return {
        activeGenerationTask,
        setActiveGenerationTask,
        contextMenuSelectedNodes,
        setContextMenuSelectedNodes,
        pageGenerateDialogOpen,
        setPageGenerateDialogOpen,
        handleSyncFromWP,
        handleUpdateToWP,
        handlePublish,
        handleUnpublish,
        handleGenerateContent,
        handlePauseGeneration,
        handleResumeGeneration,
        handleCancelGeneration,
    };
}
