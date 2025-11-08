import { dto } from "@/wailsjs/wailsjs/go/models";
import { Site, mapSite } from "@/models/sites";

export interface HealthCheckHistory {
    id: number;
    siteId: number;
    checkedAt: string;
    status: string;
    responseTimeMs: number;
    statusCode: number;
    errorMessage: string;
}

export function mapHealthHistory(x: dto.HealthCheckHistory): HealthCheckHistory {
    return {
        id: x.id,
        siteId: x.siteId,
        checkedAt: x.checkedAt,
        status: x.status,
        responseTimeMs: x.responseTimeMs,
        statusCode: x.statusCode,
        errorMessage: x.errorMessage,
    };
}

export interface AutoCheckResult {
    unhealthy: Site[];
    recovered: Site[];
}

export function mapAutoCheckResult(x: dto.AutoCheckResult): AutoCheckResult {
    const unhealthy = (x.unhealthy || []).map(mapSite);
    const recovered = (x.recovered || []).map(mapSite);
    return { unhealthy, recovered };
}
