import { Button } from "@/components/ui/button";
import {
    RiArticleLine,
    RiTimerLine,
    RiLightbulbLine,
    RiChatThreadLine,
} from "@remixicon/react";
import { cn } from "@/lib/utils";

interface SiteNavigationProps {
    onViewArticles: () => void;
    onViewJobs: () => void;
    onViewTopics?: () => void;
    onViewCategories?: () => void;
}

export function SiteNavigation({
    onViewArticles,
    onViewJobs,
    onViewTopics,
    onViewCategories,
}: SiteNavigationProps) {
    const navItems = [
        {
            icon: RiArticleLine,
            label: "Articles",
            onClick: onViewArticles,
            color: "text-blue-500 border-blue-500 hover:bg-blue-500/10"
        },
        {
            icon: RiTimerLine,
            label: "Jobs",
            onClick: onViewJobs,
            color: "text-purple-500 border-purple-500 hover:bg-purple-500/10"
        },
        {
            icon: RiChatThreadLine,
            label: "Categories",
            onClick: onViewCategories,
            color: "text-green-500 border-green-500 hover:bg-green-500/10"
        },
        {
            icon: RiLightbulbLine,
            label: "Topics",
            onClick: onViewTopics,
            color: "text-orange-500 border-orange-500 hover:bg-orange-500/10"
        }
    ];

    return (
        <div className="space-y-4">
            <h2 className="text-2xl font-bold tracking-tight">Site Sections</h2>
            <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
                {navItems.map((item, index) => (
                    <Button
                        key={index}
                        variant="outline"
                        onClick={item.onClick}
                        className={cn(
                            "h-16 flex items-center justify-center gap-3",
                            "border-2 transition-all duration-200 hover:scale-105",
                            "bg-background hover:shadow-lg",
                            item.color
                        )}
                    >
                        <item.icon className="w-5 h-5" />
                        <span className="font-semibold">{item.label}</span>
                    </Button>
                ))}
            </div>
        </div>
    );
}