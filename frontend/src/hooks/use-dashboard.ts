import { useState, useEffect, useCallback } from "react";
import { statsService } from "@/services/stats";
import { DashboardSummary } from "@/models/stats";
import { useApiCall } from "@/hooks/use-api-call";
import { DASHBOARD_AUTO_REFRESH_INTERVAL } from "@/constants/dashboard";

export function useDashboard(autoRefresh: boolean = true) {
    const [data, setData] = useState<DashboardSummary | null>(null);
    const { execute, isLoading, error } = useApiCall();
    const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

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

    useEffect(() => {
        if (!autoRefresh) return;

        loadData();

        const interval = setInterval(() => {
            loadData();
        }, DASHBOARD_AUTO_REFRESH_INTERVAL);

        return () => clearInterval(interval);
    }, [autoRefresh, loadData]);

    return {
        data,
        isLoading: isLoading && !data,
        error,
        lastUpdated,
        refresh: loadData
    };
}