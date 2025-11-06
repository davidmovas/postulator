import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
    LineChart,
    Line,
    XAxis,
    CartesianGrid,
} from "recharts";
import {
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
    ChartLegend,
    ChartLegendContent,
    ChartConfig,
} from "@/components/ui/chart";
import { Statistics, Category } from "@/models/categories";
import { format, parseISO } from "date-fns";

interface CategoryStatsTrendsProps {
    stats: Statistics[];
    categories: Category[];
}

interface ChartDataItem {
    date: string;
    [key: string]: number | string; // Динамические ключи для категорий
}

export function CategoryStatsTrends({ stats, categories }: CategoryStatsTrendsProps) {
    // Создаем slug из имени категории для использования как ключ
    const createCategorySlug = (name: string): string => {
        return name
            .toLowerCase()
            .trim()
            .replace(/[^\w\s-]/g, '')
            .replace(/[\s_-]+/g, '_')
            .replace(/^-+|-+$/g, '');
    };

    const chartData = stats.reduce((acc: ChartDataItem[], stat) => {
        const date = format(parseISO(stat.date), 'MMM dd');
        const existingDate = acc.find(item => item.date === date);
        const category = categories.find(cat => cat.id === stat.categoryId);

        if (!category) return acc; // Пропускаем если категория не найдена

        const categoryKey = createCategorySlug(category.name);

        if (existingDate) {
            existingDate[categoryKey] = stat.articlesPublished;
        } else {
            const newEntry: ChartDataItem = { date };
            newEntry[categoryKey] = stat.articlesPublished;
            acc.push(newEntry);
        }

        return acc;
    }, []);

    // Получаем уникальные категории из данных статистики
    const uniqueCategoryIds = [...new Set(stats.map(stat => stat.categoryId))].slice(0, 5);

    // Фильтруем категории которые есть в статистике
    const categoriesInStats = categories.filter(cat =>
        uniqueCategoryIds.includes(cat.id)
    );

    // Создаем конфигурацию для chart с именами категорий
    const chartConfig = categoriesInStats.reduce((config, category, index) => {
        const colors = [
            "hsl(220, 70%, 50%)",    // blue
            "hsl(0, 70%, 50%)",      // red
            "hsl(120, 70%, 40%)",    // green
            "hsl(45, 70%, 50%)",     // orange
            "hsl(280, 70%, 60%)"     // purple
        ];

        const categoryKey = createCategorySlug(category.name);

        config[categoryKey] = {
            label: category.name,
            color: colors[index % colors.length]
        };
        return config;
    }, {} as ChartConfig);

    return (
        <Card>
            <CardHeader className="pb-4">
                <CardTitle className="text-lg">Category Usage Over Time</CardTitle>
            </CardHeader>
            <CardContent>
                <ChartContainer config={chartConfig} className="min-h-[200px] w-full">
                    <LineChart accessibilityLayer data={chartData}>
                        <CartesianGrid vertical={false} />
                        <XAxis
                            dataKey="date"
                            tickLine={false}
                            axisLine={false}
                            tickMargin={8}
                            fontSize={12}
                        />
                        <ChartTooltip
                            content={<ChartTooltipContent />}
                        />
                        <ChartLegend
                            content={<ChartLegendContent />}
                            verticalAlign="top"
                        />
                        {categoriesInStats.map((category) => {
                            const categoryKey = createCategorySlug(category.name);
                            return (
                                <Line
                                    key={category.id}
                                    type="monotone"
                                    dataKey={categoryKey}
                                    stroke={`var(--color-${categoryKey})`}
                                    strokeWidth={2}
                                    dot={{
                                        fill: `var(--color-${categoryKey})`,
                                        strokeWidth: 2,
                                        r: 3
                                    }}
                                    activeDot={{ r: 5 }}
                                />
                            );
                        })}
                    </LineChart>
                </ChartContainer>

                {categoriesInStats.length === 0 && (
                    <div className="text-center py-8 text-muted-foreground">
                        No category usage data available for the selected period.
                    </div>
                )}
            </CardContent>
        </Card>
    );
}