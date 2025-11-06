import { useState, useCallback } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Category, Statistics } from "@/models/categories";
import { CategoryStatsOverview } from "./category-stats-overview";
import { CategoryStatsTrends } from "./category-stats-trends";
import { CategoryStatsFilters } from "./category-stats-filters";
import { RiBarChart2Line, RiInformationLine } from "@remixicon/react";
import { DateRange } from "react-day-picker";

interface CategoryStatsProps {
    siteId: number;
    selectedCategoryId?: number;
    categories: Category[];
    siteStats: Statistics[];
    categoryStats: Statistics[];
    totalCategories?: number;
    activeCategories?: number;
    onStatsUpdate: (from: string, to: string, categoryId?: number) => void;
}

export function CategoryStats({
    siteId,
    selectedCategoryId,
    siteStats,
    categories,
    categoryStats,
    totalCategories,
    activeCategories,
    onStatsUpdate
}: CategoryStatsProps) {
    const [dateRange, setDateRange] = useState<DateRange>();

    // Определяем какие данные показывать
    const statsData = selectedCategoryId ? categoryStats : siteStats;
    const hasStats = statsData.length > 0;

    const handleDateRangeChange = useCallback(async (range: DateRange | undefined) => {
        setDateRange(range);

        if (range?.from && range?.to) {
            const fromStr = range.from.toISOString().split('T')[0];
            const toStr = range.to.toISOString().split('T')[0];

            onStatsUpdate(fromStr, toStr, selectedCategoryId);
        }
    }, [selectedCategoryId, onStatsUpdate]);

    if (!hasStats) {
        return (
            <Card>
                <CardContent className="pt-6">
                    <div className="text-center py-8">
                        <RiBarChart2Line className="w-12 h-12 text-muted-foreground/50 mx-auto mb-4" />
                        <h3 className="text-lg font-medium mb-2">
                            {selectedCategoryId ? "Category Analytics" : "Categories Analytics"}
                        </h3>
                        <p className="text-muted-foreground text-sm">
                            {selectedCategoryId
                                ? "Usage analytics will appear here once articles are published in this category."
                                : "Category performance analytics will appear here once categories are being used."
                            }
                        </p>
                    </div>
                </CardContent>
            </Card>
        );
    }

    return (
        <div className="space-y-6">
            <CategoryStatsFilters
                dateRange={
                    dateRange?.from && dateRange?.to
                        ? { from: dateRange.from, to: dateRange.to }
                        : undefined
                }
                onDateRangeChange={handleDateRangeChange}
                selectedCategoryId={selectedCategoryId}
            />

            <CategoryStatsOverview
                stats={statsData}
                totalCategories={totalCategories}
                activeCategories={activeCategories}
            />

            <CategoryStatsTrends
                stats={statsData}
                categories={categories}
            />

            {/* Информация о выбранной категории */}
            {selectedCategoryId && siteStats.length > 0 && (
                <Card className="bg-muted/50">
                    <CardContent className="pt-6">
                        <div className="flex items-start gap-3">
                            <RiInformationLine className="w-5 h-5 text-muted-foreground mt-0.5" />
                            <div className="space-y-1">
                                <p className="text-sm font-medium">Viewing Single Category</p>
                                <p className="text-sm text-muted-foreground">
                                    Showing usage analytics for this specific category.
                                    The graph shows article publication patterns over time.
                                </p>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            )}
        </div>
    );
}