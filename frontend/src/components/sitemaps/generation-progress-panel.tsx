"use client";

import { useMemo, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { GenerationTask, GenerationNodeInfo } from "@/models/sitemaps";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import {
    Tooltip,
    TooltipContent,
    TooltipTrigger,
} from "@/components/ui/tooltip";
import {
    Loader2,
    Pause,
    Play,
    XCircle,
    ChevronUp,
    ChevronDown,
    AlertCircle,
    Clock,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface GenerationProgressPanelProps {
    task: GenerationTask;
    onPause: () => void;
    onResume: () => void;
    onCancel: () => void;
}

export function GenerationProgressPanel({
    task,
    onPause,
    onResume,
    onCancel,
}: GenerationProgressPanelProps) {
    const [expanded, setExpanded] = useState(true);

    // Filter nodes: show only active (generating/publishing), pending (up to 4), and failed
    const visibleNodes = useMemo(() => {
        if (!task.nodes) return [];

        const active: GenerationNodeInfo[] = [];
        const pending: GenerationNodeInfo[] = [];
        const failed: GenerationNodeInfo[] = [];

        for (const node of task.nodes) {
            if (node.status === "generating" || node.status === "publishing") {
                active.push(node);
            } else if (node.status === "pending") {
                pending.push(node);
            } else if (node.status === "failed") {
                failed.push(node);
            }
            // Skip completed nodes
        }

        // Show: all active, up to 4 pending, all failed
        return [...active, ...pending.slice(0, 4), ...failed];
    }, [task.nodes]);


    if (task.status !== "running" && task.status !== "paused") {
        return null;
    }

    return (
        <div className="border-b bg-muted/30 shrink-0">
            <div
                className="px-3 py-1.5 flex items-center justify-between cursor-pointer hover:bg-muted/50 transition-colors"
                onClick={() => setExpanded(!expanded)}
            >
                <div className="flex items-center gap-3">
                    {task.status === "running" ? (
                        <Loader2 className="h-4 w-4 animate-spin text-purple-500" />
                    ) : (
                        <Pause className="h-4 w-4 text-yellow-500" />
                    )}
                    <span className="text-sm font-medium">
                        {task.status === "running" ? "Generating content..." : "Generation paused"}
                    </span>
                    {task.failedNodes > 0 && (
                        <span className="text-sm text-red-500">
                            ({task.failedNodes} failed)
                        </span>
                    )}
                </div>
                <div className="flex items-center gap-3">
                    <span className="text-sm font-medium tabular-nums min-w-[80px] text-right">
                        {task.processedNodes} / {task.totalNodes}
                    </span>
                    <div className="w-48">
                        <Progress
                            value={(task.processedNodes / task.totalNodes) * 100}
                            className="h-2.5"
                        />
                    </div>
                    <div className="flex items-center gap-1">
                        {task.status === "running" ? (
                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-6 w-6"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onPause();
                                }}
                            >
                                <Pause className="h-3 w-3" />
                            </Button>
                        ) : (
                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-6 w-6"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onResume();
                                }}
                            >
                                <Play className="h-3 w-3" />
                            </Button>
                        )}
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6 text-red-500 hover:text-red-600"
                            onClick={(e) => {
                                e.stopPropagation();
                                onCancel();
                            }}
                        >
                            <XCircle className="h-3 w-3" />
                        </Button>
                    </div>
                    {expanded ? (
                        <ChevronUp className="h-4 w-4 text-muted-foreground" />
                    ) : (
                        <ChevronDown className="h-4 w-4 text-muted-foreground" />
                    )}
                </div>
            </div>
            {expanded && visibleNodes.length > 0 && (
                <div className="px-3 pb-2">
                    <div className="flex flex-wrap gap-1.5 text-xs">
                        <AnimatePresence mode="popLayout">
                            {visibleNodes.map((node) => (
                                <NodeBadge key={node.nodeId} node={node} />
                            ))}
                        </AnimatePresence>
                    </div>
                </div>
            )}
        </div>
    );
}

function NodeBadge({ node }: { node: GenerationNodeInfo }) {
    const badge = (
        <motion.div
            layout
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.8 }}
            transition={{ duration: 0.2 }}
            className={cn(
                "px-2 py-1 rounded flex items-center gap-1.5 max-w-[200px]",
                node.status === "failed" && "bg-red-500/10 text-red-700",
                node.status === "generating" && "bg-blue-500/10 text-blue-700",
                node.status === "publishing" && "bg-yellow-500/10 text-yellow-700",
                node.status === "pending" && "bg-muted text-muted-foreground"
            )}
        >
            {node.status === "generating" && <Loader2 className="h-3 w-3 animate-spin shrink-0" />}
            {node.status === "publishing" && <Loader2 className="h-3 w-3 animate-spin shrink-0" />}
            {node.status === "pending" && <Clock className="h-3 w-3 shrink-0 opacity-50" />}
            {node.status === "failed" && <AlertCircle className="h-3 w-3 shrink-0" />}
            <span className="truncate">{node.title}</span>
        </motion.div>
    );

    // Use Tooltip for failed nodes to show error message
    if (node.status === "failed" && node.error) {
        return (
            <Tooltip>
                <TooltipTrigger asChild>{badge}</TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-[300px]">
                    <p className="text-red-600 font-medium">Error:</p>
                    <p className="text-sm">{node.error}</p>
                </TooltipContent>
            </Tooltip>
        );
    }

    return badge;
}
