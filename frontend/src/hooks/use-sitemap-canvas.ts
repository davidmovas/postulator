import { useCallback, useState, useRef } from "react";
import { Node, Edge, Connection, useReactFlow, OnConnectStart, OnConnectEnd } from "@xyflow/react";
import { sitemapService } from "@/services/sitemaps";
import { SitemapNode } from "@/models/sitemaps";
import { getLayoutedElements } from "@/lib/sitemap-editor";

interface UseSitemapCanvasProps {
    sitemapNodes: SitemapNode[];
    nodes: Node[];
    edges: Edge[];
    setNodes: React.Dispatch<React.SetStateAction<Node[]>>;
    setEdges: React.Dispatch<React.SetStateAction<Edge[]>>;
    execute: <T>(fn: () => Promise<T>, options?: { errorTitle?: string }) => Promise<T | null>;
    loadData: (preservePositions?: boolean) => Promise<void>;
    refreshHistory: () => Promise<void>;
    getAllDescendantIds: (nodeId: number) => number[];
    setParentNodeId: (id: number | undefined) => void;
    setCreateDialogOpen: (open: boolean) => void;
    setSelectedNode: (node: SitemapNode | null) => void;
    setEditDialogOpen: (open: boolean) => void;
    setHasUnsavedChanges: (value: boolean) => void;
}

export function useSitemapCanvas({
    sitemapNodes,
    nodes,
    edges,
    setNodes,
    setEdges,
    execute,
    loadData,
    refreshHistory,
    getAllDescendantIds,
    setParentNodeId,
    setCreateDialogOpen,
    setSelectedNode,
    setEditDialogOpen,
    setHasUnsavedChanges,
}: UseSitemapCanvasProps) {
    const { fitView } = useReactFlow();
    const connectingNodeId = useRef<string | null>(null);

    const [contextMenuNode, setContextMenuNode] = useState<SitemapNode | null>(null);
    const [contextMenuPosition, setContextMenuPosition] = useState<{ x: number; y: number } | null>(null);
    const [edgeContextMenuPosition, setEdgeContextMenuPosition] = useState<{ x: number; y: number } | null>(null);
    const [selectedEdgeId, setSelectedEdgeId] = useState<string | null>(null);
    const [sidebarSelectedNodeIds, setSidebarSelectedNodeIds] = useState<Set<number>>(new Set());

    const onConnectStart: OnConnectStart = useCallback((_event, { nodeId }) => {
        connectingNodeId.current = nodeId;
    }, []);

    const onConnect = useCallback(async (params: Connection) => {
        if (!params.source || !params.target) return;

        const sourceId = Number(params.source);
        const targetId = Number(params.target);

        await execute(
            () => sitemapService.moveNode({ nodeId: targetId, newParentId: sourceId }),
            { errorTitle: "Failed to create connection" }
        );

        await loadData(true);
        await refreshHistory();
    }, [execute, loadData, refreshHistory]);

    const onConnectEnd: OnConnectEnd = useCallback(async (event, _connectionState) => {
        if (!connectingNodeId.current) return;

        const target = event.target as Element;
        const sourceId = Number(connectingNodeId.current);

        const nodeElement = target.closest('.react-flow__node');
        if (nodeElement) {
            const targetNodeId = nodeElement.getAttribute('data-id');
            if (targetNodeId && targetNodeId !== connectingNodeId.current) {
                const targetId = Number(targetNodeId);

                await execute(
                    () => sitemapService.moveNode({ nodeId: targetId, newParentId: sourceId }),
                    { errorTitle: "Failed to create connection" }
                );

                await loadData(true);
                await refreshHistory();
            }
            connectingNodeId.current = null;
            return;
        }

        const targetIsPane = target.classList.contains("react-flow__pane");
        if (targetIsPane) {
            setParentNodeId(sourceId);
            setCreateDialogOpen(true);
        }

        connectingNodeId.current = null;
    }, [execute, loadData, refreshHistory, setParentNodeId, setCreateDialogOpen]);

    const onNodeClick = useCallback((event: React.MouseEvent, node: Node) => {
        if (event.shiftKey) {
            event.preventDefault();
            const nodeId = Number(node.id);
            const allIds = getAllDescendantIds(nodeId);

            const allSelected = allIds.every((id) =>
                nodes.find((n) => n.id === String(id))?.selected
            );

            if (allSelected) {
                setNodes((nds) => nds.map((n) => ({
                    ...n,
                    selected: allIds.includes(Number(n.id)) ? false : n.selected,
                })));
            } else {
                setNodes((nds) => nds.map((n) => ({
                    ...n,
                    selected: allIds.includes(Number(n.id)) ? true : n.selected,
                })));
            }
        }
    }, [getAllDescendantIds, nodes, setNodes]);

    const onNodeDoubleClick = useCallback((_: React.MouseEvent, node: Node) => {
        const sitemapNode = sitemapNodes.find((n) => n.id === Number(node.id));
        if (sitemapNode) {
            setSelectedNode(sitemapNode);
            setEditDialogOpen(true);
        }
    }, [sitemapNodes, setSelectedNode, setEditDialogOpen]);

    const onNodeContextMenu = useCallback((event: React.MouseEvent, node: Node) => {
        event.preventDefault();
        const sitemapNode = sitemapNodes.find((n) => n.id === Number(node.id));
        if (sitemapNode) {
            setContextMenuNode(sitemapNode);
            setContextMenuPosition({ x: event.clientX, y: event.clientY });
        }
    }, [sitemapNodes]);

    const onPaneContextMenu = useCallback((event: MouseEvent | React.MouseEvent) => {
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

    const handleDeleteEdge = useCallback(async () => {
        if (!selectedEdgeId) return;

        const match = selectedEdgeId.match(/^e(\d+)-(\d+)$/);
        if (!match) return;

        const childId = Number(match[2]);

        await execute(
            () => sitemapService.moveNode({ nodeId: childId, newParentId: undefined }),
            { errorTitle: "Failed to remove connection" }
        );

        closeEdgeContextMenu();
        await loadData(true);
        await refreshHistory();
    }, [selectedEdgeId, execute, closeEdgeContextMenu, loadData, refreshHistory]);

    const handleAutoLayout = useCallback(() => {
        const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(nodes, edges);
        setNodes(layoutedNodes);
        setEdges(layoutedEdges);
        setHasUnsavedChanges(true);
        setTimeout(() => fitView({ duration: 300, padding: 0.3 }), 50);
    }, [edges, fitView, nodes, setEdges, setNodes, setHasUnsavedChanges]);

    const handleSelectionChange = useCallback(({ nodes: selectedNodes }: { nodes: Node[] }) => {
        const selectedIds = selectedNodes.map((n) => Number(n.id));
        setSidebarSelectedNodeIds(new Set(selectedIds));
    }, []);

    const handleSidebarNodesSelect = useCallback((nodeIds: number[]) => {
        setSidebarSelectedNodeIds(new Set(nodeIds));
        setNodes((nds) => nds.map((n) => ({
            ...n,
            selected: nodeIds.includes(Number(n.id)),
        })));
    }, [setNodes]);

    return {
        contextMenuNode,
        contextMenuPosition,
        edgeContextMenuPosition,
        selectedEdgeId,
        sidebarSelectedNodeIds,
        onConnectStart,
        onConnect,
        onConnectEnd,
        onNodeClick,
        onNodeDoubleClick,
        onNodeContextMenu,
        onPaneContextMenu,
        onEdgeContextMenu,
        closeContextMenu,
        closeEdgeContextMenu,
        handleDeleteEdge,
        handleAutoLayout,
        handleSelectionChange,
        handleSidebarNodesSelect,
    };
}
