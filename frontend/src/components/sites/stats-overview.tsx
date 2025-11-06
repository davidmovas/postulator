import { SiteStats } from "@/models/stats";
import {
    RiArticleLine,
    RiCloseCircleLine,
    RiFileTextLine,
    RiLink
} from "@remixicon/react";
import { StatsCard } from "@/components/dashboard/stats-card";

interface StatsOverviewProps {
    stats: SiteStats;
}

export function StatsOverview({ stats }: StatsOverviewProps) {
    const cards = [
        {
            title: "Total Articles",
            value: stats.articlesPublished || 0,
            icon: <RiArticleLine className="w-4 h-4" />,
            className: "border-l-purple-500"
        },
        {
            title: "Failed Articles",
            value: stats.articlesFailed || 0,
            icon: <RiCloseCircleLine className="w-4 h-4" />,
            className: "border-l-red-500"
        },
        {
            title: "Total Words",
            value: stats.totalWords || 0,
            icon: <RiFileTextLine className="w-4 h-4" />,
            className: "border-l-green-500"
        },
        {
            title: "Links Created",
            value: (stats.internalLinksCreated || 0) + (stats.externalLinksCreated || 0),
            icon: <RiLink className="w-4 h-4" />,
            className: "border-l-blue-500"
        }
    ];

    return (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {cards.map((card, index) => (
                <StatsCard
                    key={index}
                    title={card.title}
                    value={card.value}
                    icon={card.icon}
                    className={card.className}
                />
            ))}
        </div>
    );
}