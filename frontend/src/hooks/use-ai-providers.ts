"use client";

import { useState, useEffect } from "react";
import { Provider } from "@/models/providers";
import { providerService } from "@/services/providers";
import { useApiCall } from "./use-api-call";
import { useContextModal } from "@/context/modal-context";

export function useAiProviders() {
    const [providers, setProviders] = useState<Provider[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const { execute } = useApiCall();

    const {
        editProviderModal,
        confirmationModal
    } = useContextModal();

    const loadProviders = async () => {
        setIsLoading(true);
        try {
            const data = await providerService.listProviders();
            setProviders(data);
        } catch (error) {
            console.error("Failed to load providers:", error);
        } finally {
            setIsLoading(false);
        }
    };

    const toggleProviderStatus = async (provider: Provider) => {
        await execute(
            () => providerService.setProviderStatus(provider.id, !provider.isActive),
            {
                successMessage: `Provider ${!provider.isActive ? "activated" : "deactivated"} successfully`,
                showSuccessToast: true,
                onSuccess: loadProviders
            }
        );
    };

    const handleEditProvider = async (provider: Provider) => {
        editProviderModal.open(provider);
    };

    const handleDeleteProvider = async (provider: Provider) => {
        confirmationModal.open({
            title: "Delete Provider",
            description: `Are you sure you want to delete "${provider.name}"? This action cannot be undone.`,
            confirmText: "Delete",
            cancelText: "Cancel",
            variant: "destructive",
            onConfirm: async () => {
                await execute(
                    () => providerService.deleteProvider(provider.id),
                    {
                        showSuccessToast: false,
                        onSuccess: loadProviders
                    }
                );
            }
        });
    };

    useEffect(() => {
        loadProviders();
    }, []);

    return {
        providers,
        isLoading,
        loadProviders,
        toggleProviderStatus,
        handleDeleteProvider,
        handleEditProvider
    };
}