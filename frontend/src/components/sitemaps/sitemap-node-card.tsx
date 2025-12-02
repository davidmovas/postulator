"use client";

import { memo } from "react";
import { Handle, Position, NodeProps } from "@xyflow/react";
import { Card, CardContent } from "@/components/ui/card";
import { SitemapNode, NodeContentStatus } from "@/models/sitemaps";
import { Home } from "lucide-react";
import { cn } from "@/lib/utils";
import { NodeHoverCard } from "./node-hover-card";

interface SitemapNodeData extends SitemapNode {
    label: string;
}

// Status color classes for border and hover
const getStatusClasses = (status: NodeContentStatus | undefined, isRoot: boolean) => {
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
    const statusClasses = getStatusClasses(data.contentStatus, data.isRoot);

    const cardContent = (
        <>
            {/* Left handle - target (hidden for root) */}
            {!data.isRoot && (
                <Handle
                    type="target"
                    position={Position.Left}
                    className="!bg-primary !w-2.5 !h-2.5"
                />
            )}
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
                            {!data.isRoot && (
                                <p className="text-xs text-muted-foreground truncate">
                                    /{data.slug}
                                </p>
                            )}
                        </div>
                    </div>
                </CardContent>
            </Card>
            {/* Right handle - source */}
            <Handle
                type="source"
                position={Position.Right}
                className="!bg-primary !w-2.5 !h-2.5"
            />
        </>
    );

    return (
        <NodeHoverCard node={data} delay={400}>
            {cardContent}
        </NodeHoverCard>
    );
}

export const SitemapNodeCard = memo(SitemapNodeCardComponent);
