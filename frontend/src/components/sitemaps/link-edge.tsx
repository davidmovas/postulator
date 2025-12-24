"use client";

import { memo } from "react";
import {
    EdgeProps,
    getBezierPath,
    BaseEdge,
} from "@xyflow/react";
import { LinkStatus, LinkSource } from "@/models/linking";

export interface LinkEdgeData extends Record<string, unknown> {
    linkId: number;
    status: LinkStatus;
    linkSource: LinkSource;
    anchorText?: string;
    confidence?: number;
}

// Color mapping for link statuses - distinct from hierarchy (gray dashed)
const statusColors: Record<LinkStatus, { stroke: string; fill: string }> = {
    planned: { stroke: "#f59e0b", fill: "#fbbf24" },       // amber/orange - AI suggestions
    approved: { stroke: "#22c55e", fill: "#4ade80" },      // green - approved
    rejected: { stroke: "#ef4444", fill: "#f87171" },      // red - rejected
    applying: { stroke: "#3b82f6", fill: "#60a5fa" },      // blue - in progress
    applied: { stroke: "#8b5cf6", fill: "#a78bfa" },       // purple - done
    failed: { stroke: "#dc2626", fill: "#f87171" },        // dark red - failed
};

// Arrow marker for directional links
function getMarkerEnd(status: LinkStatus) {
    return `url(#link-arrow-${status})`;
}

function LinkEdgeComponent(props: EdgeProps) {
    const {
        id,
        sourceX,
        sourceY,
        targetX,
        targetY,
        sourcePosition,
        targetPosition,
        style = {},
        data,
        selected,
    } = props;

    const edgeData = data as LinkEdgeData | undefined;
    const status = edgeData?.status || "planned";
    const colors = statusColors[status];

    const [edgePath] = getBezierPath({
        sourceX,
        sourceY,
        sourcePosition,
        targetX,
        targetY,
        targetPosition,
        curvature: 0.25,
    });

    // All links are solid - hierarchy edges are dashed gray, link edges are solid colored
    // No labels to avoid clutter with many links

    return (
        <>
            {/* SVG Definitions for arrow markers */}
            <svg style={{ position: "absolute", width: 0, height: 0 }}>
                <defs>
                    {Object.entries(statusColors).map(([statusKey, colorSet]) => (
                        <marker
                            key={`link-arrow-${statusKey}`}
                            id={`link-arrow-${statusKey}`}
                            viewBox="0 0 10 10"
                            refX="8"
                            refY="5"
                            markerWidth="6"
                            markerHeight="6"
                            orient="auto-start-reverse"
                        >
                            <path
                                d="M 0 0 L 10 5 L 0 10 z"
                                fill={colorSet.fill}
                            />
                        </marker>
                    ))}
                </defs>
            </svg>

            <BaseEdge
                id={id}
                path={edgePath}
                style={{
                    ...style,
                    stroke: colors.stroke,
                    strokeWidth: selected ? 3 : 2,
                }}
                markerEnd={getMarkerEnd(status)}
            />
        </>
    );
}

export const LinkEdge = memo(LinkEdgeComponent);
