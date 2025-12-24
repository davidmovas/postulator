import dagre from "dagre";
import { Node, Edge, MarkerType } from "@xyflow/react";
import { SitemapNode } from "@/models/sitemaps";

export const NODE_WIDTH = 200;
export const NODE_HEIGHT = 60;

export const getLayoutedElements = (
    nodes: Node[],
    edges: Edge[],
    direction = "LR"
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

// Layout specifically for links mode - considers both hierarchy and link edges
// Hierarchy edges define the primary structure, link edges influence positioning
export const getLinksLayoutedElements = (
    nodes: Node[],
    hierarchyEdges: Edge[],
    linkEdges: Edge[] = [],
    direction = "TB" // Top to bottom works better for links visualization
) => {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setDefaultEdgeLabel(() => ({}));
    // Wider spacing for links mode to accommodate additional link edges
    dagreGraph.setGraph({
        rankdir: direction,
        nodesep: 100,  // Horizontal spacing between nodes
        ranksep: 180,  // Vertical spacing between ranks
        edgesep: 50,   // Spacing between edges
    });

    // Set nodes with extra height for link badges
    nodes.forEach((node) => {
        dagreGraph.setNode(node.id, {
            width: NODE_WIDTH + 20,  // Extra width for handles
            height: NODE_HEIGHT + 30, // Extra height for link badges
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
                x: nodeWithPosition.x - (NODE_WIDTH + 20) / 2,
                y: nodeWithPosition.y - (NODE_HEIGHT + 30) / 2,
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
