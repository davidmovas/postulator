"use client";

import { Prompt } from "@/models/prompts";
import { PromptCard } from "./prompt-card";

interface PromptsListProps {
    prompts: Prompt[];
    onEdit: (prompt: Prompt) => void;
    onDelete: (prompt: Prompt) => void;
    isLoading?: boolean;
}

export function PromptsList({ prompts, onEdit, onDelete, isLoading = false }: PromptsListProps) {
    if (isLoading) {
        return (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {[...Array(6)].map((_, i) => (
                    <div key={i} className="animate-pulse">
                        <div className="h-80 bg-muted rounded-lg"></div>
                    </div>
                ))}
            </div>
        );
    }

    if (prompts.length === 0) {
        return (
            <div className="text-center py-12">
                <div className="text-muted-foreground">
                    No prompts found. Create your first prompt to get started.
                </div>
            </div>
        );
    }

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {prompts.map((prompt) => (
                <PromptCard
                    key={prompt.id}
                    prompt={prompt}
                    onEdit={onEdit}
                    onDelete={onDelete}
                />
            ))}
        </div>
    );
}