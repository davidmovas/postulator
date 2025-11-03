import { dto } from "@/wailsjs/wailsjs/go/models";

export interface Topic {
    id: number;
    title: string;
    createdAt: string;
}

export interface TopicCreateInput {
    title: string;
}

export interface TopicUpdateInput extends Partial<TopicCreateInput> {
    id: number;
}

export interface BatchResult {
    created: number;
    skipped: number;
    skippedTitles: string[];
    createdTopics: Topic[];
}

export function mapTopic(x: dto.Topic): Topic {
    return {
        id: x.id,
        title: x.title,
        createdAt: x.createdAt,
    };
}

export function mapBatchResult(x: dto.BatchResult): BatchResult {
    return {
        created: x.created,
        skipped: x.skipped,
        skippedTitles: x.skippedTitles || [],
        createdTopics: (x.createdTopics || []).map(mapTopic),
    };
}