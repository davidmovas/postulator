"use client";

import { memo, useCallback } from "react";
import { Handle, Position, NodeProps } from "@xyflow/react";
import { Card, CardContent } from "@/components/ui/card";
import { SitemapNode, NodeContentStatus } from "@/models/sitemaps";
import { Home } from "lucide-react";
import { cn } from "@/lib/utils";
import { NodeHoverCard } from "./node-hover-card";
import { BrowserOpenURL } from "@/wailsjs/wailsjs/runtime/runtime";

interface SitemapNodeData extends SitemapNode {
    label: string;
    siteUrl?: string;
}

// Status color classes for border and hover
const getStatusClasses = (status: NodeContentStatus | undefined, isRoot: boolean, isModified: boolean) => {
    // Modified nodes have orange border (priority over other statuses)
    if (isModified) {
        return {
            border: "border-l-orange-500",
            hover: "hover:border-orange-500",
        };
    }
    if (isRoot) {
        return {
            border: "border-l-primary",
            hover: "hover:border-primary",
        };
    }
    switch (status) {
        case "published":
            return {
                border: "border-l-green-500",
                hover: "hover:border-green-500",
            };
        case "draft":
            return {
                border: "border-l-yellow-500",
                hover: "hover:border-yellow-500",
            };
        case "pending":
            return {
                border: "border-l-blue-500",
                hover: "hover:border-blue-500",
            };
        default:
            return {
                border: "border-l-muted-foreground/30",
                hover: "hover:border-muted-foreground/50",
            };
    }
};

function SitemapNodeCardComponent({ data, selected }: NodeProps<SitemapNodeData>) {
    const statusClasses = getStatusClasses(data.contentStatus, data.isRoot, data.isModified);

    // Check if the node is clickable (published status)
    const isClickable = data.contentStatus === "published" && data.siteUrl;

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
                        "w-[200px] cursor-pointer transition-colors duration-200",
                        "border-l-4",
                        statusClasses.border,
                        statusClasses.hover,
                        selected && "ring-2 ring-primary"
                    )}
                >
                    <CardContent className="p-3">
                        <div className="flex items-start gap-2">
                            {data.isRoot && (
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

export const SitemapNodeCard = memo(SitemapNodeCardComponent);
