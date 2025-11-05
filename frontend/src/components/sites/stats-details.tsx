import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
    RiArrowDownSLine,
    RiArrowRightSLine,
    RiTable2,
    RiBarChart2Line,
} from "@remixicon/react";
import { format } from "date-fns";
import { SiteStats } from "@/models/stats";
import { DataTable } from "@/components/table/data-table";
import { statsColumns } from "@/components/sites/stats-table";

interface StatsDetailsProps {
    dailyStats: SiteStats[];
    expandedSections: string[];
    onExpandedChange: (sections: string[]) => void;
}

export function StatsDetails({
    dailyStats,
    expandedSections,
    onExpandedChange
}: StatsDetailsProps) {
    const hasDailyStats = dailyStats.length > 0;

    const analytics = hasDailyStats ? {
        totalArticles: dailyStats.reduce((sum, day) => sum + day.articlesPublished, 0),
        totalFailed: dailyStats.reduce((sum, day) => sum + day.articlesFailed, 0),
        successRate: ((dailyStats.reduce((sum, day) => sum + day.articlesPublished, 0) /
            (dailyStats.reduce((sum, day) => sum + day.articlesPublished + day.articlesFailed, 0) || 1)) * 100).toFixed(1),
        avgWordsPerArticle: dailyStats.reduce((sum, day) => sum + day.articlesPublished, 0) > 0
            ? Math.round(dailyStats.reduce((sum, day) => sum + day.totalWords, 0) /
                dailyStats.reduce((sum, day) => sum + day.articlesPublished, 0))
            : 0,
        bestDay: dailyStats.reduce((best, day) =>
                day.articlesPublished > best.articlesPublished ? day : best,
            { articlesPublished: 0, date: '' }
        )
    } : null;

    const hasAnalytics = analytics && analytics.totalArticles > 0;

    const sections = [
        {
            id: "performance-analytics",
            title: "Performance Analytics",
            icon: RiBarChart2Line,
            description: "Success rates and averages",
            hasData: hasAnalytics
        },
        {
            id: "daily-breakdown",
            title: "Daily Breakdown",
            icon: RiTable2,
            description: "Detailed statistics by day",
            hasData: hasDailyStats
        },
    ];

    const visibleSections = sections.filter(section => section.hasData);

    if (visibleSections.length === 0) {
        return null;
    }

    const toggleSection = (sectionId: string) => {
        if (expandedSections.includes(sectionId)) {
            onExpandedChange(expandedSections.filter(id => id !== sectionId));
        } else {
            onExpandedChange([...expandedSections, sectionId]);
        }
    };

    const renderDailyBreakdown = () => (
        <div className="space-y-4">
            <DataTable
                columns={statsColumns}
                data={dailyStats}
                searchKey="date"
                searchPlaceholder="Search dates..."
                isLoading={false}
                emptyMessage="No daily statistics available for the selected period."
                showPagination={true}
                defaultPageSize={25}
                enableViewOption={false}
            />
        </div>
    );

    const renderPerformanceAnalytics = () => (
        hasAnalytics && (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                <div className="space-y-2 p-4 border rounded-lg">
                    <div className="text-sm font-medium text-muted-foreground">Success Rate</div>
                    <div className="text-2xl font-bold text-green-600">{analytics.successRate}%</div>
                    <div className="text-xs text-muted-foreground">
                        {analytics.totalArticles} published / {analytics.totalFailed} failed
                    </div>
                </div>

                <div className="space-y-2 p-4 border rounded-lg">
                    <div className="text-sm font-medium text-muted-foreground">Avg Words/Article</div>
                    <div className="text-2xl font-bold text-blue-600">{analytics.avgWordsPerArticle}</div>
                    <div className="text-xs text-muted-foreground">
                        Average content length
                    </div>
                </div>

                <div className="space-y-2 p-4 border rounded-lg">
                    <div className="text-sm font-medium text-muted-foreground">Best Day</div>
                    <div className="text-2xl font-bold text-purple-600">
                        {analytics.bestDay.articlesPublished}
                    </div>
                    <div className="text-xs text-muted-foreground">
                        {analytics.bestDay.date && format(new Date(analytics.bestDay.date), 'MMM dd, yyyy')}
                    </div>
                </div>
            </div>
        )
    );

    return (
        <div className="space-y-4">
            {visibleSections.map((section) => (
                <Card key={section.id}>
                    <CardHeader className="pb-3">
                        <div className="flex items-center justify-between">
                            <div className="flex items-center gap-3">
                                <section.icon className="w-5 h-5 text-muted-foreground" />
                                <div>
                                    <CardTitle className="text-lg">{section.title}</CardTitle>
                                    <p className="text-sm text-muted-foreground">
                                        {section.description}
                                    </p>
                                </div>
                            </div>
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => toggleSection(section.id)}
                                className="h-8 w-8 p-0"
                            >
                                {expandedSections.includes(section.id) ? (
                                    <RiArrowDownSLine className="w-4 h-4" />
                                ) : (
                                    <RiArrowRightSLine className="w-4 h-4" />
                                )}
                            </Button>
                        </div>
                    </CardHeader>

                    {expandedSections.includes(section.id) && (
                        <CardContent className="pt-0">
                            {section.id === "daily-breakdown" && renderDailyBreakdown()}
                            {section.id === "performance-analytics" && renderPerformanceAnalytics()}
                        </CardContent>
                    )}
                </Card>
            ))}
        </div>
    );
}