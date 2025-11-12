"use client";

import { useState, useMemo } from "react";
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
import { useApiCall } from "@/hooks/use-api-call";
import { promptService } from "@/services/prompts";
import { PromptCreateInput } from "@/models/prompts";
import { extractPlaceholdersFromPrompts } from "@/lib/prompt-utils";
import { MessageSquare, User, Tag } from "lucide-react";

interface CreatePromptModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSuccess?: () => void;
}

export function CreatePromptModal({ open, onOpenChange, onSuccess }: CreatePromptModalProps) {
    const { execute, isLoading } = useApiCall();

    const [formData, setFormData] = useState<PromptCreateInput>({
        name: "",
        systemPrompt: "",
        userPrompt: "",
        placeholders: []
    });

    const detectedPlaceholders = useMemo(() => {
        return extractPlaceholdersFromPrompts(formData.systemPrompt, formData.userPrompt);
    }, [formData.systemPrompt, formData.userPrompt]);

    const resetForm = () => {
        setFormData({
            name: "",
            systemPrompt: "",
            userPrompt: "",
            placeholders: []
        });
    };

    const isFormValid = formData.name.trim() &&
        formData.systemPrompt.trim() &&
        formData.userPrompt.trim();

    const handleSubmit = async () => {
        if (!isFormValid) return;

        const result = await execute<void>(
            () => promptService.createPrompt({
                ...formData,
                placeholders: detectedPlaceholders // Используем автоматически найденные плейсхолдеры
            }),
            {
                successMessage: "Prompt created successfully",
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

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Create New Prompt</DialogTitle>
                    <DialogDescription>
                        Create a new AI prompt with system and user instructions. Placeholders like {"{{variable}}"} will be automatically detected.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-6 py-4">
                    {/* Name */}
                    <div className="space-y-2">
                        <Label htmlFor="name">Prompt Name</Label>
                        <Input
                            id="name"
                            placeholder="e.g., Article Generator, Social Media Post"
                            value={formData.name}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                name: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    {/* System Prompt */}
                    <div className="space-y-2">
                        <div className="flex items-center gap-2">
                            <MessageSquare className="h-4 w-4 text-muted-foreground" />
                            <Label htmlFor="systemPrompt">System Prompt</Label>
                        </div>
                        <Textarea
                            id="systemPrompt"
                            placeholder="You are a helpful assistant that generates content..."
                            value={formData.systemPrompt}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                systemPrompt: e.target.value
                            }))}
                            rows={8}
                            disabled={isLoading}
                            className="resize-none font-mono text-sm"
                        />
                    </div>

                    {/* User Prompt */}
                    <div className="space-y-2">
                        <div className="flex items-center gap-2">
                            <User className="h-4 w-4 text-muted-foreground" />
                            <Label htmlFor="userPrompt">User Prompt</Label>
                        </div>
                        <Textarea
                            id="userPrompt"
                            placeholder="Write an article about {{topic}} with {{wordCount}} words..."
                            value={formData.userPrompt}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                userPrompt: e.target.value
                            }))}
                            rows={8}
                            disabled={isLoading}
                            className="resize-none font-mono text-sm"
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
                        {isLoading ? "Creating..." : "Create Prompt"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}