import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    CreateArticle,
    CreateAndPublishArticle,
    CreateDraft,
    DeleteArticle,
    DeleteFromWordPress,
    GenerateContent,
    GetArticle,
    ImportAllFromSite,
    ImportFromWordPress,
    ListArticles,
    PublishDraft,
    PublishToWordPress,
    SyncFromWordPress,
    UpdateArticle,
    UpdateAndSyncArticle,
    UpdateInWordPress,
    BulkDeleteArticles,
    BulkPublishToWordPress,
    BulkDeleteFromWordPress,
} from "@/wailsjs/wailsjs/go/handlers/ArticlesHandler";
import {
    mapArticle,
    mapArticleListResult,
    mapGenerateContentResult,
    Article,
    ArticleCreateInput,
    ArticleUpdateInput,
    ArticleListFilter,
    ArticleListResult,
    GenerateContentInput,
    GenerateContentResult,
} from "@/models/articles";
import { unwrapResponse } from "@/lib/api-utils";

export const articleService = {
    async createArticle(input: ArticleCreateInput): Promise<void> {
        const payload = new dto.Article({
            siteId: input.siteId,
            topicId: input.topicId,
            title: input.title,
            content: input.content,
            excerpt: input.excerpt,
            wpCategoryIds: input.wpCategoryIds,
            wpTagIds: input.wpTagIds,
            slug: input.slug,
            metaDescription: input.metaDescription,
        });

        const response = await CreateArticle(payload);
        unwrapResponse<string>(response);
    },

    async createAndPublishArticle(input: ArticleCreateInput): Promise<Article> {
        const payload = new dto.Article({
            siteId: input.siteId,
            topicId: input.topicId,
            title: input.title,
            content: input.content,
            excerpt: input.excerpt,
            wpCategoryIds: input.wpCategoryIds,
            wpTagIds: input.wpTagIds,
            slug: input.slug,
            metaDescription: input.metaDescription,
        });

        const response = await CreateAndPublishArticle(payload);
        const article = unwrapResponse<dto.Article>(response);
        return mapArticle(article);
    },

    async getArticle(id: number): Promise<Article> {
        const response = await GetArticle(id);
        const article = unwrapResponse<dto.Article>(response);
        return mapArticle(article);
    },

    async listArticles(filter: ArticleListFilter): Promise<ArticleListResult> {
        const payload = new dto.ArticleListFilter({
            siteId: filter.siteId,
            status: filter.status,
            source: filter.source,
            categoryId: filter.categoryId,
            search: filter.search,
            sortBy: filter.sortBy || 'created_at',
            sortOrder: filter.sortOrder || 'desc',
            limit: filter.limit,
            offset: filter.offset,
        });

        const response = await ListArticles(payload);
        const result = unwrapResponse<dto.ArticleListResult>(response);
        return mapArticleListResult(result);
    },

    async updateArticle(input: ArticleUpdateInput): Promise<void> {
        const article = await this.getArticle(input.id);

        // Handle featured media - null means explicitly clear, undefined means keep existing
        const featuredMediaId = input.featuredMediaId === null
            ? undefined
            : (input.featuredMediaId ?? article.featuredMediaId);
        const featuredMediaUrl = input.featuredMediaUrl === null
            ? undefined
            : (input.featuredMediaUrl ?? article.featuredMediaUrl);

        const payload = new dto.Article({
            id: input.id,
            siteId: article.siteId,
            topicId: article.topicId,
            title: input.title ?? article.title,
            originalTitle: article.originalTitle,
            content: input.content ?? article.content,
            excerpt: input.excerpt ?? article.excerpt,
            wpCategoryIds: input.wpCategoryIds ?? article.wpCategoryIds,
            wpTagIds: input.wpTagIds ?? article.wpTagIds,
            status: input.status ?? article.status,
            slug: input.slug ?? article.slug,
            metaDescription: input.metaDescription ?? article.metaDescription,
            featuredMediaId,
            featuredMediaUrl,
        });

        const response = await UpdateArticle(payload);
        unwrapResponse<string>(response);
    },

    async updateAndSyncArticle(input: ArticleUpdateInput): Promise<Article> {
        const article = await this.getArticle(input.id);

        // Handle featured media - null means explicitly clear, undefined means keep existing
        const featuredMediaId = input.featuredMediaId === null
            ? undefined
            : (input.featuredMediaId ?? article.featuredMediaId);
        const featuredMediaUrl = input.featuredMediaUrl === null
            ? undefined
            : (input.featuredMediaUrl ?? article.featuredMediaUrl);

        const payload = new dto.Article({
            id: input.id,
            siteId: article.siteId,
            topicId: article.topicId,
            title: input.title ?? article.title,
            originalTitle: article.originalTitle,
            content: input.content ?? article.content,
            excerpt: input.excerpt ?? article.excerpt,
            wpCategoryIds: input.wpCategoryIds ?? article.wpCategoryIds,
            wpTagIds: input.wpTagIds ?? article.wpTagIds,
            status: input.status ?? article.status,
            slug: input.slug ?? article.slug,
            metaDescription: input.metaDescription ?? article.metaDescription,
            featuredMediaId,
            featuredMediaUrl,
        });

        const response = await UpdateAndSyncArticle(payload);
        const result = unwrapResponse<dto.Article>(response);
        return mapArticle(result);
    },

    async deleteArticle(id: number): Promise<void> {
        const response = await DeleteArticle(id);
        unwrapResponse<string>(response);
    },

    async bulkDeleteArticles(ids: number[]): Promise<void> {
        const response = await BulkDeleteArticles(ids);
        unwrapResponse<string>(response);
    },

    async importFromWordPress(siteId: number, wpPostId: number): Promise<Article> {
        const response = await ImportFromWordPress(siteId, wpPostId);
        const article = unwrapResponse<dto.Article>(response);
        return mapArticle(article);
    },

    async importAllFromSite(siteId: number): Promise<number> {
        const response = await ImportAllFromSite(siteId);
        return unwrapResponse<number>(response);
    },

    async syncFromWordPress(siteId: number): Promise<void> {
        const response = await SyncFromWordPress(siteId);
        unwrapResponse<string>(response);
    },

    async publishToWordPress(id: number): Promise<void> {
        const response = await PublishToWordPress(id);
        unwrapResponse<string>(response);
    },

    async bulkPublishToWordPress(ids: number[]): Promise<number> {
        const response = await BulkPublishToWordPress(ids);
        return unwrapResponse<number>(response);
    },

    async updateInWordPress(id: number): Promise<void> {
        const response = await UpdateInWordPress(id);
        unwrapResponse<string>(response);
    },

    async deleteFromWordPress(id: number): Promise<void> {
        const response = await DeleteFromWordPress(id);
        unwrapResponse<string>(response);
    },

    async bulkDeleteFromWordPress(ids: number[]): Promise<number> {
        const response = await BulkDeleteFromWordPress(ids);
        return unwrapResponse<number>(response);
    },

    async createDraft(execId: number, title: string, content: string): Promise<Article> {
        const response = await CreateDraft(execId, title, content);
        const article = unwrapResponse<dto.Article>(response);
        return mapArticle(article);
    },

    async publishDraft(id: number): Promise<void> {
        const response = await PublishDraft(id);
        unwrapResponse<string>(response);
    },

    async generateContent(input: GenerateContentInput): Promise<GenerateContentResult> {
        const payload = new dto.GenerateContentInput({
            siteId: input.siteId,
            providerId: input.providerId,
            promptId: input.promptId,
            topicId: input.topicId,
            customTopicTitle: input.customTopicTitle || '',
            placeholderValues: input.placeholderValues,
            useWebSearch: input.useWebSearch || false,
        });

        const response = await GenerateContent(payload);
        const result = unwrapResponse<dto.GenerateContentResult>(response);
        return mapGenerateContentResult(result);
    },
};
