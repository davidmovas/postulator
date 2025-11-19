import {
    dto,
} from "@/wailsjs/wailsjs/go/models";
import { unwrapArrayResponse, unwrapResponse } from "@/lib/api-utils";
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
        const payload = unwrapResponse<dto.PaginatedResponse__github_com_davidmovas_postulator_internal_dto_HealthCheckHistory_>(response);
        const items = (payload.items || []).map(mapHealthHistory);
        return {
            items,
            total: payload.total ?? 0,
            limit: payload.limit ?? pageSize,
            offset: payload.offset ?? (Math.max(1, page) - 1) * pageSize,
            hasMore: Boolean(payload.hasMore),
        };
    },

    async checkAutoDetailed(): Promise<AutoCheckResult> {
        const response = await CheckAuto();
        const payload = unwrapResponse<dto.AutoCheckResult>(response);
        return mapAutoCheckResult(payload);
    },
};