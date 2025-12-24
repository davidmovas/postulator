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
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
    AlertCircle,
    CheckCircle2,
    Loader2,
    Sparkles,
    ChevronDown,
    ChevronUp,
    FileText,
    Eye,
    XCircle,
} from "lucide-react";
import { sitemapService } from "@/services/sitemaps";
import { promptService } from "@/services/prompts";
import { providerService } from "@/services/providers";
import {
    GenerateSitemapStructureResult,
    TitleInput,
} from "@/models/sitemaps";
import { Prompt } from "@/models/prompts";
import { Provider } from "@/models/providers";
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from "@/components/ui/collapsible";

// Props for creating a new sitemap via AI generation
interface GenerateDialogCreateProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    mode: "create";
    siteId: number;
    sitemapName: string;
    onSuccess: (result: GenerateSitemapStructureResult) => void;
}

// Props for adding to existing sitemap via AI generation
interface GenerateDialogAddProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    mode: "add";
    sitemapId: number;
    parentNodeIds?: number[];
    onSuccess: (result: GenerateSitemapStructureResult) => void;
}

type GenerateDialogProps = GenerateDialogCreateProps | GenerateDialogAddProps;

type GenerateState = "idle" | "loading" | "success" | "error" | "cancelled";

interface ParsedTitle {
    title: string;
    keywords: string[];
}

function parseTitlesInput(input: string): ParsedTitle[] {
    return input
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line.length > 0)
        .map((line) => {
            const parts = line.split("|").map((p) => p.trim());
            const title = parts[0];
            const keywords =
                parts.length > 1
                    ? parts[1]
                          .split(",")
                          .map((k) => k.trim())
                          .filter((k) => k.length > 0)
                    : [];
            return { title, keywords };
        });
}

export function GenerateDialog(props: GenerateDialogProps) {
    const { open, onOpenChange, onSuccess } = props;
    const [generateState, setGenerateState] = useState<GenerateState>("idle");
    const [result, setResult] = useState<GenerateSitemapStructureResult | null>(null);
    const [error, setError] = useState<string | null>(null);

    // Form state
    const [titlesInput, setTitlesInput] = useState("");
    const [providerId, setProviderId] = useState<number | null>(null);
    const [promptId, setPromptId] = useState<number | null>(null);
    const [maxDepth, setMaxDepth] = useState<number>(0);
    const [includeExistingTree, setIncludeExistingTree] = useState(false);
    const [activeTab, setActiveTab] = useState<"input" | "preview">("input");

    // Data
    const [providers, setProviders] = useState<Provider[]>([]);
    const [prompts, setPrompts] = useState<Prompt[]>([]);
    const [isLoadingData, setIsLoadingData] = useState(true);

    // Advanced options
    const [showAdvanced, setShowAdvanced] = useState(false);

    // Parse titles for preview
    const parsedTitles = useMemo(() => parseTitlesInput(titlesInput), [titlesInput]);

    // Load providers and prompts
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

            // Filter active providers
            const activeProviders = providersData.filter((p) => p.isActive);
            setProviders(activeProviders);

            // Filter prompts by sitemap_gen category
            const sitemapPrompts = promptsData.filter((p) => p.category === "sitemap_gen");
            setPrompts(sitemapPrompts);

            // Set defaults
            if (activeProviders.length > 0 && !providerId) {
                setProviderId(activeProviders[0].id);
            }
            if (sitemapPrompts.length > 0 && !promptId) {
                setPromptId(sitemapPrompts[0].id);
            }
        } catch {
            // Error already handled by Promise rejection
        } finally {
            setIsLoadingData(false);
        }
    };

    const handleGenerate = async () => {
        if (!providerId || !promptId || parsedTitles.length === 0) {
            setError("Please fill in all required fields");
            return;
        }

        setGenerateState("loading");
        setError(null);

        try {
            const titles: TitleInput[] = parsedTitles.map((t) => ({
                title: t.title,
                keywords: t.keywords.length > 0 ? t.keywords : undefined,
            }));

            let generateResult: GenerateSitemapStructureResult;

            if (props.mode === "create") {
                generateResult = await sitemapService.generateSitemapStructure({
                    siteId: props.siteId,
                    name: props.sitemapName,
                    promptId,
                    providerId,
                    titles,
                    maxDepth: maxDepth > 0 ? maxDepth : undefined,
                    includeExistingTree: false,
                });
            } else {
                generateResult = await sitemapService.generateSitemapStructure({
                    sitemapId: props.sitemapId,
                    promptId,
                    providerId,
                    titles,
                    parentNodeIds: props.parentNodeIds,
                    maxDepth: maxDepth > 0 ? maxDepth : undefined,
                    includeExistingTree,
                });
            }

            setResult(generateResult);
            setGenerateState("success");
            onSuccess(generateResult);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Generation failed");
            setGenerateState("error");
        }
    };

    const handleCancel = async () => {
        try {
            await sitemapService.cancelSitemapGeneration();
            setGenerateState("cancelled");
        } catch {
            // Cancel failed silently - user will see generation continues
        }
    };

    const handleClose = () => {
        // Don't cancel generation when closing dialog - let it continue in background
        setGenerateState("idle");
        setResult(null);
        setError(null);
        setTitlesInput("");
        setMaxDepth(0);
        setIncludeExistingTree(false);
        setShowAdvanced(false);
        setActiveTab("input");
        onOpenChange(false);
    };

    const canGenerate =
        providerId !== null &&
        promptId !== null &&
        parsedTitles.length > 0 &&
        !isLoadingData;

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="max-w-3xl h-[85vh] flex flex-col">
                <DialogHeader className="flex-shrink-0">
                    <DialogTitle className="flex items-center gap-2">
                        <Sparkles className="h-5 w-5" />
                        {props.mode === "create"
                            ? "Generate Sitemap Structure"
                            : "Generate & Add Nodes"}
                    </DialogTitle>
                    <DialogDescription>
                        {props.mode === "create"
                            ? "Use AI to create a hierarchical sitemap structure from your titles."
                            : "Use AI to generate new nodes and add them to your sitemap."}
                    </DialogDescription>
                </DialogHeader>

                <div className="flex-1 min-h-0 overflow-hidden">
                    {generateState === "idle" && (
                        <>
                            {isLoadingData ? (
                                <div className="flex items-center justify-center h-full">
                                    <Loader2 className="h-6 w-6 animate-spin" />
                                </div>
                            ) : (
                                <div className="flex flex-col h-full gap-4">
                                    {/* Provider & Prompt Selection - Fixed at top */}
                                    <div className="flex-shrink-0 grid grid-cols-2 gap-4">
                                        {/* Provider Selection */}
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

                                        {/* Prompt Selection */}
                                        <div className="space-y-2">
                                            <Label>Prompt Template *</Label>
                                            <Select
                                                value={promptId?.toString() || ""}
                                                onValueChange={(v) => setPromptId(Number(v))}
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

                                    {/* Tabs for Input/Preview */}
                                    <Tabs
                                        defaultValue="input"
                                        value={activeTab}
                                        onValueChange={(v) => setActiveTab(v as "input" | "preview")}
                                        className="flex-1 flex flex-col min-h-0"
                                    >
                                        <TabsList className="flex-shrink-0 grid w-full grid-cols-2">
                                            <TabsTrigger value="input" className="gap-2">
                                                <FileText className="h-4 w-4" />
                                                Input
                                            </TabsTrigger>
                                            <TabsTrigger value="preview" className="gap-2">
                                                <Eye className="h-4 w-4" />
                                                Preview
                                            </TabsTrigger>
                                        </TabsList>

                                        <TabsContent value="input" className="flex-1 min-h-0 mt-4">
                                            <div className="h-full flex flex-col gap-2">
                                                <Label>Titles *</Label>
                                                <Textarea
                                                    placeholder={`Enter titles, one per line.
Optionally add keywords after | separator:

How to Train a Dog | training, pets, dogs
Best Dog Food Brands
Dog Health Tips | health, veterinary
Understanding Dog Behavior | behavior, psychology
Puppy Care Guide | puppies, new pet owner`}
                                                    value={titlesInput}
                                                    onChange={(e) => setTitlesInput(e.target.value)}
                                                    className="flex-1 min-h-0 font-mono text-sm resize-none"
                                                />
                                                <p className="text-xs text-muted-foreground flex-shrink-0">
                                                    Format: Title | keyword1, keyword2 (keywords are optional)
                                                </p>
                                            </div>
                                        </TabsContent>

                                        <TabsContent value="preview" className="flex-1 min-h-0 mt-4">
                                            {parsedTitles.length === 0 ? (
                                                <div className="h-full flex items-center justify-center text-muted-foreground">
                                                    <div className="text-center">
                                                        <FileText className="h-12 w-12 mx-auto mb-2 opacity-50" />
                                                        <p>No titles entered yet</p>
                                                        <p className="text-sm">Switch to Input tab to add titles</p>
                                                    </div>
                                                </div>
                                            ) : (
                                                <ScrollArea className="h-full border rounded-md">
                                                    <Table>
                                                        <TableHeader className="sticky top-0 bg-background">
                                                            <TableRow>
                                                                <TableHead className="w-12">#</TableHead>
                                                                <TableHead>Title</TableHead>
                                                                <TableHead className="w-1/3">Keywords</TableHead>
                                                            </TableRow>
                                                        </TableHeader>
                                                        <TableBody>
                                                            {parsedTitles.map((t, i) => (
                                                                <TableRow
                                                                    key={i}
                                                                    className={i % 2 === 1 ? "bg-muted/30" : ""}
                                                                >
                                                                    <TableCell className="text-muted-foreground font-mono">
                                                                        {i + 1}
                                                                    </TableCell>
                                                                    <TableCell className="font-medium">
                                                                        {t.title}
                                                                    </TableCell>
                                                                    <TableCell>
                                                                        {t.keywords.length > 0 ? (
                                                                            <div className="flex flex-wrap gap-1">
                                                                                {t.keywords.map((k, ki) => (
                                                                                    <Badge
                                                                                        key={ki}
                                                                                        variant="secondary"
                                                                                        className="text-xs"
                                                                                    >
                                                                                        {k}
                                                                                    </Badge>
                                                                                ))}
                                                                            </div>
                                                                        ) : (
                                                                            <span className="text-muted-foreground text-xs">
                                                                                -
                                                                            </span>
                                                                        )}
                                                                    </TableCell>
                                                                </TableRow>
                                                            ))}
                                                        </TableBody>
                                                    </Table>
                                                </ScrollArea>
                                            )}
                                        </TabsContent>
                                    </Tabs>

                                    {/* Advanced Options - Fixed at bottom */}
                                    <div className="flex-shrink-0">
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
                                            <CollapsibleContent className="space-y-4 pt-2">
                                                <div className="grid grid-cols-2 gap-4">
                                                    {/* Max Depth */}
                                                    <div className="space-y-2">
                                                        <Label>Max Depth</Label>
                                                        <Input
                                                            type="number"
                                                            min={0}
                                                            max={10}
                                                            value={maxDepth}
                                                            onChange={(e) =>
                                                                setMaxDepth(Number(e.target.value))
                                                            }
                                                            placeholder="0 = unlimited"
                                                        />
                                                        <p className="text-xs text-muted-foreground">
                                                            Limit hierarchy depth (0 = no limit)
                                                        </p>
                                                    </div>

                                                    {/* Include Existing Tree (only for add mode) */}
                                                    {props.mode === "add" && (
                                                        <div className="space-y-2">
                                                            <Label>Include Existing Structure</Label>
                                                            <div className="flex items-center h-10">
                                                                <Switch
                                                                    checked={includeExistingTree}
                                                                    onCheckedChange={setIncludeExistingTree}
                                                                />
                                                            </div>
                                                            <p className="text-xs text-muted-foreground">
                                                                Send current sitemap to AI for context
                                                            </p>
                                                        </div>
                                                    )}
                                                </div>
                                            </CollapsibleContent>
                                        </Collapsible>
                                    </div>
                                </div>
                            )}
                        </>
                    )}

                    {generateState === "loading" && (
                        <div className="flex flex-col items-center justify-center h-full gap-4">
                            <Loader2 className="h-12 w-12 animate-spin text-primary" />
                            <div className="text-center">
                                <p className="text-lg font-medium">Generating structure...</p>
                                <p className="text-sm text-muted-foreground mt-1">
                                    AI is organizing {parsedTitles.length} titles into a
                                    hierarchical structure
                                </p>
                            </div>
                            <Button
                                variant="outline"
                                onClick={handleCancel}
                                className="mt-4"
                            >
                                <XCircle className="mr-2 h-4 w-4" />
                                Cancel Generation
                            </Button>
                        </div>
                    )}

                    {generateState === "success" && result && (
                        <div className="flex flex-col items-center justify-center h-full gap-6">
                            <CheckCircle2 className="h-16 w-16 text-green-500" />
                            <p className="text-lg font-medium">Generation completed!</p>

                            <div className="grid grid-cols-2 gap-6 w-full max-w-sm">
                                <div className="bg-muted/50 rounded-lg p-4 text-center">
                                    <p className="text-3xl font-bold text-green-600">
                                        {result.nodesCreated}
                                    </p>
                                    <p className="text-sm text-muted-foreground">Nodes created</p>
                                </div>
                                <div className="bg-muted/50 rounded-lg p-4 text-center">
                                    <p className="text-3xl font-bold">
                                        {(result.durationMs / 1000).toFixed(1)}s
                                    </p>
                                    <p className="text-sm text-muted-foreground">Duration</p>
                                </div>
                            </div>
                        </div>
                    )}

                    {generateState === "error" && (
                        <div className="flex flex-col items-center justify-center h-full gap-4">
                            <AlertCircle className="h-16 w-16 text-destructive" />
                            <div className="text-center max-w-md">
                                <p className="text-lg font-medium text-destructive">Generation failed</p>
                                <p className="text-sm text-muted-foreground mt-2">{error}</p>
                            </div>
                            <Button variant="outline" onClick={() => setGenerateState("idle")}>
                                Try again
                            </Button>
                        </div>
                    )}

                    {generateState === "cancelled" && (
                        <div className="flex flex-col items-center justify-center h-full gap-4">
                            <XCircle className="h-16 w-16 text-muted-foreground" />
                            <div className="text-center max-w-md">
                                <p className="text-lg font-medium">Generation cancelled</p>
                                <p className="text-sm text-muted-foreground mt-2">
                                    The generation was stopped. No changes were made.
                                </p>
                            </div>
                            <Button variant="outline" onClick={() => setGenerateState("idle")}>
                                Start over
                            </Button>
                        </div>
                    )}
                </div>

                <DialogFooter className="flex-shrink-0">
                    {generateState === "idle" && (
                        <>
                            <Button variant="outline" onClick={handleClose}>
                                Cancel
                            </Button>
                            <Button onClick={handleGenerate} disabled={!canGenerate}>
                                <Sparkles className="mr-2 h-4 w-4" />
                                Generate ({parsedTitles.length} titles)
                            </Button>
                        </>
                    )}
                    {(generateState === "success" || generateState === "cancelled") && (
                        <Button onClick={handleClose}>Close</Button>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
