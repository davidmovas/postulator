import { unwrapArrayResponse, unwrapResponse } from "@/lib/utils/error-handling";
import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    CreateCategory,
    CreateInWordPress,
    DeleteCategory,
    DeleteInWordPress,
    GetCategory,
    GetSiteStatistics,
    GetStatistics,
    ListSiteCategories,
    SyncFromWordPress,
    UpdateCategory,
    UpdateInWordPress
} from "@/wailsjs/wailsjs/go/handlers/CategoriesHandler";
import {
    mapCategory,
    mapStatistics,
    Category,
    CategoryCreateInput,
    CategoryUpdateInput,
    Statistics
} from "@/models/categories";

export const categoryService = {
    async createCategory(input: CategoryCreateInput): Promise<void> {
        const payload = new dto.Category({
            siteId: input.siteId,
            wpCategoryId: input.wpCategoryId,
            name: input.name,
            slug: input.slug,
            description: input.description,
        });

        const response = await CreateCategory(payload);
        unwrapResponse<string>(response);
    },

    async getCategory(id: number): Promise<Category> {
        const response = await GetCategory(id);
        const category = unwrapResponse<dto.Category>(response);
        return mapCategory(category);
    },

    async listSiteCategories(siteId: number): Promise<Category[]> {
        const response = await ListSiteCategories(siteId);
        const categories = unwrapArrayResponse<dto.Category>(response);
        return categories.map(mapCategory);
    },

    async updateCategory(input: CategoryUpdateInput): Promise<void> {
        const payload = new dto.Category({
            id: input.id,
            siteId: input.siteId,
            wpCategoryId: input.wpCategoryId,
            name: input.name,
            slug: input.slug,
            description: input.description,
        });

        const response = await UpdateCategory(payload);
        unwrapResponse<string>(response);
    },

    async deleteCategory(id: number): Promise<void> {
        const response = await DeleteCategory(id);
        unwrapResponse<string>(response);
    },

    async syncFromWordPress(siteId: number): Promise<void> {
        const response = await SyncFromWordPress(siteId);
        unwrapResponse<string>(response);
    },

    async createInWordPress(input: CategoryCreateInput): Promise<void> {
        const payload = new dto.Category({
            siteId: input.siteId,
            wpCategoryId: input.wpCategoryId,
            name: input.name,
            slug: input.slug,
            description: input.description,
        });

        const response = await CreateInWordPress(payload);
        unwrapResponse<string>(response);
    },

    async updateInWordPress(input: CategoryUpdateInput): Promise<void> {
        const payload = new dto.Category({
            id: input.id,
            siteId: input.siteId,
            wpCategoryId: input.wpCategoryId,
            name: input.name,
            slug: input.slug,
            description: input.description,
        });

        const response = await UpdateInWordPress(payload);
        unwrapResponse<string>(response);
    },

    async deleteInWordPress(categoryId: number): Promise<void> {
        const response = await DeleteInWordPress(categoryId);
        unwrapResponse<string>(response);
    },

    async getStatistics(categoryId: number, from: string, to: string): Promise<Statistics[]> {
        const response = await GetStatistics(categoryId, from, to);
        const stats = unwrapArrayResponse<dto.Statistics>(response);
        return stats.map(mapStatistics);
    },

    async getSiteStatistics(siteId: number, from: string, to: string): Promise<Statistics[]> {
        const response = await GetSiteStatistics(siteId, from, to);
        const stats = unwrapArrayResponse<dto.Statistics>(response);
        return stats.map(mapStatistics);
    },
};