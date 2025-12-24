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
} from "@/models/sitemaps";
import { Provider } from "@/models/providers";
import { Prompt } from "@/models/prompts";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
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
    const [language, setLanguage] = useState("English");

    const [wordCount, setWordCount] = useState("800-1200");
    const [writingStyle, setWritingStyle] = useState<WritingStyle>("professional");
    const [contentTone, setContentTone] = useState<ContentTone>("informative");
    const [customInstructions, setCustomInstructions] = useState("");
    const [includeLinks, setIncludeLinks] = useState(false);

    const [providers, setProviders] = useState<Provider[]>([]);
    const [prompts, setPrompts] = useState<Prompt[]>([]);
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

    const handleGenerate = async () => {
        if (!providerId || nodesToGenerate.length === 0) {
            setError("Please select a provider and ensure there are nodes to generate");
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
            const result = await sitemapService.startPageGeneration({
                sitemapId,
                nodeIds: nodesToGenerate.map((n) => n.id),
                providerId,
                promptId: promptId || undefined,
                publishAs,
                placeholders: {
                    language,
                },
                contentSettings: {
                    wordCount,
                    writingStyle,
                    contentTone,
                    customInstructions: customInstructions || undefined,
                    includeLinks: includeLinks || undefined,
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

    const canGenerate = providerId !== null && nodesToGenerate.length > 0 && !isLoadingData;
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

                                    <div className="space-y-2">
                                        <Label>Content Language</Label>
                                        <Input
                                            value={language}
                                            onChange={(e) => setLanguage(e.target.value)}
                                            placeholder="English"
                                        />
                                    </div>
                                </div>

                                {/* Content Settings */}
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-2">
                                        <Label>Word Count</Label>
                                        <Input
                                            value={wordCount}
                                            onChange={(e) => setWordCount(e.target.value)}
                                            placeholder="800-1200 or 1000"
                                        />
                                        <p className="text-xs text-muted-foreground">
                                            Target: exact number or range
                                        </p>
                                    </div>

                                    <div className="space-y-2">
                                        <Label>Writing Style</Label>
                                        <Select
                                            value={writingStyle}
                                            onValueChange={(v) => setWritingStyle(v as WritingStyle)}
                                        >
                                            <SelectTrigger>
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="professional">Professional</SelectItem>
                                                <SelectItem value="casual">Casual</SelectItem>
                                                <SelectItem value="formal">Formal</SelectItem>
                                                <SelectItem value="friendly">Friendly</SelectItem>
                                                <SelectItem value="technical">Technical</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                </div>

                                <div className="space-y-2">
                                    <Label>Content Tone</Label>
                                    <Select
                                        value={contentTone}
                                        onValueChange={(v) => setContentTone(v as ContentTone)}
                                    >
                                        <SelectTrigger>
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent>
                                            <SelectItem value="informative">Informative</SelectItem>
                                            <SelectItem value="persuasive">Persuasive</SelectItem>
                                            <SelectItem value="educational">Educational</SelectItem>
                                            <SelectItem value="engaging">Engaging</SelectItem>
                                            <SelectItem value="authoritative">Authoritative</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </div>

                                {/* Include Links Option */}
                                <div className="flex items-center gap-3 py-2 px-3 rounded-md border bg-muted/30">
                                    <Checkbox
                                        id="includeLinks"
                                        checked={includeLinks}
                                        onCheckedChange={(checked) => setIncludeLinks(checked === true)}
                                        disabled={!hasApprovedLinks}
                                    />
                                    <div className="flex-1">
                                        <Label
                                            htmlFor="includeLinks"
                                            className={cn(
                                                "flex items-center gap-2 cursor-pointer",
                                                !hasApprovedLinks && "text-muted-foreground cursor-not-allowed"
                                            )}
                                        >
                                            <Link2 className="h-4 w-4" />
                                            Include Internal Links
                                        </Label>
                                        <p className="text-xs text-muted-foreground mt-0.5">
                                            {hasApprovedLinks
                                                ? "Embed approved internal links in generated content"
                                                : "No approved links available (approve links in Links Mode first)"}
                                        </p>
                                    </div>
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

                        {task.status === "running" && (
                            <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                                <Loader2 className="h-4 w-4 animate-spin" />
                                Generating content...
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
