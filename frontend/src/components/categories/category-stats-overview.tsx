import { Statistics } from "@/models/categories";
import { StatsCard } from "@/components/dashboard/stats-card";
import { RiFolderLine, RiArticleLine } from "@remixicon/react";
import { TrendingUpIcon } from "lucide-react";

interface CategoryStatsOverviewProps {
    stats: Statistics[];
    totalCategories?: number;
    activeCategories?: number;
}

export function CategoryStatsOverview({
      stats,
      totalCategories,
      activeCategories
  }: CategoryStatsOverviewProps) {
    const totalArticles = stats.reduce((sum, day) => sum + day.articlesPublished, 0);

    const activeDays = stats.filter(day => day.articlesPublished > 0).length;
    const utilizationRate = stats.length > 0 ? Math.round((activeDays / stats.length) * 100) : 0;

    const avgArticlesPerDay = stats.length > 0 ? Math.round(totalArticles / stats.length) : 0;

    const cards = [
        {
            title: "Category Usage",
            value: utilizationRate,
            suffix: "%",
            icon: <TrendingUpIcon className="w-4 h-4" />,
            className: "border-l-blue-500",
            description: "Days with category activity"
        },
        {
            title: "Total Articles",
            value: totalArticles,
            icon: <RiArticleLine className="w-4 h-4" />,
            className: "border-l-green-500",
            description: "Across all categories"
        },
        {
            title: "Avg Daily Usage",
            value: avgArticlesPerDay,
            icon: <RiFolderLine className="w-4 h-4" />,
            className: "border-l-purple-500",
            description: "Articles per day"
        }
    ];

    return (
        <div className="grid gap-4 md:grid-cols-3">
            {cards.map((card, index) => (
                <StatsCard
                    key={index}
                    title={card.title}
                    value={card.value}
                    icon={card.icon}
                    className={card.className}
                    description={card.description}
                />
            ))}
        </div>
    );
}