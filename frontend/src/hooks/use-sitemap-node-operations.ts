import { useCallback, useState } from "react";
import { Node } from "@xyflow/react";
import { sitemapService } from "@/services/sitemaps";
import { SitemapNode, CreateNodeInput } from "@/models/sitemaps";

interface UseSitemapNodeOperationsProps {
    sitemapId: number;
    sitemapNodes: SitemapNode[];
    nodes: Node[];
    execute: <T>(fn: () => Promise<T>, options?: { errorTitle?: string }) => Promise<T | null>;
    loadData: (preservePositions?: boolean) => Promise<void>;
    refreshHistory: () => Promise<void>;
}

export function useSitemapNodeOperations({
    sitemapId,
    sitemapNodes,
    nodes,
    execute,
    loadData,
    refreshHistory,
}: UseSitemapNodeOperationsProps) {
    const [selectedNode, setSelectedNode] = useState<SitemapNode | null>(null);
    const [editDialogOpen, setEditDialogOpen] = useState(false);
    const [createDialogOpen, setCreateDialogOpen] = useState(false);
    const [parentNodeId, setParentNodeId] = useState<number | undefined>();

    const handleCreateNode = useCallback(async (input: CreateNodeInput) => {
        const result = await execute<SitemapNode>(
            () => sitemapService.createNode(input),
            { errorTitle: "Failed to create node" }
        );

        if (result) {
            setCreateDialogOpen(false);
            await loadData();
            await refreshHistory();
        }
    }, [execute, loadData, refreshHistory]);

    const handleUpdateNode = useCallback(async () => {
        setEditDialogOpen(false);
        await loadData();
        await refreshHistory();
    }, [loadData, refreshHistory]);

    const handleDeleteNode = useCallback(async (nodeId: number) => {
        const node = sitemapNodes.find((n) => n.id === nodeId);
        if (!node || node.isRoot) return;

        await execute(
            () => sitemapService.deleteNode(nodeId),
            { errorTitle: "Failed to delete node" }
        );

        setEditDialogOpen(false);
        await loadData();
        await refreshHistory();
    }, [sitemapNodes, execute, loadData, refreshHistory]);

    const handleDeleteSelectedNodes = useCallback(async () => {
        const selectedNodeIds = nodes
            .filter((n) => n.selected)
            .map((n) => Number(n.id));

        if (selectedNodeIds.length === 0) return;

        const nonRootNodeIds = selectedNodeIds.filter((id) => {
            const node = sitemapNodes.find((n) => n.id === id);
            return node && !node.isRoot;
        });

        if (nonRootNodeIds.length === 0) return;

        const selectedSet = new Set(nonRootNodeIds);
        const nodesToDelete = nonRootNodeIds.filter((id) => {
            const node = sitemapNodes.find((n) => n.id === id);
            if (!node) return false;

            let currentNode = node;
            while (currentNode.parentId) {
                if (selectedSet.has(currentNode.parentId)) {
                    return false;
                }
                currentNode = sitemapNodes.find((n) => n.id === currentNode.parentId)!;
                if (!currentNode) break;
            }
            return true;
        });

        if (nodesToDelete.length === 0) return;

        for (const nodeId of nodesToDelete) {
            await execute(
                () => sitemapService.deleteNode(nodeId),
                { errorTitle: "Failed to delete node" }
            );
        }

        await loadData();
        await refreshHistory();
    }, [nodes, sitemapNodes, execute, loadData, refreshHistory]);

    const handleAddChild = useCallback((parentId: number) => {
        setParentNodeId(parentId);
        setCreateDialogOpen(true);
    }, []);

    const handleAddNode = useCallback((parentId?: number) => {
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
        setParentNodeId(undefined);
        setCreateDialogOpen(true);
    }, []);

    const handleEditNode = useCallback((node: SitemapNode) => {
        setSelectedNode(node);
        setEditDialogOpen(true);
    }, []);

    const getAllDescendantIds = useCallback((nodeId: number): number[] => {
        const ids: number[] = [nodeId];
        const children = sitemapNodes.filter((n) => n.parentId === nodeId);
        for (const child of children) {
            ids.push(...getAllDescendantIds(child.id));
        }
        return ids;
    }, [sitemapNodes]);

    const getSelectedSitemapNodes = useCallback((): SitemapNode[] => {
        const selectedFlowNodeIds = nodes
            .filter((n) => n.selected)
            .map((n) => Number(n.id));
        return sitemapNodes.filter((n) => selectedFlowNodeIds.includes(n.id));
    }, [nodes, sitemapNodes]);

    return {
        selectedNode,
        setSelectedNode,
        editDialogOpen,
        setEditDialogOpen,
        createDialogOpen,
        setCreateDialogOpen,
        parentNodeId,
        setParentNodeId,
        handleCreateNode,
        handleUpdateNode,
        handleDeleteNode,
        handleDeleteSelectedNodes,
        handleAddChild,
        handleAddNode,
        handleAddOrphanNode,
        handleEditNode,
        getAllDescendantIds,
        getSelectedSitemapNodes,
    };
}
