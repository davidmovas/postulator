"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Save, RefreshCw, RotateCcw, FileText, Eye } from "lucide-react";
import { RiWordpressFill } from "@remixicon/react";
import { useApiCall } from "@/hooks/use-api-call";
import { articleService } from "@/services/articles";
import { categoryService } from "@/services/categories";
import { Article } from "@/models/articles";
import { useArticleForm, EditorMode, ViewMode } from "@/hooks/use-article-form";
import { BlockEditor } from "./block-editor";
import { HtmlEditor } from "./html-editor";
import { SeoSection } from "./sections/seo-section";
import { FeaturedImageSection } from "./sections/featured-image-section";
import { CategoriesTagsSection } from "./sections/categories-tags-section";
import { PublishingSection } from "./sections/publishing-section";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";

interface ArticleEditorProps {
    siteId: number;
    siteName?: string;
    siteUrl?: string;
    article?: Article | null;
}

interface Category {
    id: number;
    name: string;
    slug?: string;
}

export function ArticleEditor({
    siteId,
    siteName,
    siteUrl,
    article,
}: ArticleEditorProps) {
    const router = useRouter();
    const { execute, isLoading: isApiLoading } = useApiCall();

    const {
        formData,
        updateFormData,
        resetForm,
        getCreateInput,
        getUpdateInput,
        isDirty,
        setIsDirty,
        wordCount,
        charCount,
        isValid,
        isEditMode,
    } = useArticleForm({ siteId, article });

    const [viewMode, setViewMode] = useState<ViewMode>("edit");
    const [editorMode, setEditorMode] = useState<EditorMode>("visual");
    const [categories, setCategories] = useState<Category[] | null>(null);
    const [showUnsavedDialog, setShowUnsavedDialog] = useState(false);
    const [pendingNavigation, setPendingNavigation] = useState<string | null>(null);
    const [isSaving, setIsSaving] = useState(false);
    const [isPublishing, setIsPublishing] = useState(false);

    // Load categories
    useEffect(() => {
        const loadCategories = async () => {
            const result = await categoryService.listSiteCategories(siteId);
            setCategories(result || []);
        };
        loadCategories();
    }, [siteId]);

    const handleSyncCategories = useCallback(async () => {
        await execute(
            () => categoryService.syncFromWordPress(siteId),
            {
                successMessage: "Categories synced successfully",
                showSuccessToast: true,
            }
        );
        const result = await categoryService.listSiteCategories(siteId);
        setCategories(result || []);
    }, [siteId, execute]);

    const handleSave = useCallback(async () => {
        if (!isValid) return;

        setIsSaving(true);
        try {
            if (isEditMode) {
                const input = getUpdateInput();
                if (!input) return;

                await execute(
                    () => articleService.updateArticle(input),
                    {
                        successMessage: "Article saved",
                        showSuccessToast: true,
                        errorTitle: "Failed to save article",
                    }
                );
            } else {
                const input = getCreateInput();
                const result = await execute(
                    () => articleService.createArticle(input),
                    {
                        successMessage: "Article created",
                        showSuccessToast: true,
                        errorTitle: "Failed to create article",
                    }
                );

                if (result) {
                    router.push(`/sites/${siteId}/articles`);
                    return;
                }
            }
            setIsDirty(false);
        } finally {
            setIsSaving(false);
        }
    }, [isValid, isEditMode, getUpdateInput, getCreateInput, execute, router, siteId, setIsDirty]);

    const handleSync = useCallback(async () => {
        if (!article) return;

        setIsPublishing(true);
        try {
            await execute(
                () => articleService.updateInWordPress(article.id),
                {
                    successMessage: "Synced with WordPress",
                    showSuccessToast: true,
                    errorTitle: "Failed to sync",
                }
            );
        } finally {
            setIsPublishing(false);
        }
    }, [article, execute]);

    const handlePublish = useCallback(async () => {
        if (!article) return;

        setIsPublishing(true);
        try {
            await execute(
                () => articleService.publishToWordPress(article.id),
                {
                    successMessage: "Published to WordPress",
                    showSuccessToast: true,
                    errorTitle: "Failed to publish",
                }
            );
        } finally {
            setIsPublishing(false);
        }
    }, [article, execute]);

    const handleNavigate = useCallback((path: string) => {
        if (isDirty) {
            setPendingNavigation(path);
            setShowUnsavedDialog(true);
        } else {
            router.push(path);
        }
    }, [isDirty, router]);

    const handleConfirmNavigation = useCallback(() => {
        if (pendingNavigation) {
            router.push(pendingNavigation);
        }
        setShowUnsavedDialog(false);
        setPendingNavigation(null);
    }, [pendingNavigation, router]);

    const isPublished = article && article.wpPostId > 0;

    return (
        <div className="p-6 space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">
                        {isEditMode ? "Edit Article" : "New Article"}
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {siteName || `Site #${siteId}`}
                        {isDirty && <span className="text-amber-500"> • Unsaved changes</span>}
                    </p>
                </div>

                <div className="flex items-center gap-3">
                    {/* View Mode Tabs */}
                    <Tabs defaultValue="edit" value={viewMode} onValueChange={(v) => setViewMode(v as ViewMode)}>
                        <TabsList>
                            <TabsTrigger value="edit" className="gap-2">
                                <FileText className="h-4 w-4" />
                                Edit
                            </TabsTrigger>
                            <TabsTrigger value="preview" className="gap-2">
                                <Eye className="h-4 w-4" />
                                Preview
                            </TabsTrigger>
                        </TabsList>
                    </Tabs>

                    {isDirty && (
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={resetForm}
                            disabled={isSaving}
                        >
                            <RotateCcw className="h-4 w-4 mr-2" />
                            Reset
                        </Button>
                    )}

                    <Button
                        onClick={handleSave}
                        disabled={!isValid || isSaving || isApiLoading}
                    >
                        <Save className="h-4 w-4 mr-2" />
                        {isSaving ? "Saving..." : "Save"}
                    </Button>

                    {isEditMode && isPublished && (
                        <Button
                            variant="wordpress"
                            onClick={handleSync}
                            disabled={isPublishing || isApiLoading}
                        >
                            <RiWordpressFill className="h-4 w-4 mr-2" />
                            {isPublishing ? "Syncing..." : "Sync"}
                        </Button>
                    )}
                </div>
            </div>

            {viewMode === "edit" ? (
                /* Edit Mode */
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                    {/* Main Content - Left Side */}
                    <div className="lg:col-span-2 space-y-6">
                        {/* Title */}
                        <Card>
                            <CardHeader className="pb-3">
                                <CardTitle>Title</CardTitle>
                            </CardHeader>
                            <CardContent>
                                <Input
                                    value={formData.title}
                                    onChange={(e) => updateFormData({ title: e.target.value })}
                                    disabled={isSaving}
                                    placeholder="Enter article title"
                                    className="text-lg"
                                />
                            </CardContent>
                        </Card>

                        {/* Content Editor */}
                        <Card>
                            <CardHeader className="pb-3">
                                <div className="flex items-center justify-between">
                                    <div>
                                        <CardTitle>Content</CardTitle>
                                        <CardDescription>
                                            {wordCount} words • {charCount} characters
                                        </CardDescription>
                                    </div>
                                    <Tabs defaultValue="visual" value={editorMode} onValueChange={(v) => setEditorMode(v as EditorMode)}>
                                        <TabsList>
                                            <TabsTrigger value="visual">Visual</TabsTrigger>
                                            <TabsTrigger value="html">HTML</TabsTrigger>
                                        </TabsList>
                                    </Tabs>
                                </div>
                            </CardHeader>
                            <CardContent>
                                {editorMode === "visual" ? (
                                    <BlockEditor
                                        content={formData.content}
                                        onChange={(html) => updateFormData({ content: html })}
                                        disabled={isSaving}
                                    />
                                ) : (
                                    <HtmlEditor
                                        content={formData.content}
                                        onChange={(html) => updateFormData({ content: html })}
                                        disabled={isSaving}
                                    />
                                )}
                            </CardContent>
                        </Card>

                        {/* Excerpt */}
                        <Card>
                            <CardHeader className="pb-3">
                                <CardTitle>Excerpt</CardTitle>
                                <CardDescription>
                                    Brief summary of the article
                                </CardDescription>
                            </CardHeader>
                            <CardContent>
                                <Textarea
                                    value={formData.excerpt}
                                    onChange={(e) => updateFormData({ excerpt: e.target.value })}
                                    disabled={isSaving}
                                    placeholder="Write a short summary..."
                                    rows={3}
                                />
                            </CardContent>
                        </Card>

                        {/* SEO */}
                        <SeoSection
                            formData={formData}
                            onUpdate={updateFormData}
                            disabled={isSaving}
                            siteUrl={siteUrl}
                        />
                    </div>

                    {/* Sidebar - Right Side */}
                    <div className="space-y-6">
                        <PublishingSection
                            formData={formData}
                            onUpdate={updateFormData}
                            disabled={isSaving}
                            isPublished={!!isPublished}
                            wpPostUrl={article?.wpPostUrl}
                            wpPostId={article?.wpPostId}
                            createdAt={article?.createdAt}
                            updatedAt={article?.updatedAt}
                            onPublish={isEditMode && !isPublished ? handlePublish : undefined}
                            onSync={isEditMode && isPublished ? handleSync : undefined}
                            isPublishing={isPublishing}
                        />

                        <FeaturedImageSection
                            formData={formData}
                            onUpdate={updateFormData}
                            disabled={isSaving}
                        />

                        <CategoriesTagsSection
                            formData={formData}
                            onUpdate={updateFormData}
                            categories={categories}
                            disabled={isSaving}
                            onSyncCategories={handleSyncCategories}
                        />
                    </div>
                </div>
            ) : (
                /* Preview Mode */
                <Card>
                    <CardHeader>
                        <CardTitle>{formData.title || "Untitled Article"}</CardTitle>
                        {formData.excerpt && (
                            <CardDescription className="text-base">
                                {formData.excerpt}
                            </CardDescription>
                        )}
                    </CardHeader>
                    <CardContent>
                        {formData.featuredMediaUrl && (
                            <img
                                src={formData.featuredMediaUrl}
                                alt={formData.title}
                                className="w-full max-h-96 object-cover rounded-lg mb-6"
                            />
                        )}
                        <div
                            className="prose prose-lg dark:prose-invert max-w-none"
                            dangerouslySetInnerHTML={{ __html: formData.content }}
                        />
                    </CardContent>
                </Card>
            )}

            {/* Unsaved Changes Dialog */}
            <AlertDialog open={showUnsavedDialog} onOpenChange={setShowUnsavedDialog}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Unsaved Changes</AlertDialogTitle>
                        <AlertDialogDescription>
                            You have unsaved changes. Are you sure you want to leave? Your changes will be lost.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel onClick={() => setPendingNavigation(null)}>
                            Cancel
                        </AlertDialogCancel>
                        <AlertDialogAction onClick={handleConfirmNavigation}>
                            Leave
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
