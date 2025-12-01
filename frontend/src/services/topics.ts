import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    AssignToSite,
    CreateTopic,
    CreateTopics,
    DeleteTopic,
    GenerateVariations,
    GetNextTopicForJob,
    GetOrGenerateVariation, GetSelectableSiteTopics,
    GetSiteTopics,
    GetUnusedSiteTopics,
    GetTopic,
    ListTopics,
    MarkTopicUsed,
    GetJobRemainingTopics,
    UnassignFromSite,
    UpdateTopic, CreateAndAssignToSite
} from "@/wailsjs/wailsjs/go/handlers/TopicsHandler";
import {
    mapTopic,
    mapBatchResult,
    Topic,
    TopicCreateInput,
    TopicUpdateInput,
    BatchResult,
    JobTopicsStatus,
    mapJobTopicsStatus,
} from "@/models/topics";
import { unwrapArrayResponse, unwrapResponse, unwrapTopicsResponse } from "@/lib/api-utils";
import { ImportResult, mapImportResult } from "@/models/importer";
import { ImportAndAssignToSite } from "@/wailsjs/wailsjs/go/handlers/ImporterHandler";

export const topicService = {
    async getTopic(id: number): Promise<Topic> {
        const response = await GetTopic(id);
        const topic = unwrapResponse<dto.Topic>(response);
        return mapTopic(topic);
    },

    async listTopics(): Promise<Topic[]> {
        const response = await ListTopics();
        const topics = unwrapArrayResponse<dto.Topic>(response);
        return topics.map(mapTopic);
    },

    async updateTopic(input: TopicUpdateInput): Promise<void> {
        const payload = new dto.Topic({
            id: input.id,
            title: input.title,
        });

        const response = await UpdateTopic(payload);
        unwrapResponse<string>(response);
    },

    async createTopic(input: TopicCreateInput): Promise<void> {
        const payload = new dto.Topic({
            title: input.title,
        });

        const response = await CreateTopic(payload);
        unwrapResponse<string>(response);
    },

    async createTopics(topics: TopicCreateInput[]): Promise<BatchResult> {
        const payload = topics.map(topic => new dto.Topic({
            title: topic.title,
        }));

        const response = await CreateTopics(payload);
        const result = unwrapResponse<dto.BatchResult>(response);
        return mapBatchResult(result);
    },

    async createAndAssignToSite(siteId: number, topics: TopicCreateInput[]): Promise<ImportResult> {
        const payload = topics.map(topic => new dto.Topic({
            title: topic.title,
        }));

        const response = await CreateAndAssignToSite(siteId, payload);
        const result = unwrapResponse<dto.ImportResult>(response);
        return mapImportResult(result);
    },

    async deleteTopic(id: number): Promise<void> {
        const response = await DeleteTopic(id);
        unwrapResponse<string>(response);
    },

    async assignToSite(siteId: number, topicIds: number[]): Promise<void> {
        const response = await AssignToSite(siteId, topicIds);
        unwrapResponse<string>(response);
    },

    async unassignFromSite(siteId: number, topicIds: number[]): Promise<void> {
        const response = await UnassignFromSite(siteId, topicIds);
        unwrapResponse<string>(response);
    },

    async getSiteTopics(siteId: number): Promise<Topic[]> {
        const response = await GetSiteTopics(siteId);
        const topics = unwrapArrayResponse<dto.Topic>(response);
        return topics.map(mapTopic);
    },

    async getUnusedSiteTopics(siteId: number): Promise<Topic[]> {
        const response = await GetUnusedSiteTopics(siteId);
        const topics = unwrapArrayResponse<dto.Topic>(response);
        return topics.map(mapTopic);
    },

    async getSelectableTopics(siteId: number, strategy: string): Promise<Topic[]> {
        const response = await GetSelectableSiteTopics(siteId, strategy);
        const topics = unwrapArrayResponse<dto.Topic>(response);
        return topics.map(mapTopic);
    },

    async getNextTopicForJob(jobId: number): Promise<Topic | null> {
        const response = await GetNextTopicForJob(jobId);
        const topic = unwrapResponse<dto.Topic | null>(response, { allowNull: true });
        return topic ? mapTopic(topic) : null;
    },

    async generateVariations(topicId: number, count: number, jobId: number): Promise<Topic[]> {
        const response = await GenerateVariations(topicId, count, jobId);
        const topics = unwrapArrayResponse<dto.Topic>(response);
        return topics.map(mapTopic);
    },

    async getOrGenerateVariation(topicId: number, jobId: number, categoryId: number): Promise<Topic> {
        const response = await GetOrGenerateVariation(topicId, jobId, categoryId);
        const topic = unwrapResponse<dto.Topic>(response);
        return mapTopic(topic);
    },

    async markTopicUsed(topicId: number, jobId: number): Promise<void> {
        const response = await MarkTopicUsed(topicId, jobId);
        unwrapResponse<string>(response);
    },

    async getJobRemainingTopics(jobId: number): Promise<JobTopicsStatus> {
        const response = await GetJobRemainingTopics(jobId);

        const defaultValue: dto.JobTopicsStatus = {
            count: 0,
            topics: [],
            convertValues(a: any, classs: any, asMap: boolean = false): any {}
        };

        const payload = unwrapResponse<dto.JobTopicsStatus>(response, {
            suppressErrors: true,
            defaultValue: defaultValue
        });
        return mapJobTopicsStatus(payload);
    },
};