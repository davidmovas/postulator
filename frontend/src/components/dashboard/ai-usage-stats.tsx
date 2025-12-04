"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
    PieChart,
    Pie,
    Cell,
    ResponsiveContainer,
    Tooltip,
    Legend,
} from "recharts";
import { aiUsageService } from "@/services/aiusage";
import { AIUsageSummary, AIUsageByProvider, AIUsageByOperation } from "@/models/aiusage";
import { subDays } from "date-fns";
import { Zap, DollarSign, Hash, Activity } from "lucide-react";
import { toGoDateFormat } from "@/lib/time";

const COLORS = ["hsl(199, 89%, 48%)", "hsl(142, 76%, 36%)", "hsl(38, 92%, 50%)", "hsl(280, 87%, 65%)"];

export function AIUsageStats() {
    const [summary, setSummary] = useState<AIUsageSummary | null>(null);
    const [byProvider, setByProvider] = useState<AIUsageByProvider[]>([]);
    const [byOperation, setByOperation] = useState<AIUsageByOperation[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        const loadStats = async () => {
            try {
                const to = new Date();
                const from = subDays(to, 30);
                const fromStr = toGoDateFormat(from);
                const toStr = toGoDateFormat(to);

                const [summaryData, providerData, operationData] = await Promise.all([
                    aiUsageService.getSummary(0, fromStr, toStr),
                    aiUsageService.getUsageByProvider(0, fromStr, toStr),
                    aiUsageService.getUsageByOperation(0, fromStr, toStr),
                ]);

                setSummary(summaryData);
                setByProvider(providerData);
                setByOperation(operationData);
            } catch (error) {
                console.error("Failed to load AI usage stats:", error);
            } finally {
                setIsLoading(false);
            }
        };
        loadStats();
    }, []);

    if (isLoading) {
        return (
            <Card>
                <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium flex items-center gap-2">
                        <Zap className="h-4 w-4" />
                        AI Usage (30 days)
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="h-[120px] bg-muted/30 rounded animate-pulse" />
                </CardContent>
            </Card>
        );
    }

    if (!summary || summary.totalRequests === 0) {
        return (
            <Card>
                <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium flex items-center gap-2">
                        <Zap className="h-4 w-4" />
                        AI Usage (30 days)
                    </CardTitle>
                </CardHeader>
                <CardContent className="py-6 text-center text-muted-foreground text-sm">
                    No AI usage data available yet. Generate articles to see usage statistics.
                </CardContent>
            </Card>
        );
    }

    const providerChartData = byProvider.map((p, i) => ({
        name: `${p.providerName} (${p.modelName})`,
        value: p.totalTokens,
        color: COLORS[i % COLORS.length],
    }));

    const operationChartData = byOperation.map((o, i) => ({
        name: formatOperationType(o.operationType),
        value: o.requestCount,
        color: COLORS[i % COLORS.length],
    }));

    return (
        <Card>
            <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium flex items-center gap-2">
                    <Zap className="h-4 w-4" />
                    AI Usage (30 days)
                </CardTitle>
            </CardHeader>
            <CardContent>
                <div className="grid gap-6 md:grid-cols-3">
                    {/* Summary Stats */}
                    <div className="space-y-3">
                        <div className="flex items-center gap-3 p-3 bg-muted/50 rounded-lg">
                            <Activity className="h-4 w-4 text-blue-500" />
                            <div>
                                <p className="text-xs text-muted-foreground">Requests</p>
                                <p className="text-lg font-semibold">{summary.totalRequests.toLocaleString()}</p>
                            </div>
                        </div>
                        <div className="flex items-center gap-3 p-3 bg-muted/50 rounded-lg">
                            <Hash className="h-4 w-4 text-green-500" />
                            <div>
                                <p className="text-xs text-muted-foreground">Total Tokens</p>
                                <p className="text-lg font-semibold">{formatTokens(summary.totalTokens)}</p>
                            </div>
                        </div>
                        <div className="flex items-center gap-3 p-3 bg-muted/50 rounded-lg">
                            <DollarSign className="h-4 w-4 text-yellow-500" />
                            <div>
                                <p className="text-xs text-muted-foreground">Estimated Cost</p>
                                <p className="text-lg font-semibold">${summary.totalCostUsd.toFixed(4)}</p>
                            </div>
                        </div>
                        {summary.errorCount > 0 && (
                            <div className="text-xs text-muted-foreground">
                                Success rate: {((summary.successCount / summary.totalRequests) * 100).toFixed(1)}%
                            </div>
                        )}
                    </div>

                    {/* Provider Distribution */}
                    {providerChartData.length > 0 && (
                        <div>
                            <p className="text-xs text-muted-foreground mb-2">By Provider (tokens)</p>
                            <div className="h-[140px]">
                                <ResponsiveContainer width="100%" height="100%">
                                    <PieChart>
                                        <Pie
                                            data={providerChartData}
                                            cx="50%"
                                            cy="50%"
                                            innerRadius={30}
                                            outerRadius={50}
                                            paddingAngle={2}
                                            dataKey="value"
                                        >
                                            {providerChartData.map((entry, index) => (
                                                <Cell key={`cell-${index}`} fill={entry.color} />
                                            ))}
                                        </Pie>
                                        <Tooltip
                                            contentStyle={{
                                                backgroundColor: "hsl(var(--popover))",
                                                border: "1px solid hsl(var(--border))",
                                                borderRadius: "6px",
                                                fontSize: "11px",
                                            }}
                                            formatter={(value: number) => [formatTokens(value), "Tokens"]}
                                        />
                                        <Legend
                                            wrapperStyle={{ fontSize: "10px" }}
                                            formatter={(value) => <span className="text-muted-foreground">{value}</span>}
                                        />
                                    </PieChart>
                                </ResponsiveContainer>
                            </div>
                        </div>
                    )}

                    {/* Operation Distribution */}
                    {operationChartData.length > 0 && (
                        <div>
                            <p className="text-xs text-muted-foreground mb-2">By Operation (requests)</p>
                            <div className="h-[140px]">
                                <ResponsiveContainer width="100%" height="100%">
                                    <PieChart>
                                        <Pie
                                            data={operationChartData}
                                            cx="50%"
                                            cy="50%"
                                            innerRadius={30}
                                            outerRadius={50}
                                            paddingAngle={2}
                                            dataKey="value"
                                        >
                                            {operationChartData.map((entry, index) => (
                                                <Cell key={`cell-${index}`} fill={entry.color} />
                                            ))}
                                        </Pie>
                                        <Tooltip
                                            contentStyle={{
                                                backgroundColor: "hsl(var(--popover))",
                                                border: "1px solid hsl(var(--border))",
                                                borderRadius: "6px",
                                                fontSize: "11px",
                                            }}
                                        />
                                        <Legend
                                            wrapperStyle={{ fontSize: "10px" }}
                                            formatter={(value) => <span className="text-muted-foreground">{value}</span>}
                                        />
                                    </PieChart>
                                </ResponsiveContainer>
                            </div>
                        </div>
                    )}
                </div>
            </CardContent>
        </Card>
    );
}

function formatTokens(tokens: number): string {
    if (tokens >= 1000000) {
        return `${(tokens / 1000000).toFixed(2)}M`;
    }
    if (tokens >= 1000) {
        return `${(tokens / 1000).toFixed(1)}K`;
    }
    return tokens.toString();
}

function formatOperationType(type: string): string {
    return type
        .split("_")
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(" ");
}
