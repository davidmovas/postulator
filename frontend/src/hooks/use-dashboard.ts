import { useState, useEffect, useCallback } from "react";
import { statsService } from "@/services/stats";
import { settingsService } from "@/services/settings";
import { DashboardSummary } from "@/models/stats";
import { DashboardSettings } from "@/models/settings";
import { useApiCall } from "@/hooks/use-api-call";

export function useDashboard() {
    const [data, setData] = useState<DashboardSummary | null>(null);
    const [settings, setSettings] = useState<DashboardSettings | null>(null);
    const { execute, isLoading, error } = useApiCall();
    const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

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

    const loadData = useCallback(async () => {
        const result = await execute<DashboardSummary>(
            () => statsService.getDashboardSummary(),
            {
                showSuccessToast: false,
                errorTitle: "Failed to refresh dashboard"
            }
        );

        if (result) {
            setData(result);
            setLastUpdated(new Date());
        }
    }, [execute]);

    // Load settings on mount
    useEffect(() => {
        loadSettings();
    }, [loadSettings]);

    // Auto-refresh based on settings
    useEffect(() => {
        if (!settings?.autoRefreshEnabled) {
            loadData(); // Load once
            return;
        }

        loadData();

        const intervalMs = (settings.autoRefreshInterval || 30) * 1000;
        const interval = setInterval(() => {
            loadData();
        }, intervalMs);

        return () => clearInterval(interval);
    }, [settings?.autoRefreshEnabled, settings?.autoRefreshInterval, loadData]);

    return {
        data,
        isLoading: isLoading && !data,
        error,
        lastUpdated,
        refresh: loadData,
        settings,
        refreshSettings: loadSettings
    };
}