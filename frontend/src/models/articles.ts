import { dto } from "@/wailsjs/wailsjs/go/models";

export interface Article {
    id: number;
    siteId: number;
    jobId?: number;
    topicId: number;
    title: string;
    originalTitle: string;
    content: string;
    excerpt?: string;
    wpPostId: number;
    wpPostUrl: string;
    wpCategoryIds: number[];
    status: string;
    wordCount?: number;
    source: string;
    isEdited: boolean;
    createdAt: string;
    publishedAt?: string;
    updatedAt: string;
    lastSyncedAt?: string;
}

export interface WPInfoUpdate {
    id: number;
    wpPostId: number;
    wpPostUrl: string;
    status: string;
    publishedAt?: string;
}

export interface ArticleCreateInput {
    siteId: number;
    topicId: number;
    title: string;
    content: string;
    excerpt?: string;
    wpCategoryIds: number[];
}

export function mapArticle(x: dto.Article): Article {
    return {
        id: x.id,
        siteId: x.siteId,
        jobId: x.jobId,
        topicId: x.topicId,
        title: x.title,
        originalTitle: x.originalTitle,
        content: x.content,
        excerpt: x.excerpt,
        wpPostId: x.wpPostId,
        wpPostUrl: x.wpPostUrl,
        wpCategoryIds: x.wpCategoryIds || [],
        status: x.status,
        wordCount: x.wordCount,
        source: x.source,
        isEdited: x.isEdited,
        createdAt: x.createdAt,
        publishedAt: x.publishedAt,
        updatedAt: x.updatedAt,
        lastSyncedAt: x.lastSyncedAt,
    };
}

/*
export function mapWPInfoUpdate(x: dto.WPInfoUpdate): WPInfoUpdate {
    return {
        id: x.id,
        wpPostId: x.wpPostId,
        wpPostUrl: x.wpPostUrl,
        status: x.status,
        publishedAt: x.publishedAt,
    };
}*/
