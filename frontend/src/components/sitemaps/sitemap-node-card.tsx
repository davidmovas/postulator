"use client";

import { memo, useCallback } from "react";
import { Handle, Position, NodeProps } from "@xyflow/react";
import { Card, CardContent } from "@/components/ui/card";
import { SitemapNode, NodeGenerationStatus, NodePublishStatus } from "@/models/sitemaps";
import { Home, Loader2, Upload } from "lucide-react";
import { cn } from "@/lib/utils";
import { NodeHoverCard } from "./node-hover-card";
import { BrowserOpenURL } from "@/wailsjs/wailsjs/runtime/runtime";

interface SitemapNodeData extends SitemapNode {
    label: string;
    siteUrl?: string;
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

function SitemapNodeCardComponent({ data, selected }: NodeProps<SitemapNodeData>) {
    // Debug: log when component renders with status
    // console.log(`[SitemapNodeCard] id=${data.id} publishStatus=${data.publishStatus} genStatus=${data.generationStatus}`);
    const statusClasses = getStatusClasses(
        data.generationStatus,
        data.publishStatus,
        data.isRoot,
        data.isModifiedLocally
    );

    // Check if the node is clickable (published status or has WP URL)
    const isClickable = (data.publishStatus === "published" || data.wpUrl) && data.siteUrl;

    // Display just the slug with leading slash in the card
    const displaySlug = data.isRoot ? "/" : `/${data.slug}`;

    // Build full URL for the node
    const getFullUrl = useCallback(() => {
        if (!data.siteUrl) return null;
        // Remove trailing slash from siteUrl
        const baseUrl = data.siteUrl.replace(/\/$/, "");
        // For root node, just return base URL
        if (data.isRoot) {
            return baseUrl;
        }
        // Use the path for full URL, normalize to avoid double slashes
        const path = (data.path || `/${data.slug}`).replace(/^\/+/, "/");
        return `${baseUrl}${path}`;
    }, [data.siteUrl, data.isRoot, data.path, data.slug]);

    const handleSlugClick = useCallback((e: React.MouseEvent) => {
        e.stopPropagation(); // Prevent node selection
        const url = getFullUrl();
        if (url) {
            BrowserOpenURL(url);
        }
    }, [getFullUrl]);

    return (
        <>
            {/* Left handle - target (hidden for root) - MUST be direct child for React Flow */}
            {!data.isRoot && (
                <Handle
                    type="target"
                    position={Position.Left}
                    className="!bg-primary !w-2.5 !h-2.5"
                />
            )}

            {/* Card wrapped in hover card for detailed info */}
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
                            {/* Status indicator for active states */}
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

            {/* Right handle - source - MUST be direct child for React Flow */}
            <Handle
                type="source"
                position={Position.Right}
                className="!bg-primary !w-2.5 !h-2.5"
            />
        </>
    );
}

// Custom comparison to ensure status changes trigger re-render
const arePropsEqual = (
    prevProps: NodeProps<SitemapNodeData>,
    nextProps: NodeProps<SitemapNodeData>
) => {
    // Check selection state
    if (prevProps.selected !== nextProps.selected) return false;

    // Check key data properties that affect rendering
    const prev = prevProps.data;
    const next = nextProps.data;

    return (
        prev.id === next.id &&
        prev.title === next.title &&
        prev.slug === next.slug &&
        prev.path === next.path &&
        prev.isRoot === next.isRoot &&
        prev.publishStatus === next.publishStatus &&
        prev.generationStatus === next.generationStatus &&
        prev.designStatus === next.designStatus &&
        prev.isModifiedLocally === next.isModifiedLocally &&
        prev.wpUrl === next.wpUrl &&
        prev.siteUrl === next.siteUrl
    );
};

export const SitemapNodeCard = memo(SitemapNodeCardComponent, arePropsEqual);
