import { Card, CardContent } from "@/components/ui/card";
import { formatDateTime } from "@/lib/time";
import { Site } from "@/models/sites";

interface SiteInfoProps {
    site: Site;
}

export function SiteInfo({ site }: SiteInfoProps) {
    const infoItems = [
        {
            label: "Created Date",
            value: formatDateTime(site.createdAt) || "Unknown",
        },
        {
            label: "Last Updated",
            value: formatDateTime(site.updatedAt) || "Unknown",
        },
        {
            label: "Last Health Check",
            value: formatDateTime(site.lastHealthCheck) || "Never",
        },
        {
            label: "Health Status",
            value: site.healthStatus,
        }
    ];

    return (
        <Card>
            <CardContent className="pt-6">
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                    {infoItems.map((item, index) => (
                        <div key={index} className="space-y-1">
                            <div className="text-sm font-medium text-muted-foreground">
                                {item.label}
                            </div>
                            <div className="text-sm">
                                {item.value}
                            </div>
                            {index < infoItems.length - 1 && (
                                <div className="hidden lg:block absolute right-0 top-1/2 transform -translate-y-1/2 h-8 w-px bg-border" />
                            )}
                        </div>
                    ))}
                </div>
            </CardContent>
        </Card>
    );
}