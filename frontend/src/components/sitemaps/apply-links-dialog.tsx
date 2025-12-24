"use client";

import { useState, useEffect, useMemo, useRef } from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
    DialogDescription,
} from "@/components/ui/dialog";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { AlertCircle, CheckCircle2, Loader2, Link2, XCircle, FileText } from "lucide-react";
import { EventsOn } from "@/wailsjs/wailsjs/runtime/runtime";
import { providerService } from "@/services/providers";
import { linkingService } from "@/services/linking";
import { Provider } from "@/models/providers";
import {
    PlannedLink,
    ApplyLinksResult,
    ApplyStartedEvent,
    ApplyProgressEvent,
    ApplyCompletedEvent,
    ApplyFailedEvent,
    ApplyCancelledEvent,
    PageProcessingEvent,
    PageCompletedEvent,
    PageFailedEvent,
} from "@/models/linking";
import { SitemapNode } from "@/models/sitemaps";
import { cn } from "@/lib/utils";

interface ApplyLinksDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    planId: number;
    approvedLinks: PlannedLink[];
    selectedNodes: SitemapNode[];
    sitemapNodes: SitemapNode[];
    onSuccess: () => void;
}

type ApplyState = "idle" | "running" | "success" | "error" | "cancelled";

interface PageStatus {
    nodeId: number;
    title: string;
    status: "pending" | "processing" | "completed" | "failed";
    appliedLinks?: number;
    failedLinks?: number;
    error?: string;
}

export function ApplyLinksDialog({
    open,
    onOpenChange,
    planId,
    approvedLinks,
    selectedNodes,
    sitemapNodes,
    onSuccess,
}: ApplyLinksDialogProps) {
    const [applyState, setApplyState] = useState<ApplyState>("idle");
    const [error, setError] = useState<string | null>(null);
    const [result, setResult] = useState<ApplyLinksResult | null>(null);
    const [showCancelConfirm, setShowCancelConfirm] = useState(false);

    const [providerId, setProviderId] = useState<number | null>(null);
    const [providers, setProviders] = useState<Provider[]>([]);
    const [isLoadingData, setIsLoadingData] = useState(true);

    // Use ref to track running state to avoid stale closure issues
    const isRunningRef = useRef(false);
    const isCompletedRef = useRef(false);

    // Progress tracking
    const [taskId, setTaskId] = useState<string | null>(null);
    const [totalPages, setTotalPages] = useState(0);
    const [processedPages, setProcessedPages] = useState(0);
    const [appliedLinks, setAppliedLinks] = useState(0);
    const [failedLinks, setFailedLinks] = useState(0);
    const [pageStatuses, setPageStatuses] = useState<Map<number, PageStatus>>(new Map());
    const [currentPageTitle, setCurrentPageTitle] = useState<string | null>(null);

    // Get selected node IDs for filtering (only non-root nodes with WP content)
    const selectedNodeIds = useMemo(() => {
        // Filter selected nodes: must have WP page (has content)
        const nodesWithContent = selectedNodes.filter((n) => !n.isRoot && n.wpPageId != null);
        return new Set(nodesWithContent.map((n) => n.id));
    }, [selectedNodes]);

    // Get links that can be applied (approved status, source has WP page)
    // If nodes are selected, only include links where source is in selected nodes
    const linksToApply = useMemo(() => {
        return approvedLinks.filter((link) => {
            // Source node must have WP page to apply links
            const sourceNode = sitemapNodes.find((n) => n.id === link.sourceNodeId);
            if (sourceNode?.wpPageId == null) return false;

            // If nodes are selected, only include links from selected source nodes
            if (selectedNodeIds.size > 0) {
                return selectedNodeIds.has(link.sourceNodeId);
            }

            return true;
        });
    }, [approvedLinks, sitemapNodes, selectedNodeIds]);

    // Group links by source page for display
    const linksBySource = useMemo(() => {
        const grouped = new Map<number, { node: SitemapNode; links: PlannedLink[] }>();

        for (const link of linksToApply) {
            const sourceNode = sitemapNodes.find((n) => n.id === link.sourceNodeId);
            if (!sourceNode) continue;

            const existing = grouped.get(link.sourceNodeId);
            if (existing) {
                existing.links.push(link);
            } else {
                grouped.set(link.sourceNodeId, { node: sourceNode, links: [link] });
            }
        }

        return Array.from(grouped.values());
    }, [linksToApply, sitemapNodes]);

    // Calculate progress percentage
    const progressPercent = useMemo(() => {
        if (totalPages === 0) return 0;
        return Math.round((processedPages / totalPages) * 100);
    }, [processedPages, totalPages]);

    useEffect(() => {
        if (open) {
            loadData();
        }
    }, [open]);

    // Event subscriptions
    useEffect(() => {
        if (!open) return;

        const cleanupFns: (() => void)[] = [];

        // Apply started
        cleanupFns.push(
            EventsOn("linking.apply.started", (data: ApplyStartedEvent) => {
                setTaskId(data.TaskID);
                setTotalPages(data.TotalPages);
                setProcessedPages(0);
                setAppliedLinks(0);
                setFailedLinks(0);
                setPageStatuses(new Map());
            })
        );

        // Apply progress
        cleanupFns.push(
            EventsOn("linking.apply.progress", (data: ApplyProgressEvent) => {
                setProcessedPages(data.ProcessedPages);
                setAppliedLinks(data.AppliedLinks);
                setFailedLinks(data.FailedLinks);
                if (data.CurrentPage) {
                    setCurrentPageTitle(data.CurrentPage.Title);
                }
            })
        );

        // Apply completed
        cleanupFns.push(
            EventsOn("linking.apply.completed", (data: ApplyCompletedEvent) => {
                // Mark as completed to prevent double handling
                if (isCompletedRef.current) return;
                isCompletedRef.current = true;
                isRunningRef.current = false;

                setResult({
                    totalLinks: data.TotalLinks,
                    appliedLinks: data.AppliedLinks,
                    failedLinks: data.FailedLinks,
                });
                setApplyState("success");
                onSuccess();
            })
        );

        // Apply failed
        cleanupFns.push(
            EventsOn("linking.apply.failed", (data: ApplyFailedEvent) => {
                isRunningRef.current = false;
                isCompletedRef.current = true;
                setError(data.Error);
                setApplyState("error");
            })
        );

        // Page processing
        cleanupFns.push(
            EventsOn("linking.page.processing", (data: PageProcessingEvent) => {
                setCurrentPageTitle(data.Title);
                setPageStatuses((prev) => {
                    const newMap = new Map(prev);
                    newMap.set(data.NodeID, {
                        nodeId: data.NodeID,
                        title: data.Title,
                        status: "processing",
                    });
                    return newMap;
                });
            })
        );

        // Page completed
        cleanupFns.push(
            EventsOn("linking.page.completed", (data: PageCompletedEvent) => {
                setPageStatuses((prev) => {
                    const newMap = new Map(prev);
                    newMap.set(data.NodeID, {
                        nodeId: data.NodeID,
                        title: data.Title,
                        status: "completed",
                        appliedLinks: data.AppliedLinks,
                        failedLinks: data.FailedLinks,
                    });
                    return newMap;
                });
            })
        );

        // Page failed
        cleanupFns.push(
            EventsOn("linking.page.failed", (data: PageFailedEvent) => {
                setPageStatuses((prev) => {
                    const newMap = new Map(prev);
                    newMap.set(data.NodeID, {
                        nodeId: data.NodeID,
                        title: data.Title,
                        status: "failed",
                        error: data.Error,
                    });
                    return newMap;
                });
            })
        );

        // Apply cancelled
        cleanupFns.push(
            EventsOn("linking.apply.cancelled", (data: ApplyCancelledEvent) => {
                isRunningRef.current = false;
                isCompletedRef.current = true;
                setProcessedPages(data.ProcessedPages);
                setAppliedLinks(data.AppliedLinks);
                setApplyState("cancelled");
            })
        );

        return () => {
            cleanupFns.forEach((cleanup) => cleanup());
        };
    }, [open, onSuccess]);

    const loadData = async () => {
        setIsLoadingData(true);
        try {
            const providersData = await providerService.listProviders();
            const activeProviders = providersData.filter((p) => p.isActive);
            setProviders(activeProviders);

            if (activeProviders.length > 0 && !providerId) {
                setProviderId(activeProviders[0].id);
            }
        } catch (err) {
            console.error("Failed to load providers:", err);
        } finally {
            setIsLoadingData(false);
        }
    };

    const handleApply = async () => {
        if (!providerId) {
            setError("Please select a provider");
            return;
        }

        if (linksToApply.length === 0) {
            setError("No links available to apply");
            return;
        }

        // Reset refs for new operation
        isRunningRef.current = true;
        isCompletedRef.current = false;

        setApplyState("running");
        setError(null);
        setTaskId(null);
        setTotalPages(linksBySource.length);
        setProcessedPages(0);
        setAppliedLinks(0);
        setFailedLinks(0);
        setPageStatuses(new Map());

        try {
            const applyResult = await linkingService.applyLinks({
                planId,
                linkIds: linksToApply.map((l) => l.id),
                providerId,
            });

            // Use ref to check if still running (not already handled by event)
            // This is a fallback in case events don't work
            if (isRunningRef.current && !isCompletedRef.current) {
                isRunningRef.current = false;
                isCompletedRef.current = true;
                setResult(applyResult);
                setApplyState("success");
                onSuccess();
            }
        } catch (err) {
            if (isRunningRef.current) {
                isRunningRef.current = false;
                setError(err instanceof Error ? err.message : "Failed to apply links");
                setApplyState("error");
            }
        }
    };

    const handleClose = () => {
        // Show confirmation when running
        if (applyState === "running") {
            setShowCancelConfirm(true);
            return;
        }

        resetAndClose();
    };

    const resetAndClose = () => {
        // Reset refs
        isRunningRef.current = false;
        isCompletedRef.current = false;

        setApplyState("idle");
        setError(null);
        setResult(null);
        setTaskId(null);
        setPageStatuses(new Map());
        setShowCancelConfirm(false);
        onOpenChange(false);
    };

    const handleConfirmCancel = async () => {
        // Call cancel on the backend
        try {
            await linkingService.cancelApply(planId);
        } catch {
            // Ignore errors - operation may have already completed
        }
        // Mark as completed to prevent event handlers from updating state
        isCompletedRef.current = true;
        isRunningRef.current = false;
        resetAndClose();
    };

    const handleStop = () => {
        // Show confirmation before stopping
        setShowCancelConfirm(true);
    };

    const canApply = providerId !== null && linksToApply.length > 0 && !isLoadingData;

    // Render page status list
    const renderPageStatuses = () => {
        const statuses = Array.from(pageStatuses.values());
        if (statuses.length === 0) return null;

        return (
            <div className="w-full mt-4">
                <ScrollArea className="h-48">
                    <div className="space-y-2">
                        {statuses.map((page) => (
                            <div
                                key={page.nodeId}
                                className={cn(
                                    "flex items-center justify-between p-2 rounded-lg text-sm",
                                    page.status === "processing" && "bg-blue-50 dark:bg-blue-950",
                                    page.status === "completed" && "bg-green-50 dark:bg-green-950",
                                    page.status === "failed" && "bg-red-50 dark:bg-red-950"
                                )}
                            >
                                <div className="flex items-center gap-2 flex-1 min-w-0">
                                    {page.status === "processing" && (
                                        <Loader2 className="h-4 w-4 animate-spin text-blue-500 flex-shrink-0" />
                                    )}
                                    {page.status === "completed" && (
                                        <CheckCircle2 className="h-4 w-4 text-green-500 flex-shrink-0" />
                                    )}
                                    {page.status === "failed" && (
                                        <XCircle className="h-4 w-4 text-red-500 flex-shrink-0" />
                                    )}
                                    <span className="truncate">{page.title}</span>
                                </div>
                                <div className="flex-shrink-0 ml-2">
                                    {page.status === "completed" && (
                                        <span className="text-xs text-muted-foreground">
                                            {page.appliedLinks} applied
                                            {page.failedLinks ? `, ${page.failedLinks} failed` : ""}
                                        </span>
                                    )}
                                    {page.status === "failed" && (
                                        <span className="text-xs text-red-600 truncate max-w-32">
                                            {page.error}
                                        </span>
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>
                </ScrollArea>
            </div>
        );
    };

    return (
        <>
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent
                className="max-w-lg"
                onPointerDownOutside={(e) => {
                    if (applyState === "running") e.preventDefault();
                }}
                onEscapeKeyDown={(e) => {
                    if (applyState === "running") {
                        e.preventDefault();
                        setShowCancelConfirm(true);
                    }
                }}
            >
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Link2 className="h-5 w-5" />
                        Apply Links to WordPress
                    </DialogTitle>
                    <DialogDescription>
                        Insert approved links into existing WordPress page content using AI.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    {applyState === "idle" && (
                        <>
                            {isLoadingData ? (
                                <div className="flex items-center justify-center py-8">
                                    <Loader2 className="h-6 w-6 animate-spin" />
                                </div>
                            ) : (
                                <>
                                    <div className="bg-muted/50 rounded-lg p-4">
                                        <div className="flex items-center justify-between mb-2">
                                            <span className="text-sm font-medium">Links to apply</span>
                                            <Badge variant="secondary">
                                                {linksToApply.length} links in {linksBySource.length} pages
                                            </Badge>
                                        </div>
                                        {selectedNodeIds.size > 0 && (
                                            <p className="text-xs text-blue-600 mb-2">
                                                Filtering by {selectedNodeIds.size} selected page(s) with content
                                            </p>
                                        )}
                                        {linksBySource.length > 0 ? (
                                            <ScrollArea className="h-32">
                                                <div className="space-y-2">
                                                    {linksBySource.map(({ node, links }) => (
                                                        <div
                                                            key={node.id}
                                                            className="text-sm border-l-2 border-primary/30 pl-2"
                                                        >
                                                            <div className="font-medium truncate">
                                                                {node.title}
                                                            </div>
                                                            <div className="text-xs text-muted-foreground">
                                                                {links.length} outgoing link{links.length !== 1 ? "s" : ""}
                                                            </div>
                                                        </div>
                                                    ))}
                                                </div>
                                            </ScrollArea>
                                        ) : (
                                            <p className="text-sm text-muted-foreground">
                                                No links available. Make sure source pages are published to WordPress.
                                            </p>
                                        )}
                                        {approvedLinks.length > linksToApply.length && (
                                            <p className="text-xs text-amber-600 mt-2">
                                                {approvedLinks.length - linksToApply.length} approved link(s) skipped
                                                (source pages not published to WordPress)
                                            </p>
                                        )}
                                    </div>

                                    <div className="space-y-2">
                                        <Label>AI Provider *</Label>
                                        <Select
                                            value={providerId?.toString() || ""}
                                            onValueChange={(v) => setProviderId(Number(v))}
                                        >
                                            <SelectTrigger>
                                                <SelectValue placeholder="Select provider" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                {providers.map((p) => (
                                                    <SelectItem key={p.id} value={p.id.toString()}>
                                                        {p.name} ({p.type})
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                        {providers.length === 0 && (
                                            <p className="text-xs text-destructive">
                                                No active providers found
                                            </p>
                                        )}
                                    </div>

                                    {error && (
                                        <div className="flex items-center gap-2 text-destructive text-sm">
                                            <AlertCircle className="h-4 w-4" />
                                            {error}
                                        </div>
                                    )}
                                </>
                            )}
                        </>
                    )}

                    {applyState === "running" && (
                        <div className="space-y-4">
                            <div className="flex flex-col items-center justify-center py-4 gap-2">
                                <Loader2 className="h-10 w-10 animate-spin text-primary" />
                                <p className="text-lg font-medium">Applying links...</p>
                                {currentPageTitle && (
                                    <p className="text-sm text-muted-foreground flex items-center gap-1">
                                        <FileText className="h-4 w-4" />
                                        {currentPageTitle}
                                    </p>
                                )}
                            </div>

                            <div className="space-y-2">
                                <div className="flex items-center justify-between text-sm">
                                    <span>Progress</span>
                                    <span className="text-muted-foreground">
                                        {processedPages} / {totalPages} pages
                                    </span>
                                </div>
                                <Progress value={progressPercent} className="h-2" />
                            </div>

                            <div className="flex justify-center gap-6 text-sm">
                                <div className="flex items-center gap-1">
                                    <CheckCircle2 className="h-4 w-4 text-green-500" />
                                    <span>{appliedLinks} applied</span>
                                </div>
                                {failedLinks > 0 && (
                                    <div className="flex items-center gap-1">
                                        <XCircle className="h-4 w-4 text-red-500" />
                                        <span>{failedLinks} failed</span>
                                    </div>
                                )}
                            </div>

                            <div className="flex justify-center pt-2">
                                <Button variant="outline" onClick={handleStop}>
                                    Stop
                                </Button>
                            </div>

                            {renderPageStatuses()}
                        </div>
                    )}

                    {applyState === "cancelled" && (
                        <div className="space-y-4">
                            <div className="flex flex-col items-center justify-center py-6 gap-4">
                                <AlertCircle className="h-16 w-16 text-yellow-500" />
                                <div className="text-center">
                                    <p className="text-lg font-medium">Operation stopped</p>
                                    <div className="mt-2 space-y-1 text-sm text-muted-foreground">
                                        <p>Applied: {appliedLinks} links before stopping</p>
                                    </div>
                                </div>
                            </div>
                            {renderPageStatuses()}
                        </div>
                    )}

                    {applyState === "success" && result && (
                        <div className="space-y-4">
                            <div className="flex flex-col items-center justify-center py-6 gap-4">
                                <CheckCircle2 className="h-16 w-16 text-green-500" />
                                <div className="text-center">
                                    <p className="text-lg font-medium">Links applied!</p>
                                    <div className="mt-2 space-y-1 text-sm text-muted-foreground">
                                        <p>Applied: {result.appliedLinks} / {result.totalLinks}</p>
                                        {result.failedLinks > 0 && (
                                            <p className="text-amber-600">
                                                Failed: {result.failedLinks}
                                            </p>
                                        )}
                                    </div>
                                </div>
                            </div>
                            {renderPageStatuses()}
                        </div>
                    )}

                    {applyState === "error" && (
                        <div className="flex flex-col items-center justify-center py-8 gap-4">
                            <AlertCircle className="h-16 w-16 text-destructive" />
                            <div className="text-center max-w-md">
                                <p className="text-lg font-medium text-destructive">Failed to apply links</p>
                                <p className="text-sm text-muted-foreground mt-2">{error}</p>
                            </div>
                            <Button variant="outline" onClick={() => setApplyState("idle")}>
                                Try again
                            </Button>
                        </div>
                    )}
                </div>

                <DialogFooter>
                    {applyState === "idle" && (
                        <>
                            <Button variant="outline" onClick={handleClose}>
                                Cancel
                            </Button>
                            <Button onClick={handleApply} disabled={!canApply}>
                                <Link2 className="mr-2 h-4 w-4" />
                                Apply {linksToApply.length} Links
                            </Button>
                        </>
                    )}
                    {applyState === "running" && (
                        <p className="text-sm text-muted-foreground">
                            Please wait, applying links...
                        </p>
                    )}
                    {(applyState === "success" || applyState === "cancelled") && (
                        <Button onClick={handleClose}>Close</Button>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>

        <AlertDialog open={showCancelConfirm} onOpenChange={setShowCancelConfirm}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Stop Applying Links?</AlertDialogTitle>
                    <AlertDialogDescription>
                        Link insertion is in progress. Stopping will cancel the operation.
                        Links already applied will be kept.
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>Continue</AlertDialogCancel>
                    <AlertDialogAction onClick={handleConfirmCancel}>
                        Stop
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
        </>
    );
}
