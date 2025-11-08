import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    CheckAllHealth,
    CheckHealth,
    CreateSite, DeleteSite,
    GetSite,
    GetSiteWithPassword,
    ListSites,
    UpdateSite,
    UpdateSitePassword,
} from "@/wailsjs/wailsjs/go/handlers/SitesHandler";
import { mapSite, Site, SiteCreateInput, SiteUpdateInput } from "@/models/sites";
import { unwrapArrayResponse, unwrapResponse } from "@/lib/api-utils";

export const siteService = {
    async createSite(input: SiteCreateInput): Promise<string> {
        const payload = new dto.Site({
            name: input.name,
            url: input.url,
            wpUsername: input.wpUsername,
            wpPassword: input.wpPassword,
            autoHealthCheck: input.autoHealthCheck,
        });

        const response = await CreateSite(payload);
        return unwrapResponse<string>(response);
    },

    async getSite(id: number): Promise<Site> {
        const response = await GetSite(id);
        const site = unwrapResponse<dto.Site>(response);
        return mapSite(site);
    },

    async getSiteWithPassword(id: number): Promise<Site> {
        const response = await GetSiteWithPassword(id);
        const site = unwrapResponse<dto.Site>(response);
        return mapSite(site);
    },

    async listSites(): Promise<Site[]> {
        const response = await ListSites();
        const sites = unwrapArrayResponse<dto.Site>(response);
        return sites.map(mapSite);
    },

    async updateSite(input: SiteUpdateInput): Promise<void> {
        const payload = new dto.Site({
            id: input.id,
            name: input.name,
            url: input.url,
            wpUsername: input.wpUsername,
            wpPassword: input.wpPassword,
            status: input.status,
            autoHealthCheck: input.autoHealthCheck,
        });

        const response = await UpdateSite(payload);
        unwrapResponse<string>(response);
    },

    async updateSitePassword(id: number, password: string): Promise<void> {
        const response = await UpdateSitePassword(id, password);
        unwrapResponse<string>(response);
    },

    async deleteSite(id: number): Promise<void> {
        const response = await DeleteSite(id);
        unwrapResponse<string>(response);
    },

    async checkHealth(siteId: number): Promise<string> {
        const response = await CheckHealth(siteId);
        return unwrapResponse<string>(response);
    },

    async checkAllHealth(): Promise<string> {
        const response = await CheckAllHealth();
        return unwrapResponse<string>(response);
    },
};