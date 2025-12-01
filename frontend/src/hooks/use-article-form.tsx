"use client";

import { useState, useCallback, useMemo } from "react";
import { Article, ArticleCreateInput, ArticleUpdateInput } from "@/models/articles";

export type EditorMode = "visual" | "html";
export type ViewMode = "edit" | "preview";

export interface ArticleFormData {
    title: string;
    content: string;
    excerpt: string;
    slug: string;
    metaDescription: string;
    status: "draft" | "published";
    categoryIds: number[];
    tagIds: number[];
    featuredMediaId?: number;
    featuredMediaUrl?: string;
    author?: number;
    publishedAt?: string;
}

interface UseArticleFormOptions {
    siteId: number;
    article?: Article | null;
}

const defaultFormData: ArticleFormData = {
    title: "",
    content: "",
    excerpt: "",
    slug: "",
    metaDescription: "",
    status: "published",
    categoryIds: [],
    tagIds: [],
    featuredMediaId: undefined,
    featuredMediaUrl: undefined,
    author: undefined,
    publishedAt: undefined,
};

export function useArticleForm({ siteId, article }: UseArticleFormOptions) {
    const [formData, setFormData] = useState<ArticleFormData>(() => {
        if (article) {
            return {
                title: article.title || "",
                content: article.content || "",
                excerpt: article.excerpt || "",
                slug: article.slug || "",
                metaDescription: article.metaDescription || "",
                status: article.status as "draft" | "published",
                categoryIds: article.wpCategoryIds || [],
                tagIds: article.wpTagIds || [],
                featuredMediaId: article.featuredMediaId,
                featuredMediaUrl: article.featuredMediaUrl,
                author: article.author,
                publishedAt: article.publishedAt,
            };
        }
        return defaultFormData;
    });

    const [editorMode, setEditorMode] = useState<EditorMode>("visual");
    const [isLoading, setIsLoading] = useState(false);
    const [isDirty, setIsDirty] = useState(false);

    const updateFormData = useCallback((updates: Partial<ArticleFormData>) => {
        setFormData(prev => ({ ...prev, ...updates }));
        setIsDirty(true);
    }, []);

    const resetForm = useCallback(() => {
        if (article) {
            setFormData({
                title: article.title || "",
                content: article.content || "",
                excerpt: article.excerpt || "",
                slug: article.slug || "",
                metaDescription: article.metaDescription || "",
                status: article.status as "draft" | "published",
                categoryIds: article.wpCategoryIds || [],
                tagIds: article.wpTagIds || [],
                featuredMediaId: article.featuredMediaId,
                featuredMediaUrl: article.featuredMediaUrl,
                author: article.author,
                publishedAt: article.publishedAt,
            });
        } else {
            setFormData(defaultFormData);
        }
        setIsDirty(false);
    }, [article]);

    const getCreateInput = useCallback((): ArticleCreateInput => {
        return {
            siteId,
            title: formData.title.trim(),
            content: formData.content.trim(),
            excerpt: formData.excerpt.trim() || undefined,
            slug: formData.slug.trim() || undefined,
            metaDescription: formData.metaDescription.trim() || undefined,
        };
    }, [siteId, formData]);

    const getUpdateInput = useCallback((): ArticleUpdateInput | null => {
        if (!article) return null;
        return {
            id: article.id,
            title: formData.title.trim(),
            content: formData.content.trim(),
            excerpt: formData.excerpt.trim() || undefined,
            slug: formData.slug.trim() || undefined,
            metaDescription: formData.metaDescription.trim() || undefined,
        };
    }, [article, formData]);

    const wordCount = useMemo(() => {
        return formData.content.split(/\s+/).filter(w => w.length > 0).length;
    }, [formData.content]);

    const charCount = useMemo(() => {
        return formData.content.length;
    }, [formData.content]);

    const isValid = useMemo(() => {
        return formData.title.trim().length > 0 && formData.content.trim().length > 0;
    }, [formData.title, formData.content]);

    return {
        formData,
        updateFormData,
        resetForm,
        getCreateInput,
        getUpdateInput,
        editorMode,
        setEditorMode,
        isLoading,
        setIsLoading,
        isDirty,
        setIsDirty,
        wordCount,
        charCount,
        isValid,
        isEditMode: !!article,
    };
}
