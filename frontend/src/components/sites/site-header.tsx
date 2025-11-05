import { Site } from "@/models/sites";
import { ExternalLink } from "lucide-react";
import SiteStatusBadge from "@/components/sites/site-status-badge";
import HealthIndicator from "@/components/sites/site-health-badge";

interface SiteHeaderProps {
    site: Site;
}

export function SiteHeader({ site }: SiteHeaderProps) {
    return (
        <div className="space-y-4">
            <div className="flex items-center gap-3">
                <h1 className="text-3xl font-bold tracking-tight">{site.name}</h1>
                <SiteStatusBadge status={site.status} />
                <HealthIndicator status={site.healthStatus} />
            </div>

            <div className="flex items-center gap-4 text-sm text-muted-foreground">
                <div className="flex items-center gap-2">
                    <ExternalLink className="h-4 w-4" />
                    <a
                        href={site.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="hover:text-blue-600 hover:underline"
                    >
                        {site.url}
                    </a>
                </div>

                <div className="h-4 w-px bg-border" />

                <div>
                    Username: <span className="font-medium text-foreground">{site.wpUsername}</span>
                </div>
            </div>
        </div>
    );
}