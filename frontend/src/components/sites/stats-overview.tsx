import { SiteStats } from "@/models/stats";
import {
    RiArticleLine,
    RiCloseCircleLine,
    RiFileTextLine,
    RiLink
} from "@remixicon/react";
import { Card } from "@/components/ui/card";

interface StatsOverviewProps {
    stats: SiteStats;
}

export function StatsOverview({ stats }: StatsOverviewProps) {
    const cards = [
        {
            title: "Total Articles",
            value: stats.articlesPublished?.toString() || "0",
            icon: RiArticleLine,
            color: "text-purple-500",
            bgColor: "bg-purple-500/10",
            borderColor: "border-purple-500/50"
        },
        {
            title: "Failed Articles",
            value: stats.articlesFailed?.toString() || "0",
            icon: RiCloseCircleLine,
            color: "text-red-500",
            bgColor: "bg-red-500/10",
            borderColor: "border-red-500/50"
        },
        {
            title: "Total Words",
            value: stats.totalWords ? stats.totalWords.toLocaleString() : "0",
            icon: RiFileTextLine,
            color: "text-green-500",
            bgColor: "bg-green-500/10",
            borderColor: "border-green-500/50"
        },
        {
            title: "Links Created",
            value: ((stats.internalLinksCreated || 0) + (stats.externalLinksCreated || 0)).toString(),
            icon: RiLink,
            color: "text-blue-500",
            bgColor: "bg-blue-500/10",
            borderColor: "border-blue-500/50"
        }
    ];

    return (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {cards.map((card, index) => (
                <Card key={index} className="h-full">
                    <div key={index} className="relative p-6 border rounded-xl bg-gradient-to-br from-sidebar/60 to-sidebar group hover:shadow-md transition-all duration-200">
                        <div className="flex items-center gap-4">
                            {/* Icon */}
                            <div className={`size-12 shrink-0 rounded-full ${card.bgColor} border ${card.borderColor} flex items-center justify-center ${card.color}`}>
                                <card.icon className="w-6 h-6" />
                            </div>

                            {/* Content */}
                            <div className="flex-1">
                                <div className="text-sm font-medium text-muted-foreground/60 uppercase tracking-wider">
                                    {card.title}
                                </div>
                                <div className="text-2xl font-bold mt-1">{card.value}</div>
                            </div>
                        </div>
                    </div>
                </Card>
            ))}
        </div>
    );
}