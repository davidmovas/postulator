import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    GetSummary,
    GetUsageByPeriod,
    GetUsageByOperation,
    GetUsageByProvider,
    GetUsageBySite,
} from "@/wailsjs/wailsjs/go/handlers/AIUsageHandler";
import {
    AIUsageSummary,
    AIUsageByPeriod,
    AIUsageByOperation,
    AIUsageByProvider,
    AIUsageBySite,
    mapAIUsageSummary,
    mapAIUsageByPeriod,
    mapAIUsageByOperation,
    mapAIUsageByProvider,
    mapAIUsageBySite,
} from "@/models/aiusage";
import { unwrapArrayResponse, unwrapResponse } from "@/lib/api-utils";

export const aiUsageService = {
    async getSummary(siteId: number, from: string, to: string): Promise<AIUsageSummary> {
        const response = await GetSummary(siteId, from, to);
        const summary = unwrapResponse<dto.AIUsageSummary>(response);
        return mapAIUsageSummary(summary);
    },

    async getUsageByPeriod(siteId: number, from: string, to: string, groupBy: string = "day"): Promise<AIUsageByPeriod[]> {
        const response = await GetUsageByPeriod(siteId, from, to, groupBy);
        const usage = unwrapArrayResponse<dto.AIUsageByPeriod>(response);
        return usage.map(mapAIUsageByPeriod);
    },

    async getUsageByOperation(siteId: number, from: string, to: string): Promise<AIUsageByOperation[]> {
        const response = await GetUsageByOperation(siteId, from, to);
        const usage = unwrapArrayResponse<dto.AIUsageByOperation>(response);
        return usage.map(mapAIUsageByOperation);
    },

    async getUsageByProvider(siteId: number, from: string, to: string): Promise<AIUsageByProvider[]> {
        const response = await GetUsageByProvider(siteId, from, to);
        const usage = unwrapArrayResponse<dto.AIUsageByProvider>(response);
        return usage.map(mapAIUsageByProvider);
    },

    async getUsageBySite(from: string, to: string): Promise<AIUsageBySite[]> {
        const response = await GetUsageBySite(from, to);
        const usage = unwrapArrayResponse<dto.AIUsageBySite>(response);
        return usage.map(mapAIUsageBySite);
    },
};
