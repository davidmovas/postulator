"use client";

import { Button } from "@/components/ui/button";
import { useAiProviders } from "@/hooks/use-ai-providers";
import { Plus, RefreshCw } from "lucide-react";
import { useContextModal } from "@/context/modal-context";
import { ProvidersList } from "@/components/ai-providers/providers-list";
import { CreateProviderModal } from "@/components/ai-providers/modals/create-provider-modal";
import { EditProviderModal } from "@/components/ai-providers/modals/edit-provider-modal";
import { ConfirmationModal } from "@/modals/confirm-modal";

export default function ProvidersPage() {
    const {
        providers,
        isLoading,
        loadProviders,
        toggleProviderStatus,
        handleDeleteProvider,
        handleEditProvider
    } = useAiProviders();

    const {
        createProviderModal,
        editProviderModal,
        confirmationModal
    } = useContextModal();

    const handleCreateProvider = () => {
        createProviderModal.open();
    };

    return (
        <div className="p-6 space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">AI Providers</h1>
                    <p className="text-muted-foreground mt-2">
                        Manage your AI model providers and configurations
                    </p>
                </div>

                <div className="flex items-center gap-3">
                    <Button
                        variant="outline"
                        onClick={loadProviders}
                        disabled={isLoading}
                    >
                        <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
                        Refresh
                    </Button>

                    <Button onClick={handleCreateProvider}>
                        <Plus className="h-4 w-4 mr-2" />
                        Add Provider
                    </Button>
                </div>
            </div>

            <div className="flex items-center gap-4 py-4">
                <div className="text-sm text-muted-foreground">
                    {providers.length} provider{providers.length !== 1 ? 's' : ''} configured
                </div>
                <div className="flex-1 border-t" />
            </div>

            <ProvidersList
                providers={providers}
                onEdit={handleEditProvider}
                onDelete={handleDeleteProvider}
                onToggleStatus={toggleProviderStatus}
                isLoading={isLoading}
            />

            {/* Modals */}
            <CreateProviderModal
                open={createProviderModal.isOpen}
                onOpenChange={createProviderModal.close}
                onSuccess={loadProviders}
            />

            <EditProviderModal
                open={editProviderModal.isOpen}
                onOpenChange={editProviderModal.close}
                provider={editProviderModal.provider}
                onSuccess={loadProviders}
            />

            <ConfirmationModal
                open={confirmationModal.isOpen}
                onOpenChange={confirmationModal.close}
                data={confirmationModal.data}
            />
        </div>
    );
}