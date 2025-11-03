import { dto } from "@/wailsjs/wailsjs/go/models";

export interface SiteStats {
    id: number;
    siteId: number;
    date: string;
    articlesPublished: number;
    articlesFailed: number;
    totalWords: number;
    internalLinksCreated: number;
    externalLinksCreated: number;
}

export interface DashboardSummary {
    totalSites: number;
    activeSites: number;
    unhealthySites: number;
    totalJobs: number;
    activeJobs: number;
    pausedJobs: number;
    pendingValidations: number;
    executionsToday: number;
    failedExecutionsToday: number;
}

export function mapSiteStats(x: dto.SiteStats): SiteStats {
    return {
        id: x.id,
        siteId: x.siteId,
        date: x.date,
        articlesPublished: x.articlesPublished,
        articlesFailed: x.articlesFailed,
        totalWords: x.totalWords,
        internalLinksCreated: x.internalLinksCreated,
        externalLinksCreated: x.externalLinksCreated,
    };
}

export function mapDashboardSummary(x: dto.DashboardSummary): DashboardSummary {
    return {
        totalSites: x.totalSites,
        activeSites: x.activeSites,
        unhealthySites: x.unhealthySites,
        totalJobs: x.totalJobs,
        activeJobs: x.activeJobs,
        pausedJobs: x.pausedJobs,
        pendingValidations: x.pendingValidations,
        executionsToday: x.executionsToday,
        failedExecutionsToday: x.failedExecutionsToday,
    };
}