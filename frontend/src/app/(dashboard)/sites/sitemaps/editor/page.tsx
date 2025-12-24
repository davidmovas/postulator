"use client";

import { useState, useEffect, useCallback, Suspense, useRef, useMemo } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
    ReactFlow,
    Background,
    BackgroundVariant,
    Panel,
    NodeTypes,
    EdgeTypes,
    ReactFlowProvider,
    SelectionMode,
    ConnectionMode,
    useUpdateNodeInternals,
    useReactFlow,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { useHotkeys, HotkeyConfig } from "@/hooks/use-hotkeys";
import { useSitemapEditorData } from "@/hooks/use-sitemap-editor-data";
import { useSitemapNodeOperations } from "@/hooks/use-sitemap-node-operations";
import { useSitemapWPOperations } from "@/hooks/use-sitemap-wp-operations";
import { useSitemapCanvas } from "@/hooks/use-sitemap-canvas";
import { sitemapService } from "@/services/sitemaps";
import { CreateNodeInput } from "@/models/sitemaps";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
    ResizableHandle,
    ResizablePanel,
    ResizablePanelGroup,
} from "@/components/ui/resizable";
import { SitemapNodeCard } from "@/components/sitemaps/sitemap-node-card";
import { LinkEdge } from "@/components/sitemaps/link-edge";
import { LinkContextMenu } from "@/components/sitemaps/link-context-menu";
import { useSitemapLinking } from "@/hooks/use-sitemap-linking";
import { NodeEditDialog } from "@/components/sitemaps/node-edit-dialog";
import { NodeLinksDialog } from "@/components/sitemaps/node-links-dialog";
import { SitemapSidebar } from "@/components/sitemaps/sitemap-sidebar";
import { CanvasControls } from "@/components/sitemaps/canvas-controls";
import { CanvasContextMenu } from "@/components/sitemaps/canvas-context-menu";
import { EdgeContextMenu } from "@/components/sitemaps/edge-context-menu";
import { HotkeysDialog } from "@/components/sitemaps/hotkeys-dialog";
import { BulkCreateDialog } from "@/components/sitemaps/bulk-create-dialog";
import { CommandPalette } from "@/components/sitemaps/command-palette";
import { ImportDialog } from "@/components/sitemaps/import-dialog";
import { ScanDialog } from "@/components/sitemaps/scan-dialog";
import { GenerateDialog } from "@/components/sitemaps/generate-dialog";
import { PageGenerateDialog } from "@/components/sitemaps/page-generate-dialog";
import { SuggestLinksDialog } from "@/components/sitemaps/suggest-links-dialog";
import { ApplyLinksDialog } from "@/components/sitemaps/apply-links-dialog";
import { EditorHeader, EditorMode } from "@/components/sitemaps/editor-header";
import { GenerationProgressPanel } from "@/components/sitemaps/generation-progress-panel";
import { createNodesFromPaths } from "@/lib/sitemap-utils";
import { getLinksLayoutedElements } from "@/lib/sitemap-editor";

const nodeTypes: NodeTypes = {
    sitemapNode: SitemapNodeCard,
};

const edgeTypes: EdgeTypes = {
    linkEdge: LinkEdge,
};

function SitemapEditorFlow() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const siteId = Number(searchParams.get("id"));
    const sitemapId = Number(searchParams.get("sitemapId"));
    const reactFlowWrapper = useRef<HTMLDivElement>(null);
    const searchInputRef = useRef<HTMLInputElement>(null);

    const editorData = useSitemapEditorData({ siteId, sitemapId });
    const {
        site,
        sitemap,
        sitemapNodes,
        nodes,
        edges,
        setNodes,
        setEdges,
        onNodesChange,
        onEdgesChange,
        isLoading,
        hasUnsavedChanges,
        setHasUnsavedChanges,
        history,
        loadData,
        handleSavePositions,
        execute,
    } = editorData;

    const nodeOps = useSitemapNodeOperations({
        sitemapId,
        sitemapNodes,
        nodes,
        execute,
        loadData,
        refreshHistory: history.refreshState,
    });

    const wpOps = useSitemapWPOperations({
        siteId,
        sitemapId,
        execute,
        loadData,
    });

    const canvas = useSitemapCanvas({
        sitemapNodes,
        nodes,
        edges,
        setNodes,
        setEdges,
        execute,
        loadData,
        refreshHistory: history.refreshState,
        getAllDescendantIds: nodeOps.getAllDescendantIds,
        setParentNodeId: nodeOps.setParentNodeId,
        setCreateDialogOpen: nodeOps.setCreateDialogOpen,
        setSelectedNode: nodeOps.setSelectedNode,
        setEditDialogOpen: nodeOps.setEditDialogOpen,
        setHasUnsavedChanges,
    });

    const [editorMode, setEditorMode] = useState<EditorMode>("map");
    const updateNodeInternals = useUpdateNodeInternals();
    const { fitView } = useReactFlow();

    // Update all node internals when editor mode changes (handles change)
    useEffect(() => {
        if (nodes.length > 0) {
            // Small delay to ensure React has re-rendered the nodes with new handles
            const timer = setTimeout(() => {
                const nodeIds = nodes.map((node) => node.id);
                updateNodeInternals(nodeIds);
            }, 50);
            return () => clearTimeout(timer);
        }
    }, [editorMode, nodes, updateNodeInternals]);

    // Separate positions for links mode (doesn't affect map mode positions)
    const [linkModePositions, setLinkModePositions] = useState<Map<string, { x: number; y: number }>>(new Map());
    const [linkModeInitialized, setLinkModeInitialized] = useState(false);

    // Initialize link mode positions from current map positions when first entering links mode
    useEffect(() => {
        if (editorMode === "links" && !linkModeInitialized && nodes.length > 0) {
            const positions = new Map<string, { x: number; y: number }>();
            nodes.forEach((node) => {
                positions.set(node.id, { ...node.position });
            });
            setLinkModePositions(positions);
            setLinkModeInitialized(true);
        }
    }, [editorMode, linkModeInitialized, nodes]);

    // Reset link mode initialization when nodes change significantly (e.g., reload)
    useEffect(() => {
        if (nodes.length === 0) {
            setLinkModeInitialized(false);
        }
    }, [nodes.length]);

    // Linking mode hook - only manages edges and link counts
    const linking = useSitemapLinking({
        sitemapId,
        siteId,
        enabled: editorMode === "links",
    });

    // Get approved links for ApplyLinksDialog
    const approvedLinks = useMemo(() => {
        return linking.links.filter((link) => link.status === "approved");
    }, [linking.links]);

    // Handle node position changes in links mode
    const handleLinkModeNodesChange = useCallback((changes: any[]) => {
        changes.forEach((change: any) => {
            if (change.type === "position" && change.position) {
                setLinkModePositions((prev) => {
                    const newMap = new Map(prev);
                    newMap.set(change.id, change.position);
                    return newMap;
                });
            }
        });
    }, []);

    // Unified handlers that check mode internally - React Flow needs stable handler references
    const handleConnectStart = useCallback(
        (event: React.MouseEvent | React.TouchEvent, params: { nodeId: string | null; handleId: string | null; handleType: "source" | "target" | null }) => {
            if (editorMode === "links") {
                linking.onConnectStart(event as any, params);
            } else {
                canvas.onConnectStart(event as any, params);
            }
        },
        [editorMode, linking.onConnectStart, canvas.onConnectStart]
    );

    const handleConnect = useCallback(
        (connection: any) => {
            if (editorMode === "links") {
                linking.onConnect(connection);
            } else {
                canvas.onConnect(connection);
            }
        },
        [editorMode, linking.onConnect, canvas.onConnect]
    );

    const handleConnectEnd = useCallback(
        (event: MouseEvent | TouchEvent) => {
            if (editorMode === "links") {
                linking.onConnectEnd(event);
            } else {
                canvas.onConnectEnd(event);
            }
        },
        [editorMode, linking.onConnectEnd, canvas.onConnectEnd]
    );

    // Links mode - hovered node for highlighting connections
    const [hoveredNodeId, setHoveredNodeId] = useState<number | null>(null);
    const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null);
    const HOVER_DELAY_MS = 200; // Delay before highlighting to avoid flickering

    // Compute nodes with mode-specific data and positions
    // Key includes editorMode to force re-mount when mode changes (handles need to re-register)
    const displayNodes = useMemo(() => {
        // Build sets of related nodes for highlighting
        const incomingSources = new Set<number>(); // Nodes that link TO the hovered node
        const outgoingTargets = new Set<number>(); // Nodes that hovered node links TO

        if (hoveredNodeId && linking.linkGraph) {
            linking.linkGraph.edges.forEach((edge) => {
                if (edge.targetNodeId === hoveredNodeId) {
                    incomingSources.add(edge.sourceNodeId);
                }
                if (edge.sourceNodeId === hoveredNodeId) {
                    outgoingTargets.add(edge.targetNodeId);
                }
            });
        }

        return nodes.map((node) => {
            const nodeId = Number(node.id);
            const linkCounts = linking.linkCountsMap.get(nodeId);
            // Use link mode positions if in links mode and initialized
            const position = editorMode === "links" && linkModeInitialized
                ? (linkModePositions.get(node.id) || node.position)
                : node.position;

            // Highlight state for linking mode
            const isHovered = hoveredNodeId === nodeId;
            const isIncomingSource = incomingSources.has(nodeId); // This node links TO hovered
            const isOutgoingTarget = outgoingTargets.has(nodeId); // Hovered links TO this node
            // Dim nodes that are not related when there's a hovered node
            const isDimmed = editorMode === "links" && hoveredNodeId !== null &&
                !isHovered && !isIncomingSource && !isOutgoingTarget;

            return {
                ...node,
                position,
                draggable: true, // Allow dragging in both modes
                data: {
                    ...node.data,
                    editorMode,
                    siteUrl: site?.url,
                    outgoingLinkCount: linkCounts?.outgoing || 0,
                    incomingLinkCount: linkCounts?.incoming || 0,
                    // Highlight info for linking mode
                    isHovered,
                    isIncomingSource,
                    isOutgoingTarget,
                    isDimmed,
                },
            };
        });
    }, [nodes, editorMode, linking.linkCountsMap, linking.linkGraph, site?.url, linkModePositions, linkModeInitialized, hoveredNodeId]);

    // Hierarchy edges (серые пунктирные) для режима links - показывают структуру сайтмапы
    const hierarchyEdges = useMemo(() => {
        if (editorMode !== "links") return [];
        return edges.map((edge) => ({
            ...edge,
            id: `hierarchy-${edge.id}`,
            sourceHandle: "bottom", // Parent connects from bottom
            targetHandle: "top",    // Child connects to top
            type: "default",
            animated: false,
            selectable: false,
            focusable: false,
            style: { stroke: "#9ca3af", strokeWidth: 1.5, strokeDasharray: "4,4", opacity: 0.6 },
            markerEnd: {
                type: "arrowclosed" as const,
                width: 10,
                height: 10,
                color: "#9ca3af",
            },
        }));
    }, [editorMode, edges]);

    // Выбираем какие edges показывать в зависимости от режима
    const displayEdges = useMemo(() => {
        if (editorMode === "links") {
            // Apply highlight styles to link edges when related to hovered node
            const highlightedLinkEdges = linking.linkEdges.map((edge) => {
                if (!hoveredNodeId) return edge;

                const sourceId = Number(edge.source);
                const targetId = Number(edge.target);
                const isIncoming = targetId === hoveredNodeId; // This edge points TO hovered
                const isOutgoing = sourceId === hoveredNodeId; // This edge FROM hovered

                if (isIncoming) {
                    // Cyan - edge coming into hovered node (links TO hovered)
                    return {
                        ...edge,
                        style: { ...edge.style, stroke: "#22d3ee", strokeWidth: 3 },
                        markerEnd: { ...edge.markerEnd, color: "#22d3ee" },
                        zIndex: 10,
                    };
                }
                if (isOutgoing) {
                    // Emerald - edge going out from hovered node (hovered links TO)
                    return {
                        ...edge,
                        style: { ...edge.style, stroke: "#34d399", strokeWidth: 3 },
                        markerEnd: { ...edge.markerEnd, color: "#34d399" },
                        zIndex: 10,
                    };
                }
                // Dim non-related edges when hovering
                return {
                    ...edge,
                    style: { ...edge.style, opacity: 0.5 },
                };
            });

            // В режиме links показываем и hierarchy (серые) и link edges (цветные)
            return [...hierarchyEdges, ...highlightedLinkEdges];
        }
        return edges;
    }, [editorMode, hierarchyEdges, linking.linkEdges, edges, hoveredNodeId]);

    const [bulkCreateDialogOpen, setBulkCreateDialogOpen] = useState(false);
    const [importDialogOpen, setImportDialogOpen] = useState(false);
    const [scanDialogOpen, setScanDialogOpen] = useState(false);
    const [generateDialogOpen, setGenerateDialogOpen] = useState(false);
    const [suggestLinksDialogOpen, setSuggestLinksDialogOpen] = useState(false);
    const [applyLinksDialogOpen, setApplyLinksDialogOpen] = useState(false);
    const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
    const [hotkeysDialogOpen, setHotkeysDialogOpen] = useState(false);
    const [showUnsavedDialog, setShowUnsavedDialog] = useState(false);
    const [pendingNavigation, setPendingNavigation] = useState<string | null>(null);
    // Links mode - node links dialog
    const [nodeLinksDialogOpen, setNodeLinksDialogOpen] = useState(false);
    const [selectedNodeForLinks, setSelectedNodeForLinks] = useState<number | null>(null);

    const handleUndo = useCallback(async () => {
        const success = await history.undo();
        if (success) {
            await loadData(true);
        }
    }, [history, loadData]);

    const handleRedo = useCallback(async () => {
        const success = await history.redo();
        if (success) {
            await loadData(true);
        }
    }, [history, loadData]);

    // Auto layout for links mode - considers both hierarchy and link edges
    const handleLinkModeAutoLayout = useCallback(() => {
        if (nodes.length === 0) return;

        // Pass both hierarchy edges and link edges for smart layout
        const newPositions = getLinksLayoutedElements(nodes, edges, linking.linkEdges);
        setLinkModePositions(newPositions);
    }, [nodes, edges, linking.linkEdges]);

    // Combined auto layout handler that works for both modes
    const handleAutoLayout = useCallback(() => {
        if (editorMode === "links") {
            handleLinkModeAutoLayout();
        } else {
            canvas.handleAutoLayout();
        }
    }, [editorMode, handleLinkModeAutoLayout, canvas]);

    // Handler for node double-click in links mode - opens links dialog
    const handleLinksNodeDoubleClick = useCallback((event: React.MouseEvent, node: any) => {
        const nodeId = Number(node.id);
        setSelectedNodeForLinks(nodeId);
        setNodeLinksDialogOpen(true);
    }, []);

    // Handlers for node hover in links mode - for highlighting connections
    // With delay to avoid flickering and skip nodes without any links
    const handleNodeMouseEnter = useCallback((event: React.MouseEvent, node: any) => {
        if (editorMode !== "links") return;

        // Clear any pending timeout
        if (hoverTimeoutRef.current) {
            clearTimeout(hoverTimeoutRef.current);
            hoverTimeoutRef.current = null;
        }

        const nodeId = Number(node.id);
        const linkCounts = linking.linkCountsMap.get(nodeId);

        // Skip highlighting for nodes that have no links at all
        if (!linkCounts || (linkCounts.incoming === 0 && linkCounts.outgoing === 0)) {
            return;
        }

        // Apply delay before highlighting
        hoverTimeoutRef.current = setTimeout(() => {
            setHoveredNodeId(nodeId);
        }, HOVER_DELAY_MS);
    }, [editorMode, linking.linkCountsMap]);

    const handleNodeMouseLeave = useCallback(() => {
        if (editorMode !== "links") return;

        // Clear pending timeout if mouse leaves before delay
        if (hoverTimeoutRef.current) {
            clearTimeout(hoverTimeoutRef.current);
            hoverTimeoutRef.current = null;
        }

        setHoveredNodeId(null);
    }, [editorMode]);

    // Clean up timeout on unmount or mode change
    useEffect(() => {
        return () => {
            if (hoverTimeoutRef.current) {
                clearTimeout(hoverTimeoutRef.current);
            }
        };
    }, [editorMode]);

    // Node click handler for links mode - supports multi-select and toggle
    const handleLinksNodeClick = useCallback((event: React.MouseEvent, node: any) => {
        const isShiftClick = event.shiftKey;
        const isCtrlClick = event.ctrlKey || event.metaKey;

        if (isShiftClick) {
            // Shift+Click: Select node and all its descendants (like map mode)
            event.preventDefault();
            const nodeId = Number(node.id);
            const allIds = nodeOps.getAllDescendantIds(nodeId);

            // Check if all descendants are already selected
            const allSelected = allIds.every((id) =>
                nodes.find((n) => n.id === String(id))?.selected
            );

            if (allSelected) {
                // Deselect all descendants
                setNodes((nds) => nds.map((n) => ({
                    ...n,
                    selected: allIds.includes(Number(n.id)) ? false : n.selected,
                })));
            } else {
                // Select all descendants
                setNodes((nds) => nds.map((n) => ({
                    ...n,
                    selected: allIds.includes(Number(n.id)) ? true : n.selected,
                })));
            }
        } else if (isCtrlClick) {
            // Ctrl+Click: Toggle single node selection
            setNodes((nds) =>
                nds.map((n) => {
                    if (n.id === node.id) {
                        return { ...n, selected: !n.selected };
                    }
                    return n;
                })
            );
        } else {
            // Regular click - check if this node is already selected
            setNodes((nds) => {
                const clickedNode = nds.find((n) => n.id === node.id);
                const isAlreadySelected = clickedNode?.selected;
                const hasOtherSelected = nds.some((n) => n.id !== node.id && n.selected);

                // If clicking on already selected node and no other nodes selected - deselect
                // If clicking on already selected node but others are selected - select only this
                // If clicking on unselected node - select only this
                if (isAlreadySelected && !hasOtherSelected) {
                    // Deselect the node
                    return nds.map((n) => ({ ...n, selected: false }));
                } else {
                    // Select only this node
                    return nds.map((n) => ({ ...n, selected: n.id === node.id }));
                }
            });
        }
    }, [setNodes, nodes, nodeOps]);

    // Handler for "Go to node" from links dialog - centers view on node
    const handleGoToNode = useCallback((nodeId: number) => {
        setNodeLinksDialogOpen(false);
        // Small delay to let dialog close
        setTimeout(() => {
            fitView({
                nodes: [{ id: String(nodeId) }],
                duration: 300,
                padding: 0.5,
                maxZoom: 1,
            });
        }, 100);
    }, [fitView]);

    // Get the selected node for links dialog
    const selectedNodeForLinksData = useMemo(() => {
        if (!selectedNodeForLinks) return null;
        return sitemapNodes.find((n) => n.id === selectedNodeForLinks) || null;
    }, [selectedNodeForLinks, sitemapNodes]);

    const handleNavigateBack = useCallback(() => {
        if (wpOps.activeGenerationTask &&
            (wpOps.activeGenerationTask.status === "running" || wpOps.activeGenerationTask.status === "paused")) {
            return;
        }
        const backUrl = `/sites/sitemaps?id=${siteId}`;
        if (hasUnsavedChanges) {
            setPendingNavigation(backUrl);
            setShowUnsavedDialog(true);
        } else {
            router.push(backUrl);
        }
    }, [hasUnsavedChanges, router, siteId, wpOps.activeGenerationTask]);

    const confirmNavigation = useCallback(() => {
        setShowUnsavedDialog(false);
        if (pendingNavigation) {
            router.push(pendingNavigation);
        }
    }, [pendingNavigation, router]);

    const handleBulkCreate = useCallback(async (paths: string[]) => {
        if (!sitemapId) return;

        const createNodeFn = async (input: CreateNodeInput) => {
            const result = await sitemapService.createNode(input);
            return result;
        };

        try {
            for await (const progress of createNodesFromPaths(
                paths,
                sitemapId,
                sitemapNodes,
                createNodeFn
            )) {
            }
            await loadData();
        } catch (error) {
            console.error("Failed to create nodes:", error);
        }
    }, [sitemapId, sitemapNodes, loadData]);

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === "Tab" && !e.repeat) {
                const target = e.target as HTMLElement;
                if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable) {
                    return;
                }
                e.preventDefault();
                setCommandPaletteOpen(true);
            }
        };

        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    }, []);

    const hotkeys = useMemo<HotkeyConfig[]>(() => [
        {
            key: "s",
            ctrl: true,
            description: "Save layout",
            category: "General",
            action: () => {
                if (hasUnsavedChanges) handleSavePositions();
            },
        },
        {
            key: "l",
            ctrl: true,
            description: "Auto layout",
            category: "Layout",
            action: handleAutoLayout,
        },
        {
            key: "n",
            ctrl: true,
            description: "Add new node",
            category: "Nodes",
            action: () => nodeOps.handleAddNode(),
        },
        {
            key: "Delete",
            description: "Delete selected nodes",
            category: "Nodes",
            action: nodeOps.handleDeleteSelectedNodes,
        },
        {
            key: "Backspace",
            description: "Delete selected nodes",
            category: "Nodes",
            action: nodeOps.handleDeleteSelectedNodes,
        },
        {
            key: "Escape",
            description: "Deselect all",
            category: "Selection",
            action: () => {
                setNodes((nds) => nds.map((n) => ({ ...n, selected: false })));
            },
        },
        {
            key: "f",
            shift: true,
            description: "Focus search",
            category: "Navigation",
            action: () => {
                searchInputRef.current?.focus();
            },
        },
        {
            key: "b",
            ctrl: true,
            description: "Bulk create nodes",
            category: "Nodes",
            action: () => setBulkCreateDialogOpen(true),
        },
        {
            key: "i",
            ctrl: true,
            description: "Import from file",
            category: "Nodes",
            action: () => setImportDialogOpen(true),
        },
        {
            key: "k",
            ctrl: true,
            description: "Scan from WordPress",
            category: "Nodes",
            action: () => setScanDialogOpen(true),
        },
        {
            key: "z",
            ctrl: true,
            description: "Undo",
            category: "History",
            action: handleUndo,
        },
        {
            key: "z",
            ctrl: true,
            shift: true,
            description: "Redo",
            category: "History",
            action: handleRedo,
        },
        {
            key: "g",
            ctrl: true,
            description: "Generate page content",
            category: "Content",
            action: () => wpOps.setPageGenerateDialogOpen(true),
        },
        {
            key: "Tab",
            description: "Command palette",
            category: "General",
            action: () => {},
        },
    ], [hasUnsavedChanges, handleSavePositions, handleAutoLayout, nodeOps, setNodes, handleUndo, handleRedo, wpOps]);

    useHotkeys(hotkeys);

    if (isLoading && !sitemap) {
        return (
            <div className="h-screen flex items-center justify-center">
                <div className="text-muted-foreground">Loading sitemap...</div>
            </div>
        );
    }

    return (
        <div className="h-full flex flex-col overflow-hidden">
            <EditorHeader
                site={site}
                sitemap={sitemap}
                hasUnsavedChanges={hasUnsavedChanges}
                canUndo={history.canUndo}
                canRedo={history.canRedo}
                activeGenerationTask={wpOps.activeGenerationTask}
                hotkeys={hotkeys}
                editorMode={editorMode}
                onModeChange={setEditorMode}
                onNavigateBack={handleNavigateBack}
                onUndo={handleUndo}
                onRedo={handleRedo}
                onAutoLayout={handleAutoLayout}
                onSave={handleSavePositions}
                // Map mode actions
                onAddNode={() => nodeOps.handleAddNode()}
                onBulkCreate={() => setBulkCreateDialogOpen(true)}
                onImport={() => setImportDialogOpen(true)}
                onScan={() => setScanDialogOpen(true)}
                onGenerateStructure={() => setGenerateDialogOpen(true)}
                onGeneratePages={() => wpOps.setPageGenerateDialogOpen(true)}
                // Links mode actions
                onSuggestLinks={() => setSuggestLinksDialogOpen(true)}
                onApplyLinks={() => setApplyLinksDialogOpen(true)}
                onApproveAllLinks={linking.approveAllLinks}
                onRejectAllLinks={linking.rejectAllLinks}
                onClearAILinks={linking.clearAILinks}
                linkStats={linking.linkStats}
            />

            {wpOps.activeGenerationTask && (
                <GenerationProgressPanel
                    task={wpOps.activeGenerationTask}
                    onPause={wpOps.handlePauseGeneration}
                    onResume={wpOps.handleResumeGeneration}
                    onCancel={wpOps.handleCancelGeneration}
                />
            )}

            <ResizablePanelGroup direction="horizontal" className="flex-1 min-h-0">
                <ResizablePanel defaultSize={20} minSize={15} maxSize={35} className="overflow-hidden">
                    <SitemapSidebar
                        nodes={sitemapNodes}
                        selectedNodeIds={canvas.sidebarSelectedNodeIds}
                        onNodeSelect={(node) => {
                            nodeOps.setSelectedNode(node);
                            nodeOps.setEditDialogOpen(true);
                        }}
                        onNodesSelect={canvas.handleSidebarNodesSelect}
                        onAddChild={nodeOps.handleAddChild}
                        searchInputRef={searchInputRef}
                    />
                </ResizablePanel>

                <ResizableHandle withHandle />

                <ResizablePanel defaultSize={80}>
                    <div className="h-full" ref={reactFlowWrapper}>
                        <ReactFlow
                            nodes={displayNodes}
                            edges={displayEdges}
                            onNodesChange={editorMode === "map" ? onNodesChange : handleLinkModeNodesChange}
                            onEdgesChange={editorMode === "map" ? onEdgesChange : undefined}
                            onConnectStart={handleConnectStart}
                            onConnect={handleConnect}
                            onConnectEnd={handleConnectEnd}
                            onNodeClick={editorMode === "map" ? canvas.onNodeClick : handleLinksNodeClick}
                            onNodeDoubleClick={editorMode === "map" ? canvas.onNodeDoubleClick : handleLinksNodeDoubleClick}
                            onNodeContextMenu={editorMode === "map" ? canvas.onNodeContextMenu : undefined}
                            onPaneContextMenu={editorMode === "map" ? canvas.onPaneContextMenu : undefined}
                            onEdgeContextMenu={editorMode === "map" ? canvas.onEdgeContextMenu : linking.onEdgeContextMenu}
                            onPaneClick={() => {
                                canvas.closeContextMenu();
                                canvas.closeEdgeContextMenu();
                                linking.closeLinkContextMenu();
                            }}
                            onSelectionChange={canvas.handleSelectionChange}
                            onNodeMouseEnter={handleNodeMouseEnter}
                            onNodeMouseLeave={handleNodeMouseLeave}
                            nodeTypes={nodeTypes}
                            edgeTypes={edgeTypes}
                            fitView
                            fitViewOptions={{ padding: 0.3, maxZoom: 0.8 }}
                            snapToGrid
                            snapGrid={[15, 15]}
                            minZoom={0.1}
                            maxZoom={1.5}
                            defaultViewport={{ x: 0, y: 0, zoom: 0.7 }}
                            selectionMode={SelectionMode.Partial}
                            selectionOnDrag={false}
                            panOnDrag
                            connectionMode={ConnectionMode.Strict}
                        >
                            <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
                            <Panel position="bottom-right">
                                <CanvasControls />
                            </Panel>
                        </ReactFlow>
                    </div>
                </ResizablePanel>
            </ResizablePanelGroup>

            <CanvasContextMenu
                selectedNode={canvas.contextMenuNode}
                selectedNodes={nodeOps.getSelectedSitemapNodes()}
                position={canvas.contextMenuPosition}
                siteUrl={site?.url}
                onClose={canvas.closeContextMenu}
                onAddNode={nodeOps.handleAddNode}
                onAddOrphanNode={nodeOps.handleAddOrphanNode}
                onEditNode={nodeOps.handleEditNode}
                onDeleteNode={nodeOps.handleDeleteNode}
                onAddChildNode={nodeOps.handleAddChild}
                onSyncFromWP={wpOps.handleSyncFromWP}
                onUpdateToWP={wpOps.handleUpdateToWP}
                onGenerateContent={wpOps.handleGenerateContent}
                onPublish={wpOps.handlePublish}
                onUnpublish={wpOps.handleUnpublish}
            />
            <EdgeContextMenu
                position={canvas.edgeContextMenuPosition}
                onClose={canvas.closeEdgeContextMenu}
                onDeleteEdge={canvas.handleDeleteEdge}
            />

            {/* Link context menu for links mode */}
            {linking.linkContextMenu && (
                <LinkContextMenu
                    linkId={linking.linkContextMenu.linkId}
                    position={linking.linkContextMenu.position}
                    status={linking.linkContextMenu.status}
                    onApprove={linking.approveLink}
                    onReject={linking.rejectLink}
                    onRemove={linking.removeLink}
                    onClose={linking.closeLinkContextMenu}
                />
            )}

            {/* Node links dialog for links mode */}
            <NodeLinksDialog
                open={nodeLinksDialogOpen}
                onOpenChange={setNodeLinksDialogOpen}
                node={selectedNodeForLinksData}
                linkGraph={linking.linkGraph}
                sitemapNodes={sitemapNodes}
                onApproveLink={linking.approveLink}
                onRejectLink={linking.rejectLink}
                onRemoveLink={linking.removeLink}
                onGoToNode={handleGoToNode}
            />

            {nodeOps.selectedNode && (
                <NodeEditDialog
                    open={nodeOps.editDialogOpen}
                    onOpenChange={nodeOps.setEditDialogOpen}
                    node={nodeOps.selectedNode}
                    onUpdate={nodeOps.handleUpdateNode}
                    onDelete={() => nodeOps.handleDeleteNode(nodeOps.selectedNode!.id)}
                    onAddChild={() => nodeOps.handleAddChild(nodeOps.selectedNode!.id)}
                />
            )}

            <NodeEditDialog
                open={nodeOps.createDialogOpen}
                onOpenChange={nodeOps.setCreateDialogOpen}
                sitemapId={sitemapId}
                parentId={nodeOps.parentNodeId}
                onCreate={nodeOps.handleCreateNode}
            />

            <AlertDialog open={showUnsavedDialog} onOpenChange={setShowUnsavedDialog}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Unsaved Changes</AlertDialogTitle>
                        <AlertDialogDescription>
                            You have unsaved layout changes. Are you sure you want to leave?
                            Your changes will be lost.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Stay</AlertDialogCancel>
                        <AlertDialogAction onClick={confirmNavigation}>
                            Leave Without Saving
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>

            <BulkCreateDialog
                open={bulkCreateDialogOpen}
                onOpenChange={setBulkCreateDialogOpen}
                onSubmit={handleBulkCreate}
            />

            <ImportDialog
                open={importDialogOpen}
                onOpenChange={setImportDialogOpen}
                sitemapId={sitemapId}
                onSuccess={() => loadData()}
            />

            <ScanDialog
                open={scanDialogOpen}
                onOpenChange={setScanDialogOpen}
                mode="add"
                sitemapId={sitemapId}
                onSuccess={() => loadData()}
            />

            <GenerateDialog
                open={generateDialogOpen}
                onOpenChange={setGenerateDialogOpen}
                mode="add"
                sitemapId={sitemapId}
                onSuccess={() => loadData()}
            />

            <PageGenerateDialog
                open={wpOps.pageGenerateDialogOpen}
                onOpenChange={(open) => {
                    wpOps.setPageGenerateDialogOpen(open);
                    if (!open) {
                        wpOps.setContextMenuSelectedNodes([]);
                    }
                }}
                sitemapId={sitemapId}
                selectedNodes={
                    wpOps.contextMenuSelectedNodes.length > 0
                        ? wpOps.contextMenuSelectedNodes
                        : nodeOps.getSelectedSitemapNodes()
                }
                allNodes={sitemapNodes}
                onSuccess={() => loadData()}
                onTaskStarted={wpOps.setActiveGenerationTask}
                activeTask={wpOps.activeGenerationTask}
                hasApprovedLinks={linking.linkStats.approved > 0}
            />

            {linking.plan && (
                <SuggestLinksDialog
                    open={suggestLinksDialogOpen}
                    onOpenChange={setSuggestLinksDialogOpen}
                    planId={linking.plan.id}
                    selectedNodes={nodeOps.getSelectedSitemapNodes()}
                    allNodes={sitemapNodes}
                    onSuccess={() => linking.loadLinkingData()}
                />
            )}

            {linking.plan && (
                <ApplyLinksDialog
                    open={applyLinksDialogOpen}
                    onOpenChange={setApplyLinksDialogOpen}
                    planId={linking.plan.id}
                    approvedLinks={approvedLinks}
                    selectedNodes={nodeOps.getSelectedSitemapNodes()}
                    sitemapNodes={sitemapNodes}
                    onSuccess={() => linking.loadLinkingData()}
                />
            )}

            <CommandPalette
                open={commandPaletteOpen}
                onOpenChange={setCommandPaletteOpen}
                hasUnsavedChanges={hasUnsavedChanges}
                hasSelectedNodes={nodes.some((n) => n.selected)}
                onSave={handleSavePositions}
                onAutoLayout={handleAutoLayout}
                onAddNode={() => nodeOps.handleAddNode()}
                onBulkCreate={() => setBulkCreateDialogOpen(true)}
                onImport={() => setImportDialogOpen(true)}
                onScan={() => setScanDialogOpen(true)}
                onGeneratePages={() => wpOps.setPageGenerateDialogOpen(true)}
                onFocusSearch={() => searchInputRef.current?.focus()}
                onDeleteSelected={nodeOps.handleDeleteSelectedNodes}
                onShowHotkeys={() => setHotkeysDialogOpen(true)}
            />

            <HotkeysDialog
                hotkeys={hotkeys}
                open={hotkeysDialogOpen}
                onOpenChange={setHotkeysDialogOpen}
            />
        </div>
    );
}

function SitemapEditorContent() {
    return (
        <ReactFlowProvider>
            <SitemapEditorFlow />
        </ReactFlowProvider>
    );
}

export default function SitemapEditorPage() {
    return (
        <Suspense
            fallback={
                <div className="h-screen flex items-center justify-center">
                    <div className="text-muted-foreground">Loading...</div>
                </div>
            }
        >
            <SitemapEditorContent />
        </Suspense>
    );
}
