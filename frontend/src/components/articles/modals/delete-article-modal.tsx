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

interface DeleteArticleModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    article: Article | null;
    onSuccess?: () => void;
}

export function DeleteArticleModal({ open, onOpenChange, article, onSuccess }: DeleteArticleModalProps) {
    const { execute, isLoading } = useApiCall();
    const [deleteFromWP, setDeleteFromWP] = useState(true);

    const isPublishedToWP = article && article.wpPostId > 0;

    const handleDelete = async () => {
        if (!article) return;

        if (isPublishedToWP && deleteFromWP) {
            // Delete from WordPress (which also updates local record)
            await execute(
                () => articleService.deleteFromWordPress(article.id),
                {
                    showSuccessToast: true,
                    successMessage: "Article deleted from WordPress",
                    errorTitle: "Failed to delete article from WordPress",
                }
            );
            // Then delete local record
            await execute(
                () => articleService.deleteArticle(article.id),
                {
                    showSuccessToast: true,
                    successMessage: "Article deleted",
                    errorTitle: "Failed to delete article",
                }
            );
        } else {
            // Delete only local record
            await execute(
                () => articleService.deleteArticle(article.id),
                {
                    showSuccessToast: true,
                    successMessage: "Article deleted",
                    errorTitle: "Failed to delete article",
                }
            );
        }

        onOpenChange(false);
        onSuccess?.();
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[450px]">
                <DialogHeader>
                    <DialogTitle>Delete Article</DialogTitle>
                    <DialogDescription>
                        Are you sure you want to delete this article?
                    </DialogDescription>
                </DialogHeader>

                {article && (
                    <div className="space-y-4 py-4">
                        <div className="p-3 rounded-lg bg-muted/50">
                            <p className="font-medium line-clamp-2">{article.title}</p>
                        </div>

                        {isPublishedToWP && (
                            <div className="flex items-center justify-between p-3 rounded-lg border">
                                <div className="flex items-center gap-2">
                                    <RiWordpressFill className="w-5 h-5 text-[#21759b]" />
                                    <Label htmlFor="delete-from-wp" className="cursor-pointer">
                                        Also delete from WordPress
                                    </Label>
                                </div>
                                <Switch
                                    id="delete-from-wp"
                                    checked={deleteFromWP}
                                    onCheckedChange={setDeleteFromWP}
                                />
                            </div>
                        )}

                        {isPublishedToWP && deleteFromWP && (
                            <p className="text-sm text-red-600 dark:text-red-400">
                                This will permanently delete the article from your WordPress site. This action cannot be undone.
                            </p>
                        )}

                        {isPublishedToWP && !deleteFromWP && (
                            <p className="text-sm text-amber-600 dark:text-amber-400">
                                The article will remain on your WordPress site but will be removed from this app.
                            </p>
                        )}
                    </div>
                )}

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
                        {isLoading ? "Deleting..." : "Delete"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
