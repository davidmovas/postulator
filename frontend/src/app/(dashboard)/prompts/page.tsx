"use client";

import { Button } from "@/components/ui/button";
import { PromptsList } from "@/components/prompts/prompts-list";
import { usePrompts } from "@/hooks/use-prompts";
import { Plus } from "lucide-react";
import { useContextModal } from "@/context/modal-context";
import { CreatePromptModal } from "@/components/prompts/modals/create-prompt-modal";
import { EditPromptModal } from "@/components/prompts/modals/edit-prompt-modal";
import { RiRefreshLine } from "@remixicon/react";

export default function PromptsPage() {
    const {
        prompts,
        isLoading,
        loadPrompts,
        handleEditPrompt,
        handleDeletePrompt
    } = usePrompts();

    const { createPromptModal, editPromptModal } = useContextModal();

    const handleCreatePrompt = () => {
        createPromptModal.open();
    };

    return (
        <div className="p-6 space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">Prompts</h1>
                    <p className="text-muted-foreground mt-2">
                        Manage your AI prompts for content generation
                    </p>
                </div>

                <div className="flex items-center gap-3">
                    <Button
                        variant="outline"
                        onClick={loadPrompts}
                        disabled={isLoading}
                    >
                        <RiRefreshLine className="w-4 h-4" />
                        Refresh
                    </Button>

                    <Button onClick={handleCreatePrompt}>
                        <Plus className="h-4 w-4 mr-2" />
                        Add Prompt
                    </Button>
                </div>
            </div>

            <div className="flex items-center gap-4 py-4">
                <div className="text-sm text-muted-foreground">
                    {prompts.length} prompt{prompts.length !== 1 ? 's' : ''} configured
                </div>
                <div className="flex-1 border-t" />
            </div>

            <PromptsList
                prompts={prompts}
                onEdit={handleEditPrompt}
                onDelete={handleDeletePrompt}
                isLoading={isLoading}
            />

            {/* Modals */}
            <CreatePromptModal
                open={createPromptModal.isOpen}
                onOpenChange={(open) => (open ? createPromptModal.open() : createPromptModal.close())}
                onSuccess={loadPrompts}
            />

            <EditPromptModal
                open={editPromptModal.isOpen}
                onOpenChange={(open) => (open ? editPromptModal.open(editPromptModal.prompt!) : editPromptModal.close())}
                prompt={editPromptModal.prompt}
                onSuccess={loadPrompts}
            />
        </div>
    );
}