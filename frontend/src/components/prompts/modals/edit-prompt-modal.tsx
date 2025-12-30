"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useApiCall } from "@/hooks/use-api-call";
import { promptService } from "@/services/prompts";
import { Prompt, PromptUpdateInput, PromptCategory, PROMPT_CATEGORIES, ContextConfig, isV2Prompt } from "@/models/prompts";
import { ContextConfigEditor } from "@/components/prompts/context-config/context-config-editor";
import { FileText, FolderOpen, Lock, Settings2 } from "lucide-react";

interface EditPromptModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    prompt: Prompt | null;
    onSuccess?: () => void;
}

export function EditPromptModal({ open, onOpenChange, prompt, onSuccess }: EditPromptModalProps) {
    const { execute, isLoading } = useApiCall();

    const [formData, setFormData] = useState<PromptUpdateInput>({
        id: 0,
        name: "",
        category: "post_gen",
        version: 2,
        instructions: "",
        contextConfig: {},
    });

    useEffect(() => {
        if (prompt) {
            setFormData({
                id: prompt.id,
                name: prompt.name,
                category: prompt.category,
                version: prompt.version || 2,
                instructions: prompt.instructions || "",
                contextConfig: prompt.contextConfig || {},
                // Keep legacy fields for backward compatibility
                systemPrompt: prompt.systemPrompt,
                userPrompt: prompt.userPrompt,
                placeholders: prompt.placeholders,
            });
        }
    }, [prompt]);

    const resetForm = () => {
        setFormData({
            id: 0,
            name: "",
            category: "post_gen",
            version: 2,
            instructions: "",
            contextConfig: {},
        });
    };

    const isFormValid = formData.name?.trim() && formData.instructions?.trim();

    const handleSubmit = async () => {
        if (!isFormValid || !prompt) return;

        const result = await execute<void>(
            () => promptService.updatePrompt(formData),
            {
                successMessage: "Prompt updated successfully",
                showSuccessToast: true
            }
        );

        if (result !== null) {
            onOpenChange(false);
            resetForm();
            onSuccess?.();
        }
    };

    const handleOpenChange = (newOpen: boolean) => {
        if (!newOpen) {
            resetForm();
        }
        onOpenChange(newOpen);
    };

    const handleCategoryChange = (value: PromptCategory) => {
        setFormData(prev => ({
            ...prev,
            category: value,
            contextConfig: {}, // Reset config when category changes
        }));
    };

    const handleContextConfigChange = (config: ContextConfig) => {
        setFormData(prev => ({
            ...prev,
            contextConfig: config,
        }));
    };

    if (!prompt) return null;

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Edit Prompt</DialogTitle>
                    <DialogDescription>
                        Update your AI prompt instructions and context configuration.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-6 py-4">
                    {/* Builtin Warning */}
                    {prompt.isBuiltin && (
                        <Alert>
                            <Lock className="h-4 w-4" />
                            <AlertDescription>
                                This is a builtin prompt. You can customize its content, but cannot delete it or change its category.
                            </AlertDescription>
                        </Alert>
                    )}

                    {/* Name and Category Row */}
                    <div className="grid grid-cols-2 gap-4">
                        {/* Name */}
                        <div className="space-y-2">
                            <Label htmlFor="edit-name">Prompt Name</Label>
                            <Input
                                id="edit-name"
                                placeholder="e.g., SEO Article Writer"
                                value={formData.name || ""}
                                onChange={(e) => setFormData(prev => ({
                                    ...prev,
                                    name: e.target.value
                                }))}
                                disabled={isLoading}
                            />
                        </div>

                        {/* Category */}
                        <div className="space-y-2">
                            <div className="flex items-center gap-2">
                                <FolderOpen className="h-4 w-4 text-muted-foreground" />
                                <Label htmlFor="edit-category">Category</Label>
                            </div>
                            <Select
                                value={formData.category}
                                onValueChange={handleCategoryChange}
                                disabled={isLoading || prompt.isBuiltin}
                            >
                                <SelectTrigger>
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    {Object.entries(PROMPT_CATEGORIES).map(([value, label]) => (
                                        <SelectItem key={value} value={value}>
                                            {label}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                    </div>

                    {/* Instructions */}
                    <div className="space-y-2">
                        <div className="flex items-center gap-2">
                            <FileText className="h-4 w-4 text-muted-foreground" />
                            <Label htmlFor="edit-instructions">Instructions</Label>
                        </div>
                        <Textarea
                            id="edit-instructions"
                            placeholder="You are an SEO copywriter. Generate engaging content that is optimized for search engines..."
                            value={formData.instructions || ""}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                instructions: e.target.value
                            }))}
                            rows={10}
                            disabled={isLoading}
                            className="resize-y min-h-[200px] font-mono text-sm"
                        />
                        <p className="text-xs text-muted-foreground">
                            Write your instructions for the AI. This becomes the system prompt.
                            Focus on describing the desired behavior, style, and output format.
                        </p>
                    </div>

                    <Separator />

                    {/* Context Configuration */}
                    <div className="space-y-3">
                        <div className="flex items-center gap-2">
                            <Settings2 className="h-4 w-4 text-muted-foreground" />
                            <Label>Context Fields</Label>
                        </div>
                        <p className="text-sm text-muted-foreground">
                            Configure which data is included when using this prompt.
                            These settings can be overridden at usage time.
                        </p>
                        <div className="border rounded-lg p-4 bg-muted/30">
                            <ContextConfigEditor
                                category={formData.category || "post_gen"}
                                config={formData.contextConfig || {}}
                                onChange={handleContextConfigChange}
                                disabled={isLoading}
                            />
                        </div>
                    </div>
                </div>

                <DialogFooter>
                    <Button
                        variant="outline"
                        onClick={() => handleOpenChange(false)}
                        disabled={isLoading}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        disabled={!isFormValid || isLoading}
                    >
                        {isLoading ? "Updating..." : "Update Prompt"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
