"use client";

import { useState, useEffect, useMemo, useCallback } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { statsService } from "@/services/stats";
import { SiteStats } from "@/models/stats";
import { toGoDateFormat } from "@/lib/time";
import { useApiCall } from "@/hooks/use-api-call";
import { StatsFilters } from "@/components/sites/stats-filters";
import {
    RiFileTextLine,
    RiArticleLine,
    RiLink,
    RiBarChart2Line,
    RiExternalLinkLine,
    RiLinksLine,
} from "@remixicon/react";

export function DashboardCharts() {
    const { execute } = useApiCall();
    const [dateRange, setDateRange] = useState<{ from: Date; to: Date }>();
    const [stats, setStats] = useState<SiteStats[]>([]);
    const [isLoading, setIsLoading] = useState(false);

    // Initial range: last 14 days
    useEffect(() => {
        const to = new Date();
        const from = new Date();
        from.setDate(from.getDate() - 14);
        setDateRange({ from, to });
    }, []);

    const loadData = useCallback(
        async (from: Date, to: Date) => {
            const fromStr = toGoDateFormat(from);
            const toStr = toGoDateFormat(to);
            setIsLoading(true);
            try {
                const data = await execute(
                    () => statsService.getGlobalStatistics(fromStr, toStr),
                    { errorTitle: "Failed to load statistics" }
                );
                if (data) setStats(data);
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

    const totals = useMemo(() => {
        return stats.reduce(
            (acc, stat) => ({
                published: acc.published + stat.articlesPublished,
                failed: acc.failed + stat.articlesFailed,
                words: acc.words + stat.totalWords,
                internalLinks: acc.internalLinks + (stat.internalLinksCreated || 0),
                externalLinks: acc.externalLinks + (stat.externalLinksCreated || 0),
            }),
            { published: 0, failed: 0, words: 0, internalLinks: 0, externalLinks: 0 }
        );
    }, [stats]);

    const hasData = stats.length > 0 && (totals.published > 0 || totals.failed > 0);

    // Don't render anything if no data
    if (!hasData && !isLoading) {
        return null;
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <RiArticleLine className="w-5 h-5 text-blue-500" />
                    <h2 className="text-lg font-semibold whitespace-nowrap">Content Statistics</h2>
                </div>
                <StatsFilters
                    dateRange={dateRange}
                    onDateRangeChange={(range) => {
                        if (range?.from && range?.to) setDateRange(range);
                    }}
                />
            </div>

            {isLoading && !hasData && (
                <div className="grid gap-4 grid-cols-2 md:grid-cols-3 lg:grid-cols-6">
                    {[...Array(6)].map((_, i) => (
                        <Card key={i}>
                            <CardContent className="pt-4 pb-4">
                                <div className="h-16 bg-muted/30 rounded animate-pulse" />
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}

            {hasData && (
                <div className="grid gap-4 grid-cols-2 md:grid-cols-3 lg:grid-cols-6">
                    <Card className="border-l-4 border-l-green-500">
                        <CardContent className="pt-4 pb-4">
                            <div className="flex items-center gap-3">
                                <div className="p-2 rounded-lg bg-green-500/10">
                                    <RiFileTextLine className="w-5 h-5 text-green-500" />
                                </div>
                                <div>
                                    <p className="text-xs text-muted-foreground">Published</p>
                                    <p className="text-2xl font-bold">
                                        {totals.published.toLocaleString()}
                                    </p>
                                </div>
                            </div>
                        </CardContent>
                    </Card>

                    {totals.failed > 0 && (
                        <Card className="border-l-4 border-l-red-500">
                            <CardContent className="pt-4 pb-4">
                                <div className="flex items-center gap-3">
                                    <div className="p-2 rounded-lg bg-red-500/10">
                                        <RiFileTextLine className="w-5 h-5 text-red-500" />
                                    </div>
                                    <div>
                                        <p className="text-xs text-muted-foreground">Failed</p>
                                        <p className="text-2xl font-bold">
                                            {totals.failed.toLocaleString()}
                                        </p>
                                    </div>
                                </div>
                            </CardContent>
                        </Card>
                    )}

                    <Card className="border-l-4 border-l-blue-500">
                        <CardContent className="pt-4 pb-4">
                            <div className="flex items-center gap-3">
                                <div className="p-2 rounded-lg bg-blue-500/10">
                                    <RiBarChart2Line className="w-5 h-5 text-blue-500" />
                                </div>
                                <div>
                                    <p className="text-xs text-muted-foreground">Words</p>
                                    <p className="text-2xl font-bold">
                                        {totals.words.toLocaleString()}
                                    </p>
                                </div>
                            </div>
                        </CardContent>
                    </Card>

                    <Card className="border-l-4 border-l-purple-500">
                        <CardContent className="pt-4 pb-4">
                            <div className="flex items-center gap-3">
                                <div className="p-2 rounded-lg bg-purple-500/10">
                                    <RiLink className="w-5 h-5 text-purple-500" />
                                </div>
                                <div>
                                    <p className="text-xs text-muted-foreground">Total Links</p>
                                    <p className="text-2xl font-bold">
                                        {(totals.internalLinks + totals.externalLinks).toLocaleString()}
                                    </p>
                                </div>
                            </div>
                        </CardContent>
                    </Card>

                    <Card className="border-l-4 border-l-cyan-500">
                        <CardContent className="pt-4 pb-4">
                            <div className="flex items-center gap-3">
                                <div className="p-2 rounded-lg bg-cyan-500/10">
                                    <RiLinksLine className="w-5 h-5 text-cyan-500" />
                                </div>
                                <div>
                                    <p className="text-xs text-muted-foreground">Internal Links</p>
                                    <p className="text-2xl font-bold">
                                        {totals.internalLinks.toLocaleString()}
                                    </p>
                                </div>
                            </div>
                        </CardContent>
                    </Card>

                    <Card className="border-l-4 border-l-orange-500">
                        <CardContent className="pt-4 pb-4">
                            <div className="flex items-center gap-3">
                                <div className="p-2 rounded-lg bg-orange-500/10">
                                    <RiExternalLinkLine className="w-5 h-5 text-orange-500" />
                                </div>
                                <div>
                                    <p className="text-xs text-muted-foreground">External Links</p>
                                    <p className="text-2xl font-bold">
                                        {totals.externalLinks.toLocaleString()}
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
