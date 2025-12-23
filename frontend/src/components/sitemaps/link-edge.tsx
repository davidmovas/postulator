"use client";

import { memo } from "react";
import {
    EdgeProps,
    getBezierPath,
    EdgeLabelRenderer,
    BaseEdge,
} from "@xyflow/react";
import { LinkStatus, LinkSource } from "@/models/linking";
import { cn } from "@/lib/utils";

interface LinkEdgeData {
    linkId: number;
    status: LinkStatus;
    linkSource: LinkSource;
    anchorText?: string;
    confidence?: number;
}

// Color mapping for link statuses
const statusColors: Record<LinkStatus, { stroke: string; fill: string }> = {
    planned: { stroke: "#6b7280", fill: "#9ca3af" },       // gray
    approved: { stroke: "#22c55e", fill: "#4ade80" },      // green
    rejected: { stroke: "#ef4444", fill: "#f87171" },      // red
    applying: { stroke: "#3b82f6", fill: "#60a5fa" },      // blue
    applied: { stroke: "#8b5cf6", fill: "#a78bfa" },       // purple
    failed: { stroke: "#f97316", fill: "#fb923c" },        // orange
};

// Arrow marker for directional links
function getMarkerEnd(status: LinkStatus) {
    return `url(#link-arrow-${status})`;
}

function LinkEdgeComponent({
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
}: EdgeProps<LinkEdgeData>) {
    const status = data?.status || "planned";
    const linkSource = data?.linkSource || "manual";
    const colors = statusColors[status];

    const [edgePath, labelX, labelY] = getBezierPath({
        sourceX,
        sourceY,
        sourcePosition,
        targetX,
        targetY,
        targetPosition,
        curvature: 0.25,
    });

    // Dashed line for AI-suggested links, solid for manual
    const strokeDasharray = linkSource === "ai" ? "5,5" : undefined;

    // Show anchor text or confidence for AI links
    const label = data?.anchorText
        ? data.anchorText
        : data?.confidence
            ? `${Math.round(data.confidence * 100)}%`
            : null;

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
                    strokeDasharray,
                }}
                markerEnd={getMarkerEnd(status)}
            />

            {/* Label for anchor text or confidence */}
            {label && (
                <EdgeLabelRenderer>
                    <div
                        style={{
                            position: "absolute",
                            transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
                            pointerEvents: "all",
                        }}
                        className={cn(
                            "px-2 py-0.5 rounded text-xs font-medium",
                            "bg-background border shadow-sm",
                            selected && "ring-2 ring-primary"
                        )}
                    >
                        {label}
                    </div>
                </EdgeLabelRenderer>
            )}
        </>
    );
}

export const LinkEdge = memo(LinkEdgeComponent);
