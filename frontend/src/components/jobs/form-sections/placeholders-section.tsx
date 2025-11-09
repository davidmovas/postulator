"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { JobCreateInput } from "@/models/jobs";
import { extractPlaceholdersFromPrompts } from "@/lib/prompt-utils";
import { useMemo } from "react";

interface PlaceholdersSectionProps {
    formData: Partial<JobCreateInput>;
    onUpdate: (updates: Partial<JobCreateInput>) => void;
    prompts: any[];
}

export function PlaceholdersSection({ formData, onUpdate, prompts }: PlaceholdersSectionProps) {
    const selectedPrompt = prompts.find(p => p.id === formData.promptId);

    const placeholders = useMemo(() => {
        if (!selectedPrompt) return [];
        return extractPlaceholdersFromPrompts(
            selectedPrompt.systemPrompt,
            selectedPrompt.userPrompt
        );
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
                    Set values for placeholders used in the selected prompt
                </CardFooter>
            </CardHeader>
            <CardContent className="space-y-4">
                {/* Selected Prompt Info */}
                <div className="p-3 bg-muted/50 rounded-md">
                    <div className="font-medium">{selectedPrompt.name}</div>
                    <div className="text-sm text-muted-foreground mt-1">
                        {placeholders.length} placeholder{placeholders.length !== 1 ? 's' : ''} detected
                    </div>
                </div>

                {/* Placeholder Inputs */}
                {placeholders.length > 0 ? (
                    <div className="space-y-4">
                        {placeholders.map((placeholder, index) => (
                            <div key={index} className="space-y-2">
                                <Label htmlFor={`placeholder-${placeholder}`}>
                                    <Badge variant="secondary" className="font-mono mr-2">
                                        {placeholder}
                                    </Badge>
                                    Value
                                </Label>
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
                        No placeholders found in the selected prompt
                    </div>
                )}
            </CardContent>
        </Card>
    );
}