import { dto } from "@/wailsjs/wailsjs/go/models";

export interface Category {
    id: number;
    siteId: number;
    wpCategoryId: number;
    name: string;
    slug?: string;
    description?: string;
    count: number;
    createdAt: string;
    updatedAt: string;
}

export interface CategoryCreateInput {
    siteId: number;
    wpCategoryId: number;
    name: string;
    slug?: string;
    description?: string;
}

export interface Statistics {
    categoryId: number;
    date: string;
    articlesPublished: number;
    totalWords: number;
}

export function mapCategory(x: dto.Category): Category {
    return {
        id: x.id,
        siteId: x.siteId,
        wpCategoryId: x.wpCategoryId,
        name: x.name,
        slug: x.slug,
        description: x.description,
        count: x.count,
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}

export function mapStatistics(x: dto.Statistics): Statistics {
    return {
        categoryId: x.categoryId,
        date: x.date,
        articlesPublished: x.articlesPublished,
        totalWords: x.totalWords,
    };
}