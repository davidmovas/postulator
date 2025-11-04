import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    CreateArticle,
    CreateDraft,
    DeleteArticle,
    DeleteFromWordPress,
    GetArticle,
    ImportAllFromSite,
    ImportFromWordPress,
    ListArticles,
    PublishDraft,
    PublishToWordPress,
    SyncFromWordPress,
    UpdateArticle,
    UpdateInWordPress
} from "@/wailsjs/wailsjs/go/handlers/ArticlesHandler";
import {
    mapArticle,
    Article,
    ArticleCreateInput,
    ArticleUpdateInput,
    WPInfoUpdate
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
        });

        const response = await CreateArticle(payload);
        unwrapResponse<string>(response);
    },

    async getArticle(id: number): Promise<Article> {
        const response = await GetArticle(id);
        const article = unwrapResponse<dto.Article>(response);
        return mapArticle(article);
    },

    async listArticles(siteId: number, limit: number, offset: number): Promise<{ items: Article[], total: number }> {
        const response = await ListArticles(siteId, limit, offset);
        const paginated = unwrapResponse<any>(response);
        return {
            items: (paginated.items || []).map(mapArticle),
            total: paginated.total || 0
        };
    },

    async updateArticle(input: ArticleUpdateInput): Promise<void> {
        const payload = new dto.Article({
            id: input.id,
            siteId: input.siteId,
            topicId: input.topicId,
            title: input.title,
            content: input.content,
            excerpt: input.excerpt,
            wpCategoryIds: input.wpCategoryIds,
            status: input.status,
        });

        const response = await UpdateArticle(payload);
        unwrapResponse<string>(response);
    },

    async deleteArticle(id: number): Promise<void> {
        const response = await DeleteArticle(id);
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

    async publishToWordPress(input: ArticleUpdateInput): Promise<void> {
        const payload = new dto.Article({
            id: input.id,
            siteId: input.siteId,
            topicId: input.topicId,
            title: input.title,
            content: input.content,
            excerpt: input.excerpt,
            wpCategoryIds: input.wpCategoryIds,
            status: input.status,
        });

        const response = await PublishToWordPress(payload);
        unwrapResponse<string>(response);
    },

    async updateInWordPress(input: WPInfoUpdate): Promise<void> {
        const payload = new dto.Article({
            id: input.id,
            wpPostId: input.wpPostId,
            wpPostUrl: input.wpPostUrl,
            status: input.status,
            publishedAt: input.publishedAt,
        });

        const response = await UpdateInWordPress(payload);
        unwrapResponse<string>(response);
    },

    async deleteFromWordPress(id: number): Promise<void> {
        const response = await DeleteFromWordPress(id);
        unwrapResponse<string>(response);
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
};