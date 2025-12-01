"use client";

import { useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Search, Sparkles, RotateCcw } from "lucide-react";
import { ArticleFormData } from "@/hooks/use-article-form";
import { cn } from "@/lib/utils";

interface SeoSectionProps {
    formData: ArticleFormData;
    onUpdate: (updates: Partial<ArticleFormData>) => void;
    disabled?: boolean;
    onAiOptimize?: () => void;
    isAiLoading?: boolean;
    siteUrl?: string;
}

export function SeoSection({
    formData,
    onUpdate,
    disabled = false,
    onAiOptimize,
    isAiLoading = false,
    siteUrl = "example.com",
}: SeoSectionProps) {
    const metaDescriptionLength = formData.metaDescription.length;
    const isMetaDescriptionOptimal = metaDescriptionLength >= 120 && metaDescriptionLength <= 160;
    const isMetaDescriptionWarning = metaDescriptionLength > 0 && metaDescriptionLength < 120;
    const isMetaDescriptionError = metaDescriptionLength > 160;

    const generateSlug = useCallback(() => {
        const slug = formData.title
            .toLowerCase()
            .replace(/[^a-z0-9\s-]/g, "")
            .replace(/\s+/g, "-")
            .replace(/-+/g, "-")
            .trim();
        onUpdate({ slug });
    }, [formData.title, onUpdate]);

    const handleSlugChange = useCallback((value: string) => {
        // Auto-format slug as user types
        const formattedSlug = value
            .toLowerCase()
            .replace(/\s+/g, "-")
            .replace(/[^a-z0-9-]/g, "");
        onUpdate({ slug: formattedSlug });
    }, [onUpdate]);

    return (
        <Card>
            <CardHeader>
                <div className="flex items-center justify-between">
                    <div>
                        <CardTitle className="flex items-center gap-2">
                            <Search className="h-5 w-5" />
                            SEO Settings
                        </CardTitle>
                        <CardDescription>
                            Optimize your article for search engines
                        </CardDescription>
                    </div>
                    {onAiOptimize && (
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={onAiOptimize}
                            disabled={disabled || isAiLoading || !formData.content}
                        >
                            <Sparkles className="h-4 w-4 mr-2" />
                            AI Optimize
                        </Button>
                    )}
                </div>
            </CardHeader>
            <CardContent className="space-y-6">
                {/* URL Slug */}
                <div className="space-y-2">
                    <Label htmlFor="seo-slug">URL Slug</Label>
                    <div className="flex gap-2">
                        <Input
                            id="seo-slug"
                            value={formData.slug}
                            onChange={(e) => handleSlugChange(e.target.value)}
                            disabled={disabled}
                            placeholder="article-url-slug"
                        />
                        <Button
                            type="button"
                            variant="outline"
                            size="icon"
                            onClick={generateSlug}
                            disabled={disabled || !formData.title}
                            title="Generate from title"
                        >
                            <RotateCcw className="h-4 w-4" />
                        </Button>
                    </div>
                    {formData.slug && (
                        <div className="text-xs text-muted-foreground bg-muted px-3 py-2 rounded-md font-mono">
                            {siteUrl}/.../{formData.slug}
                        </div>
                    )}
                    <p className="text-xs text-muted-foreground">
                        The URL-friendly version of the title. Leave empty to auto-generate.
                    </p>
                </div>

                {/* Meta Description */}
                <div className="space-y-2">
                    <div className="flex items-center justify-between">
                        <Label htmlFor="seo-meta">Meta Description</Label>
                        <Badge
                            variant="secondary"
                            className={cn(
                                "text-xs",
                                isMetaDescriptionOptimal && "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
                                isMetaDescriptionWarning && "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
                                isMetaDescriptionError && "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                            )}
                        >
                            {metaDescriptionLength}/160
                        </Badge>
                    </div>
                    <Textarea
                        id="seo-meta"
                        value={formData.metaDescription}
                        onChange={(e) => onUpdate({ metaDescription: e.target.value.slice(0, 200) })}
                        disabled={disabled}
                        placeholder="Brief description for search engines (recommended: 120-160 characters)"
                        rows={3}
                    />
                    <p className="text-xs text-muted-foreground">
                        {isMetaDescriptionOptimal && "Good length for search results."}
                        {isMetaDescriptionWarning && "Consider adding more detail (aim for 120-160 characters)."}
                        {isMetaDescriptionError && "Too long - will be truncated in search results."}
                        {metaDescriptionLength === 0 && "Describe your article for better SEO."}
                    </p>
                </div>

                {/* Search Engine Preview */}
                <div className="space-y-2">
                    <Label>Search Engine Preview</Label>
                    <div className="p-4 bg-muted rounded-lg space-y-1 border">
                        <div className="text-blue-600 dark:text-blue-400 text-lg font-medium line-clamp-1 hover:underline cursor-pointer">
                            {formData.title || "Article Title"}
                        </div>
                        <div className="text-green-700 dark:text-green-500 text-sm">
                            {siteUrl}/.../{formData.slug || "article-slug"}
                        </div>
                        <div className="text-muted-foreground text-sm line-clamp-2">
                            {formData.metaDescription || formData.excerpt || "No description provided. The first part of your content will be shown instead."}
                        </div>
                    </div>
                </div>

                {/* SEO Score (optional visual indicator) */}
                <div className="space-y-2">
                    <Label>SEO Score</Label>
                    <div className="space-y-2">
                        <SeoCheckItem
                            label="Title is set"
                            checked={formData.title.length > 0}
                        />
                        <SeoCheckItem
                            label="Title length (50-60 chars)"
                            checked={formData.title.length >= 50 && formData.title.length <= 60}
                            warning={formData.title.length > 0 && (formData.title.length < 50 || formData.title.length > 60)}
                        />
                        <SeoCheckItem
                            label="Meta description is set"
                            checked={formData.metaDescription.length > 0}
                        />
                        <SeoCheckItem
                            label="Meta description length (120-160 chars)"
                            checked={isMetaDescriptionOptimal}
                            warning={isMetaDescriptionWarning}
                        />
                        <SeoCheckItem
                            label="URL slug is set"
                            checked={formData.slug.length > 0}
                        />
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}

interface SeoCheckItemProps {
    label: string;
    checked: boolean;
    warning?: boolean;
}

function SeoCheckItem({ label, checked, warning }: SeoCheckItemProps) {
    return (
        <div className="flex items-center gap-2 text-sm">
            <div
                className={cn(
                    "w-2 h-2 rounded-full",
                    checked && "bg-green-500",
                    warning && "bg-yellow-500",
                    !checked && !warning && "bg-muted-foreground/30"
                )}
            />
            <span className={cn(
                checked ? "text-foreground" : "text-muted-foreground"
            )}>
                {label}
            </span>
        </div>
    );
}
