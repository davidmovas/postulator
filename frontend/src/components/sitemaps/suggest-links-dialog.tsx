"use client";

import { useState, useEffect, useMemo } from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
    DialogDescription,
} from "@/components/ui/dialog";
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
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { AlertCircle, CheckCircle2, Loader2, Sparkles, ChevronDown, ChevronUp } from "lucide-react";
import { providerService } from "@/services/providers";
import { promptService } from "@/services/prompts";
import { linkingService } from "@/services/linking";
import { Provider } from "@/models/providers";
import { Prompt } from "@/models/prompts";
import { SitemapNode } from "@/models/sitemaps";

interface SuggestLinksDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    planId: number;
    selectedNodes: SitemapNode[];
    allNodes: SitemapNode[];
    onSuccess: () => void;
}

type SuggestState = "idle" | "loading" | "success" | "error";

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

    const [providerId, setProviderId] = useState<number | null>(null);
    const [promptId, setPromptId] = useState<number | null>(null);
    const [feedback, setFeedback] = useState("");
    const [maxIncoming, setMaxIncoming] = useState("");
    const [maxOutgoing, setMaxOutgoing] = useState("");

    const [providers, setProviders] = useState<Provider[]>([]);
    const [prompts, setPrompts] = useState<Prompt[]>([]);
    const [isLoadingData, setIsLoadingData] = useState(true);
    const [showAdvanced, setShowAdvanced] = useState(false);

    const nodesToAnalyze = useMemo(() => {
        if (selectedNodes.length > 0) {
            return selectedNodes.filter((n) => !n.isRoot);
        }
        return allNodes.filter((n) => !n.isRoot);
    }, [selectedNodes, allNodes]);

    useEffect(() => {
        if (open) {
            loadData();
        }
    }, [open]);

    const loadData = async () => {
        setIsLoadingData(true);
        try {
            const [providersData, promptsData] = await Promise.all([
                providerService.listProviders(),
                promptService.listPrompts(),
            ]);

            const activeProviders = providersData.filter((p) => p.isActive);
            setProviders(activeProviders);
            setPrompts(promptsData);

            if (activeProviders.length > 0 && !providerId) {
                setProviderId(activeProviders[0].id);
            }
        } catch (err) {
            console.error("Failed to load data:", err);
        } finally {
            setIsLoadingData(false);
        }
    };

    const handleSuggest = async () => {
        if (!providerId) {
            setError("Please select a provider");
            return;
        }

        if (nodesToAnalyze.length < 2) {
            setError("Need at least 2 nodes to suggest links");
            return;
        }

        setSuggestState("loading");
        setError(null);

        try {
            await linkingService.suggestLinks({
                planId,
                providerId,
                promptId: promptId || undefined,
                nodeIds: nodesToAnalyze.map((n) => n.id),
                feedback: feedback.trim() || undefined,
                maxIncoming: maxIncoming ? parseInt(maxIncoming, 10) : undefined,
                maxOutgoing: maxOutgoing ? parseInt(maxOutgoing, 10) : undefined,
            });

            setSuggestState("success");
            onSuccess();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to generate suggestions");
            setSuggestState("error");
        }
    };

    const handleClose = () => {
        setSuggestState("idle");
        setError(null);
        setFeedback("");
        setMaxIncoming("");
        setMaxOutgoing("");
        onOpenChange(false);
    };

    const canSuggest = providerId !== null && nodesToAnalyze.length >= 2 && !isLoadingData;

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="max-w-lg">
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
                                            <Label>Prompt Template</Label>
                                            <Select
                                                value={promptId?.toString() || "default"}
                                                onValueChange={(v) => setPromptId(v === "default" ? null : Number(v))}
                                            >
                                                <SelectTrigger>
                                                    <SelectValue />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    <SelectItem value="default">Default (Built-in)</SelectItem>
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
                        <div className="flex flex-col items-center justify-center py-8 gap-4">
                            <Loader2 className="h-12 w-12 animate-spin text-primary" />
                            <div className="text-center">
                                <p className="text-lg font-medium">Analyzing {nodesToAnalyze.length} pages...</p>
                                <p className="text-sm text-muted-foreground mt-1">
                                    AI is generating link suggestions
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
                    {suggestState === "success" && (
                        <Button onClick={handleClose}>Close</Button>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
