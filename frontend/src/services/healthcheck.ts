import {
    dto,
} from "@/wailsjs/wailsjs/go/models";
import { unwrapArrayResponse, unwrapPaginatedResponse, unwrapResponse } from "@/lib/api-utils";
import { AutoCheckResult, HealthCheckHistory, mapAutoCheckResult, mapHealthHistory } from "@/models/healthcheck";
import { PaginatedResponse } from "@/models/common";
import { CheckAuto, CheckSite, GetHistory, GetHistoryByPeriod } from "@/wailsjs/wailsjs/go/handlers/HealthCheckHandler";

export const healthcheckService = {
    async checkHealth(siteId: number): Promise<string> {
        const response = await CheckSite(siteId);
        unwrapResponse<dto.HealthCheckHistory>(response);
        return "Site health checked";
    },

    async checkAllHealth(): Promise<string> {
        const response = await CheckAuto();
        unwrapResponse<dto.AutoCheckResult>(response);
        return "All sites health checked";
    },

    async getHistory(siteId: number, limit: number): Promise<HealthCheckHistory[]> {
        const response = await GetHistory(siteId, limit);
        const list = unwrapArrayResponse<dto.HealthCheckHistory>(response);
        return list.map(mapHealthHistory);
    },

    async getHistoryByPeriod(siteId: number, from: string, to: string, page: number, pageSize: number): Promise<PaginatedResponse<HealthCheckHistory>> {
        const response = await GetHistoryByPeriod(siteId, from, to, page, pageSize);
        return unwrapPaginatedResponse<HealthCheckHistory, any>(response, mapHealthHistory);
    },

    async checkAutoDetailed(): Promise<AutoCheckResult> {
        const response = await CheckAuto();
        const payload = unwrapResponse<dto.AutoCheckResult>(response);
        return mapAutoCheckResult(payload);
    },
};