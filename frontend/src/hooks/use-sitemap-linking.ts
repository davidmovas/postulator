import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import { Edge, Connection, EdgeMouseHandler, OnConnectStart, OnConnectEnd } from "@xyflow/react";
import { linkingService } from "@/services/linking";
import { LinkPlan, PlannedLink, LinkGraph, GraphEdge, LinkStatus } from "@/models/linking";

interface UseSitemapLinkingProps {
    sitemapId: number;
    siteId: number;
    enabled: boolean;
}

interface LinkCounts {
    outgoing: number;
    incoming: number;
}

interface LinkContextMenuState {
    linkId: number;
    position: { x: number; y: number };
    status: LinkStatus;
}

interface UseSitemapLinkingReturn {
    // State
    plan: LinkPlan | null;
    links: PlannedLink[];
    linkGraph: LinkGraph | null;
    isLoading: boolean;
    error: string | null;

    // Link counts per node (nodeId -> counts)
    linkCountsMap: Map<number, LinkCounts>;

    // React Flow edges for linking mode
    linkEdges: Edge[];

    // Context menu
    linkContextMenu: LinkContextMenuState | null;
    onEdgeContextMenu: EdgeMouseHandler;
    closeLinkContextMenu: () => void;

    // Connection handlers
    onConnectStart: OnConnectStart;
    onConnectEnd: OnConnectEnd;

    // Operations
    loadLinkingData: () => Promise<void>;
    addLink: (sourceNodeId: number, targetNodeId: number) => Promise<PlannedLink | null>;
    removeLink: (linkId: number) => Promise<boolean>;
    approveLink: (linkId: number) => Promise<boolean>;
    rejectLink: (linkId: number) => Promise<boolean>;
    onConnect: (connection: Connection) => Promise<void>;
}

// Стили линий по статусу
const getLinkEdgeStyle = (status: LinkStatus) => {
    switch (status) {
        case "planned":
            return { stroke: "#f59e0b", strokeWidth: 2 }; // Orange
        case "approved":
            return { stroke: "#22c55e", strokeWidth: 2 }; // Green
        case "rejected":
            return { stroke: "#ef4444", strokeWidth: 2, strokeDasharray: "5,5" }; // Red dashed
        case "applied":
            return { stroke: "#3b82f6", strokeWidth: 2 }; // Blue
        case "applying":
            return { stroke: "#8b5cf6", strokeWidth: 2 }; // Purple
        case "failed":
            return { stroke: "#dc2626", strokeWidth: 2, strokeDasharray: "3,3" }; // Dark red dashed
        default:
            return { stroke: "#f59e0b", strokeWidth: 2 };
    }
};

// Цвет маркера (стрелки) по статусу
const getMarkerColor = (status: LinkStatus) => {
    switch (status) {
        case "planned": return "#f59e0b";
        case "approved": return "#22c55e";
        case "rejected": return "#ef4444";
        case "applied": return "#3b82f6";
        case "applying": return "#8b5cf6";
        case "failed": return "#dc2626";
        default: return "#f59e0b";
    }
};

export function useSitemapLinking({
    sitemapId,
    siteId,
    enabled,
}: UseSitemapLinkingProps): UseSitemapLinkingReturn {
    const [plan, setPlan] = useState<LinkPlan | null>(null);
    const [links, setLinks] = useState<PlannedLink[]>([]);
    const [linkGraph, setLinkGraph] = useState<LinkGraph | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [linkContextMenu, setLinkContextMenu] = useState<LinkContextMenuState | null>(null);
    const connectingNodeId = useRef<string | null>(null);

    // Загрузка данных плана линковки
    const loadLinkingData = useCallback(async () => {
        if (!sitemapId || !siteId) return;

        setIsLoading(true);
        setError(null);

        try {
            const activePlan = await linkingService.getOrCreateActivePlan(sitemapId, siteId);
            setPlan(activePlan);

            const planLinks = await linkingService.getLinks(activePlan.id);
            setLinks(planLinks);

            const graph = await linkingService.getLinkGraph(activePlan.id);
            setLinkGraph(graph);
        } catch (err) {
            console.error("Failed to load linking data:", err);
            setError(err instanceof Error ? err.message : "Failed to load linking data");
        } finally {
            setIsLoading(false);
        }
    }, [sitemapId, siteId]);

    // Загружаем данные когда режим включается
    useEffect(() => {
        if (enabled) {
            loadLinkingData();
        }
    }, [enabled, loadLinkingData]);

    // Карта количества линков на ноду
    const linkCountsMap = useMemo<Map<number, LinkCounts>>(() => {
        const map = new Map<number, LinkCounts>();
        if (linkGraph) {
            linkGraph.nodes.forEach((graphNode) => {
                map.set(graphNode.nodeId, {
                    outgoing: graphNode.outgoingLinkCount,
                    incoming: graphNode.incomingLinkCount,
                });
            });
        }
        return map;
    }, [linkGraph]);

    // Конвертируем линки в React Flow edges
    const linkEdges = useMemo<Edge[]>(() => {
        if (!enabled || !linkGraph) return [];

        return linkGraph.edges.map((edge: GraphEdge) => {
            const style = getLinkEdgeStyle(edge.status);
            const markerColor = getMarkerColor(edge.status);

            return {
                id: `link-${edge.id}`,
                source: String(edge.sourceNodeId),
                target: String(edge.targetNodeId),
                type: "default", // Bezier curves
                animated: edge.status === "applying",
                selectable: true,
                style,
                markerEnd: {
                    type: "arrowclosed" as const,
                    width: 12,
                    height: 12,
                    color: markerColor,
                },
                data: {
                    linkId: edge.id,
                    status: edge.status,
                    linkSource: edge.source,
                    anchorText: edge.anchorText,
                    confidence: edge.confidence,
                },
            };
        });
    }, [enabled, linkGraph]);

    // Контекстное меню для edge
    const onEdgeContextMenu: EdgeMouseHandler = useCallback((event, edge) => {
        // Только для link edges (не hierarchy)
        if (!edge.id.startsWith("link-")) return;

        event.preventDefault();

        const linkId = edge.data?.linkId;
        const status = edge.data?.status;

        if (linkId) {
            setLinkContextMenu({
                linkId,
                position: { x: event.clientX, y: event.clientY },
                status,
            });
        }
    }, []);

    const closeLinkContextMenu = useCallback(() => {
        setLinkContextMenu(null);
    }, []);

    // Добавить новый линк
    const addLink = useCallback(async (sourceNodeId: number, targetNodeId: number): Promise<PlannedLink | null> => {
        if (!plan) return null;

        try {
            const newLink = await linkingService.addLink({
                planId: plan.id,
                sourceNodeId,
                targetNodeId,
            });

            // Обновляем данные
            await loadLinkingData();
            return newLink;
        } catch (err) {
            console.error("Failed to add link:", err);
            setError(err instanceof Error ? err.message : "Failed to add link");
            return null;
        }
    }, [plan, loadLinkingData]);

    // Удалить линк
    const removeLink = useCallback(async (linkId: number): Promise<boolean> => {
        try {
            await linkingService.removeLink(linkId);
            closeLinkContextMenu();
            await loadLinkingData();
            return true;
        } catch (err) {
            console.error("Failed to remove link:", err);
            setError(err instanceof Error ? err.message : "Failed to remove link");
            return false;
        }
    }, [loadLinkingData, closeLinkContextMenu]);

    // Одобрить линк
    const approveLink = useCallback(async (linkId: number): Promise<boolean> => {
        try {
            await linkingService.approveLink(linkId);
            closeLinkContextMenu();
            await loadLinkingData();
            return true;
        } catch (err) {
            console.error("Failed to approve link:", err);
            setError(err instanceof Error ? err.message : "Failed to approve link");
            return false;
        }
    }, [loadLinkingData, closeLinkContextMenu]);

    // Отклонить линк
    const rejectLink = useCallback(async (linkId: number): Promise<boolean> => {
        try {
            await linkingService.rejectLink(linkId);
            closeLinkContextMenu();
            await loadLinkingData();
            return true;
        } catch (err) {
            console.error("Failed to reject link:", err);
            setError(err instanceof Error ? err.message : "Failed to reject link");
            return false;
        }
    }, [loadLinkingData, closeLinkContextMenu]);

    // Начало создания связи
    const onConnectStart: OnConnectStart = useCallback((event, { nodeId, handleId, handleType }) => {
        console.log("[Linking] onConnectStart:", { nodeId, handleId, handleType });
        connectingNodeId.current = nodeId;
    }, []);

    // Конец создания связи (если не попали в target)
    const onConnectEnd: OnConnectEnd = useCallback(() => {
        connectingNodeId.current = null;
    }, []);

    // Обработка соединения - создаёт линк
    // connection.source = нода откуда НАЧАЛИ тянуть (содержит ссылку)
    // connection.target = нода куда ЗАКОНЧИЛИ тянуть (на неё ссылаемся)
    const onConnect = useCallback(async (connection: Connection) => {
        if (!connection.source || !connection.target) return;

        const sourceId = Number(connection.source);
        const targetId = Number(connection.target);

        // Нельзя ссылаться на себя
        if (sourceId === targetId) return;

        // Создаём линк: sourceId ссылается на targetId
        await addLink(sourceId, targetId);
        connectingNodeId.current = null;
    }, [addLink]);

    return {
        plan,
        links,
        linkGraph,
        isLoading,
        error,
        linkCountsMap,
        linkEdges,
        linkContextMenu,
        onEdgeContextMenu,
        closeLinkContextMenu,
        onConnectStart,
        onConnectEnd,
        loadLinkingData,
        addLink,
        removeLink,
        approveLink,
        rejectLink,
        onConnect,
    };
}
