import { useState, useEffect, useCallback } from "react";
import { settingsService } from "@/services/settings";
import { DashboardSettings, DashboardSettingsUpdateInput } from "@/models/settings";
import { useApiCall } from "@/hooks/use-api-call";

export function useDashboardSettings() {
    const [settings, setSettings] = useState<DashboardSettings | null>(null);
    const { execute, isLoading } = useApiCall();

    const loadSettings = useCallback(async () => {
        const result = await execute<DashboardSettings>(
            () => settingsService.getDashboardSettings(),
            {
                showSuccessToast: false,
                errorTitle: "Failed to load dashboard settings"
            }
        );

        if (result) {
            setSettings(result);
        }
    }, [execute]);

    const updateSettings = useCallback(async (input: DashboardSettingsUpdateInput) => {
        if (!settings) return;

        await execute<string>(
            () => settingsService.updateDashboardSettings(input, settings),
            {
                showSuccessToast: false,
                errorTitle: "Failed to update dashboard settings"
            }
        );

        // Update local state optimistically
        setSettings(prev => prev ? { ...prev, ...input } : prev);
    }, [execute, settings]);

    useEffect(() => {
        loadSettings();
    }, [loadSettings]);

    return {
        settings,
        isLoading: isLoading && !settings,
        updateSettings,
        refresh: loadSettings
    };
}
