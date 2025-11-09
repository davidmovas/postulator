"use client";

import { useState, useEffect } from "react";
import { Prompt } from "@/models/prompts";
import { promptService } from "@/services/prompts";
import { useContextModal } from "@/context/modal-context";

export function usePrompts() {
    const [prompts, setPrompts] = useState<Prompt[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const { editPromptModal, confirmationModal } = useContextModal();

    const loadPrompts = async () => {
        setIsLoading(true);
        try {
            const data = await promptService.listPrompts();
            setPrompts(data);
        } catch (error) {
            console.error("Failed to load prompts:", error);
        } finally {
            setIsLoading(false);
        }
    };

    const handleEditPrompt = (prompt: Prompt) => {
        editPromptModal.open(prompt);
    };

    const handleDeletePrompt = (prompt: Prompt) => {
        confirmationModal.open({
            title: "Delete prompt?",
            description: `This action cannot be undone. This will permanently delete the prompt "${prompt.name}".`,
            confirmText: "Delete",
            cancelText: "Cancel",
            variant: "destructive",
            onConfirm: async () => {
                await promptService.deletePrompt(prompt.id);
                await loadPrompts();
            }
        });
    };

    useEffect(() => {
        loadPrompts();
    }, []);

    return {
        prompts,
        isLoading,
        loadPrompts,
        handleEditPrompt,
        handleDeletePrompt
    };
}