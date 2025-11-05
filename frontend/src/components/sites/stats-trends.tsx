import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
    BarChart,
    Bar,
    LineChart,
    Line,
    XAxis,
    CartesianGrid,
} from "recharts";
import {
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
    ChartConfig,
} from "@/components/ui/chart";
import { SiteStats } from "@/models/stats";
import { format, parseISO } from "date-fns";

interface StatsTrendsProps {
    dailyStats: SiteStats[];
}

export function StatsTrends({ dailyStats }: StatsTrendsProps) {
    const chartData = dailyStats.map(stat => ({
        date: stat.date,
        displayDate: format(parseISO(stat.date), 'MMM dd'),
        published: stat.articlesPublished,
        failed: stat.articlesFailed,
        words: stat.totalWords,
        links: (stat.internalLinksCreated || 0) + (stat.externalLinksCreated || 0),
        successRate: stat.articlesPublished + stat.articlesFailed > 0
            ? Math.round((stat.articlesPublished / (stat.articlesPublished + stat.articlesFailed)) * 100)
            : 0
    }));

    const publicationsChartConfig = {
        published: {
            label: "Published",
            color: "hsl(145, 85%, 50%)",
        },
        failed: {
            label: "Failed",
            color: "hsl(4, 95%, 62%)",
        },
    } satisfies ChartConfig;

    const contentChartConfig = {
        words: {
            label: "Words",
            color: "hsl(199, 100%, 65%)",
        },
        links: {
            label: "Links",
            color: "hsl(290, 100%, 68%)",
        },
    } satisfies ChartConfig;

    const successChartConfig = {
        successRate: {
            label: "Success Rate",
            color: "hsl(38, 100%, 60%)",
        },
    } satisfies ChartConfig;
    return (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {/* График публикаций */}
            <Card>
                <CardHeader className="pb-4">
                    <CardTitle className="text-lg">Articles Published</CardTitle>
                </CardHeader>
                <CardContent>
                    <ChartContainer config={publicationsChartConfig} className="min-h-[200px] w-full">
                        <BarChart accessibilityLayer data={chartData}>
                            <CartesianGrid vertical={false} />
                            <XAxis
                                dataKey="displayDate"
                                tickLine={false}
                                axisLine={false}
                                tickMargin={8}
                                fontSize={12}
                            />
                            <ChartTooltip content={<ChartTooltipContent />} />
                            <Bar dataKey="published" fill="var(--color-published)" radius={4} />
                            <Bar dataKey="failed" fill="var(--color-failed)" radius={4} />
                        </BarChart>
                    </ChartContainer>
                </CardContent>
            </Card>

            {/* График контента */}
            <Card>
                <CardHeader className="pb-4">
                    <CardTitle className="text-lg">Content & Links</CardTitle>
                </CardHeader>
                <CardContent>
                    <ChartContainer config={contentChartConfig} className="min-h-[200px] w-full">
                        <BarChart accessibilityLayer data={chartData}>
                            <CartesianGrid vertical={false} />
                            <XAxis
                                dataKey="displayDate"
                                tickLine={false}
                                axisLine={false}
                                tickMargin={8}
                                fontSize={12}
                            />
                            <ChartTooltip content={<ChartTooltipContent />} />
                            <Bar dataKey="words" fill="var(--color-words)" radius={4} />
                            <Bar dataKey="links" fill="var(--color-links)" radius={4} />
                        </BarChart>
                    </ChartContainer>
                </CardContent>
            </Card>

            {/* График успеха */}
            <Card>
                <CardHeader className="pb-4">
                    <CardTitle className="text-lg">Success Rate</CardTitle>
                </CardHeader>
                <CardContent>
                    <ChartContainer config={successChartConfig} className="min-h-[200px] w-full">
                        <LineChart accessibilityLayer data={chartData}>
                            <CartesianGrid vertical={false} />
                            <XAxis
                                dataKey="displayDate"
                                tickLine={false}
                                axisLine={false}
                                tickMargin={8}
                                fontSize={12}
                            />
                            <ChartTooltip
                                content={<ChartTooltipContent />}
                                formatter={(value) => [`${value}%`, "Success Rate"]}
                            />
                            <Line
                                dataKey="successRate"
                                type="monotone"
                                stroke="var(--color-successRate)"
                                strokeWidth={2}
                                dot={{ fill: "var(--color-successRate)", strokeWidth: 2, r: 3 }}
                                activeDot={{ r: 5 }}
                            />
                        </LineChart>
                    </ChartContainer>
                </CardContent>
            </Card>
        </div>
    );
}