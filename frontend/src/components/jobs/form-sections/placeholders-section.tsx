"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { JobCreateInput } from "@/models/jobs";
import { extractPlaceholdersFromPrompts } from "@/lib/prompt-utils";
import { isV2Prompt, Prompt, ContextConfig } from "@/models/prompts";
import { ContextConfigEditor } from "@/components/prompts/context-config/context-config-editor";
import { useMemo, useState, useEffect } from "react";

interface PlaceholdersSectionProps {
    formData: Partial<JobCreateInput>;
    onUpdate: (updates: Partial<JobCreateInput>) => void;
    prompts: Prompt[] | null;
}

export function PlaceholdersSection({ formData, onUpdate, prompts }: PlaceholdersSectionProps) {
    const selectedPrompt = prompts?.find(p => p.id === formData.promptId);

    const EXCLUDED_KEYS = ["title", "topic"]; // These will be filled dynamically during execution

    // For v2 prompts, placeholders are not needed - context fields are configured in the prompt
    const isV2 = selectedPrompt && isV2Prompt(selectedPrompt);

    // Context overrides for v2 prompts
    const [contextOverrides, setContextOverrides] = useState<ContextConfig>({});
    // Additional instructions for v2 prompts (job-specific, not saved to prompt)
    const [customInstructions, setCustomInstructions] = useState<string>("");

    // Initialize context overrides and custom instructions when prompt changes
    useEffect(() => {
        if (selectedPrompt && isV2Prompt(selectedPrompt) && selectedPrompt.contextConfig) {
            setContextOverrides(selectedPrompt.contextConfig);
        } else {
            setContextOverrides({});
        }
        // Initialize customInstructions from existing formData if available
        setCustomInstructions(formData.placeholdersValues?.customInstructions || "");
    }, [selectedPrompt?.id]);

    // Handle context config changes - convert to placeholdersValues for backend compatibility
    const handleContextConfigChange = (config: ContextConfig) => {
        setContextOverrides(config);
        updatePlaceholdersFromConfig(config, customInstructions);
    };

    // Handle custom instructions changes
    const handleCustomInstructionsChange = (value: string) => {
        setCustomInstructions(value);
        updatePlaceholdersFromConfig(contextOverrides, value);
    };

    // Convert context config and custom instructions to placeholdersValues format
    const updatePlaceholdersFromConfig = (config: ContextConfig, instructions: string) => {
        const placeholderValues: Record<string, string> = {};
        Object.entries(config).forEach(([key, value]) => {
            if (value.enabled && value.value) {
                placeholderValues[key] = value.value;
            }
        });
        // Add custom instructions if provided
        if (instructions.trim()) {
            placeholderValues.customInstructions = instructions.trim();
        }
        onUpdate({ placeholdersValues: placeholderValues });
    };

    const placeholders = useMemo(() => {
        if (!selectedPrompt || isV2) return [];
        const keys = extractPlaceholdersFromPrompts(
            selectedPrompt.systemPrompt || "",
            selectedPrompt.userPrompt || ""
        );
        return keys.filter(k => !EXCLUDED_KEYS.includes(k.toLowerCase()));
    }, [selectedPrompt, isV2]);

    const updatePlaceholderValue = (placeholder: string, value: string) => {
        const currentValues = formData.placeholdersValues || {};
        onUpdate({
            placeholdersValues: {
                ...currentValues,
                [placeholder]: value
            }
        });
    };

    if (!selectedPrompt) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Prompt Placeholders</CardTitle>
                    <CardFooter>
                        Select a prompt to configure placeholder values
                    </CardFooter>
                </CardHeader>
                <CardContent>
                    <div className="text-center py-6 text-muted-foreground">
                        Please select a prompt first
                    </div>
                </CardContent>
            </Card>
        );
    }

    // For v2 prompts, show ContextConfigEditor
    if (isV2 && selectedPrompt) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Content Settings</CardTitle>
                    <CardFooter>
                        Configure settings for content generation. These are based on the prompt&apos;s context configuration.
                    </CardFooter>
                </CardHeader>
                <CardContent className="space-y-6">
                    <div className="p-3 bg-muted/50 rounded-md">
                        <div className="font-medium">{selectedPrompt.name}</div>
                        <div className="text-sm text-muted-foreground mt-1">
                            Adjust values below to customize generation
                        </div>
                    </div>
                    <ContextConfigEditor
                        category="post_gen"
                        mode="override"
                        baseConfig={selectedPrompt.contextConfig}
                        config={contextOverrides}
                        onChange={handleContextConfigChange}
                    />

                    <Separator />

                    {/* Additional Instructions - job-specific */}
                    <div className="space-y-3">
                        <div>
                            <Label htmlFor="customInstructions" className="text-sm font-medium">
                                Additional Instructions
                                <span className="text-muted-foreground font-normal ml-2">(Optional)</span>
                            </Label>
                            <p className="text-xs text-muted-foreground mt-1">
                                Add temporary instructions specific to this job. These will be appended to the prompt and won&apos;t affect the base prompt template.
                            </p>
                        </div>
                        <Textarea
                            id="customInstructions"
                            placeholder="e.g., Focus on technical details, use a formal tone, include code examples..."
                            value={customInstructions}
                            onChange={(e) => handleCustomInstructionsChange(e.target.value)}
                            rows={3}
                            className="resize-none"
                        />
                    </div>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Prompt Placeholders</CardTitle>
                <CardFooter>
                    Set values for placeholders used in the selected prompt. Note: the following fields are filled automatically during job execution and do not need manual input: <span className="font-mono">{EXCLUDED_KEYS.join(", ")}</span>
                </CardFooter>
            </CardHeader>
            <CardContent className="space-y-4">
                {/* Selected Prompt Info */}
                <div className="p-3 bg-muted/50 rounded-md">
                    <div className="font-medium">{selectedPrompt.name}</div>
                    <div className="text-sm text-muted-foreground mt-1">
                        {placeholders.length} placeholder{placeholders.length !== 1 ? 's' : ''} to fill
                    </div>
                </div>

                {/* Placeholder Inputs */}
                {placeholders.length > 0 ? (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        {placeholders.map((placeholder, index) => (
                            <div key={index} className="space-y-2">
                                <div className="text-xs font-mono inline-flex items-center px-2 py-1 rounded bg-muted text-muted-foreground/90 w-fit">
                                    {placeholder}
                                </div>
                                <Input
                                    id={`placeholder-${placeholder}`}
                                    placeholder={`Enter value for ${placeholder}`}
                                    value={formData.placeholdersValues?.[placeholder] || ""}
                                    onChange={(e) => updatePlaceholderValue(placeholder, e.target.value)}
                                />
                            </div>
                        ))}
                    </div>
                ) : (
                    <div className="text-center py-6 text-muted-foreground">
                        No placeholders to fill (some fields are auto-filled during execution)
                    </div>
                )}
            </CardContent>
        </Card>
    );
}