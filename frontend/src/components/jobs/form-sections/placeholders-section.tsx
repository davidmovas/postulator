"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { JobCreateInput } from "@/models/jobs";
import { extractPlaceholdersFromPrompts } from "@/lib/prompt-utils";
import { useMemo } from "react";

interface PlaceholdersSectionProps {
    formData: Partial<JobCreateInput>;
    onUpdate: (updates: Partial<JobCreateInput>) => void;
    prompts: any[] | null;
}

export function PlaceholdersSection({ formData, onUpdate, prompts }: PlaceholdersSectionProps) {
    const selectedPrompt = prompts?.find(p => p.id === formData.promptId);

    const EXCLUDED_KEYS = ["title", "topic"]; // These will be filled dynamically during execution

    const placeholders = useMemo(() => {
        if (!selectedPrompt) return [];
        const keys = extractPlaceholdersFromPrompts(
            selectedPrompt.systemPrompt,
            selectedPrompt.userPrompt
        );
        return keys.filter(k => !EXCLUDED_KEYS.includes(k.toLowerCase()));
    }, [selectedPrompt]);

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