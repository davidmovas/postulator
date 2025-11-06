import { Card, CardContent } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { formatDateTime } from "@/lib/time";
import { Site } from "@/models/sites";
import {
    RiCalendarLine,
    RiRefreshLine,
    RiHeartPulseLine,
    RiTimeLine
} from "@remixicon/react";
import { cn } from "@/lib/utils";

interface SiteInfoProps {
    site: Site;
}

export function SiteInfo({ site }: SiteInfoProps) {
    const infoItems = [
        {
            label: "Created",
            value: formatDateTime(site.createdAt) || "Unknown",
            icon: RiCalendarLine,
        },
        {
            label: "Updated",
            value: formatDateTime(site.updatedAt) || "Unknown",
            icon: RiTimeLine,
        },
        {
            label: "Last Check",
            value: formatDateTime(site.lastHealthCheck) || "Never",
            icon: RiRefreshLine,
        },
        {
            label: "Health",
            value: site.healthStatus,
            icon: RiHeartPulseLine,
        }
    ];

    const formatHealthStatus = (status: string) => {
        return status.charAt(0).toUpperCase() + status.slice(1);
    };

    return (
        <Card className="bg-card">
            <CardContent className="p-3">
                <div className="flex items-center divide-x divide-border/50">
                    {infoItems.map((item, index) => (
                        <div key={index} className="flex-1 flex flex-col items-center text-center px-3 first:pl-0 last:pr-0">
                            <div className="flex flex-col items-center gap-1">
                                <div className="flex items-center gap-1">
                                    <item.icon className="w-3.5 h-3.5 text-muted-foreground" />
                                    <div className="text-xs text-muted-foreground font-medium">
                                        {item.label}
                                    </div>
                                </div>
                                <div className={cn(
                                    "text-sm font-semibold",
                                    item.label === "Health" && (
                                        site.healthStatus === "healthy" ? "text-green-600" :
                                            site.healthStatus === "unhealthy" ? "text-red-600" : "text-yellow-600"
                                    )
                                )}>
                                    {item.label === "Health" ? formatHealthStatus(item.value) : item.value}
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            </CardContent>
        </Card>
    );
}