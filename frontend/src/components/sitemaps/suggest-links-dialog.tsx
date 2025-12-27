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
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
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
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { AlertCircle, CheckCircle2, Loader2, Sparkles, ChevronDown, ChevronUp, Link2 } from "lucide-react";
import { EventsOn } from "@/wailsjs/wailsjs/runtime/runtime";
import { providerService } from "@/services/providers";
import { promptService } from "@/services/prompts";
import { linkingService } from "@/services/linking";
import { Provider } from "@/models/providers";
import { Prompt, isV2Prompt } from "@/models/prompts";
import { SitemapNode } from "@/models/sitemaps";
import {
    SuggestStartedEvent,
    SuggestProgressEvent,
    SuggestCompletedEvent,
    SuggestFailedEvent,
    SuggestCancelledEvent,
} from "@/models/linking";

interface SuggestLinksDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    planId: number;
    selectedNodes: SitemapNode[];
    allNodes: SitemapNode[];
    onSuccess: () => void;
}

type SuggestState = "idle" | "loading" | "success" | "error" | "cancelled";

export function SuggestLinksDialog({
    open,
    onOpenChange,
    planId,
    selectedNodes,
    allNodes,
    onSuccess,
}: SuggestLinksDialogProps) {
    const [suggestState, setSuggestState] = useState<SuggestState>("idle");
    const [error, setError] = useState<string | null>(null);
    const [showCancelConfirm, setShowCancelConfirm] = useState(false);

    const [providerId, setProviderId] = useState<number | null>(null);
    const [promptId, setPromptId] = useState<number | null>(null);
    const [feedback, setFeedback] = useState("");
    const [maxIncoming, setMaxIncoming] = useState("");
    const [maxOutgoing, setMaxOutgoing] = useState("");

    const [providers, setProviders] = useState<Provider[]>([]);
    const [prompts, setPrompts] = useState<Prompt[]>([]);
    const [isLoadingData, setIsLoadingData] = useState(true);
    const [showAdvanced, setShowAdvanced] = useState(false);

    // Progress tracking
    const [currentBatch, setCurrentBatch] = useState(0);
    const [totalBatches, setTotalBatches] = useState(0);
    const [processedNodes, setProcessedNodes] = useState(0);
    const [totalNodes, setTotalNodes] = useState(0);
    const [linksCreated, setLinksCreated] = useState(0);

    // Refs to avoid stale closures
    const isRunningRef = useRef(false);
    const isCompletedRef = useRef(false);

    const nodesToAnalyze = useMemo(() => {
        if (selectedNodes.length > 0) {
            return selectedNodes.filter((n) => !n.isRoot);
        }
        return allNodes.filter((n) => !n.isRoot);
    }, [selectedNodes, allNodes]);

    // Get selected prompt
    const selectedPrompt = useMemo(() => {
        if (!promptId || prompts.length === 0) return null;
        return prompts.find(p => p.id === promptId) || null;
    }, [promptId, prompts]);

    // Apply context config values from a prompt
    const applyPromptConfig = (prompt: Prompt) => {
        if (!isV2Prompt(prompt) || !prompt.contextConfig) return;

        const config = prompt.contextConfig;

        if (config.maxIncoming?.enabled && config.maxIncoming.value) {
            setMaxIncoming(config.maxIncoming.value);
        }
        if (config.maxOutgoing?.enabled && config.maxOutgoing.value) {
            setMaxOutgoing(config.maxOutgoing.value);
        }
    };

    // Handle prompt selection change
    const handlePromptChange = (newPromptId: number) => {
        setPromptId(newPromptId);
        const prompt = prompts.find(p => p.id === newPromptId);
        if (prompt) {
            applyPromptConfig(prompt);
        }
    };

    useEffect(() => {
        if (open) {
            loadData();
        }
    }, [open]);

    // Event subscriptions
    useEffect(() => {
        if (!open) return;

        const cleanupFns: (() => void)[] = [];

        // Suggest started
        cleanupFns.push(
            EventsOn("linking.suggest.started", (data: SuggestStartedEvent) => {
                setTotalNodes(data.TotalNodes);
                setTotalBatches(data.TotalBatches);
                setCurrentBatch(0);
                setProcessedNodes(0);
                setLinksCreated(0);
            })
        );

        // Suggest progress
        cleanupFns.push(
            EventsOn("linking.suggest.progress", (data: SuggestProgressEvent) => {
                setCurrentBatch(data.CurrentBatch);
                setTotalBatches(data.TotalBatches);
                setProcessedNodes(data.ProcessedNodes);
                setTotalNodes(data.TotalNodes);
                setLinksCreated(data.LinksCreated);
            })
        );

        // Suggest completed
        cleanupFns.push(
            EventsOn("linking.suggest.completed", (data: SuggestCompletedEvent) => {
                if (isCompletedRef.current) return;
                isCompletedRef.current = true;
                isRunningRef.current = false;

                setLinksCreated(data.LinksCreated);
                setSuggestState("success");
                onSuccess();
            })
        );

        // Suggest failed
        cleanupFns.push(
            EventsOn("linking.suggest.failed", (data: SuggestFailedEvent) => {
                isRunningRef.current = false;
                isCompletedRef.current = true;
                setError(data.Error);
                setSuggestState("error");
            })
        );

        // Suggest cancelled
        cleanupFns.push(
            EventsOn("linking.suggest.cancelled", (data: SuggestCancelledEvent) => {
                isRunningRef.current = false;
                isCompletedRef.current = true;
                setLinksCreated(data.LinksCreated);
                setProcessedNodes(data.ProcessedNodes);
                setSuggestState("cancelled");
            })
        );

        return () => {
            cleanupFns.forEach((cleanup) => cleanup());
        };
    }, [open, onSuccess]);

    const loadData = async () => {
        setIsLoadingData(true);
        try {
            const [providersData, promptsData] = await Promise.all([
                providerService.listProviders(),
                promptService.listPromptsByCategory("link_suggest"),
            ]);

            const activeProviders = providersData.filter((p) => p.isActive);
            setProviders(activeProviders);
            setPrompts(promptsData);

            if (activeProviders.length > 0 && !providerId) {
                setProviderId(activeProviders[0].id);
            }

            // Set default prompt (first builtin or first available)
            if (promptsData.length > 0 && !promptId) {
                const builtin = promptsData.find(p => p.isBuiltin);
                const defaultPrompt = builtin || promptsData[0];
                setPromptId(defaultPrompt.id);
                applyPromptConfig(defaultPrompt);
            }
        } catch (err) {
            console.error("Failed to load data:", err);
        } finally {
            setIsLoadingData(false);
        }
    };

    const handleSuggest = async () => {
        if (!providerId || !promptId) {
            setError("Please select a provider and prompt");
            return;
        }

        if (nodesToAnalyze.length < 2) {
            setError("Need at least 2 nodes to suggest links");
            return;
        }

        // Reset refs
        isRunningRef.current = true;
        isCompletedRef.current = false;

        setSuggestState("loading");
        setError(null);
        setCurrentBatch(0);
        setTotalBatches(0);
        setProcessedNodes(0);
        setTotalNodes(nodesToAnalyze.length);
        setLinksCreated(0);

        try {
            await linkingService.suggestLinks({
                planId,
                providerId,
                promptId,
                nodeIds: nodesToAnalyze.map((n) => n.id),
                feedback: feedback.trim() || undefined,
                maxIncoming: maxIncoming ? parseInt(maxIncoming, 10) : undefined,
                maxOutgoing: maxOutgoing ? parseInt(maxOutgoing, 10) : undefined,
            });

            // Fallback if events don't work
            if (isRunningRef.current && !isCompletedRef.current) {
                isRunningRef.current = false;
                isCompletedRef.current = true;
                setSuggestState("success");
                onSuccess();
            }
        } catch (err) {
            if (isRunningRef.current) {
                isRunningRef.current = false;
                setError(err instanceof Error ? err.message : "Failed to generate suggestions");
                setSuggestState("error");
            }
        }
    };

    const handleClose = () => {
        // Show confirmation when loading
        if (suggestState === "loading") {
            setShowCancelConfirm(true);
            return;
        }

        resetAndClose();
    };

    const resetAndClose = () => {
        // Reset refs
        isRunningRef.current = false;
        isCompletedRef.current = false;

        setSuggestState("idle");
        setError(null);
        setFeedback("");
        setMaxIncoming("");
        setMaxOutgoing("");
        setCurrentBatch(0);
        setTotalBatches(0);
        setProcessedNodes(0);
        setLinksCreated(0);
        setShowCancelConfirm(false);
        onOpenChange(false);
    };

    const handleConfirmCancel = async () => {
        // Call cancel on the backend
        try {
            await linkingService.cancelSuggest(planId);
        } catch {
            // Ignore errors - operation may have already completed
        }
        // Mark as completed to prevent event handlers from updating state
        isCompletedRef.current = true;
        isRunningRef.current = false;
        resetAndClose();
    };

    const handleStop = async () => {
        // Show confirmation before stopping
        setShowCancelConfirm(true);
    };

    const canSuggest = providerId !== null && promptId !== null && nodesToAnalyze.length >= 2 && !isLoadingData;

    // Calculate progress percentage
    const progressPercent = useMemo(() => {
        if (totalNodes === 0) return 0;
        return Math.round((processedNodes / totalNodes) * 100);
    }, [processedNodes, totalNodes]);

    return (
        <>
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent
                className="max-w-lg"
                onPointerDownOutside={(e) => {
                    if (suggestState === "loading") e.preventDefault();
                }}
                onEscapeKeyDown={(e) => {
                    if (suggestState === "loading") {
                        e.preventDefault();
                        setShowCancelConfirm(true);
                    }
                }}
            >
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Sparkles className="h-5 w-5" />
                        AI Suggest Links
                    </DialogTitle>
                    <DialogDescription>
                        Use AI to analyze your sitemap and suggest internal links.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    {suggestState === "idle" && (
                        <>
                            {isLoadingData ? (
                                <div className="flex items-center justify-center py-8">
                                    <Loader2 className="h-6 w-6 animate-spin" />
                                </div>
                            ) : (
                                <>
                                    <div className="bg-muted/50 rounded-lg p-4">
                                        <div className="flex items-center justify-between">
                                            <span className="text-sm font-medium">Pages to analyze</span>
                                            <Badge variant="secondary">
                                                {nodesToAnalyze.length} pages
                                            </Badge>
                                        </div>
                                        {nodesToAnalyze.length > 0 && (
                                            <ScrollArea className="h-20 mt-2">
                                                <div className="space-y-1">
                                                    {nodesToAnalyze.slice(0, 8).map((node) => (
                                                        <div
                                                            key={node.id}
                                                            className="text-xs text-muted-foreground truncate"
                                                        >
                                                            {node.path || node.title}
                                                        </div>
                                                    ))}
                                                    {nodesToAnalyze.length > 8 && (
                                                        <div className="text-xs text-muted-foreground">
                                                            ...and {nodesToAnalyze.length - 8} more
                                                        </div>
                                                    )}
                                                </div>
                                            </ScrollArea>
                                        )}
                                        {selectedNodes.length === 0 && (
                                            <p className="text-xs text-muted-foreground mt-2">
                                                Tip: Select specific nodes in the editor to analyze only those
                                            </p>
                                        )}
                                    </div>

                                    <div className="grid grid-cols-2 gap-4">
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

                                        <div className="space-y-2">
                                            <Label>Prompt Template *</Label>
                                            <Select
                                                value={promptId?.toString() || ""}
                                                onValueChange={(v) => handlePromptChange(Number(v))}
                                            >
                                                <SelectTrigger>
                                                    <SelectValue placeholder="Select prompt" />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    {prompts.map((p) => (
                                                        <SelectItem key={p.id} value={p.id.toString()}>
                                                            {p.name}
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                        </div>
                                    </div>

                                    <Collapsible open={showAdvanced} onOpenChange={setShowAdvanced}>
                                        <CollapsibleTrigger asChild>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                className="w-full justify-between"
                                            >
                                                Advanced Options
                                                {showAdvanced ? (
                                                    <ChevronUp className="h-4 w-4" />
                                                ) : (
                                                    <ChevronDown className="h-4 w-4" />
                                                )}
                                            </Button>
                                        </CollapsibleTrigger>
                                        <CollapsibleContent className="pt-2 space-y-4">
                                            <div className="grid grid-cols-2 gap-4">
                                                <div className="space-y-2">
                                                    <Label>Max Outgoing Links</Label>
                                                    <Input
                                                        type="number"
                                                        min="0"
                                                        placeholder="0 = no limit"
                                                        value={maxOutgoing}
                                                        onChange={(e) => setMaxOutgoing(e.target.value)}
                                                    />
                                                    <p className="text-xs text-muted-foreground">
                                                        Per page limit
                                                    </p>
                                                </div>
                                                <div className="space-y-2">
                                                    <Label>Max Incoming Links</Label>
                                                    <Input
                                                        type="number"
                                                        min="0"
                                                        placeholder="0 = no limit"
                                                        value={maxIncoming}
                                                        onChange={(e) => setMaxIncoming(e.target.value)}
                                                    />
                                                    <p className="text-xs text-muted-foreground">
                                                        Per page limit
                                                    </p>
                                                </div>
                                            </div>

                                            <div className="space-y-2">
                                                <Label>Additional Instructions</Label>
                                                <Textarea
                                                    placeholder="e.g. Focus on creating links from service pages to blog posts..."
                                                    value={feedback}
                                                    onChange={(e) => setFeedback(e.target.value)}
                                                    rows={3}
                                                />
                                                <p className="text-xs text-muted-foreground">
                                                    Guide the AI with specific linking preferences
                                                </p>
                                            </div>
                                        </CollapsibleContent>
                                    </Collapsible>

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

                    {suggestState === "loading" && (
                        <div className="space-y-4">
                            <div className="flex flex-col items-center justify-center py-4 gap-2">
                                <Loader2 className="h-10 w-10 animate-spin text-primary" />
                                <p className="text-lg font-medium">Analyzing pages...</p>
                                {totalBatches > 1 && (
                                    <p className="text-sm text-muted-foreground">
                                        Batch {currentBatch} of {totalBatches}
                                    </p>
                                )}
                            </div>

                            <div className="space-y-2">
                                <div className="flex items-center justify-between text-sm">
                                    <span>Progress</span>
                                    <span className="text-muted-foreground">
                                        {processedNodes} / {totalNodes} pages
                                    </span>
                                </div>
                                <Progress value={progressPercent} className="h-2" />
                            </div>

                            <div className="flex justify-center gap-6 text-sm">
                                <div className="flex items-center gap-1">
                                    <Link2 className="h-4 w-4 text-primary" />
                                    <span>{linksCreated} links suggested</span>
                                </div>
                            </div>

                            <div className="flex justify-center pt-2">
                                <Button variant="outline" onClick={handleStop}>
                                    Stop
                                </Button>
                            </div>
                        </div>
                    )}

                    {suggestState === "cancelled" && (
                        <div className="flex flex-col items-center justify-center py-8 gap-4">
                            <AlertCircle className="h-16 w-16 text-yellow-500" />
                            <div className="text-center">
                                <p className="text-lg font-medium">Operation stopped</p>
                                <p className="text-sm text-muted-foreground mt-1">
                                    {linksCreated} link suggestions created before stopping
                                </p>
                            </div>
                        </div>
                    )}

                    {suggestState === "success" && (
                        <div className="flex flex-col items-center justify-center py-8 gap-4">
                            <CheckCircle2 className="h-16 w-16 text-green-500" />
                            <div className="text-center">
                                <p className="text-lg font-medium">Suggestions generated!</p>
                                <p className="text-sm text-muted-foreground mt-1">
                                    {linksCreated} link suggestions created
                                </p>
                                <p className="text-xs text-muted-foreground mt-2">
                                    Review the suggested links in the editor
                                </p>
                            </div>
                        </div>
                    )}

                    {suggestState === "error" && (
                        <div className="flex flex-col items-center justify-center py-8 gap-4">
                            <AlertCircle className="h-16 w-16 text-destructive" />
                            <div className="text-center max-w-md">
                                <p className="text-lg font-medium text-destructive">Generation failed</p>
                                <p className="text-sm text-muted-foreground mt-2">{error}</p>
                            </div>
                            <Button variant="outline" onClick={() => setSuggestState("idle")}>
                                Try again
                            </Button>
                        </div>
                    )}
                </div>

                <DialogFooter>
                    {suggestState === "idle" && (
                        <>
                            <Button variant="outline" onClick={handleClose}>
                                Cancel
                            </Button>
                            <Button onClick={handleSuggest} disabled={!canSuggest}>
                                <Sparkles className="mr-2 h-4 w-4" />
                                Generate ({nodesToAnalyze.length} pages)
                            </Button>
                        </>
                    )}
                    {suggestState === "loading" && (
                        <p className="text-sm text-muted-foreground">
                            Please wait, generating suggestions...
                        </p>
                    )}
                    {(suggestState === "success" || suggestState === "error" || suggestState === "cancelled") && (
                        <Button onClick={handleClose}>Close</Button>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>

        <AlertDialog open={showCancelConfirm} onOpenChange={setShowCancelConfirm}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Stop Generation?</AlertDialogTitle>
                    <AlertDialogDescription>
                        Link suggestion is in progress. Stopping will cancel the operation.
                        Links already suggested will be kept.
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
