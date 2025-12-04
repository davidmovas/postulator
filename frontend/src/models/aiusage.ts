import { dto } from "@/wailsjs/wailsjs/go/models";

export interface AIUsageLog {
    id: number;
    siteId: number;
    operationType: string;
    providerName: string;
    modelName: string;
    inputTokens: number;
    outputTokens: number;
    totalTokens: number;
    costUsd: number;
    durationMs: number;
    success: boolean;
    errorMessage?: string;
    metadata?: string;
    createdAt: string;
}

export interface AIUsageSummary {
    totalRequests: number;
    totalTokens: number;
    totalInputTokens: number;
    totalOutputTokens: number;
    totalCostUsd: number;
    successCount: number;
    errorCount: number;
}

export interface AIUsageByPeriod {
    period: string;
    totalTokens: number;
    totalCostUsd: number;
    requestCount: number;
}

export interface AIUsageByOperation {
    operationType: string;
    totalTokens: number;
    totalCostUsd: number;
    requestCount: number;
}

export interface AIUsageByProvider {
    providerName: string;
    modelName: string;
    totalTokens: number;
    totalCostUsd: number;
    requestCount: number;
}

export interface AIUsageBySite {
    siteId: number;
    siteName: string;
    totalTokens: number;
    totalCostUsd: number;
    requestCount: number;
}

export function mapAIUsageSummary(x: dto.AIUsageSummary): AIUsageSummary {
    return {
        totalRequests: x.totalRequests,
        totalTokens: x.totalTokens,
        totalInputTokens: x.totalInputTokens,
        totalOutputTokens: x.totalOutputTokens,
        totalCostUsd: x.totalCostUsd,
        successCount: x.successCount,
        errorCount: x.errorCount,
    };
}

export function mapAIUsageByPeriod(x: dto.AIUsageByPeriod): AIUsageByPeriod {
    return {
        period: x.period,
        totalTokens: x.totalTokens,
        totalCostUsd: x.totalCostUsd,
        requestCount: x.requestCount,
    };
}

export function mapAIUsageByOperation(x: dto.AIUsageByOperation): AIUsageByOperation {
    return {
        operationType: x.operationType,
        totalTokens: x.totalTokens,
        totalCostUsd: x.totalCostUsd,
        requestCount: x.requestCount,
    };
}

export function mapAIUsageByProvider(x: dto.AIUsageByProvider): AIUsageByProvider {
    return {
        providerName: x.providerName,
        modelName: x.modelName,
        totalTokens: x.totalTokens,
        totalCostUsd: x.totalCostUsd,
        requestCount: x.requestCount,
    };
}

export function mapAIUsageBySite(x: dto.AIUsageBySite): AIUsageBySite {
    return {
        siteId: x.siteId,
        siteName: x.siteName,
        totalTokens: x.totalTokens,
        totalCostUsd: x.totalCostUsd,
        requestCount: x.requestCount,
    };
}
