import { useState, useCallback } from "react";
import { StatsOverview } from "./stats-overview";
import { StatsTrends } from "./stats-trends";
import { StatsFilters } from "./stats-filters";
import { StatsDetails } from "./stats-details";
import { Card, CardContent } from "@/components/ui/card";
import { RiBarChart2Line, RiInformationLine } from "@remixicon/react";
import { DateRange } from "react-day-picker";
import { SiteStats } from "@/models/stats";
import { toGoDateFormat } from "@/lib/time";
import { statsService } from "@/services/stats";
import { useApiCall } from "@/hooks/use-api-call";

interface SiteStatsProps {
    siteId: number;
    totalStats: SiteStats | null;
    dailyStats: SiteStats[];
    onStatsUpdate?: (dailyStats: SiteStats[]) => void;
}

export function SiteStatistics({ siteId, totalStats, dailyStats, onStatsUpdate }: SiteStatsProps) {
    const [dateRange, setDateRange] = useState<DateRange>();
    const [expandedSections, setExpandedSections] = useState<string[]>([]);
    const { execute } = useApiCall();

    const handleDateRangeChange = useCallback(async (range: DateRange | undefined) => {
        setDateRange(range);

        if (range?.from && range?.to) {
            const fromStr = toGoDateFormat(range.from);
            const toStr = toGoDateFormat(range.to);

            const dailyStatsResult = await execute<SiteStats[]>(
                () => statsService.getSiteStatistics(siteId, fromStr, toStr),
                {
                    errorTitle: "Failed to load daily statistics",
                }
            );

            onStatsUpdate?.(dailyStatsResult || []);
        }
    }, [siteId, execute, onStatsUpdate]);

    const hasTotalStats = totalStats !== null;
    const hasDailyStats = dailyStats.length > 0;
    const hasAnyStats = hasTotalStats || hasDailyStats;

    if (!hasAnyStats) {
        return (
            <Card>
                <CardContent className="pt-6">
                    <div className="text-center py-8">
                        <RiBarChart2Line className="w-12 h-12 text-muted-foreground/50 mx-auto mb-4" />
                        <h3 className="text-lg font-medium mb-2">No Statistics Available</h3>
                        <p className="text-muted-foreground text-sm">
                            Statistics will appear here once you start publishing content.
                        </p>
                    </div>
                </CardContent>
            </Card>
        );
    }

    return (
        <div className="space-y-6">
            <StatsFilters
                dateRange={dateRange}
                onDateRangeChange={handleDateRangeChange}
            />

            {hasTotalStats && totalStats && (
                <StatsOverview stats={totalStats} />
            )}

            {hasDailyStats && (
                <StatsTrends
                    dailyStats={dailyStats}
                />
            )}

            {hasDailyStats && (
                <StatsDetails
                    dailyStats={dailyStats}
                    expandedSections={expandedSections}
                    onExpandedChange={setExpandedSections}
                />
            )}

            {(hasTotalStats && !hasDailyStats) && (
                <Card className="bg-muted/50">
                    <CardContent className="pt-6">
                        <div className="flex items-start gap-3">
                            <RiInformationLine className="w-5 h-5 text-muted-foreground mt-0.5" />
                            <div className="space-y-1">
                                <p className="text-sm font-medium">Partial Data Available</p>
                                <p className="text-sm text-muted-foreground">
                                    You have total statistics, but daily breakdown will appear after content publication.
                                </p>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            )}
        </div>
    );
}