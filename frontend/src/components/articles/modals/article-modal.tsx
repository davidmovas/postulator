"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useApiCall } from "@/hooks/use-api-call";
import { articleService } from "@/services/articles";
import { Article, ArticleCreateInput, ArticleUpdateInput } from "@/models/articles";
import { ScrollArea } from "@/components/ui/scroll-area";

interface ArticleModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    article?: Article | null;
    siteId: number;
    onSuccess?: () => void;
}

export function ArticleModal({ open, onOpenChange, article, siteId, onSuccess }: ArticleModalProps) {
    const { execute, isLoading } = useApiCall();
    const isEditMode = !!article;

    const [formData, setFormData] = useState({
        title: "",
        content: "",
        excerpt: "",
        slug: "",
        metaDescription: "",
    });

    useEffect(() => {
        if (article) {
            setFormData({
                title: article.title || "",
                content: article.content || "",
                excerpt: article.excerpt || "",
                slug: article.slug || "",
                metaDescription: article.metaDescription || "",
            });
        } else {
            setFormData({
                title: "",
                content: "",
                excerpt: "",
                slug: "",
                metaDescription: "",
            });
        }
    }, [article, open]);

    const isFormValid = formData.title.trim().length > 0 && formData.content.trim().length > 0;

    const handleSubmit = async () => {
        if (!isFormValid) return;

        if (isEditMode && article) {
            const input: ArticleUpdateInput = {
                id: article.id,
                title: formData.title.trim(),
                content: formData.content.trim(),
                excerpt: formData.excerpt.trim() || undefined,
                slug: formData.slug.trim() || undefined,
                metaDescription: formData.metaDescription.trim() || undefined,
            };

            const result = await execute<void>(
                () => articleService.updateArticle(input),
                {
                    successMessage: "Article updated successfully",
                    showSuccessToast: true,
                    errorTitle: "Failed to update article",
                }
            );

            if (result !== null) {
                onOpenChange(false);
                onSuccess?.();
            }
        } else {
            const input: ArticleCreateInput = {
                siteId,
                title: formData.title.trim(),
                content: formData.content.trim(),
                excerpt: formData.excerpt.trim() || undefined,
                slug: formData.slug.trim() || undefined,
                metaDescription: formData.metaDescription.trim() || undefined,
            };

            const result = await execute<void>(
                () => articleService.createArticle(input),
                {
                    successMessage: "Article created successfully",
                    showSuccessToast: true,
                    errorTitle: "Failed to create article",
                }
            );

            if (result !== null) {
                onOpenChange(false);
                onSuccess?.();
            }
        }
    };

    const wordCount = formData.content.split(/\s+/).filter(w => w.length > 0).length;

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[900px] max-h-[90vh] flex flex-col">
                <DialogHeader>
                    <DialogTitle>{isEditMode ? "Edit Article" : "Create Article"}</DialogTitle>
                    <DialogDescription>
                        {isEditMode
                            ? "Update your article content and SEO settings."
                            : "Create a new article for your site."}
                    </DialogDescription>
                </DialogHeader>

                <Tabs defaultValue="content" className="flex-1 overflow-hidden flex flex-col">
                    <TabsList className="grid w-full grid-cols-2">
                        <TabsTrigger value="content">Content</TabsTrigger>
                        <TabsTrigger value="seo">SEO</TabsTrigger>
                    </TabsList>

                    <ScrollArea className="flex-1 pr-4">
                        <TabsContent value="content" className="space-y-4 mt-4">
                            <div className="space-y-2">
                                <Label htmlFor="article-title">
                                    Title <span className="text-red-600">*</span>
                                </Label>
                                <Input
                                    id="article-title"
                                    value={formData.title}
                                    onChange={(e) => setFormData(prev => ({
                                        ...prev,
                                        title: e.target.value,
                                    }))}
                                    disabled={isLoading}
                                    placeholder="Enter article title"
                                />
                            </div>

                            <div className="space-y-2">
                                <div className="flex items-center justify-between">
                                    <Label htmlFor="article-content">
                                        Content <span className="text-red-600">*</span>
                                    </Label>
                                    <span className="text-xs text-muted-foreground">
                                        {wordCount} words
                                    </span>
                                </div>
                                <Textarea
                                    id="article-content"
                                    value={formData.content}
                                    onChange={(e) => setFormData(prev => ({
                                        ...prev,
                                        content: e.target.value,
                                    }))}
                                    disabled={isLoading}
                                    placeholder="Write your article content here... (HTML is supported)"
                                    className="min-h-[300px] font-mono text-sm"
                                />
                                <p className="text-xs text-muted-foreground">
                                    You can use HTML tags for formatting.
                                </p>
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="article-excerpt">Excerpt</Label>
                                <Textarea
                                    id="article-excerpt"
                                    value={formData.excerpt}
                                    onChange={(e) => setFormData(prev => ({
                                        ...prev,
                                        excerpt: e.target.value,
                                    }))}
                                    disabled={isLoading}
                                    placeholder="Brief summary of the article (optional)"
                                    rows={3}
                                />
                            </div>
                        </TabsContent>

                        <TabsContent value="seo" className="space-y-4 mt-4">
                            <div className="space-y-2">
                                <Label htmlFor="article-slug">URL Slug</Label>
                                <Input
                                    id="article-slug"
                                    value={formData.slug}
                                    onChange={(e) => setFormData(prev => ({
                                        ...prev,
                                        slug: e.target.value.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, ''),
                                    }))}
                                    disabled={isLoading}
                                    placeholder="article-url-slug"
                                />
                                <p className="text-xs text-muted-foreground">
                                    The URL-friendly version of the title. Leave empty to auto-generate.
                                </p>
                            </div>

                            <div className="space-y-2">
                                <div className="flex items-center justify-between">
                                    <Label htmlFor="article-meta">Meta Description</Label>
                                    <span className="text-xs text-muted-foreground">
                                        {formData.metaDescription.length}/160
                                    </span>
                                </div>
                                <Textarea
                                    id="article-meta"
                                    value={formData.metaDescription}
                                    onChange={(e) => setFormData(prev => ({
                                        ...prev,
                                        metaDescription: e.target.value.slice(0, 160),
                                    }))}
                                    disabled={isLoading}
                                    placeholder="Brief description for search engines (recommended: 150-160 characters)"
                                    rows={3}
                                />
                            </div>

                            {/* Preview */}
                            <div className="space-y-2">
                                <Label>Search Engine Preview</Label>
                                <div className="p-4 bg-muted rounded-lg space-y-1">
                                    <div className="text-blue-600 dark:text-blue-400 text-lg font-medium line-clamp-1">
                                        {formData.title || "Article Title"}
                                    </div>
                                    <div className="text-green-700 dark:text-green-500 text-sm">
                                        example.com/{formData.slug || "article-slug"}
                                    </div>
                                    <div className="text-muted-foreground text-sm line-clamp-2">
                                        {formData.metaDescription || formData.excerpt || "No description provided. The first part of your content will be shown instead."}
                                    </div>
                                </div>
                            </div>
                        </TabsContent>
                    </ScrollArea>
                </Tabs>

                <DialogFooter className="mt-4">
                    <Button
                        variant="outline"
                        onClick={() => onOpenChange(false)}
                        disabled={isLoading}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        disabled={!isFormValid || isLoading}
                    >
                        {isLoading
                            ? (isEditMode ? "Updating..." : "Creating...")
                            : (isEditMode ? "Update Article" : "Create Article")}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
