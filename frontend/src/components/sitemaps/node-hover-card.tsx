"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { createPortal } from "react-dom";
import { Badge } from "@/components/ui/badge";
import { SitemapNode, NodeGenerationStatus, NodePublishStatus } from "@/models/sitemaps";
import {
    Globe,
    FileText,
    Tag,
    Clock,
    CheckCircle2,
    AlertCircle,
    Circle,
    Sparkles,
    Loader2,
    XCircle,
    Upload,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface NodeHoverCardProps {
    node: SitemapNode;
    children: React.ReactNode;
    delay?: number;
}

const getStatusInfo = (
    generationStatus: NodeGenerationStatus,
    publishStatus: NodePublishStatus,
    isModifiedLocally: boolean
) => {
    // Modified locally has priority
    if (isModifiedLocally) {
        return {
            label: "Modified Locally",
            icon: Upload,
            className: "text-orange-500",
        };
    }
    // Publish status has highest priority
    switch (publishStatus) {
        case "published":
            return {
                label: "Published",
                icon: CheckCircle2,
                className: "text-green-500",
            };
        case "draft":
            return {
                label: "WP Draft",
                icon: AlertCircle,
                className: "text-yellow-500",
            };
        case "pending":
            return {
                label: "WP Pending",
                icon: Clock,
                className: "text-blue-500",
            };
        case "publishing":
            return {
                label: "Publishing...",
                icon: Loader2,
                className: "text-cyan-500",
            };
        case "failed":
            return {
                label: "Publish Failed",
                icon: XCircle,
                className: "text-red-500",
            };
    }
    // Then generation status
    switch (generationStatus) {
        case "generated":
            return {
                label: "Generated",
                icon: Sparkles,
                className: "text-purple-500",
            };
        case "generating":
            return {
                label: "Generating...",
                icon: Loader2,
                className: "text-purple-400",
            };
        case "queued":
            return {
                label: "Queued",
                icon: Clock,
                className: "text-slate-400",
            };
        case "failed":
            return {
                label: "Generation Failed",
                icon: XCircle,
                className: "text-red-400",
            };
    }
    // Default
    return {
        label: "No content",
        icon: Circle,
        className: "text-muted-foreground",
    };
};

export function NodeHoverCard({
    node,
    children,
    delay = 500,
}: NodeHoverCardProps) {
    const [isVisible, setIsVisible] = useState(false);
    const [position, setPosition] = useState({ x: 0, y: 0 });
    const [mounted, setMounted] = useState(false);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    const mousePositionRef = useRef({ x: 0, y: 0 });

    useEffect(() => {
        setMounted(true);
        return () => setMounted(false);
    }, []);

    const handleMouseMove = useCallback((e: React.MouseEvent) => {
        // Store the actual mouse position
        mousePositionRef.current = { x: e.clientX, y: e.clientY };
    }, []);

    const showCard = useCallback(() => {
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
        }

        timeoutRef.current = setTimeout(() => {
            // Use stored mouse position + offset
            // Position below and to the right to avoid context menu overlap
            const tooltipWidth = 280;
            const tooltipHeight = 250;

            let x = mousePositionRef.current.x + 20;
            let y = mousePositionRef.current.y + 20;

            // Check if tooltip would go off screen to the right
            if (x + tooltipWidth > window.innerWidth - 20) {
                x = mousePositionRef.current.x - tooltipWidth - 15;
            }

            // Check if tooltip would go off screen at bottom
            if (y + tooltipHeight > window.innerHeight - 20) {
                y = window.innerHeight - tooltipHeight - 20;
            }

            // Ensure not above viewport
            if (y < 20) {
                y = 20;
            }

            setPosition({ x, y });
            setIsVisible(true);
        }, delay);
    }, [delay]);

    const hideCard = useCallback(() => {
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
            timeoutRef.current = null;
        }
        setIsVisible(false);
    }, []);

    useEffect(() => {
        return () => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }
        };
    }, []);

    // Don't show hover card for root node
    if (node.isRoot) {
        return <>{children}</>;
    }

    const statusInfo = getStatusInfo(
        node.generationStatus,
        node.publishStatus,
        node.isModifiedLocally
    );
    const StatusIcon = statusInfo.icon;

    const tooltipContent = isVisible && mounted && (
        <div
            className="fixed z-[9999] w-[280px] bg-popover border rounded-lg shadow-lg p-3 space-y-3 animate-in fade-in-0 zoom-in-95 duration-150"
            style={{
                left: position.x,
                top: position.y,
                pointerEvents: 'none',
            }}
        >
            {/* Title */}
            <div>
                <p className="font-medium text-sm">{node.title}</p>
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground mt-0.5">
                    <Globe className="h-3 w-3" />
                    <span className="truncate">/{node.slug}</span>
                </div>
            </div>

            {/* Description */}
            {node.description && (
                <div className="flex items-start gap-1.5 text-xs">
                    <FileText className="h-3 w-3 text-muted-foreground mt-0.5 shrink-0" />
                    <p className="text-muted-foreground line-clamp-2">
                        {node.description}
                    </p>
                </div>
            )}

            {/* Status */}
            <div className="flex items-center gap-1.5">
                <StatusIcon className={cn("h-3.5 w-3.5", statusInfo.className)} />
                <span className="text-xs">{statusInfo.label}</span>
            </div>

            {/* Keywords */}
            {node.keywords && node.keywords.length > 0 && (
                <div className="space-y-1.5">
                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                        <Tag className="h-3 w-3" />
                        <span>Keywords ({node.keywords.length})</span>
                    </div>
                    <div className="flex flex-wrap gap-1">
                        {node.keywords.map((keyword, index) => (
                            <Badge
                                key={`kw-${index}-${keyword}`}
                                variant="secondary"
                                className="text-[10px] px-1.5 py-0 h-5"
                            >
                                {keyword}
                            </Badge>
                        ))}
                    </div>
                </div>
            )}

            {/* Stats row */}
            {node.contentType !== "none" && (
                <div className="pt-1 border-t text-[10px] text-muted-foreground capitalize">
                    {node.contentType}
                </div>
            )}
        </div>
    );

    return (
        <>
            <div
                onMouseEnter={showCard}
                onMouseMove={handleMouseMove}
                onMouseLeave={hideCard}
                style={{ pointerEvents: 'auto' }}
            >
                {children}
            </div>
            {mounted && typeof document !== 'undefined' && createPortal(tooltipContent, document.body)}
        </>
    );
}
