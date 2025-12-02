"use client";

import { useState, useEffect, useCallback, Suspense, useRef, useMemo } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
    ReactFlow,
    Background,
    useNodesState,
    useEdgesState,
    addEdge,
    Connection,
    Edge,
    Node,
    BackgroundVariant,
    Panel,
    NodeTypes,
    useReactFlow,
    ReactFlowProvider,
    MarkerType,
    SelectionMode,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "dagre";

import { useApiCall } from "@/hooks/use-api-call";
import { useHotkeys, HotkeyConfig } from "@/hooks/use-hotkeys";
import { sitemapService } from "@/services/sitemaps";
import { siteService } from "@/services/sites";
import {
    Sitemap,
    SitemapNode,
    SitemapWithNodes,
    CreateNodeInput,
} from "@/models/sitemaps";
import { Site } from "@/models/sites";
import { Button } from "@/components/ui/button";
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
import {
    ArrowLeft,
    Plus,
    Save,
    LayoutGrid,
    ListPlus,
} from "lucide-react";
import { SitemapNodeCard } from "@/components/sitemaps/sitemap-node-card";
import { NodeEditDialog } from "@/components/sitemaps/node-edit-dialog";
import { SitemapSidebar } from "@/components/sitemaps/sitemap-sidebar";
import { CanvasControls } from "@/components/sitemaps/canvas-controls";
import { CanvasContextMenu } from "@/components/sitemaps/canvas-context-menu";
import { EdgeContextMenu } from "@/components/sitemaps/edge-context-menu";
import { HotkeysDialog } from "@/components/sitemaps/hotkeys-dialog";
import { BulkCreateDialog } from "@/components/sitemaps/bulk-create-dialog";
import { CommandPalette } from "@/components/sitemaps/command-palette";
import { createNodesFromPaths } from "@/lib/sitemap-utils";
import { cn } from "@/lib/utils";

// Custom node types
const nodeTypes: NodeTypes = {
    sitemapNode: SitemapNodeCard,
};

// Dagre layout configuration - horizontal (left-to-right)
const NODE_WIDTH = 200;
const NODE_HEIGHT = 60;

const getLayoutedElements = (
    nodes: Node[],
    edges: Edge[],
    direction = "LR" // Changed to left-to-right
) => {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setDefaultEdgeLabel(() => ({}));
    dagreGraph.setGraph({ rankdir: direction, nodesep: 40, ranksep: 100 });

    nodes.forEach((node) => {
        dagreGraph.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
    });

    edges.forEach((edge) => {
        dagreGraph.setEdge(edge.source, edge.target);
    });

    dagre.layout(dagreGraph);

    const newNodes = nodes.map((node) => {
        const nodeWithPosition = dagreGraph.node(node.id);
        const newNode = {
            ...node,
            targetPosition: "left",
            sourcePosition: "right",
            position: {
                x: nodeWithPosition.x - NODE_WIDTH / 2,
                y: nodeWithPosition.y - NODE_HEIGHT / 2,
            },
        };

        return newNode;
    });

    return { nodes: newNodes as Node[], edges };
};

// Convert sitemap nodes to React Flow nodes
const convertToFlowNodes = (sitemapNodes: SitemapNode[]): Node[] => {
    return sitemapNodes.map((node) => ({
        id: String(node.id),
        type: "sitemapNode",
        position: {
            x: node.positionX || 0,
            y: node.positionY || 0,
        },
        data: {
            ...node,
            label: node.title,
        },
    }));
};

// Convert sitemap nodes to React Flow edges with smooth bezier curves
const convertToFlowEdges = (sitemapNodes: SitemapNode[]): Edge[] => {
    const edges: Edge[] = [];

    sitemapNodes.forEach((node) => {
        if (node.parentId) {
            edges.push({
                id: `e${node.parentId}-${node.id}`,
                source: String(node.parentId),
                target: String(node.id),
                type: "default", // bezier curve
                animated: false,
                style: { stroke: "#888", strokeWidth: 1.5 },
                markerEnd: {
                    type: MarkerType.ArrowClosed,
                    width: 12,
                    height: 12,
                    color: "#888",
                },
            });
        }
    });

    return edges;
};

function SitemapEditorFlow() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const siteId = Number(searchParams.get("id"));
    const sitemapId = Number(searchParams.get("sitemapId"));
    const reactFlowWrapper = useRef<HTMLDivElement>(null);

    const { execute, isLoading } = useApiCall();
    const { fitView, screenToFlowPosition } = useReactFlow();

    const [site, setSite] = useState<Site | null>(null);
    const [sitemap, setSitemap] = useState<Sitemap | null>(null);
    const [sitemapNodes, setSitemapNodes] = useState<SitemapNode[]>([]);

    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);

    const [selectedNode, setSelectedNode] = useState<SitemapNode | null>(null);
    const [editDialogOpen, setEditDialogOpen] = useState(false);
    const [createDialogOpen, setCreateDialogOpen] = useState(false);
    const [parentNodeId, setParentNodeId] = useState<number | undefined>();
    const [contextMenuNode, setContextMenuNode] = useState<SitemapNode | null>(null);
    const [contextMenuPosition, setContextMenuPosition] = useState<{ x: number; y: number } | null>(null);
    const [edgeContextMenuPosition, setEdgeContextMenuPosition] = useState<{ x: number; y: number } | null>(null);
    const [selectedEdgeId, setSelectedEdgeId] = useState<string | null>(null);
    const [bulkCreateDialogOpen, setBulkCreateDialogOpen] = useState(false);
    const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
    const [hotkeysDialogOpen, setHotkeysDialogOpen] = useState(false);
    const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
    const [sidebarSelectedNodeIds, setSidebarSelectedNodeIds] = useState<Set<number>>(new Set());

    // Refs
    const searchInputRef = useRef<HTMLInputElement>(null);
    const [showUnsavedDialog, setShowUnsavedDialog] = useState(false);
    const [pendingNavigation, setPendingNavigation] = useState<string | null>(null);
    const [initialPositions, setInitialPositions] = useState<Map<string, { x: number; y: number }>>(new Map());

    const loadData = useCallback(async () => {
        const [siteResult, sitemapResult] = await Promise.all([
            execute<Site>(() => siteService.getSite(siteId), {
                errorTitle: "Failed to load site",
            }),
            execute<SitemapWithNodes>(
                () => sitemapService.getSitemapWithNodes(sitemapId),
                {
                    errorTitle: "Failed to load sitemap",
                }
            ),
        ]);

        if (siteResult) setSite(siteResult);
        if (sitemapResult) {
            setSitemap(sitemapResult.sitemap);
            setSitemapNodes(sitemapResult.nodes);

            // Convert to React Flow format
            const flowNodes = convertToFlowNodes(sitemapResult.nodes);
            const flowEdges = convertToFlowEdges(sitemapResult.nodes);

            // Check if ANY node is missing positions - if so, run auto layout
            const hasNodesWithoutPositions = sitemapResult.nodes.some(
                (n) => !n.isRoot && (n.positionX === undefined || n.positionY === undefined || (n.positionX === 0 && n.positionY === 0))
            );

            if (hasNodesWithoutPositions && flowNodes.length > 0) {
                const { nodes: layoutedNodes, edges: layoutedEdges } =
                    getLayoutedElements(flowNodes, flowEdges);
                setNodes(layoutedNodes);
                setEdges(layoutedEdges);
                // Store initial positions
                const positions = new Map<string, { x: number; y: number }>();
                layoutedNodes.forEach((n) => positions.set(n.id, { ...n.position }));
                setInitialPositions(positions);
                // Mark as having unsaved changes since we auto-layouted
                setHasUnsavedChanges(true);
            } else {
                setNodes(flowNodes);
                setEdges(flowEdges);
                // Store initial positions
                const positions = new Map<string, { x: number; y: number }>();
                flowNodes.forEach((n) => positions.set(n.id, { ...n.position }));
                setInitialPositions(positions);
                setHasUnsavedChanges(false);
            }
        }
    }, [execute, setEdges, setNodes, siteId, sitemapId]);

    useEffect(() => {
        if (siteId && sitemapId) {
            loadData();
        }
    }, [siteId, sitemapId, loadData]);

    // Detect position changes
    useEffect(() => {
        if (initialPositions.size === 0) return;

        const hasChanges = nodes.some((node) => {
            const initial = initialPositions.get(node.id);
            if (!initial) return false;
            return Math.abs(node.position.x - initial.x) > 1 || Math.abs(node.position.y - initial.y) > 1;
        });

        setHasUnsavedChanges(hasChanges);
    }, [nodes, initialPositions]);

    // Handle navigation with unsaved changes
    const handleNavigateBack = useCallback(() => {
        const backUrl = `/sites/sitemaps?id=${siteId}`;
        if (hasUnsavedChanges) {
            setPendingNavigation(backUrl);
            setShowUnsavedDialog(true);
        } else {
            router.push(backUrl);
        }
    }, [hasUnsavedChanges, router, siteId]);

    const confirmNavigation = useCallback(() => {
        setShowUnsavedDialog(false);
        if (pendingNavigation) {
            router.push(pendingNavigation);
        }
    }, [pendingNavigation, router]);

    // Track connection source when drag starts
    const connectingNodeId = useRef<string | null>(null);

    const onConnectStart = useCallback((_: React.MouseEvent | React.TouchEvent, { nodeId }: { nodeId: string | null }) => {
        connectingNodeId.current = nodeId;
    }, []);

    const onConnect = useCallback(
        (params: Connection) => setEdges((eds) => addEdge(params, eds)),
        [setEdges]
    );

    // Auto-create node when dragging from handle and dropping on empty space
    const onConnectEnd = useCallback(
        (event: MouseEvent | TouchEvent) => {
            if (!connectingNodeId.current) return;

            const targetIsPane = (event.target as Element).classList.contains("react-flow__pane");
            if (!targetIsPane) return;

            // Get position from event
            const clientX = "changedTouches" in event ? event.changedTouches[0].clientX : event.clientX;
            const clientY = "changedTouches" in event ? event.changedTouches[0].clientY : event.clientY;

            // Open create dialog with the connecting node as parent
            const parentId = Number(connectingNodeId.current);
            setParentNodeId(parentId);
            setCreateDialogOpen(true);

            connectingNodeId.current = null;
        },
        []
    );

    // Get all descendant IDs of a node
    const getAllDescendantIds = useCallback((nodeId: number): number[] => {
        const ids: number[] = [nodeId];
        const children = sitemapNodes.filter((n) => n.parentId === nodeId);
        for (const child of children) {
            ids.push(...getAllDescendantIds(child.id));
        }
        return ids;
    }, [sitemapNodes]);

    // Handle Shift+click on canvas node to select node and all descendants
    const onNodeClick = useCallback(
        (event: React.MouseEvent, node: Node) => {
            if (event.shiftKey) {
                event.preventDefault();
                const nodeId = Number(node.id);
                const allIds = getAllDescendantIds(nodeId);

                // Check if all are already selected
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
            }
            // Regular click handled by React Flow default behavior
        },
        [getAllDescendantIds, nodes, setNodes]
    );

    const onNodeDoubleClick = useCallback(
        (_: React.MouseEvent, node: Node) => {
            const sitemapNode = sitemapNodes.find((n) => n.id === Number(node.id));
            if (sitemapNode) {
                setSelectedNode(sitemapNode);
                setEditDialogOpen(true);
            }
        },
        [sitemapNodes]
    );

    const onNodeContextMenu = useCallback(
        (event: React.MouseEvent, node: Node) => {
            event.preventDefault();
            const sitemapNode = sitemapNodes.find((n) => n.id === Number(node.id));
            if (sitemapNode) {
                setContextMenuNode(sitemapNode);
                setContextMenuPosition({ x: event.clientX, y: event.clientY });
            }
        },
        [sitemapNodes]
    );

    const onPaneContextMenu = useCallback((event: React.MouseEvent) => {
        event.preventDefault();
        setContextMenuNode(null);
        setContextMenuPosition({ x: event.clientX, y: event.clientY });
    }, []);

    const onEdgeContextMenu = useCallback((event: React.MouseEvent, edge: Edge) => {
        event.preventDefault();
        setSelectedEdgeId(edge.id);
        setEdgeContextMenuPosition({ x: event.clientX, y: event.clientY });
    }, []);

    const closeContextMenu = useCallback(() => {
        setContextMenuPosition(null);
        setContextMenuNode(null);
    }, []);

    const closeEdgeContextMenu = useCallback(() => {
        setEdgeContextMenuPosition(null);
        setSelectedEdgeId(null);
    }, []);

    const handleAutoLayout = useCallback(() => {
        const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
            nodes,
            edges
        );
        setNodes(layoutedNodes);
        setEdges(layoutedEdges);
        setHasUnsavedChanges(true);
        setTimeout(() => fitView({ duration: 300, padding: 0.3 }), 50);
    }, [edges, fitView, nodes, setEdges, setNodes]);

    const handleSavePositions = useCallback(async () => {
        for (const node of nodes) {
            const nodeId = Number(node.id);
            await sitemapService.updateNodePositions({
                nodeId,
                positionX: node.position.x,
                positionY: node.position.y,
            });
        }
        // Update initial positions after save
        const positions = new Map<string, { x: number; y: number }>();
        nodes.forEach((n) => positions.set(n.id, { ...n.position }));
        setInitialPositions(positions);
        setHasUnsavedChanges(false);
    }, [nodes]);


    const handleCreateNode = async (input: CreateNodeInput) => {
        const result = await execute<SitemapNode>(
            () => sitemapService.createNode(input),
            {
                successMessage: "Node created successfully",
                showSuccessToast: true,
                errorTitle: "Failed to create node",
            }
        );

        if (result) {
            setCreateDialogOpen(false);
            // loadData will automatically run auto-layout for nodes without positions
            await loadData();
        }
    };

    const handleUpdateNode = async () => {
        setEditDialogOpen(false);
        loadData();
    };

    const handleDeleteNode = async (nodeId: number) => {
        // Find the node to check if it's root
        const node = sitemapNodes.find((n) => n.id === nodeId);
        if (node?.isRoot) {
            return; // Don't delete root node
        }

        await execute(
            () => sitemapService.deleteNode(nodeId),
            {
                successMessage: "Node deleted successfully",
                showSuccessToast: true,
                errorTitle: "Failed to delete node",
            }
        );

        setEditDialogOpen(false);
        loadData();
    };

    const handleDeleteEdge = useCallback(async () => {
        if (!selectedEdgeId) return;

        // Parse edge ID to get source and target node IDs
        // Edge ID format: e{parentId}-{childId}
        const match = selectedEdgeId.match(/^e(\d+)-(\d+)$/);
        if (!match) return;

        const childId = Number(match[2]);

        // Move the child node to have no parent (becomes orphan)
        await execute(
            () => sitemapService.moveNode({ nodeId: childId, newParentId: undefined }),
            {
                successMessage: "Connection removed",
                showSuccessToast: true,
                errorTitle: "Failed to remove connection",
            }
        );

        closeEdgeContextMenu();
        loadData();
    }, [selectedEdgeId, execute, closeEdgeContextMenu, loadData]);

    // Bulk create nodes from paths
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
                // Progress updates happen here if needed
            }
            // loadData will automatically run auto-layout for nodes without positions
            await loadData();
        } catch (error) {
            console.error("Failed to create nodes:", error);
        }
    }, [sitemapId, sitemapNodes, loadData]);

    // Double-Shift detection for Command Palette
    const lastShiftPressRef = useRef<number>(0);
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === "Shift" && !e.repeat) {
                const now = Date.now();
                if (now - lastShiftPressRef.current < 400) {
                    // Double shift detected - open command palette
                    setCommandPaletteOpen(true);
                    lastShiftPressRef.current = 0;
                } else {
                    lastShiftPressRef.current = now;
                }
            }
        };

        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    }, []);

    const handleAddChild = useCallback((parentId: number) => {
        setParentNodeId(parentId);
        setCreateDialogOpen(true);
    }, []);

    const handleAddNode = useCallback((parentId?: number) => {
        // If no parent specified, find the root node
        if (parentId === undefined) {
            const rootNode = sitemapNodes.find((n) => n.isRoot);
            if (rootNode) {
                setParentNodeId(rootNode.id);
            }
        } else {
            setParentNodeId(parentId);
        }
        setCreateDialogOpen(true);
    }, [sitemapNodes]);

    const handleAddOrphanNode = useCallback(() => {
        // Create node without parent (standalone)
        setParentNodeId(undefined);
        setCreateDialogOpen(true);
    }, []);

    const handleEditNode = useCallback((node: SitemapNode) => {
        setSelectedNode(node);
        setEditDialogOpen(true);
    }, []);

    // Handle sidebar multi-select - sync with React Flow selection
    const handleSidebarNodesSelect = useCallback((nodeIds: number[]) => {
        setSidebarSelectedNodeIds(new Set(nodeIds));
        // Also sync selection to React Flow canvas
        setNodes((nds) => nds.map((n) => ({
            ...n,
            selected: nodeIds.includes(Number(n.id)),
        })));
    }, [setNodes]);

    // Sync React Flow canvas selection to sidebar (only when selection changes on canvas)
    const handleSelectionChange = useCallback(({ nodes: selectedNodes }: { nodes: Node[] }) => {
        const selectedIds = selectedNodes.map((n) => Number(n.id));
        setSidebarSelectedNodeIds(new Set(selectedIds));
    }, []);

    // Delete selected nodes
    const handleDeleteSelectedNodes = useCallback(() => {
        const selectedNodeIds = nodes
            .filter((n) => n.selected)
            .map((n) => Number(n.id));

        if (selectedNodeIds.length === 0) return;

        // Filter out root nodes
        const nodesToDelete = selectedNodeIds.filter((id) => {
            const node = sitemapNodes.find((n) => n.id === id);
            return node && !node.isRoot;
        });

        if (nodesToDelete.length > 0) {
            // Delete first selected node for now
            handleDeleteNode(nodesToDelete[0]);
        }
    }, [nodes, sitemapNodes, handleDeleteNode]);

    // Hotkeys configuration
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
            action: () => handleAddNode(),
        },
        {
            key: "Delete",
            description: "Delete selected nodes",
            category: "Nodes",
            action: handleDeleteSelectedNodes,
        },
        {
            key: "Backspace",
            description: "Delete selected nodes",
            category: "Nodes",
            action: handleDeleteSelectedNodes,
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
            key: "Shift",
            description: "Command palette (double press)",
            category: "General",
            action: () => {}, // Handled by separate effect
        },
    ], [hasUnsavedChanges, handleSavePositions, handleAutoLayout, handleAddNode, handleDeleteSelectedNodes, setNodes]);

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
            {/* Header */}
            <div className="border-b px-4 py-3 flex items-center justify-between bg-background shrink-0">
                <div className="flex items-center gap-4">
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={handleNavigateBack}
                    >
                        <ArrowLeft className="h-4 w-4" />
                    </Button>
                    <div>
                        <h1 className="text-lg font-semibold">{sitemap?.name}</h1>
                        <p className="text-sm text-muted-foreground">{site?.name}</p>
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <HotkeysDialog hotkeys={hotkeys} />
                    <Button variant="outline" size="sm" onClick={handleAutoLayout}>
                        <LayoutGrid className="mr-2 h-4 w-4" />
                        Auto Layout
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => handleAddNode()}>
                        <Plus className="mr-2 h-4 w-4" />
                        Add Node
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => setBulkCreateDialogOpen(true)}>
                        <ListPlus className="mr-2 h-4 w-4" />
                        Bulk Create
                    </Button>
                    <Button
                        variant={hasUnsavedChanges ? "default" : "outline"}
                        size="sm"
                        onClick={handleSavePositions}
                        className={cn(
                            hasUnsavedChanges && "animate-pulse"
                        )}
                    >
                        <Save className="mr-2 h-4 w-4" />
                        Save
                    </Button>
                </div>
            </div>

            {/* Main Content */}
            <ResizablePanelGroup direction="horizontal" className="flex-1 min-h-0">
                {/* Sidebar */}
                <ResizablePanel defaultSize={20} minSize={15} maxSize={35}>
                    <SitemapSidebar
                        nodes={sitemapNodes}
                        selectedNodeIds={sidebarSelectedNodeIds}
                        onNodeSelect={(node) => {
                            setSelectedNode(node);
                            setEditDialogOpen(true);
                        }}
                        onNodesSelect={handleSidebarNodesSelect}
                        onAddChild={handleAddChild}
                        searchInputRef={searchInputRef}
                    />
                </ResizablePanel>

                <ResizableHandle withHandle />

                {/* Canvas */}
                <ResizablePanel defaultSize={80}>
                    <div className="h-full" ref={reactFlowWrapper}>
                        <ReactFlow
                            nodes={nodes}
                            edges={edges}
                            onNodesChange={onNodesChange}
                            onEdgesChange={onEdgesChange}
                            onConnectStart={onConnectStart}
                            onConnect={onConnect}
                            onConnectEnd={onConnectEnd}
                            onNodeClick={onNodeClick}
                            onNodeDoubleClick={onNodeDoubleClick}
                            onNodeContextMenu={onNodeContextMenu}
                            onPaneContextMenu={onPaneContextMenu}
                            onEdgeContextMenu={onEdgeContextMenu}
                            onPaneClick={() => {
                                closeContextMenu();
                                closeEdgeContextMenu();
                            }}
                            onSelectionChange={handleSelectionChange}
                            nodeTypes={nodeTypes}
                            fitView
                            fitViewOptions={{ padding: 0.3, maxZoom: 0.8 }}
                            snapToGrid
                            snapGrid={[15, 15]}
                            minZoom={0.1}
                            maxZoom={1.5}
                            defaultViewport={{ x: 0, y: 0, zoom: 0.7 }}
                            selectionMode={SelectionMode.Partial}
                            selectionOnDrag
                            panOnDrag={[1, 2]}
                        >
                            <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
                            <Panel position="bottom-right">
                                <CanvasControls />
                            </Panel>
                        </ReactFlow>
                    </div>
                </ResizablePanel>
            </ResizablePanelGroup>

            {/* Context Menus - rendered outside ReactFlow for proper positioning */}
            <CanvasContextMenu
                selectedNode={contextMenuNode}
                position={contextMenuPosition}
                onClose={closeContextMenu}
                onAddNode={handleAddNode}
                onAddOrphanNode={handleAddOrphanNode}
                onEditNode={handleEditNode}
                onDeleteNode={handleDeleteNode}
                onAddChildNode={handleAddChild}
            />
            <EdgeContextMenu
                position={edgeContextMenuPosition}
                onClose={closeEdgeContextMenu}
                onDeleteEdge={handleDeleteEdge}
            />

            {/* Edit Dialog */}
            {selectedNode && (
                <NodeEditDialog
                    open={editDialogOpen}
                    onOpenChange={setEditDialogOpen}
                    node={selectedNode}
                    onUpdate={handleUpdateNode}
                    onDelete={() => handleDeleteNode(selectedNode.id)}
                    onAddChild={() => handleAddChild(selectedNode.id)}
                />
            )}

            {/* Create Dialog */}
            <NodeEditDialog
                open={createDialogOpen}
                onOpenChange={setCreateDialogOpen}
                sitemapId={sitemapId}
                parentId={parentNodeId}
                onCreate={handleCreateNode}
            />

            {/* Unsaved Changes Dialog */}
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

            {/* Bulk Create Dialog */}
            <BulkCreateDialog
                open={bulkCreateDialogOpen}
                onOpenChange={setBulkCreateDialogOpen}
                onSubmit={handleBulkCreate}
            />

            {/* Command Palette */}
            <CommandPalette
                open={commandPaletteOpen}
                onOpenChange={setCommandPaletteOpen}
                hasUnsavedChanges={hasUnsavedChanges}
                hasSelectedNodes={nodes.some((n) => n.selected)}
                onSave={handleSavePositions}
                onAutoLayout={handleAutoLayout}
                onAddNode={() => handleAddNode()}
                onBulkCreate={() => setBulkCreateDialogOpen(true)}
                onFocusSearch={() => searchInputRef.current?.focus()}
                onDeleteSelected={handleDeleteSelectedNodes}
                onShowHotkeys={() => setHotkeysDialogOpen(true)}
            />

            {/* Hotkeys Dialog (controlled) */}
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
