import { dto } from "@/wailsjs/wailsjs/go/models";

export type ArticleStatus = 'draft' | 'published' | 'pending' | 'private' | 'unknown';
export type ArticleSource = 'generated' | 'manual' | 'imported';

export interface Article {
    id: number;
    siteId: number;
    jobId?: number;
    topicId?: number;
    title: string;
    originalTitle: string;
    content: string;
    excerpt?: string;
    wpPostId: number;
    wpPostUrl: string;
    wpCategoryIds: number[];
    wpTagIds: number[];
    status: ArticleStatus;
    wordCount?: number;
    source: ArticleSource;
    isEdited: boolean;
    createdAt: string;
    publishedAt?: string;
    updatedAt: string;
    lastSyncedAt?: string;

    // SEO & WordPress fields
    slug?: string;
    featuredMediaId?: number;
    featuredMediaUrl?: string;
    metaDescription?: string;
    author?: number;
}

export interface ArticleListFilter {
    siteId: number;
    status?: string;
    source?: string;
    categoryId?: number;
    search?: string;
    sortBy?: string;
    sortOrder?: 'asc' | 'desc';
    limit: number;
    offset: number;
}

export interface ArticleListResult {
    articles: Article[];
    total: number;
}

export interface ArticleCreateInput {
    siteId: number;
    topicId?: number;
    title: string;
    content: string;
    excerpt?: string;
    wpCategoryIds?: number[];
    wpTagIds?: number[];
    slug?: string;
    metaDescription?: string;
}

export interface ArticleUpdateInput {
    id: number;
    title?: string;
    content?: string;
    excerpt?: string;
    wpCategoryIds?: number[];
    wpTagIds?: number[];
    status?: ArticleStatus;
    slug?: string;
    metaDescription?: string;
    featuredMediaId?: number | null;  // null means explicitly clear
    featuredMediaUrl?: string | null; // null means explicitly clear
}

export function mapArticle(x: dto.Article): Article {
    return {
        id: x.id,
        siteId: x.siteId,
        jobId: x.jobId ?? undefined,
        topicId: x.topicId ?? undefined,
        title: x.title,
        originalTitle: x.originalTitle,
        content: x.content,
        excerpt: x.excerpt ?? undefined,
        wpPostId: x.wpPostId,
        wpPostUrl: x.wpPostUrl,
        wpCategoryIds: x.wpCategoryIds || [],
        wpTagIds: x.wpTagIds || [],
        status: x.status as ArticleStatus,
        wordCount: x.wordCount ?? undefined,
        source: x.source as ArticleSource,
        isEdited: x.isEdited,
        createdAt: x.createdAt,
        publishedAt: x.publishedAt ?? undefined,
        updatedAt: x.updatedAt,
        lastSyncedAt: x.lastSyncedAt ?? undefined,
        slug: x.slug ?? undefined,
        featuredMediaId: x.featuredMediaId ?? undefined,
        featuredMediaUrl: x.featuredMediaUrl ?? undefined,
        metaDescription: x.metaDescription ?? undefined,
        author: x.author ?? undefined,
    };
}

export function mapArticleListResult(x: dto.ArticleListResult): ArticleListResult {
    return {
        articles: (x.articles || []).map(mapArticle),
        total: x.total,
    };
}

export const articleStatusLabels: Record<ArticleStatus, string> = {
    draft: 'Draft',
    published: 'Published',
    pending: 'Pending',
    private: 'Private',
    unknown: 'Unknown',
};

export const articleSourceLabels: Record<ArticleSource, string> = {
    generated: 'Generated',
    manual: 'Manual',
    imported: 'Imported',
};

export const articleStatusColors: Record<ArticleStatus, string> = {
    draft: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
    published: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
    pending: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
    private: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
    unknown: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400',
};

export interface GenerateContentInput {
    siteId: number;
    providerId: number;
    promptId: number;
    topicId?: number;
    customTopicTitle?: string;
    placeholderValues: Record<string, string>;
}

export interface GenerateContentResult {
    title: string;
    content: string;
    excerpt: string;
    metaDescription: string;
    topicId?: number;
}

export function mapGenerateContentResult(x: dto.GenerateContentResult): GenerateContentResult {
    return {
        title: x.title,
        content: x.content,
        excerpt: x.excerpt,
        metaDescription: x.metaDescription,
        topicId: x.topicId ?? undefined,
    };
}
