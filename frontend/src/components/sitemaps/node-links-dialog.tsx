"use client";

import { useMemo } from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { SitemapNode } from "@/models/sitemaps";
import { LinkGraph, GraphEdge, LinkStatus } from "@/models/linking";
import {
    ArrowRightToLine,
    ArrowLeftFromLine,
    Check,
    X,
    Trash2,
    Sparkles,
    User,
    Focus,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface NodeLinksDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    node: SitemapNode | null;
    linkGraph: LinkGraph | null;
    sitemapNodes: SitemapNode[];
    onApproveLink: (linkId: number) => Promise<boolean>;
    onRejectLink: (linkId: number) => Promise<boolean>;
    onRemoveLink: (linkId: number) => Promise<boolean>;
    onGoToNode?: (nodeId: number) => void;
}

// Status badge colors and labels
const statusConfig: Record<LinkStatus, { label: string; className: string }> = {
    planned: { label: "Planned", className: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
    approved: { label: "Approved", className: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
    rejected: { label: "Rejected", className: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
    applying: { label: "Applying", className: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
    applied: { label: "Applied", className: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200" },
    failed: { label: "Failed", className: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200" },
};

interface LinkItemProps {
    edge: GraphEdge;
    targetNode: SitemapNode | undefined;
    targetNodeId: number;
    direction: "outgoing" | "incoming";
    onApprove: (linkId: number) => Promise<boolean>;
    onReject: (linkId: number) => Promise<boolean>;
    onRemove: (linkId: number) => Promise<boolean>;
    onGoToNode?: (nodeId: number) => void;
}

function LinkItem({ edge, targetNode, targetNodeId, direction, onApprove, onReject, onRemove, onGoToNode }: LinkItemProps) {
    const status = statusConfig[edge.status];
    const canApprove = edge.status === "planned" || edge.status === "rejected";
    const canReject = edge.status === "planned" || edge.status === "approved";
    const isAI = edge.source === "ai";

    return (
        <div className="p-2 rounded-md hover:bg-muted/50 group">
            <div className="flex items-start gap-2">
                {/* Direction icon - cyan for incoming, emerald for outgoing */}
                <div className={cn(
                    "flex-shrink-0 p-1.5 rounded-md mt-0.5",
                    direction === "outgoing"
                        ? "bg-emerald-500/10 text-emerald-500"
                        : "bg-cyan-500/10 text-cyan-500"
                )}>
                    {direction === "outgoing" ? (
                        <ArrowRightToLine className="h-3.5 w-3.5" />
                    ) : (
                        <ArrowLeftFromLine className="h-3.5 w-3.5" />
                    )}
                </div>

                {/* Target node info - takes remaining space */}
                <div className="flex-1 min-w-0 overflow-hidden">
                    <div className="flex items-center gap-1.5 flex-wrap">
                        <span className="font-medium text-sm break-words">
                            {targetNode?.title || "Unknown"}
                        </span>
                        {isAI && <Sparkles className="h-3 w-3 text-purple-500 flex-shrink-0" />}
                        <Badge variant="outline" className={cn("text-xs", status.className)}>
                            {status.label}
                        </Badge>
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5">
                        <span className="break-all">/{targetNode?.slug || ""}</span>
                    </div>
                    {edge.anchorText && (
                        <div className="text-xs text-primary italic mt-0.5 break-words">
                            "{edge.anchorText}"
                        </div>
                    )}
                </div>

                {/* Actions - compact column on the right */}
                <div className="flex items-center gap-0.5 flex-shrink-0">
                    {onGoToNode && (
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={() => onGoToNode(targetNodeId)}
                            title="Go to node"
                        >
                            <Focus className="h-3.5 w-3.5 text-muted-foreground" />
                        </Button>
                    )}
                    {canApprove && (
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={() => onApprove(edge.id)}
                            title="Approve"
                        >
                            <Check className="h-3.5 w-3.5 text-green-500" />
                        </Button>
                    )}
                    {canReject && (
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={() => onReject(edge.id)}
                            title="Reject"
                        >
                            <X className="h-3.5 w-3.5 text-orange-500" />
                        </Button>
                    )}
                    <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        onClick={() => onRemove(edge.id)}
                        title="Remove"
                    >
                        <Trash2 className="h-3.5 w-3.5 text-destructive" />
                    </Button>
                </div>
            </div>
        </div>
    );
}

export function NodeLinksDialog({
    open,
    onOpenChange,
    node,
    linkGraph,
    sitemapNodes,
    onApproveLink,
    onRejectLink,
    onRemoveLink,
    onGoToNode,
}: NodeLinksDialogProps) {
    // Get outgoing and incoming links for this node
    const { outgoingLinks, incomingLinks } = useMemo(() => {
        if (!node || !linkGraph) {
            return { outgoingLinks: [], incomingLinks: [] };
        }

        const outgoing = linkGraph.edges.filter((edge) => edge.sourceNodeId === node.id);
        const incoming = linkGraph.edges.filter((edge) => edge.targetNodeId === node.id);

        return { outgoingLinks: outgoing, incomingLinks: incoming };
    }, [node, linkGraph]);

    // Helper to find node by ID
    const getNodeById = (nodeId: number) => sitemapNodes.find((n) => n.id === nodeId);

    if (!node) return null;

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle className="flex flex-wrap items-center gap-x-2 gap-y-1">
                        <span>Links for</span>
                        <span className="text-primary break-all">{node.title}</span>
                    </DialogTitle>
                </DialogHeader>

                <div className="space-y-4">
                    {/* Outgoing links section - emerald color */}
                    <div>
                        <div className="flex items-center gap-2 mb-2">
                            <div className="p-1.5 rounded-md bg-emerald-500/10">
                                <ArrowRightToLine className="h-4 w-4 text-emerald-500" />
                            </div>
                            <span className="font-medium text-sm">Outgoing Links</span>
                            <span className="text-xs text-muted-foreground">(this page links to)</span>
                            <Badge variant="secondary" className="ml-auto">
                                {outgoingLinks.length}
                            </Badge>
                        </div>

                        {outgoingLinks.length > 0 ? (
                            <ScrollArea className="h-[150px] rounded-md border">
                                <div className="p-1">
                                    {outgoingLinks.map((edge) => (
                                        <LinkItem
                                            key={edge.id}
                                            edge={edge}
                                            targetNode={getNodeById(edge.targetNodeId)}
                                            targetNodeId={edge.targetNodeId}
                                            direction="outgoing"
                                            onApprove={onApproveLink}
                                            onReject={onRejectLink}
                                            onRemove={onRemoveLink}
                                            onGoToNode={onGoToNode}
                                        />
                                    ))}
                                </div>
                            </ScrollArea>
                        ) : (
                            <div className="text-sm text-muted-foreground text-center py-4 border rounded-md bg-muted/30">
                                No outgoing links from this page
                            </div>
                        )}
                    </div>

                    <Separator />

                    {/* Incoming links section - cyan color */}
                    <div>
                        <div className="flex items-center gap-2 mb-2">
                            <div className="p-1.5 rounded-md bg-cyan-500/10">
                                <ArrowLeftFromLine className="h-4 w-4 text-cyan-500" />
                            </div>
                            <span className="font-medium text-sm">Incoming Links</span>
                            <span className="text-xs text-muted-foreground">(pages linking here)</span>
                            <Badge variant="secondary" className="ml-auto">
                                {incomingLinks.length}
                            </Badge>
                        </div>

                        {incomingLinks.length > 0 ? (
                            <ScrollArea className="h-[150px] rounded-md border">
                                <div className="p-1">
                                    {incomingLinks.map((edge) => (
                                        <LinkItem
                                            key={edge.id}
                                            edge={edge}
                                            targetNode={getNodeById(edge.sourceNodeId)}
                                            targetNodeId={edge.sourceNodeId}
                                            direction="incoming"
                                            onApprove={onApproveLink}
                                            onReject={onRejectLink}
                                            onRemove={onRemoveLink}
                                            onGoToNode={onGoToNode}
                                        />
                                    ))}
                                </div>
                            </ScrollArea>
                        ) : (
                            <div className="text-sm text-muted-foreground text-center py-4 border rounded-md bg-muted/30">
                                No incoming links to this page
                            </div>
                        )}
                    </div>

                    {/* Summary */}
                    <div className="flex items-center justify-between text-xs text-muted-foreground pt-2 border-t">
                        <span>Total: {outgoingLinks.length + incomingLinks.length} links</span>
                        <div className="flex items-center gap-3">
                            <span className="flex items-center gap-1">
                                <Sparkles className="h-3 w-3 text-purple-500" />
                                AI
                            </span>
                            <span className="flex items-center gap-1">
                                <User className="h-3 w-3 text-muted-foreground" />
                                Manual
                            </span>
                        </div>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}
