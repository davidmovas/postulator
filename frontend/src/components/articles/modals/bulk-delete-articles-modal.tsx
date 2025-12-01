"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { useApiCall } from "@/hooks/use-api-call";
import { articleService } from "@/services/articles";
import { Article } from "@/models/articles";
import { RiWordpressFill } from "@remixicon/react";
import { AlertTriangle } from "lucide-react";

interface BulkDeleteArticlesModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    articles: Article[];
    onSuccess?: () => void;
}

export function BulkDeleteArticlesModal({
    open,
    onOpenChange,
    articles,
    onSuccess
}: BulkDeleteArticlesModalProps) {
    const { execute, isLoading } = useApiCall();
    const [deleteFromWP, setDeleteFromWP] = useState(true);

    const publishedArticles = articles.filter(a => a.wpPostId > 0);
    const hasPublishedArticles = publishedArticles.length > 0;

    const handleDelete = async () => {
        if (articles.length === 0) return;

        // If deleteFromWP is enabled and there are published articles, delete from WP first
        if (deleteFromWP && hasPublishedArticles) {
            await execute(
                () => articleService.bulkDeleteFromWordPress(publishedArticles.map(a => a.id)),
                {
                    showSuccessToast: true,
                    successMessage: `Deleted ${publishedArticles.length} article(s) from WordPress`,
                    errorTitle: "Failed to delete from WordPress",
                }
            );
        }

        // Then delete all articles locally
        await execute(
            () => articleService.bulkDeleteArticles(articles.map(a => a.id)),
            {
                showSuccessToast: true,
                successMessage: `Deleted ${articles.length} article(s)`,
                errorTitle: "Failed to delete articles",
            }
        );

        onOpenChange(false);
        onSuccess?.();
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <AlertTriangle className="h-5 w-5 text-destructive" />
                        Delete {articles.length} Article{articles.length > 1 ? 's' : ''}
                    </DialogTitle>
                    <DialogDescription>
                        Are you sure you want to delete the selected articles? This action cannot be undone.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    {/* Summary */}
                    <div className="p-3 rounded-lg bg-muted/50 space-y-1">
                        <p className="text-sm">
                            <span className="font-medium">{articles.length}</span> article{articles.length > 1 ? 's' : ''} selected
                        </p>
                        {hasPublishedArticles && (
                            <p className="text-sm text-muted-foreground">
                                <span className="font-medium">{publishedArticles.length}</span> published to WordPress
                            </p>
                        )}
                    </div>

                    {/* WordPress Delete Toggle */}
                    {hasPublishedArticles && (
                        <div className="flex items-center justify-between p-3 rounded-lg border">
                            <div className="flex items-center gap-2">
                                <RiWordpressFill className="w-5 h-5 text-[#21759b]" />
                                <Label htmlFor="bulk-delete-from-wp" className="cursor-pointer">
                                    Also delete from WordPress
                                </Label>
                            </div>
                            <Switch
                                id="bulk-delete-from-wp"
                                checked={deleteFromWP}
                                onCheckedChange={setDeleteFromWP}
                            />
                        </div>
                    )}

                    {/* Warning messages */}
                    {hasPublishedArticles && deleteFromWP && (
                        <p className="text-sm text-red-600 dark:text-red-400">
                            This will permanently delete {publishedArticles.length} article{publishedArticles.length > 1 ? 's' : ''} from your WordPress site. This action cannot be undone.
                        </p>
                    )}

                    {hasPublishedArticles && !deleteFromWP && (
                        <p className="text-sm text-amber-600 dark:text-amber-400">
                            The articles will remain on your WordPress site but will be removed from this app.
                        </p>
                    )}
                </div>

                <DialogFooter>
                    <Button
                        variant="outline"
                        onClick={() => onOpenChange(false)}
                        disabled={isLoading}
                    >
                        Cancel
                    </Button>
                    <Button
                        variant="destructive"
                        onClick={handleDelete}
                        disabled={isLoading}
                    >
                        {isLoading ? "Deleting..." : `Delete ${articles.length} Article${articles.length > 1 ? 's' : ''}`}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
