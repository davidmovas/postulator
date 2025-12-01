"use client";

import { useMemo } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { FolderTree, Tags, RefreshCw } from "lucide-react";
import { ArticleFormData } from "@/hooks/use-article-form";
import { cn } from "@/lib/utils";
import { VirtualizedMultiSelect } from "@/components/ui/virtualized-multi-select";
import { RiWordpressFill } from "@remixicon/react";

interface Category {
    id: number;
    name: string;
    slug?: string;
    parentId?: number;
}

interface Tag {
    id: number;
    name: string;
    slug?: string;
}

interface CategoriesTagsSectionProps {
    formData: ArticleFormData;
    onUpdate: (updates: Partial<ArticleFormData>) => void;
    categories: Category[] | null;
    tags?: Tag[] | null;
    disabled?: boolean;
    onSyncCategories?: () => void;
    onSyncTags?: () => void;
    isSyncing?: boolean;
}

export function CategoriesTagsSection({
    formData,
    onUpdate,
    categories,
    tags,
    disabled = false,
    onSyncCategories,
    onSyncTags,
    isSyncing = false,
}: CategoriesTagsSectionProps) {
    const categoriesLoading = categories === null;
    const tagsLoading = tags === null;

    // Convert categories to options for VirtualizedMultiSelect
    const categoryOptions = useMemo(() => {
        if (!categories) return [];
        return categories.map(c => ({
            value: c.id.toString(),
            label: c.name,
        }));
    }, [categories]);

    // Convert tags to options for VirtualizedMultiSelect
    const tagOptions = useMemo(() => {
        if (!tags) return [];
        return tags.map(t => ({
            value: t.id.toString(),
            label: t.name,
        }));
    }, [tags]);

    // Convert selected IDs to string array
    const selectedCategoryValues = useMemo(() =>
        formData.categoryIds.map(id => id.toString()),
        [formData.categoryIds]
    );

    const selectedTagValues = useMemo(() =>
        formData.tagIds.map(id => id.toString()),
        [formData.tagIds]
    );

    const handleCategoryChange = (values: string[]) => {
        onUpdate({ categoryIds: values.map(v => parseInt(v)) });
    };

    const handleTagChange = (values: string[]) => {
        onUpdate({ tagIds: values.map(v => parseInt(v)) });
    };

    return (
        <Card>
            <CardHeader>
                <CardTitle className="flex items-center gap-2">
                    <FolderTree className="h-5 w-5" />
                    Categories & Tags
                </CardTitle>
                <CardDescription>
                    Organize your article with categories and tags
                </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
                {/* Categories */}
                <div className="space-y-3">
                    <div className="flex items-center justify-between">
                        <Label className="flex items-center gap-2">
                            <FolderTree className="h-4 w-4" />
                            Categories
                        </Label>
                        {onSyncCategories && (
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={onSyncCategories}
                                disabled={disabled || isSyncing}
                                className="h-7 text-xs"
                            >
                                <RefreshCw className={cn("h-3 w-3 mr-1", isSyncing && "animate-spin")} />
                                Sync
                            </Button>
                        )}
                    </div>

                    {categoriesLoading ? (
                        <div className="flex items-center justify-center h-10 text-sm text-muted-foreground">
                            Loading categories...
                        </div>
                    ) : categories && categories.length > 0 ? (
                        <VirtualizedMultiSelect
                            options={categoryOptions}
                            value={selectedCategoryValues}
                            onChange={handleCategoryChange}
                            placeholder="Select categories..."
                            searchPlaceholder="Search categories..."
                            disabled={disabled}
                        />
                    ) : (
                        <div className="flex flex-col items-center justify-center py-4 text-sm text-muted-foreground gap-2 border rounded-md">
                            <p>No categories found</p>
                            {onSyncCategories && (
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={onSyncCategories}
                                    disabled={disabled || isSyncing}
                                >
                                    <RiWordpressFill className="h-3 w-3 mr-1" />
                                    Sync
                                </Button>
                            )}
                        </div>
                    )}
                </div>

                {/* Tags */}
                {tags !== undefined && (
                    <div className="space-y-3">
                        <div className="flex items-center justify-between">
                            <Label className="flex items-center gap-2">
                                <Tags className="h-4 w-4" />
                                Tags
                            </Label>
                            {onSyncTags && (
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={onSyncTags}
                                    disabled={disabled || isSyncing}
                                    className="h-7 text-xs"
                                >
                                    <RefreshCw className={cn("h-3 w-3 mr-1", isSyncing && "animate-spin")} />
                                    Sync
                                </Button>
                            )}
                        </div>

                        {tagsLoading ? (
                            <div className="flex items-center justify-center h-10 text-sm text-muted-foreground">
                                Loading tags...
                            </div>
                        ) : tags && tags.length > 0 ? (
                            <VirtualizedMultiSelect
                                options={tagOptions}
                                value={selectedTagValues}
                                onChange={handleTagChange}
                                placeholder="Select tags..."
                                searchPlaceholder="Search tags..."
                                disabled={disabled}
                            />
                        ) : (
                            <div className="flex items-center justify-center py-4 text-sm text-muted-foreground border rounded-md">
                                No tags found
                            </div>
                        )}
                    </div>
                )}
            </CardContent>
        </Card>
    );
}
