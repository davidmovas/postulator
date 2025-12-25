import { useState, useCallback, useEffect } from "react";
import { Node, Edge, useNodesState, useEdgesState } from "@xyflow/react";
import { useApiCall } from "@/hooks/use-api-call";
import { useSitemapHistory } from "@/hooks/use-sitemap-history";
import { sitemapService } from "@/services/sitemaps";
import { siteService } from "@/services/sites";
import { Sitemap, SitemapNode, SitemapWithNodes } from "@/models/sitemaps";
import { Site } from "@/models/sites";
import { getLayoutedElements, convertToFlowNodes, convertToFlowEdges } from "@/lib/sitemap-editor";

interface UseSitemapEditorDataProps {
    siteId: number;
    sitemapId: number;
}

export function useSitemapEditorData({ siteId, sitemapId }: UseSitemapEditorDataProps) {
    const { execute, isLoading } = useApiCall();

    const [site, setSite] = useState<Site | null>(null);
    const [sitemap, setSitemap] = useState<Sitemap | null>(null);
    const [sitemapNodes, setSitemapNodes] = useState<SitemapNode[]>([]);

    const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

    const [initialPositions, setInitialPositions] = useState<Map<string, { x: number; y: number }>>(new Map());
    const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);

    const history = useSitemapHistory({ sitemapId });

    const loadData = useCallback(async (preservePositions = false) => {
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

            const flowNodes = convertToFlowNodes(sitemapResult.nodes, siteResult?.url);
            const flowEdges = convertToFlowEdges(sitemapResult.nodes);

            if (preservePositions) {
                setNodes((currentNodes) => {
                    const positionMap = new Map(currentNodes.map((n) => [n.id, n.position]));
                    return flowNodes.map((n) => ({
                        ...n,
                        position: positionMap.get(n.id) || n.position,
                    }));
                });
                setEdges(flowEdges);
                return;
            }

            const hasNodesWithoutPositions = sitemapResult.nodes.some(
                (n) => !n.isRoot && (n.positionX === undefined || n.positionY === undefined || (n.positionX === 0 && n.positionY === 0))
            );

            if (hasNodesWithoutPositions && flowNodes.length > 0) {
                const { nodes: layoutedNodes, edges: layoutedEdges } =
                    getLayoutedElements(flowNodes, flowEdges);
                setNodes(layoutedNodes);
                setEdges(layoutedEdges);
                const positions = new Map<string, { x: number; y: number }>();
                layoutedNodes.forEach((n) => positions.set(n.id, { ...n.position }));
                setInitialPositions(positions);
                setHasUnsavedChanges(true);
            } else {
                setNodes(flowNodes);
                setEdges(flowEdges);
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

    useEffect(() => {
        if (initialPositions.size === 0) return;

        const hasChanges = nodes.some((node) => {
            const initial = initialPositions.get(node.id);
            if (!initial) return false;
            return Math.abs(node.position.x - initial.x) > 1 || Math.abs(node.position.y - initial.y) > 1;
        });

        setHasUnsavedChanges(hasChanges);
    }, [nodes, initialPositions]);

    const handleSavePositions = useCallback(async () => {
        for (const node of nodes) {
            const nodeId = Number(node.id);
            await sitemapService.updateNodePositions({
                nodeId,
                positionX: node.position.x,
                positionY: node.position.y,
            });
        }
        const positions = new Map<string, { x: number; y: number }>();
        nodes.forEach((n) => positions.set(n.id, { ...n.position }));
        setInitialPositions(positions);
        setHasUnsavedChanges(false);
    }, [nodes]);

    return {
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
    };
}
