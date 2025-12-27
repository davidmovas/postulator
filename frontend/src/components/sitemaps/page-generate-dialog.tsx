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
    AlertCircle,
    CheckCircle2,
    Loader2,
    FileText,
    Pause,
    Play,
    XCircle,
    ExternalLink,
    ChevronDown,
    ChevronUp,
    Link2,
    Globe,
} from "lucide-react";
import { sitemapService } from "@/services/sitemaps";
import { providerService } from "@/services/providers";
import { promptService } from "@/services/prompts";
import {
    SitemapNode,
    GenerationTask,
    PublishAs,
    WritingStyle,
    ContentTone,
    AutoLinkMode,
} from "@/models/sitemaps";
import { Provider, Model } from "@/models/providers";
import { Prompt, isV2Prompt, ContextConfig } from "@/models/prompts";
import { Textarea } from "@/components/ui/textarea";
import { ContextConfigEditor } from "@/components/prompts/context-config/context-config-editor";
import { Checkbox } from "@/components/ui/checkbox";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
    Tooltip,
    TooltipContent,
    TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface PageGenerateDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    sitemapId: number;
    selectedNodes: SitemapNode[];
    allNodes: SitemapNode[];
    onSuccess: () => void;
    onTaskStarted?: (task: GenerationTask) => void;
    activeTask?: GenerationTask | null;
    hasApprovedLinks?: boolean; // Whether there are approved links available
}

export function PageGenerateDialog({
    open,
    onOpenChange,
    sitemapId,
    selectedNodes,
    allNodes,
    onSuccess,
    onTaskStarted,
    activeTask,
    hasApprovedLinks = false,
}: PageGenerateDialogProps) {
    const [task, setTask] = useState<GenerationTask | null>(null);
    const [error, setError] = useState<string | null>(null);

    const [providerId, setProviderId] = useState<number | null>(null);
    const [promptId, setPromptId] = useState<number | null>(null);
    const [publishAs, setPublishAs] = useState<PublishAs>("draft");
    const [contextOverrides, setContextOverrides] = useState<ContextConfig>({});
    const [customInstructions, setCustomInstructions] = useState("");
    const [useWebSearch, setUseWebSearch] = useState(false);
    const [includeLinks, setIncludeLinks] = useState(false);
    const [autoLinkMode, setAutoLinkMode] = useState<AutoLinkMode>("none");
    const [maxIncomingLinks, setMaxIncomingLinks] = useState(5);
    const [maxOutgoingLinks, setMaxOutgoingLinks] = useState(3);
    const [linkSuggestPromptId, setLinkSuggestPromptId] = useState<number | null>(null);
    const [linkApplyPromptId, setLinkApplyPromptId] = useState<number | null>(null);

    const [providers, setProviders] = useState<Provider[]>([]);
    const [prompts, setPrompts] = useState<Prompt[]>([]);
    const [linkSuggestPrompts, setLinkSuggestPrompts] = useState<Prompt[]>([]);
    const [linkApplyPrompts, setLinkApplyPrompts] = useState<Prompt[]>([]);
    const [modelsMap, setModelsMap] = useState<Record<string, Model[]>>({});
    const [isLoadingData, setIsLoadingData] = useState(true);
    const [showAdvanced, setShowAdvanced] = useState(false);

    const nodesToGenerate = useMemo(() => {
        if (selectedNodes.length > 0) {
            return selectedNodes.filter(
                (n) => !n.isRoot && n.generationStatus !== "generated"
            );
        }
        return allNodes.filter(
            (n) => !n.isRoot && n.generationStatus !== "generated"
        );
    }, [selectedNodes, allNodes]);

    // Get selected provider and check if model supports web search
    const selectedProvider = useMemo(() => {
        if (!providerId || providers.length === 0) return null;
        return providers.find(p => p.id === providerId) || null;
    }, [providerId, providers]);

    const selectedModelInfo = useMemo(() => {
        if (!selectedProvider) return null;
        const models = modelsMap[selectedProvider.type] || [];
        return models.find(m => m.id === selectedProvider.model) || null;
    }, [selectedProvider, modelsMap]);

    const supportsWebSearch = selectedModelInfo?.supportsWebSearch ?? false;

    // Get selected prompt
    const selectedPrompt = useMemo(() => {
        if (!promptId || prompts.length === 0) return null;
        return prompts.find(p => p.id === promptId) || null;
    }, [promptId, prompts]);

    // Handle prompt selection change - initialize context overrides from prompt config
    const handlePromptChange = (newPromptId: number) => {
        setPromptId(newPromptId);
        const prompt = prompts.find(p => p.id === newPromptId);
        if (prompt && isV2Prompt(prompt) && prompt.contextConfig) {
            // Initialize overrides with prompt's config values
            setContextOverrides(prompt.contextConfig);
        } else {
            setContextOverrides({});
        }
    };

    useEffect(() => {
        if (open) {
            loadData();
            // If there's an active task for this sitemap, show it
            if (activeTask && activeTask.sitemapId === sitemapId) {
                setTask(activeTask);
            }
        }
    }, [open, activeTask, sitemapId]);

    useEffect(() => {
        let interval: NodeJS.Timeout | null = null;

        if (task && (task.status === "running" || task.status === "paused")) {
            interval = setInterval(async () => {
                try {
                    const updatedTask = await sitemapService.getPageGenerationTask(task.id);
                    setTask(updatedTask);

                    if (updatedTask.status === "completed" ||
                        updatedTask.status === "failed" ||
                        updatedTask.status === "cancelled") {
                        if (interval) clearInterval(interval);
                        if (updatedTask.status === "completed") {
                            onSuccess();
                        }
                    }
                } catch (err) {
                    console.error("Failed to fetch task status:", err);
                }
            }, 2000);
        }

        return () => {
            if (interval) clearInterval(interval);
        };
    }, [task?.id, task?.status, onSuccess]);

    const loadData = async () => {
        setIsLoadingData(true);
        try {
            const [providersData, promptsData, linkSuggestData, linkApplyData] = await Promise.all([
                providerService.listProviders(),
                promptService.listPromptsByCategory("page_gen"),
                promptService.listPromptsByCategory("link_suggest"),
                promptService.listPromptsByCategory("link_apply"),
            ]);

            const activeProviders = providersData.filter((p) => p.isActive);
            setProviders(activeProviders);
            setPrompts(promptsData);
            setLinkSuggestPrompts(linkSuggestData);
            setLinkApplyPrompts(linkApplyData);

            if (activeProviders.length > 0 && !providerId) {
                setProviderId(activeProviders[0].id);
            }

            // Set default prompt (first builtin or first available)
            if (promptsData.length > 0 && !promptId) {
                const builtin = promptsData.find(p => p.isBuiltin);
                const defaultPrompt = builtin || promptsData[0];
                setPromptId(defaultPrompt.id);
                // Initialize context overrides from the prompt's config
                if (isV2Prompt(defaultPrompt) && defaultPrompt.contextConfig) {
                    setContextOverrides(defaultPrompt.contextConfig);
                }
            }

            // Set default link prompts (first builtin or first available)
            if (linkSuggestData.length > 0 && !linkSuggestPromptId) {
                const builtin = linkSuggestData.find(p => p.isBuiltin);
                setLinkSuggestPromptId(builtin?.id || linkSuggestData[0].id);
            }
            if (linkApplyData.length > 0 && !linkApplyPromptId) {
                const builtin = linkApplyData.find(p => p.isBuiltin);
                setLinkApplyPromptId(builtin?.id || linkApplyData[0].id);
            }

            // Load models for all unique provider types (for web search support check)
            const uniqueTypes = [...new Set(activeProviders.map(p => p.type))];
            const modelsPromises = uniqueTypes.map(async (type) => {
                const models = await providerService.getAvailableModels(type);
                return { type, models };
            });
            const modelsResults = await Promise.all(modelsPromises);
            const newModelsMap: Record<string, Model[]> = {};
            modelsResults.forEach(({ type, models }) => {
                newModelsMap[type] = models;
            });
            setModelsMap(newModelsMap);
        } catch (err) {
            console.error("Failed to load data:", err);
        } finally {
            setIsLoadingData(false);
        }
    };

    const handleGenerate = async () => {
        if (!providerId || !promptId || nodesToGenerate.length === 0) {
            setError("Please select a provider, prompt, and ensure there are nodes to generate");
            return;
        }

        setError(null);

        // Check if there's already an active task for this sitemap
        try {
            const activeTasks = await sitemapService.listActivePageGenerationTasks();
            const existingTask = activeTasks.find((t) => t.sitemapId === sitemapId);
            if (existingTask) {
                setTask(existingTask);
                return;
            }
        } catch (err) {
            console.error("Failed to check active tasks:", err);
        }

        try {
            // Extract values from context overrides
            const language = contextOverrides.language?.enabled ? contextOverrides.language.value || "English" : "English";
            const wordCount = contextOverrides.wordCount?.enabled ? contextOverrides.wordCount.value || "800-1200" : "800-1200";
            const writingStyle = (contextOverrides.writingStyle?.enabled ? contextOverrides.writingStyle.value || "professional" : "professional") as WritingStyle;
            const contentTone = (contextOverrides.contentTone?.enabled ? contextOverrides.contentTone.value || "informative" : "informative") as ContentTone;

            const result = await sitemapService.startPageGeneration({
                sitemapId,
                nodeIds: nodesToGenerate.map((n) => n.id),
                providerId,
                promptId,
                publishAs,
                placeholders: {
                    language,
                },
                contentSettings: {
                    wordCount,
                    writingStyle,
                    contentTone,
                    customInstructions: customInstructions || undefined,
                    useWebSearch: supportsWebSearch && useWebSearch,
                    // Only include approved links if autoLinkMode is none
                    includeLinks: autoLinkMode === "none" ? includeLinks : undefined,
                    autoLinkMode,
                    autoLinkProviderId: autoLinkMode !== "none" ? providerId : undefined,
                    autoLinkSuggestPromptId: autoLinkMode !== "none" ? linkSuggestPromptId ?? undefined : undefined,
                    autoLinkApplyPromptId: autoLinkMode === "after" ? linkApplyPromptId ?? undefined : undefined,
                    maxIncomingLinks: autoLinkMode !== "none" ? maxIncomingLinks : undefined,
                    maxOutgoingLinks: autoLinkMode !== "none" ? maxOutgoingLinks : undefined,
                },
            });

            setTask(result);
            onTaskStarted?.(result); // Notify parent to update header progress
        } catch (err) {
            setError(err instanceof Error ? err.message : "Generation failed");
        }
    };

    const handlePause = async () => {
        if (!task) return;
        try {
            await sitemapService.pausePageGeneration(task.id);
            setTask({ ...task, status: "paused" });
        } catch (err) {
            console.error("Failed to pause:", err);
        }
    };

    const handleResume = async () => {
        if (!task) return;
        try {
            await sitemapService.resumePageGeneration(task.id);
            setTask({ ...task, status: "running" });
        } catch (err) {
            console.error("Failed to resume:", err);
        }
    };

    const handleCancel = async () => {
        if (!task) return;
        try {
            await sitemapService.cancelPageGeneration(task.id);
            setTask({ ...task, status: "cancelled" });
        } catch (err) {
            console.error("Failed to cancel:", err);
        }
    };

    const handleClose = () => {
        // Allow closing the modal at any time - progress is shown in the editor header
        // Only reset task state if it's completed/failed/cancelled
        if (!task || task.status === "completed" || task.status === "failed" || task.status === "cancelled") {
            setTask(null);
            setError(null);
        }
        onOpenChange(false);
    };

    const progress = task
        ? Math.round((task.processedNodes / task.totalNodes) * 100)
        : 0;

    const canGenerate = providerId !== null && promptId !== null && nodesToGenerate.length > 0 && !isLoadingData;
    const isRunning = task?.status === "running" || task?.status === "paused";

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="max-w-2xl">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <FileText className="h-5 w-5" />
                        Generate Page Content
                    </DialogTitle>
                    <DialogDescription>
                        Generate WordPress page content for sitemap nodes using AI.
                    </DialogDescription>
                </DialogHeader>

                {!task && (
                    <div className="space-y-4">
                        {isLoadingData ? (
                            <div className="flex items-center justify-center py-8">
                                <Loader2 className="h-6 w-6 animate-spin" />
                            </div>
                        ) : (
                            <>
                                <div className="bg-muted/50 rounded-lg p-4">
                                    <div className="flex items-center justify-between">
                                        <span className="text-sm font-medium">Nodes to generate</span>
                                        <Badge variant="secondary">
                                            {nodesToGenerate.length} nodes
                                        </Badge>
                                    </div>
                                    {nodesToGenerate.length > 0 && (
                                        <ScrollArea className="h-24 mt-2">
                                            <div className="space-y-1">
                                                {nodesToGenerate.slice(0, 10).map((node) => (
                                                    <div
                                                        key={node.id}
                                                        className="text-xs text-muted-foreground truncate"
                                                    >
                                                        {node.path || node.title}
                                                    </div>
                                                ))}
                                                {nodesToGenerate.length > 10 && (
                                                    <div className="text-xs text-muted-foreground">
                                                        ...and {nodesToGenerate.length - 10} more
                                                    </div>
                                                )}
                                            </div>
                                        </ScrollArea>
                                    )}
                                </div>

                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-2">
                                        <Label>AI Provider *</Label>
                                        <Select
                                            value={providerId?.toString() || ""}
                                            onValueChange={(v) => {
                                                setProviderId(Number(v));
                                                // Reset web search when provider changes
                                                setUseWebSearch(false);
                                            }}
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

                                {/* Web Search Option */}
                                {supportsWebSearch && (
                                    <div className="flex items-center space-x-3 p-3 bg-blue-50 dark:bg-blue-950/30 rounded-lg border border-blue-200 dark:border-blue-800">
                                        <Checkbox
                                            id="useWebSearch"
                                            checked={useWebSearch}
                                            onCheckedChange={(checked) => setUseWebSearch(checked === true)}
                                        />
                                        <div className="flex-1">
                                            <Label
                                                htmlFor="useWebSearch"
                                                className="flex items-center gap-2 cursor-pointer"
                                            >
                                                <Globe className="h-4 w-4 text-blue-500" />
                                                <span className="font-medium">Enable Web Search</span>
                                            </Label>
                                            <p className="text-xs text-muted-foreground mt-1">
                                                Allow the AI to search the web for up-to-date information
                                            </p>
                                        </div>
                                    </div>
                                )}

                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-2">
                                        <Label>Publish As</Label>
                                        <Select
                                            value={publishAs}
                                            onValueChange={(v) => setPublishAs(v as PublishAs)}
                                        >
                                            <SelectTrigger>
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="draft">Draft</SelectItem>
                                                <SelectItem value="pending">Pending Review</SelectItem>
                                                <SelectItem value="publish">Published</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>

                                </div>

                                {/* Context Settings - Dynamic based on prompt config */}
                                {selectedPrompt && isV2Prompt(selectedPrompt) && (
                                    <div className="space-y-3 p-3 rounded-md border bg-muted/30">
                                        <Label className="font-medium">Content Settings</Label>
                                        <ContextConfigEditor
                                            category="page_gen"
                                            mode="override"
                                            baseConfig={selectedPrompt.contextConfig}
                                            config={contextOverrides}
                                            onChange={setContextOverrides}
                                            compact
                                        />
                                    </div>
                                )}

                                {/* Auto-Link Mode Selection */}
                                <div className="space-y-3 p-3 rounded-md border bg-muted/30">
                                    <div className="flex items-center gap-2">
                                        <Link2 className="h-4 w-4" />
                                        <Label className="font-medium">Internal Linking</Label>
                                    </div>
                                    <RadioGroup
                                        value={autoLinkMode}
                                        onValueChange={(v) => setAutoLinkMode(v as AutoLinkMode)}
                                        className="space-y-2"
                                    >
                                        <div className="flex items-start gap-2">
                                            <RadioGroupItem value="none" id="link-none" className="mt-1" />
                                            <div className="flex-1">
                                                <Label htmlFor="link-none" className="cursor-pointer font-normal">
                                                    No automatic linking
                                                </Label>
                                                <p className="text-xs text-muted-foreground">
                                                    Generate content without internal links
                                                </p>
                                            </div>
                                        </div>
                                        <div className="flex items-start gap-2">
                                            <RadioGroupItem value="before" id="link-before" className="mt-1" />
                                            <div className="flex-1">
                                                <Label htmlFor="link-before" className="cursor-pointer font-normal">
                                                    Embed links during generation
                                                </Label>
                                                <p className="text-xs text-muted-foreground">
                                                    AI suggests links before generation and embeds them in content
                                                </p>
                                            </div>
                                        </div>
                                        <div className="flex items-start gap-2">
                                            <RadioGroupItem value="after" id="link-after" className="mt-1" />
                                            <div className="flex-1">
                                                <Label htmlFor="link-after" className="cursor-pointer font-normal">
                                                    Apply links after generation
                                                </Label>
                                                <p className="text-xs text-muted-foreground">
                                                    Generate content first, then AI suggests and applies links to WordPress
                                                </p>
                                            </div>
                                        </div>
                                    </RadioGroup>

                                    {/* Link settings when auto-link mode is enabled */}
                                    {autoLinkMode !== "none" && (
                                        <div className="space-y-3 pt-2 border-t mt-3">
                                            {/* Link Suggest Prompt */}
                                            <div className="space-y-1">
                                                <Label className="text-xs">Link Suggestion Prompt</Label>
                                                <Select
                                                    value={linkSuggestPromptId?.toString() || ""}
                                                    onValueChange={(v) => setLinkSuggestPromptId(Number(v))}
                                                >
                                                    <SelectTrigger className="h-8">
                                                        <SelectValue placeholder="Select prompt..." />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        {linkSuggestPrompts.map((p) => (
                                                            <SelectItem key={p.id} value={p.id.toString()}>
                                                                {p.name}
                                                            </SelectItem>
                                                        ))}
                                                    </SelectContent>
                                                </Select>
                                            </div>

                                            {/* Link Apply Prompt - only for "after" mode */}
                                            {autoLinkMode === "after" && (
                                                <div className="space-y-1">
                                                    <Label className="text-xs">Link Insertion Prompt</Label>
                                                    <Select
                                                        value={linkApplyPromptId?.toString() || ""}
                                                        onValueChange={(v) => setLinkApplyPromptId(Number(v))}
                                                    >
                                                        <SelectTrigger className="h-8">
                                                            <SelectValue placeholder="Select prompt..." />
                                                        </SelectTrigger>
                                                        <SelectContent>
                                                            {linkApplyPrompts.map((p) => (
                                                                <SelectItem key={p.id} value={p.id.toString()}>
                                                                    {p.name}
                                                                </SelectItem>
                                                            ))}
                                                        </SelectContent>
                                                    </Select>
                                                </div>
                                            )}

                                            {/* Link limits */}
                                            <div className="grid grid-cols-2 gap-3">
                                                <div className="space-y-1">
                                                    <Label className="text-xs">Max Outgoing Links</Label>
                                                    <Input
                                                        type="number"
                                                        min={0}
                                                        max={20}
                                                        value={maxOutgoingLinks}
                                                        onChange={(e) => setMaxOutgoingLinks(Number(e.target.value))}
                                                        className="h-8"
                                                    />
                                                </div>
                                                <div className="space-y-1">
                                                    <Label className="text-xs">Max Incoming Links</Label>
                                                    <Input
                                                        type="number"
                                                        min={0}
                                                        max={20}
                                                        value={maxIncomingLinks}
                                                        onChange={(e) => setMaxIncomingLinks(Number(e.target.value))}
                                                        className="h-8"
                                                    />
                                                </div>
                                            </div>
                                        </div>
                                    )}

                                    {/* Include approved links checkbox - only show when autoLinkMode is none */}
                                    {autoLinkMode === "none" && (
                                        <div className="flex items-center gap-2 pt-2 border-t mt-3">
                                            <Checkbox
                                                id="includeLinks"
                                                checked={includeLinks}
                                                onCheckedChange={(checked) => setIncludeLinks(checked === true)}
                                                disabled={!hasApprovedLinks}
                                            />
                                            <Label
                                                htmlFor="includeLinks"
                                                className={cn(
                                                    "text-sm cursor-pointer",
                                                    !hasApprovedLinks && "text-muted-foreground cursor-not-allowed"
                                                )}
                                            >
                                                {hasApprovedLinks
                                                    ? "Include pre-approved links from linking plan"
                                                    : "No approved links available"}
                                            </Label>
                                        </div>
                                    )}
                                </div>

                                <Collapsible open={showAdvanced} onOpenChange={setShowAdvanced}>
                                    <CollapsibleTrigger asChild>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            className="w-full justify-between"
                                        >
                                            Additional Instructions
                                            {showAdvanced ? (
                                                <ChevronUp className="h-4 w-4" />
                                            ) : (
                                                <ChevronDown className="h-4 w-4" />
                                            )}
                                        </Button>
                                    </CollapsibleTrigger>
                                    <CollapsibleContent className="pt-2">
                                        <div className="space-y-2">
                                            <Textarea
                                                value={customInstructions}
                                                onChange={(e) => setCustomInstructions(e.target.value)}
                                                placeholder="Add any custom instructions for the AI (optional)..."
                                                className="min-h-[80px]"
                                            />
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
                    </div>
                )}

                {task && (
                    <div className="space-y-4">
                        <div className="space-y-2">
                            <div className="flex items-center justify-between text-sm">
                                <span>Progress</span>
                                <span className="text-muted-foreground">
                                    {task.processedNodes} / {task.totalNodes}
                                </span>
                            </div>
                            <Progress value={progress} />
                        </div>

                        <div className="grid grid-cols-4 gap-2 text-center">
                            <div className="bg-muted/50 rounded p-2">
                                <div className="text-lg font-bold">{task.processedNodes}</div>
                                <div className="text-xs text-muted-foreground">Processed</div>
                            </div>
                            <div className="bg-muted/50 rounded p-2">
                                <div className="text-lg font-bold text-green-600">
                                    {task.processedNodes - task.failedNodes - task.skippedNodes}
                                </div>
                                <div className="text-xs text-muted-foreground">Success</div>
                            </div>
                            <div className="bg-muted/50 rounded p-2">
                                <div className="text-lg font-bold text-yellow-600">
                                    {task.skippedNodes}
                                </div>
                                <div className="text-xs text-muted-foreground">Skipped</div>
                            </div>
                            <div className="bg-muted/50 rounded p-2">
                                <div className="text-lg font-bold text-red-600">
                                    {task.failedNodes}
                                </div>
                                <div className="text-xs text-muted-foreground">Failed</div>
                            </div>
                        </div>

                        {task.status === "running" && (!task.linkingPhase || task.linkingPhase === "none") && (
                            <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                                <Loader2 className="h-4 w-4 animate-spin" />
                                Generating content...
                            </div>
                        )}

                        {task.status === "running" && task.linkingPhase === "suggesting" && (
                            <div className="flex flex-col items-center gap-2 p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
                                <div className="flex items-center gap-2 text-sm text-blue-600">
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                    <Link2 className="h-4 w-4" />
                                    AI is suggesting internal links...
                                </div>
                                {task.linksCreated !== undefined && task.linksCreated > 0 && (
                                    <div className="text-xs text-blue-500">
                                        {task.linksCreated} links suggested so far
                                    </div>
                                )}
                            </div>
                        )}

                        {task.status === "running" && task.linkingPhase === "applying" && (
                            <div className="flex flex-col items-center gap-2 p-3 rounded-lg bg-purple-500/10 border border-purple-500/20">
                                <div className="flex items-center gap-2 text-sm text-purple-600">
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                    <Link2 className="h-4 w-4" />
                                    Applying links to WordPress...
                                </div>
                                {task.linksCreated !== undefined && task.linksCreated > 0 && (
                                    <div className="flex items-center gap-3 text-xs">
                                        <span className="text-purple-500">
                                            {task.linksCreated} to apply
                                        </span>
                                        {task.linksApplied !== undefined && task.linksApplied > 0 && (
                                            <span className="text-green-600">
                                                {task.linksApplied} done
                                            </span>
                                        )}
                                        {task.linksFailed !== undefined && task.linksFailed > 0 && (
                                            <span className="text-red-500">
                                                {task.linksFailed} failed
                                            </span>
                                        )}
                                    </div>
                                )}
                            </div>
                        )}

                        {task.status === "paused" && (
                            <div className="flex items-center justify-center gap-2 text-sm text-yellow-600">
                                <Pause className="h-4 w-4" />
                                Generation paused
                            </div>
                        )}

                        {task.status === "completed" && (
                            <div className="flex flex-col items-center gap-2 py-4">
                                <CheckCircle2 className="h-12 w-12 text-green-500" />
                                <p className="font-medium">Generation completed!</p>
                                {/* Show linking results if linking was performed */}
                                {(task.linksCreated !== undefined && task.linksCreated > 0) && (
                                    <div className="flex items-center gap-4 text-sm text-muted-foreground mt-2">
                                        <span className="flex items-center gap-1">
                                            <Link2 className="h-4 w-4" />
                                            {task.linksCreated} links suggested
                                        </span>
                                        {task.linksApplied !== undefined && task.linksApplied > 0 && (
                                            <span className="text-green-600">
                                                {task.linksApplied} applied
                                            </span>
                                        )}
                                        {task.linksFailed !== undefined && task.linksFailed > 0 && (
                                            <span className="text-red-600">
                                                {task.linksFailed} failed
                                            </span>
                                        )}
                                    </div>
                                )}
                            </div>
                        )}

                        {task.status === "failed" && (
                            <div className="flex flex-col items-center gap-2 py-4">
                                <AlertCircle className="h-12 w-12 text-destructive" />
                                <p className="font-medium text-destructive">Generation failed</p>
                                {task.error && (
                                    <p className="text-sm text-muted-foreground">{task.error}</p>
                                )}
                            </div>
                        )}

                        {task.status === "cancelled" && (
                            <div className="flex flex-col items-center gap-2 py-4">
                                <XCircle className="h-12 w-12 text-muted-foreground" />
                                <p className="font-medium">Generation cancelled</p>
                            </div>
                        )}

                        {task.nodes && task.nodes.length > 0 && (
                            <ScrollArea className="h-48 border rounded overflow-hidden">
                                <div className="p-2 space-y-1 overflow-hidden">
                                    {task.nodes.map((node) => (
                                        <div
                                            key={node.nodeId}
                                            className={cn(
                                                "text-sm p-2 rounded overflow-hidden",
                                                node.status === "completed" && "bg-green-500/10",
                                                node.status === "failed" && "bg-red-500/10",
                                                node.status === "generating" && "bg-blue-500/10",
                                                node.status === "publishing" && "bg-yellow-500/10"
                                            )}
                                        >
                                            <div className="flex items-center justify-between">
                                                <span className="truncate flex-1">{node.title}</span>
                                                <div className="flex items-center gap-2">
                                                    {node.status === "generating" && (
                                                        <Loader2 className="h-3 w-3 animate-spin" />
                                                    )}
                                                    {node.status === "publishing" && (
                                                        <Loader2 className="h-3 w-3 animate-spin text-yellow-500" />
                                                    )}
                                                    {node.status === "completed" && (
                                                        <CheckCircle2 className="h-3 w-3 text-green-500" />
                                                    )}
                                                    {node.status === "failed" && (
                                                        <AlertCircle className="h-3 w-3 text-red-500" />
                                                    )}
                                                    {node.wpUrl && (
                                                        <a
                                                            href={node.wpUrl}
                                                            target="_blank"
                                                            rel="noopener noreferrer"
                                                            className="text-muted-foreground hover:text-foreground"
                                                        >
                                                            <ExternalLink className="h-3 w-3" />
                                                        </a>
                                                    )}
                                                </div>
                                            </div>
                                            {node.status === "failed" && node.error && (
                                                <div
                                                    className="mt-1 text-xs text-red-500 break-words overflow-hidden"
                                                    style={{ wordBreak: "break-word", maxWidth: "100%" }}
                                                    title={node.error}
                                                >
                                                    Error: {node.error.length > 150 ? node.error.slice(0, 150) + "..." : node.error}
                                                </div>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            </ScrollArea>
                        )}
                    </div>
                )}

                <DialogFooter>
                    {!task && (
                        <>
                            <Button variant="outline" onClick={handleClose}>
                                Cancel
                            </Button>
                            <Button onClick={handleGenerate} disabled={!canGenerate}>
                                <FileText className="mr-2 h-4 w-4" />
                                Generate ({nodesToGenerate.length} pages)
                            </Button>
                        </>
                    )}
                    {isRunning && (
                        <>
                            {task?.status === "running" ? (
                                <Button variant="outline" onClick={handlePause}>
                                    <Pause className="mr-2 h-4 w-4" />
                                    Pause
                                </Button>
                            ) : (
                                <Button variant="outline" onClick={handleResume}>
                                    <Play className="mr-2 h-4 w-4" />
                                    Resume
                                </Button>
                            )}
                            <Button variant="destructive" onClick={handleCancel}>
                                <XCircle className="mr-2 h-4 w-4" />
                                Cancel
                            </Button>
                        </>
                    )}
                    {(task?.status === "completed" ||
                        task?.status === "failed" ||
                        task?.status === "cancelled") && (
                        <>
                            <Button
                                variant="outline"
                                onClick={() => {
                                    setTask(null);
                                    setError(null);
                                }}
                            >
                                New Task
                            </Button>
                            <Button onClick={handleClose}>Close</Button>
                        </>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
