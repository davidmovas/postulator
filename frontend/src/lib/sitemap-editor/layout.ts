import dagre from "dagre";
import { Node, Edge, MarkerType } from "@xyflow/react";
import { SitemapNode } from "@/models/sitemaps";

export const LAYOUT = {
    NODE: {
        WIDTH: 200,
        HEIGHT: 60,
    },
    MAP_MODE: {
        DIRECTION: "LR" as const,
        NODE_SEP: 40,
        RANK_SEP: 100,
    },
    LINKS_MODE: {
        DIRECTION: "TB" as const,
        NODE_SEP: 100,
        RANK_SEP: 180,
        EDGE_SEP: 50,
        EXTRA_WIDTH: 20,
        EXTRA_HEIGHT: 30,
    },
} as const;

export const EDGE_STYLES = {
    HIERARCHY: {
        STROKE: "#888",
        STROKE_WIDTH: 1.5,
    },
    MARKER: {
        WIDTH: 12,
        HEIGHT: 12,
    },
} as const;

// Backwards compatibility
export const NODE_WIDTH = LAYOUT.NODE.WIDTH;
export const NODE_HEIGHT = LAYOUT.NODE.HEIGHT;

export const getLayoutedElements = (
    nodes: Node[],
    edges: Edge[],
    direction = LAYOUT.MAP_MODE.DIRECTION
) => {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setDefaultEdgeLabel(() => ({}));
    dagreGraph.setGraph({
        rankdir: direction,
        nodesep: LAYOUT.MAP_MODE.NODE_SEP,
        ranksep: LAYOUT.MAP_MODE.RANK_SEP,
    });

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

// Layout specifically for links mode - considers both hierarchy and link edges
// Hierarchy edges define the primary structure, link edges influence positioning
export const getLinksLayoutedElements = (
    nodes: Node[],
    hierarchyEdges: Edge[],
    linkEdges: Edge[] = [],
    direction = LAYOUT.LINKS_MODE.DIRECTION
) => {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setDefaultEdgeLabel(() => ({}));
    dagreGraph.setGraph({
        rankdir: direction,
        nodesep: LAYOUT.LINKS_MODE.NODE_SEP,
        ranksep: LAYOUT.LINKS_MODE.RANK_SEP,
        edgesep: LAYOUT.LINKS_MODE.EDGE_SEP,
    });

    const nodeWidth = NODE_WIDTH + LAYOUT.LINKS_MODE.EXTRA_WIDTH;
    const nodeHeight = NODE_HEIGHT + LAYOUT.LINKS_MODE.EXTRA_HEIGHT;

    nodes.forEach((node) => {
        dagreGraph.setNode(node.id, {
            width: nodeWidth,
            height: nodeHeight,
        });
    });

    // Add hierarchy edges first - these define the primary structure
    hierarchyEdges.forEach((edge) => {
        dagreGraph.setEdge(edge.source, edge.target, {
            weight: 2, // Higher weight for hierarchy edges
        });
    });

    // Add link edges with lower weight - they influence positioning
    // but don't override the hierarchy structure
    linkEdges.forEach((edge) => {
        // Only add if not already an edge (avoid duplicates with hierarchy)
        if (!dagreGraph.hasEdge(edge.source, edge.target)) {
            dagreGraph.setEdge(edge.source, edge.target, {
                weight: 1, // Lower weight for link edges
            });
        }
    });

    dagre.layout(dagreGraph);

    const newPositions = new Map<string, { x: number; y: number }>();
    nodes.forEach((node) => {
        const nodeWithPosition = dagreGraph.node(node.id);
        if (nodeWithPosition) {
            newPositions.set(node.id, {
                x: nodeWithPosition.x - nodeWidth / 2,
                y: nodeWithPosition.y - nodeHeight / 2,
            });
        }
    });

    return newPositions;
};

export const convertToFlowNodes = (sitemapNodes: SitemapNode[], siteUrl?: string): Node[] => {
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
            siteUrl,
        },
    }));
};

export const convertToFlowEdges = (sitemapNodes: SitemapNode[]): Edge[] => {
    const edges: Edge[] = [];

    sitemapNodes.forEach((node) => {
        if (node.parentId) {
            edges.push({
                id: `e${node.parentId}-${node.id}`,
                source: String(node.parentId),
                target: String(node.id),
                type: "default",
                animated: false,
                style: { stroke: EDGE_STYLES.HIERARCHY.STROKE, strokeWidth: EDGE_STYLES.HIERARCHY.STROKE_WIDTH },
                markerEnd: {
                    type: MarkerType.ArrowClosed,
                    width: EDGE_STYLES.MARKER.WIDTH,
                    height: EDGE_STYLES.MARKER.HEIGHT,
                    color: EDGE_STYLES.HIERARCHY.STROKE,
                },
            });
        }
    });

    return edges;
};
