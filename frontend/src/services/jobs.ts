import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    mapJob,
    Job,
    JobCreateInput,
    JobUpdateInput
} from "@/models/jobs";
import { unwrapArrayResponse, unwrapResponse } from "@/lib/api-utils";
import {
    CreateJob,
    DeleteJob, ExecuteManually,
    GetJob,
    ListJobs,
    PauseJob,
    ResumeJob,
    UpdateJob
} from "@/wailsjs/wailsjs/go/handlers/JobsHandler";

export const jobService = {
    async createJob(input: JobCreateInput): Promise<void> {
        const payload = new dto.Job({
            name: input.name,
            siteId: input.siteId,
            promptId: input.promptId,
            aiProviderId: input.aiProviderId,
            placeholdersValues: input.placeholdersValues,
            topicStrategy: input.topicStrategy,
            categoryStrategy: input.categoryStrategy,
            requiresValidation: input.requiresValidation,
            jitterEnabled: input.jitterEnabled,
            jitterMinutes: input.jitterMinutes,
            status: input.status || 'active',
            schedule: input.schedule,
            categories: input.categories,
            topics: input.topics,
        });

        const response = await CreateJob(payload);
        unwrapResponse<string>(response);
    },

    async getJob(id: number): Promise<Job> {
        const response = await GetJob(id);
        const job = unwrapResponse<dto.Job>(response);
        return mapJob(job);
    },

    async listJobs(): Promise<Job[]> {
        const response = await ListJobs();
        const jobs = unwrapArrayResponse<dto.Job>(response);
        return jobs.map(mapJob);
    },

    async updateJob(input: JobUpdateInput): Promise<void> {
        const payload = new dto.Job({
            id: input.id,
            name: input.name,
            siteId: input.siteId,
            promptId: input.promptId,
            aiProviderId: input.aiProviderId,
            placeholdersValues: input.placeholdersValues,
            topicStrategy: input.topicStrategy,
            categoryStrategy: input.categoryStrategy,
            requiresValidation: input.requiresValidation,
            jitterEnabled: input.jitterEnabled,
            jitterMinutes: input.jitterMinutes,
            status: input.status,
            schedule: input.schedule,
            categories: input.categories,
            topics: input.topics,
        });

        const response = await UpdateJob(payload);
        unwrapResponse<string>(response);
    },

    async deleteJob(id: number): Promise<void> {
        const response = await DeleteJob(id);
        unwrapResponse<string>(response);
    },

    async pauseJob(id: number): Promise<void> {
        const response = await PauseJob(id);
        unwrapResponse<string>(response);
    },

    async resumeJob(id: number): Promise<void> {
        const response = await ResumeJob(id);
        unwrapResponse<string>(response);
    },

    async executeManually(jobId: number): Promise<void> {
        const response = await ExecuteManually(jobId);
        unwrapResponse<string>(response);
    },
};