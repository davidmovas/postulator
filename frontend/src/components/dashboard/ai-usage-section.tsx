"use client";

import { useEffect, useState, useCallback } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { StatsFilters } from "@/components/sites/stats-filters";
import { useApiCall } from "@/hooks/use-api-call";
import { aiUsageService } from "@/services/aiusage";
import { AIUsageSummary } from "@/models/aiusage";
import { toGoDateFormat } from "@/lib/time";
import {
    RiFlashlightLine,
    RiCoinLine,
    RiCloseCircleLine,
    RiStackLine,
} from "@remixicon/react";

function formatTokens(tokens: number): string {
    if (tokens >= 1000000) {
        return `${(tokens / 1000000).toFixed(2)}M`;
    }
    if (tokens >= 1000) {
        return `${(tokens / 1000).toFixed(1)}K`;
    }
    return tokens.toString();
}

export function AIUsageSection() {
    const { execute } = useApiCall();
    const [dateRange, setDateRange] = useState<{ from: Date; to: Date }>();
    const [summary, setSummary] = useState<AIUsageSummary | null>(null);
    const [isLoading, setIsLoading] = useState(false);

    // Initial range: last 30 days
    useEffect(() => {
        const to = new Date();
        const from = new Date();
        from.setDate(from.getDate() - 30);
        setDateRange({ from, to });
    }, []);

    const loadData = useCallback(
        async (from: Date, to: Date) => {
            const fromStr = toGoDateFormat(from);
            const toStr = toGoDateFormat(to);
            setIsLoading(true);
            try {
                const summaryData = await execute(
                    () => aiUsageService.getSummary(0, fromStr, toStr),
                    { errorTitle: "Failed to load AI usage summary" }
                );
                if (summaryData) setSummary(summaryData);
            } finally {
                setIsLoading(false);
            }
        },
        [execute]
    );

    useEffect(() => {
        if (dateRange?.from && dateRange?.to) {
            loadData(dateRange.from, dateRange.to);
        }
    }, [dateRange, loadData]);

    const hasData = summary && summary.totalRequests > 0;

    // Don't render anything if no data
    if (!hasData && !isLoading) {
        return null;
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <RiFlashlightLine className="w-5 h-5 text-yellow-500" />
                    <h2 className="text-lg font-semibold whitespace-nowrap">AI Usage</h2>
                </div>
                <StatsFilters
                    dateRange={dateRange}
                    onDateRangeChange={(range) => {
                        if (range?.from && range?.to) setDateRange(range);
                    }}
                />
            </div>

            {isLoading && !summary && (
                <div className="grid gap-4 grid-cols-2 md:grid-cols-4">
                    {[...Array(4)].map((_, i) => (
                        <Card key={i}>
                            <CardContent className="pt-4 pb-4">
                                <div className="h-16 bg-muted/30 rounded animate-pulse" />
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}

            {hasData && summary && (
                <div className="grid gap-4 grid-cols-2 md:grid-cols-4">
                    <Card className="border-l-4 border-l-blue-500">
                        <CardContent className="pt-4 pb-4">
                            <div className="flex items-center gap-3">
                                <div className="p-2 rounded-lg bg-blue-500/10">
                                    <RiFlashlightLine className="w-5 h-5 text-blue-500" />
                                </div>
                                <div>
                                    <p className="text-xs text-muted-foreground">Requests</p>
                                    <p className="text-2xl font-bold">
                                        {summary.totalRequests.toLocaleString()}
                                    </p>
                                </div>
                            </div>
                        </CardContent>
                    </Card>

                    <Card className="border-l-4 border-l-purple-500">
                        <CardContent className="pt-4 pb-4">
                            <div className="flex items-center gap-3">
                                <div className="p-2 rounded-lg bg-purple-500/10">
                                    <RiStackLine className="w-5 h-5 text-purple-500" />
                                </div>
                                <div>
                                    <p className="text-xs text-muted-foreground">Total Tokens</p>
                                    <p className="text-2xl font-bold">
                                        {formatTokens(summary.totalTokens)}
                                    </p>
                                </div>
                            </div>
                        </CardContent>
                    </Card>

                    <Card className="border-l-4 border-l-yellow-500">
                        <CardContent className="pt-4 pb-4">
                            <div className="flex items-center gap-3">
                                <div className="p-2 rounded-lg bg-yellow-500/10">
                                    <RiCoinLine className="w-5 h-5 text-yellow-500" />
                                </div>
                                <div>
                                    <p className="text-xs text-muted-foreground">Estimated Cost</p>
                                    <p className="text-2xl font-bold">
                                        ${summary.totalCostUsd.toFixed(2)}
                                    </p>
                                </div>
                            </div>
                        </CardContent>
                    </Card>

                    <Card className={`border-l-4 ${summary.errorCount > 0 ? "border-l-yellow-500" : "border-l-green-500"}`}>
                        <CardContent className="pt-4 pb-4">
                            <div className="flex items-center gap-3">
                                <div className={`p-2 rounded-lg ${summary.errorCount > 0 ? "bg-yellow-500/10" : "bg-green-500/10"}`}>
                                    <RiCloseCircleLine className={`w-5 h-5 ${summary.errorCount > 0 ? "text-yellow-500" : "text-green-500"}`} />
                                </div>
                                <div>
                                    <p className="text-xs text-muted-foreground">Success Rate</p>
                                    <p className={`text-2xl font-bold ${summary.errorCount > 0 ? "text-yellow-500" : "text-green-500"}`}>
                                        {((summary.successCount / summary.totalRequests) * 100).toFixed(1)}%
                                    </p>
                                </div>
                            </div>
                        </CardContent>
                    </Card>
                </div>
            )}
        </div>
    );
}
