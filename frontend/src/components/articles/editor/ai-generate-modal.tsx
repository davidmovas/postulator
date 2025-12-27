"use client";

import { useState, useCallback, useMemo, useEffect } from "react";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Checkbox } from "@/components/ui/checkbox";
import { Loader2, Sparkles, Globe } from "lucide-react";
import { Provider, Model } from "@/models/providers";
import { Prompt } from "@/models/prompts";
import { Topic } from "@/models/topics";
import { GenerateContentResult } from "@/models/articles";
import { extractPlaceholdersFromPrompts } from "@/lib/prompt-utils";
import { articleService } from "@/services/articles";
import { providerService } from "@/services/providers";
import { promptService } from "@/services/prompts";
import { topicService } from "@/services/topics";
import { toast } from "sonner";

interface AIGenerateModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: number;
    onGenerated: (result: GenerateContentResult) => void;
}

type TopicMode = "existing" | "custom";

const EXCLUDED_PLACEHOLDERS = ["title", "topic", "siteName", "siteUrl"];

export function AIGenerateModal({
    open,
    onOpenChange,
    siteId,
    onGenerated,
}: AIGenerateModalProps) {
    const [providers, setProviders] = useState<Provider[] | null>(null);
    const [prompts, setPrompts] = useState<Prompt[] | null>(null);
    const [topics, setTopics] = useState<Topic[] | null>(null);
    const [modelsMap, setModelsMap] = useState<Record<string, Model[]>>({});

    const [selectedProviderId, setSelectedProviderId] = useState<string>("");
    const [selectedPromptId, setSelectedPromptId] = useState<string>("");
    const [topicMode, setTopicMode] = useState<TopicMode>("existing");
    const [selectedTopicId, setSelectedTopicId] = useState<string>("");
    const [customTopicTitle, setCustomTopicTitle] = useState("");
    const [placeholderValues, setPlaceholderValues] = useState<Record<string, string>>({});
    const [useWebSearch, setUseWebSearch] = useState(false);

    const [isLoading, setIsLoading] = useState(false);
    const [isGenerating, setIsGenerating] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Load data when modal opens
    useEffect(() => {
        if (open) {
            loadData();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [open, siteId]);

    const loadData = async () => {
        setIsLoading(true);
        setError(null);
        try {
            const [providersData, promptsData, topicsData] = await Promise.all([
                providerService.listActiveProviders(),
                promptService.listPrompts(),
                topicService.getUnusedSiteTopics(siteId),
            ]);
            setProviders(providersData);
            setPrompts(promptsData);
            setTopics(topicsData);

            // Load models for all unique provider types
            const uniqueTypes = [...new Set(providersData.map(p => p.type))];
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
            setError("Failed to load data");
            console.error(err);
        } finally {
            setIsLoading(false);
        }
    };

    // Get selected provider and its model info
    const selectedProvider = useMemo(() => {
        if (!selectedProviderId || !providers) return null;
        return providers.find(p => p.id.toString() === selectedProviderId) || null;
    }, [selectedProviderId, providers]);

    const selectedModelInfo = useMemo(() => {
        if (!selectedProvider) return null;
        const models = modelsMap[selectedProvider.type] || [];
        return models.find(m => m.id === selectedProvider.model) || null;
    }, [selectedProvider, modelsMap]);

    const supportsWebSearch = selectedModelInfo?.supportsWebSearch ?? false;

    // Get placeholders from selected prompt
    const selectedPrompt = useMemo(() => {
        if (!selectedPromptId || !prompts) return null;
        return prompts.find(p => p.id.toString() === selectedPromptId) || null;
    }, [selectedPromptId, prompts]);

    const placeholders = useMemo(() => {
        if (!selectedPrompt) return [];
        const keys = extractPlaceholdersFromPrompts(
            selectedPrompt.systemPrompt || "",
            selectedPrompt.userPrompt || ""
        );
        return keys.filter(k => !EXCLUDED_PLACEHOLDERS.includes(k.toLowerCase()));
    }, [selectedPrompt]);

    // Reset form when modal closes
    const handleClose = useCallback(() => {
        setSelectedProviderId("");
        setSelectedPromptId("");
        setTopicMode("existing");
        setSelectedTopicId("");
        setCustomTopicTitle("");
        setPlaceholderValues({});
        setUseWebSearch(false);
        setError(null);
        onOpenChange(false);
    }, [onOpenChange]);

    // Reset web search when provider changes and doesn't support it
    const handleProviderChange = useCallback((providerId: string) => {
        setSelectedProviderId(providerId);
        // Reset web search - will be enabled again if user wants and model supports it
        setUseWebSearch(false);
    }, []);

    const updatePlaceholderValue = (key: string, value: string) => {
        setPlaceholderValues(prev => ({ ...prev, [key]: value }));
    };

    const canGenerate = useMemo(() => {
        if (!selectedProviderId || !selectedPromptId) return false;
        if (topicMode === "existing" && !selectedTopicId) return false;
        if (topicMode === "custom" && !customTopicTitle.trim()) return false;
        // Check all placeholders are filled
        for (const placeholder of placeholders) {
            if (!placeholderValues[placeholder]?.trim()) return false;
        }
        return true;
    }, [selectedProviderId, selectedPromptId, topicMode, selectedTopicId, customTopicTitle, placeholders, placeholderValues]);

    const handleGenerate = async () => {
        if (!canGenerate) return;

        setIsGenerating(true);
        setError(null);

        try {
            const input = {
                siteId,
                providerId: parseInt(selectedProviderId),
                promptId: parseInt(selectedPromptId),
                topicId: topicMode === "existing" ? parseInt(selectedTopicId) : undefined,
                customTopicTitle: topicMode === "custom" ? customTopicTitle.trim() : undefined,
                placeholderValues,
                useWebSearch: supportsWebSearch && useWebSearch,
            };

            const result = await articleService.generateContent(input);
            toast.success("Content generated successfully");
            onGenerated(result);
            handleClose();
        } catch (err: any) {
            const message = err?.message || "Failed to generate content";
            setError(message);
            toast.error(message);
        } finally {
            setIsGenerating(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Sparkles className="h-5 w-5" />
                        Generate Content with AI
                    </DialogTitle>
                    <DialogDescription>
                        Select a provider, prompt, and topic to generate article content
                    </DialogDescription>
                </DialogHeader>

                {isLoading ? (
                    <div className="flex items-center justify-center py-8">
                        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                    </div>
                ) : error && !providers ? (
                    <p className="text-sm text-destructive py-4 text-center">{error}</p>
                ) : (
                    <div className="space-y-6 py-4">
                        {/* Provider Selection */}
                        <div className="space-y-2">
                            <Label>AI Provider</Label>
                            {providers && providers.length === 0 ? (
                                <p className="text-sm text-amber-600 dark:text-amber-500">
                                    No active providers found. Please configure a provider first.
                                </p>
                            ) : (
                                <Select
                                    value={selectedProviderId}
                                    onValueChange={handleProviderChange}
                                >
                                    <SelectTrigger>
                                        <SelectValue placeholder="Select a provider" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {providers?.map((provider) => (
                                            <SelectItem
                                                key={provider.id}
                                                value={provider.id.toString()}
                                            >
                                                {provider.name} ({provider.model})
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            )}
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

                        {/* Prompt Selection */}
                        <div className="space-y-2">
                            <Label>Prompt</Label>
                            {prompts && prompts.length === 0 ? (
                                <p className="text-sm text-amber-600 dark:text-amber-500">
                                    No prompts found. Please create a prompt first.
                                </p>
                            ) : (
                                <Select
                                    value={selectedPromptId}
                                    onValueChange={(value) => {
                                        setSelectedPromptId(value);
                                        setPlaceholderValues({});
                                    }}
                                >
                                    <SelectTrigger>
                                        <SelectValue placeholder="Select a prompt" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {prompts?.map((prompt) => (
                                            <SelectItem
                                                key={prompt.id}
                                                value={prompt.id.toString()}
                                            >
                                                {prompt.name}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            )}
                        </div>

                        {/* Topic Selection */}
                        <div className="space-y-4">
                            <Label>Topic</Label>
                            <RadioGroup
                                value={topicMode}
                                onValueChange={(value) => setTopicMode(value as TopicMode)}
                                className="space-y-3"
                            >
                                <div className="flex items-center space-x-2">
                                    <RadioGroupItem value="existing" id="existing" />
                                    <Label htmlFor="existing" className="flex-1 cursor-pointer">
                                        <div className="font-medium">Select from unused topics</div>
                                        <div className="text-sm text-muted-foreground">
                                            Choose from topics not yet used for this site
                                        </div>
                                    </Label>
                                </div>

                                <div className="flex items-center space-x-2">
                                    <RadioGroupItem value="custom" id="custom" />
                                    <Label htmlFor="custom" className="flex-1 cursor-pointer">
                                        <div className="font-medium">Enter custom topic</div>
                                        <div className="text-sm text-muted-foreground">
                                            Create a new topic and assign it to this site
                                        </div>
                                    </Label>
                                </div>
                            </RadioGroup>

                            {topicMode === "existing" && (
                                <div className="pl-6">
                                    {topics && topics.length === 0 ? (
                                        <p className="text-sm text-amber-600 dark:text-amber-500">
                                            No unused topics available. Enter a custom topic instead.
                                        </p>
                                    ) : (
                                        <Select
                                            value={selectedTopicId}
                                            onValueChange={setSelectedTopicId}
                                        >
                                            <SelectTrigger>
                                                <SelectValue placeholder="Select a topic" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                {topics?.map((topic) => (
                                                    <SelectItem
                                                        key={topic.id}
                                                        value={topic.id.toString()}
                                                    >
                                                        {topic.title}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                    )}
                                </div>
                            )}

                            {topicMode === "custom" && (
                                <div className="pl-6">
                                    <Input
                                        value={customTopicTitle}
                                        onChange={(e) => setCustomTopicTitle(e.target.value)}
                                        placeholder="Enter topic title"
                                    />
                                    <p className="text-xs text-muted-foreground mt-1">
                                        This topic will be created and assigned to the site
                                    </p>
                                </div>
                            )}
                        </div>

                        {/* Placeholders */}
                        {selectedPrompt && placeholders.length > 0 && (
                            <div className="space-y-4 border-t pt-4">
                                <div>
                                    <Label className="text-base">Prompt Placeholders</Label>
                                    <p className="text-sm text-muted-foreground mt-1">
                                        Fill in the values for placeholders used in the prompt
                                    </p>
                                </div>

                                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                    {placeholders.map((placeholder) => (
                                        <div key={placeholder} className="space-y-2">
                                            <div className="text-xs font-mono inline-flex items-center px-2 py-1 rounded bg-muted text-muted-foreground/90 w-fit">
                                                {placeholder}
                                            </div>
                                            <Input
                                                placeholder={`Enter value for ${placeholder}`}
                                                value={placeholderValues[placeholder] || ""}
                                                onChange={(e) =>
                                                    updatePlaceholderValue(placeholder, e.target.value)
                                                }
                                            />
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        {selectedPrompt && placeholders.length === 0 && (
                            <div className="text-center py-4 text-muted-foreground text-sm border-t pt-4">
                                No additional placeholders to fill (standard fields are auto-filled)
                            </div>
                        )}

                        {/* Error display */}
                        {error && (
                            <p className="text-sm text-destructive">{error}</p>
                        )}
                    </div>
                )}

                <DialogFooter>
                    <Button variant="outline" onClick={handleClose} disabled={isGenerating}>
                        Cancel
                    </Button>
                    <Button
                        onClick={handleGenerate}
                        disabled={!canGenerate || isGenerating || isLoading}
                    >
                        {isGenerating ? (
                            <>
                                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                                Generating...
                            </>
                        ) : (
                            <>
                                <Sparkles className="h-4 w-4 mr-2" />
                                Generate
                            </>
                        )}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
