import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    GetDashboardSummary,
    GetSiteStatistics,
    GetTotalStatistics
} from "@/wailsjs/wailsjs/go/handlers/StatsHandler";
import {
    mapSiteStats,
    mapDashboardSummary,
    SiteStats,
    DashboardSummary
} from "@/models/stats";
import { unwrapResponse } from "@/lib/api-utils";

export const statsService = {
    async getSiteStatistics(siteId: number, from: string, to: string): Promise<SiteStats[]> {
        const response = await GetSiteStatistics(siteId, from, to);
        const stats = unwrapResponse<dto.SiteStats[]>(response);
        return stats.map(mapSiteStats);
    },

    async getTotalStatistics(siteId: number): Promise<SiteStats> {
        const response = await GetTotalStatistics(siteId);
        const stats = unwrapResponse<dto.SiteStats>(response);
        return mapSiteStats(stats);
    },

    async getDashboardSummary(): Promise<DashboardSummary> {
        const response = await GetDashboardSummary();
        const summary = unwrapResponse<dto.DashboardSummary>(response);
        return mapDashboardSummary(summary);
    },
};