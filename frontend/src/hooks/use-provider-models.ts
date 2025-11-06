"use client";

import { useState } from "react";
import { Model } from "@/models/providers";
import { providerService } from "@/services/providers";
import { useApiCall } from "./use-api-call";

export function useProviderModels() {
    const [availableModels, setAvailableModels] = useState<Model[]>([]);
    const [modelsLoading, setModelsLoading] = useState(false);
    const { execute } = useApiCall();

    const loadModels = async (providerType: string) => {
        if (!providerType) {
            setAvailableModels([]);
            return;
        }

        setModelsLoading(true);

        await execute<Model[]>(
            () => providerService.getAvailableModels(providerType),
            {
                onSuccess: (models) => {
                    setAvailableModels(models);
                },
                onError: () => {
                    setAvailableModels([]);
                }
            }
        );

        setModelsLoading(false);
    };

    return {
        availableModels,
        modelsLoading,
        loadModels
    };
}