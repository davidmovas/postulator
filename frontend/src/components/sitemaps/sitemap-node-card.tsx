"use client";

import { memo, useCallback } from "react";
import { Handle, Position, NodeProps } from "@xyflow/react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SitemapNode, NodeGenerationStatus, NodePublishStatus } from "@/models/sitemaps";
import { Home, Loader2, Upload, ArrowRightToLine, ArrowLeftFromLine } from "lucide-react";
import { cn } from "@/lib/utils";
import { NodeHoverCard } from "./node-hover-card";
import { BrowserOpenURL } from "@/wailsjs/wailsjs/runtime/runtime";

export type EditorMode = "map" | "links";

export interface SitemapNodeData extends SitemapNode {
    label: string;
    siteUrl?: string;
    editorMode?: EditorMode;
    outgoingLinkCount?: number;
    incomingLinkCount?: number;
    // Highlight state for linking mode
    isHovered?: boolean;
    isIncomingSource?: boolean;  // This node links TO the hovered node
    isOutgoingTarget?: boolean;  // Hovered node links TO this node
    isDimmed?: boolean;          // Node is not related to hovered node
}

// Status color classes for border, hover, and animations based on 3-status system
const getStatusClasses = (
    generationStatus: NodeGenerationStatus,
    publishStatus: NodePublishStatus,
    isRoot: boolean,
    isModifiedLocally: boolean
) => {
    // Modified nodes have orange border (priority over other statuses)
    if (isModifiedLocally) {
        return {
            border: "border-l-orange-500",
            hover: "hover:border-orange-500",
            animation: "",
            isActive: false,
        };
    }
    if (isRoot) {
        return {
            border: "border-l-primary",
            hover: "hover:border-primary",
            animation: "",
            isActive: false,
        };
    }
    // Publishing status - active with cyan glow
    if (publishStatus === "publishing") {
        return {
            border: "border-l-cyan-500",
            hover: "hover:border-cyan-500",
            animation: "animate-pulse shadow-[0_0_15px_rgba(6,182,212,0.5)]",
            isActive: true,
            icon: "publishing",
        };
    }
    // Publish status takes priority
    switch (publishStatus) {
        case "published":
            return {
                border: "border-l-green-500",
                hover: "hover:border-green-500",
                animation: "",
                isActive: false,
            };
        case "draft":
            return {
                border: "border-l-yellow-500",
                hover: "hover:border-yellow-500",
                animation: "",
                isActive: false,
            };
        case "pending":
            return {
                border: "border-l-blue-500",
                hover: "hover:border-blue-500",
                animation: "",
                isActive: false,
            };
        case "failed":
            return {
                border: "border-l-red-500",
                hover: "hover:border-red-500",
                animation: "",
                isActive: false,
            };
    }
    // Generating status - active with purple glow
    if (generationStatus === "generating") {
        return {
            border: "border-l-purple-500",
            hover: "hover:border-purple-500",
            animation: "animate-pulse shadow-[0_0_15px_rgba(168,85,247,0.5)]",
            isActive: true,
            icon: "generating",
        };
    }
    // Then check generation status
    switch (generationStatus) {
        case "generated":
            return {
                border: "border-l-purple-500",
                hover: "hover:border-purple-500",
                animation: "",
                isActive: false,
            };
        case "queued":
            return {
                border: "border-l-slate-400",
                hover: "hover:border-slate-400",
                animation: "",
                isActive: false,
            };
        case "failed":
            return {
                border: "border-l-red-400",
                hover: "hover:border-red-400",
                animation: "",
                isActive: false,
            };
    }
    // Default - no content
    return {
        border: "border-l-muted-foreground/30",
        hover: "hover:border-muted-foreground/50",
        animation: "",
        isActive: false,
    };
};

function SitemapNodeCardComponent(props: NodeProps) {
    const { selected } = props;
    const data = props.data as unknown as SitemapNodeData;
    const isLinkingMode = data.editorMode === "links";

    const statusClasses = getStatusClasses(
        data.generationStatus,
        data.publishStatus,
        data.isRoot,
        data.isModifiedLocally
    );

    const isClickable = (data.publishStatus === "published" || data.wpUrl) && data.siteUrl;
    const displaySlug = data.isRoot ? "/" : `/${data.slug}`;

    const getFullUrl = useCallback(() => {
        if (!data.siteUrl) return null;
        const baseUrl = data.siteUrl.replace(/\/$/, "");
        if (data.isRoot) {
            return baseUrl;
        }
        const path = (data.path || `/${data.slug}`).replace(/^\/+/, "/");
        return `${baseUrl}${path}`;
    }, [data.siteUrl, data.isRoot, data.path, data.slug]);

    const handleSlugClick = useCallback((e: React.MouseEvent) => {
        e.stopPropagation();
        const url = getFullUrl();
        if (url) {
            BrowserOpenURL(url);
        }
    }, [getFullUrl]);

    const outgoingCount = data.outgoingLinkCount || 0;
    const incomingCount = data.incomingLinkCount || 0;

    // Highlight state for linking mode
    const isHovered = data.isHovered || false;
    const isIncomingSource = data.isIncomingSource || false;  // Blue - links to hovered
    const isOutgoingTarget = data.isOutgoingTarget || false;  // Green - hovered links to
    const isDimmed = data.isDimmed || false;

    // Get highlight classes for linking mode
    // 3 clearly distinct colors: purple (hovered), cyan (incoming), emerald (outgoing)
    const getLinkHighlightClasses = () => {
        if (!isLinkingMode) return "";
        if (isHovered) {
            // Purple - the node we're hovering on
            return "ring-2 ring-purple-500 shadow-[0_0_12px_rgba(168,85,247,0.5)]";
        }
        if (isIncomingSource) {
            // Cyan - nodes that link TO the hovered node
            return "ring-2 ring-cyan-400 shadow-[0_0_10px_rgba(34,211,238,0.4)]";
        }
        if (isOutgoingTarget) {
            // Emerald - nodes that hovered node links TO
            return "ring-2 ring-emerald-400 shadow-[0_0_10px_rgba(52,211,153,0.4)]";
        }
        if (isDimmed) {
            return "opacity-50";
        }
        return "";
    };

    // Card content - shared between modes
    const cardContent = (
        <Card
            className={cn(
                "w-[200px] cursor-pointer transition-all duration-300",
                "border-l-4",
                statusClasses.border,
                statusClasses.hover,
                statusClasses.animation,
                selected && "ring-2 ring-primary",
                getLinkHighlightClasses()
            )}
        >
            <CardContent className="p-2.5">
                <div className="flex items-center gap-2">
                    {/* Left side: icon + title/slug */}
                    <div className="flex items-start gap-2 min-w-0 flex-1">
                        {statusClasses.isActive && (
                            <div className="flex-shrink-0 mt-0.5">
                                {data.generationStatus === "generating" ? (
                                    <Loader2 className="h-4 w-4 animate-spin text-purple-500" />
                                ) : data.publishStatus === "publishing" ? (
                                    <Upload className="h-4 w-4 animate-bounce text-cyan-500" />
                                ) : null}
                            </div>
                        )}
                        {data.isRoot && !statusClasses.isActive && (
                            <div className="flex-shrink-0 text-primary mt-0.5">
                                <Home className="h-4 w-4" />
                            </div>
                        )}
                        <div className="min-w-0 flex-1">
                            <p className="font-medium text-sm truncate leading-tight">{data.title}</p>
                            {isClickable ? (
                                <button
                                    onClick={handleSlugClick}
                                    className="text-xs text-primary hover:underline truncate max-w-full text-left block leading-tight"
                                    title={getFullUrl() || undefined}
                                >
                                    {displaySlug}
                                </button>
                            ) : !data.isRoot ? (
                                <p className="text-xs text-muted-foreground truncate leading-tight">
                                    {displaySlug}
                                </p>
                            ) : null}
                        </div>
                    </div>

                    {/* Right side: link counts in linking mode */}
                    {isLinkingMode && (
                        <div className="flex-shrink-0 flex flex-col items-end gap-0.5 text-xs">
                            <div className="flex items-center gap-1 text-blue-500" title="Incoming links">
                                <span className="font-medium tabular-nums">{incomingCount}</span>
                                <ArrowLeftFromLine className="h-3 w-3" />
                            </div>
                            <div className="flex items-center gap-1 text-emerald-500" title="Outgoing links">
                                <span className="font-medium tabular-nums">{outgoingCount}</span>
                                <ArrowRightToLine className="h-3 w-3" />
                            </div>
                        </div>
                    )}
                </div>
            </CardContent>
        </Card>
    );

    // Linking mode - 4 handles:
    // - Right (source) & Left (target) for internal links (LTR)
    // - Bottom (source) & Top (target) for hierarchy edges (TB)
    if (isLinkingMode) {
        const linkHandleClass = cn(
            "!w-3 !h-3 !border-2 !border-background",
            "cursor-crosshair !z-50"
        );
        const hierarchyHandleClass = cn(
            "!w-2 !h-2 !border !border-background",
            "!z-40"
        );

        return (
            <>
                {/* Top = target for hierarchy (parent connects here) */}
                <Handle
                    type="target"
                    position={Position.Top}
                    id="top"
                    isConnectable={false}
                    className={cn(hierarchyHandleClass, "!bg-gray-400")}
                />
                {/* Bottom = source for hierarchy (connects to children) */}
                <Handle
                    type="source"
                    position={Position.Bottom}
                    id="bottom"
                    isConnectable={false}
                    className={cn(hierarchyHandleClass, "!bg-gray-400")}
                />
                {/* Left = target for links (incoming links land here) */}
                <Handle
                    type="target"
                    position={Position.Left}
                    id="left"
                    isConnectable={true}
                    className={cn(linkHandleClass, "!bg-blue-500 hover:!bg-blue-400")}
                />
                {/* Right = source for links (outgoing links start here) */}
                <Handle
                    type="source"
                    position={Position.Right}
                    id="right"
                    isConnectable={true}
                    className={cn(linkHandleClass, "!bg-emerald-500 hover:!bg-emerald-400")}
                />

                {cardContent}
            </>
        );
    }

    // Map mode - original layout with NodeHoverCard
    return (
        <>
            {!data.isRoot && (
                <Handle type="target" position={Position.Left} className="!bg-primary !w-2.5 !h-2.5" />
            )}
            <Handle type="source" position={Position.Right} className="!bg-primary !w-2.5 !h-2.5" />

            <NodeHoverCard node={data} delay={500}>
                <Card
                    className={cn(
                        "w-[200px] cursor-pointer transition-all duration-300",
                        "border-l-4",
                        statusClasses.border,
                        statusClasses.hover,
                        statusClasses.animation,
                        selected && "ring-2 ring-primary"
                    )}
                >
                    <CardContent className="p-3">
                        <div className="flex items-start gap-2">
                            {statusClasses.isActive && (
                                <div className="flex-shrink-0 mt-0.5">
                                    {data.generationStatus === "generating" ? (
                                        <Loader2 className="h-4 w-4 animate-spin text-purple-500" />
                                    ) : data.publishStatus === "publishing" ? (
                                        <Upload className="h-4 w-4 animate-bounce text-cyan-500" />
                                    ) : null}
                                </div>
                            )}
                            {data.isRoot && !statusClasses.isActive && (
                                <div className="flex-shrink-0 text-primary mt-0.5">
                                    <Home className="h-4 w-4" />
                                </div>
                            )}
                            <div className="min-w-0 flex-1">
                                <p className="font-medium text-sm truncate">{data.title}</p>
                                {isClickable ? (
                                    <button
                                        onClick={handleSlugClick}
                                        className="text-xs text-primary hover:underline truncate max-w-full text-left block"
                                        title={getFullUrl() || undefined}
                                    >
                                        {displaySlug}
                                    </button>
                                ) : !data.isRoot ? (
                                    <p className="text-xs text-muted-foreground truncate">
                                        {displaySlug}
                                    </p>
                                ) : null}
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </NodeHoverCard>
        </>
    );
}

// Temporarily disabled memo to debug handle issues
export const SitemapNodeCard = SitemapNodeCardComponent;
