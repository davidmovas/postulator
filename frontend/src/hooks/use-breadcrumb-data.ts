import { useState } from "react";
import { siteService } from "@/services/sites";

export function useBreadcrumbData() {
    const [siteNames, setSiteNames] = useState<Record<number, string>>({});

    const getSiteName = async (siteId: number): Promise<string> => {
        if (siteNames[siteId]) {
            return siteNames[siteId];
        }

        try {
            const site = await siteService.getSite(siteId);
            const name = site.name;
            setSiteNames(prev => ({ ...prev, [siteId]: name }));
            return name;
        } catch {
            return `Site ${siteId}`;
        }
    };

    return { getSiteName };
}