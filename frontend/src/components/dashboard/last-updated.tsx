import { formatDistanceToNow } from "date-fns";
import { DASHBOARD_AUTO_REFRESH_INTERVAL } from "@/constants/dashboard";

interface LastUpdatedProps {
    lastUpdated: Date | null;
    isLoading: boolean;
    autoRefresh: boolean;
}

export function LastUpdated({ lastUpdated, isLoading, autoRefresh }: LastUpdatedProps) {
    if (isLoading) {
        return (
            <div className="text-sm text-muted-foreground text-right animate-pulse">
                Updating...
            </div>
        );
    }

    if (!lastUpdated) {
        return null;
    }

    const refreshIntervalSeconds = DASHBOARD_AUTO_REFRESH_INTERVAL / 1000;

    return (
        <div className="text-sm text-muted-foreground text-right">
            {autoRefresh ? (
                <>
                    Auto-refresh every {refreshIntervalSeconds} seconds
                </>
            ) : (
                `Updated ${formatDistanceToNow(lastUpdated, { addSuffix: true })}`
            )}
        </div>
    );
}