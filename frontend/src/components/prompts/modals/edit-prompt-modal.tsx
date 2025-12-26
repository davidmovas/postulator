"use client";

import { useState, useMemo, useEffect } from "react";
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
import { Badge } from "@/components/ui/badge";
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
import { Prompt, PromptUpdateInput, PromptCategory, PROMPT_CATEGORIES } from "@/models/prompts";
import { extractPlaceholdersFromPrompts } from "@/lib/prompt-utils";
import { MessageSquare, User, Tag, FolderOpen, Lock } from "lucide-react";

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
        systemPrompt: "",
        userPrompt: "",
        placeholders: []
    });

    const detectedPlaceholders = useMemo(() => {
        return extractPlaceholdersFromPrompts(formData.systemPrompt || "", formData.userPrompt || "");
    }, [formData.systemPrompt, formData.userPrompt]);

    useEffect(() => {
        if (prompt) {
            setFormData({
                id: prompt.id,
                name: prompt.name,
                category: prompt.category,
                systemPrompt: prompt.systemPrompt,
                userPrompt: prompt.userPrompt,
                placeholders: prompt.placeholders
            });
        }
    }, [prompt]);

    const resetForm = () => {
        setFormData({
            id: 0,
            name: "",
            category: "post_gen",
            systemPrompt: "",
            userPrompt: "",
            placeholders: []
        });
    };

    const isFormValid = formData.name?.trim() &&
        formData.systemPrompt?.trim() &&
        formData.userPrompt?.trim();

    const handleSubmit = async () => {
        if (!isFormValid || !prompt) return;

        const result = await execute<void>(
            () => promptService.updatePrompt({
                ...formData,
                placeholders: detectedPlaceholders
            }),
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

    if (!prompt) return null;

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Edit Prompt</DialogTitle>
                    <DialogDescription>
                        Update your AI prompt. Placeholders like {"{{variable}}"} will be automatically detected.
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
                                placeholder="e.g., Article Generator"
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
                                onValueChange={(value: PromptCategory) => setFormData(prev => ({
                                    ...prev,
                                    category: value
                                }))}
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

                    {/* System Prompt */}
                    <div className="space-y-2">
                        <div className="flex items-center gap-2">
                            <MessageSquare className="h-4 w-4 text-muted-foreground" />
                            <Label htmlFor="edit-systemPrompt">System Prompt</Label>
                        </div>
                        <Textarea
                            id="edit-systemPrompt"
                            placeholder="You are a helpful assistant that generates content..."
                            value={formData.systemPrompt || ""}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                systemPrompt: e.target.value
                            }))}
                            rows={12}
                            disabled={isLoading}
                            className="resize-y min-h-[200px] font-mono text-sm"
                        />
                    </div>

                    {/* User Prompt */}
                    <div className="space-y-2">
                        <div className="flex items-center gap-2">
                            <User className="h-4 w-4 text-muted-foreground" />
                            <Label htmlFor="edit-userPrompt">User Prompt</Label>
                        </div>
                        <Textarea
                            id="edit-userPrompt"
                            placeholder="Write an article about {{topic}} with {{wordCount}} words..."
                            value={formData.userPrompt || ""}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                userPrompt: e.target.value
                            }))}
                            rows={12}
                            disabled={isLoading}
                            className="resize-y min-h-[200px] font-mono text-sm"
                        />
                    </div>

                    {/* Detected Placeholders */}
                    {detectedPlaceholders.length > 0 && (
                        <div className="space-y-2">
                            <div className="flex items-center gap-2">
                                <Tag className="h-4 w-4 text-muted-foreground" />
                                <Label>Detected Placeholders</Label>
                            </div>
                            <div className="flex flex-wrap gap-2 p-3 bg-muted/50 rounded-md">
                                {detectedPlaceholders.map((placeholder, index) => (
                                    <Badge
                                        key={index}
                                        variant="secondary"
                                        className="font-mono text-xs"
                                    >
                                        {placeholder}
                                    </Badge>
                                ))}
                            </div>
                            <p className="text-xs text-muted-foreground">
                                Placeholders are automatically detected from {"{{variable}}"} patterns in your prompts
                            </p>
                        </div>
                    )}
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