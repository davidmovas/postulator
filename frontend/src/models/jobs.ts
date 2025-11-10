import { dto } from "@/wailsjs/wailsjs/go/models";

export interface Job {
    id: number;
    name: string;
    siteId: number;
    promptId: number;
    aiProviderId: number;
    placeholdersValues:  Record<string, string>;
    topicStrategy: string;
    categoryStrategy: string;
    requiresValidation: boolean;
    jitterEnabled: boolean;
    jitterMinutes: number;
    status: string;
    createdAt: string;
    updatedAt: string;
    schedule?: Schedule;
    state?: State;
    categories: number[];
    topics: number[];
}

export interface JobCreateInput {
    name: string;
    siteId: number;
    promptId: number;
    aiProviderId: number;
    placeholdersValues:  Record<string, string>;
    topicStrategy: string;
    categoryStrategy: string;
    requiresValidation: boolean;
    jitterEnabled: boolean;
    jitterMinutes: number;
    status?: string;
    schedule?: Schedule;
    categories: number[];
    topics: number[];
}

export interface JobUpdateInput extends Partial<JobCreateInput> {
    id: number;
    status?: string;
}

export interface Schedule {
    type: string;
    config: any;
}

export interface OnceSchedule {
    executeAt: string;
}

export interface IntervalSchedule {
    value: number;
    unit: string;
    startAt?: string;
}

export interface DailySchedule {
    hour: number;
    minute: number;
    weekdays: number[];
}

export interface State {
    jobId: number;
    lastRunAt?: string;
    nextRunAt?: string;
    totalExecutions: number;
    failedExecutions: number;
    lastCategoryIndex: number;
}

export function mapJob(x: dto.Job): Job {
    return {
        id: x.id,
        name: x.name,
        siteId: x.siteId,
        promptId: x.promptId,
        aiProviderId: x.aiProviderId,
        placeholdersValues: x.placeholdersValues,
        topicStrategy: x.topicStrategy,
        categoryStrategy: x.categoryStrategy,
        requiresValidation: x.requiresValidation,
        jitterEnabled: x.jitterEnabled,
        jitterMinutes: x.jitterMinutes,
        status: x.status,
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
        schedule: x.schedule,
        state: x.state,
        categories: x.categories || [],
        topics: x.topics || [],
    };
}

export function mapState(x: dto.State): State {
    return {
        jobId: x.jobId,
        lastRunAt: x.lastRunAt,
        nextRunAt: x.nextRunAt,
        totalExecutions: x.totalExecutions,
        failedExecutions: x.failedExecutions,
        lastCategoryIndex: x.lastCategoryIndex,
    };
}
