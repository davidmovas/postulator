import { unwrapArrayResponse, unwrapResponse } from "@/lib/utils/error-handling";
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
import { mapSite, SiteCreateInput, SiteUpdateInput } from "@/models/sites";
import Site = dto.Site;

export const siteService = {
    async createSite(input: SiteCreateInput): Promise<void> {
        const payload = new dto.Site({
            name: input.name,
            url: input.url,
            wpUsername: input.wpUsername,
            wpPassword: input.wpPassword,
        });

        const response = await CreateSite(payload);
        unwrapResponse<string>(response);
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