"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Save, RefreshCw, RotateCcw, FileText, Eye, Info, Sparkles } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
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
import { AIGenerateModal } from "./ai-generate-modal";
import { GenerateContentResult } from "@/models/articles";
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
        updateFormDataSilent,
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
    const [showAIGenerateModal, setShowAIGenerateModal] = useState(false);

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

                // Update and sync to WordPress in one call
                const result = await execute(
                    () => articleService.updateAndSyncArticle(input),
                    {
                        successMessage: "Article saved and synced to WordPress",
                        showSuccessToast: true,
                        errorTitle: "Failed to save article",
                    }
                );

                if (result) {
                    router.push(`/sites/articles?id=${siteId}`);
                }
            } else {
                const input = getCreateInput();
                // Create and publish to WordPress in one call
                const result = await execute(
                    () => articleService.createAndPublishArticle(input),
                    {
                        successMessage: "Article created and published to WordPress",
                        showSuccessToast: true,
                        errorTitle: "Failed to create article",
                    }
                );

                if (result) {
                    router.push(`/sites/articles?id=${siteId}`);
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

    const handleAIGenerated = useCallback((result: GenerateContentResult) => {
        updateFormData({
            title: result.title,
            content: result.content,
            excerpt: result.excerpt,
            metaDescription: result.metaDescription,
            topicId: result.topicId,
        });
    }, [updateFormData]);

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
                            variant="outline"
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
                                    <div className="flex items-center gap-2">
                                        <Button
                                            variant="ai"
                                            size="sm"
                                            onClick={() => setShowAIGenerateModal(true)}
                                            disabled={isSaving}
                                        >
                                            <Sparkles className="h-4 w-4 mr-2" />
                                            Generate with AI
                                        </Button>
                                        <Tabs defaultValue="visual" value={editorMode} onValueChange={(v) => setEditorMode(v as EditorMode)}>
                                            <TabsList>
                                                <TabsTrigger value="visual">Visual</TabsTrigger>
                                                <TabsTrigger value="html">HTML</TabsTrigger>
                                            </TabsList>
                                        </Tabs>
                                    </div>
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
                                        onFormat={(html) => updateFormDataSilent({ content: html })}
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
                <div className="space-y-4">
                    <Alert>
                        <Info className="h-4 w-4" />
                        <AlertDescription>
                            This preview uses default styling. The final appearance on your website may differ based on your theme&apos;s styles.
                        </AlertDescription>
                    </Alert>
                    <Card>
                        <CardHeader>
                            <CardTitle className="text-3xl">{formData.title || "Untitled Article"}</CardTitle>
                            {formData.excerpt && (
                                <CardDescription className="text-base mt-2">
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
                                className="
                                    prose prose-lg dark:prose-invert max-w-none
                                    prose-headings:font-semibold prose-headings:tracking-tight
                                    prose-h1:text-3xl prose-h1:mt-8 prose-h1:mb-4
                                    prose-h2:text-2xl prose-h2:mt-6 prose-h2:mb-3
                                    prose-h3:text-xl prose-h3:mt-5 prose-h3:mb-2
                                    prose-p:leading-7 prose-p:mb-4
                                    prose-ul:my-4 prose-ul:list-disc prose-ul:pl-6
                                    prose-ol:my-4 prose-ol:list-decimal prose-ol:pl-6
                                    prose-li:my-1
                                    prose-blockquote:border-l-4 prose-blockquote:border-primary/30 prose-blockquote:pl-4 prose-blockquote:italic prose-blockquote:my-4
                                    prose-code:bg-muted prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded prose-code:text-sm
                                    prose-pre:bg-muted prose-pre:p-4 prose-pre:rounded-lg prose-pre:overflow-x-auto
                                    prose-img:rounded-lg prose-img:my-6
                                    prose-a:text-primary prose-a:underline prose-a:underline-offset-4 hover:prose-a:text-primary/80
                                    prose-hr:my-8 prose-hr:border-border
                                    prose-table:border-collapse prose-table:w-full
                                    prose-th:border prose-th:border-border prose-th:px-4 prose-th:py-2 prose-th:bg-muted prose-th:font-semibold
                                    prose-td:border prose-td:border-border prose-td:px-4 prose-td:py-2
                                    prose-strong:font-semibold
                                    prose-em:italic
                                "
                                dangerouslySetInnerHTML={{ __html: formData.content }}
                            />
                        </CardContent>
                    </Card>
                </div>
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

            {/* AI Generate Modal */}
            <AIGenerateModal
                open={showAIGenerateModal}
                onOpenChange={setShowAIGenerateModal}
                siteId={siteId}
                onGenerated={handleAIGenerated}
            />
        </div>
    );
}
