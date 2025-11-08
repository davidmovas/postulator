"use client";

import { useState, useEffect } from "react";
import { HealthCheckSettings, HealthCheckSettingsUpdateInput } from "@/models/settings";
import { settingsService } from "@/services/settings";
import { useApiCall } from "./use-api-call";

export function useHealthCheckSettings() {
    const [settings, setSettings] = useState<HealthCheckSettings | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const { execute } = useApiCall();

    const loadSettings = async () => {
        setIsLoading(true);
        try {
            const data = await settingsService.getHealthCheckSettings();
            setSettings(data);
        } catch (error) {
            console.error("Failed to load health check settings:", error);
        } finally {
            setIsLoading(false);
        }
    };

    const updateSettings = async (input: HealthCheckSettingsUpdateInput) => {
        return await execute<string>(
            () => settingsService.updateHealthCheckSettings(input),
            {
                successMessage: "Health check settings updated successfully",
                showSuccessToast: true,
                onSuccess: () => {
                    if (settings) {
                        setSettings({
                            ...settings,
                            ...input
                        });
                    }
                    loadSettings();
                }
            }
        );
    };

    useEffect(() => {
        loadSettings();
    }, []);

    return {
        settings,
        isLoading,
        loadSettings,
        updateSettings
    };
}