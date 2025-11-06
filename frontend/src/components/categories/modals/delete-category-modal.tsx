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
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { useApiCall } from "@/hooks/use-api-call";
import { categoryService } from "@/services/categories";
import { Category } from "@/models/categories";

interface DeleteCategoryModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    category: Category | null;
    onSuccess?: () => void;
}

export function DeleteCategoryModal({
    open,
    onOpenChange,
    category,
    onSuccess
}: DeleteCategoryModalProps) {
    const { execute, isLoading } = useApiCall();
    const [deleteFromWordPress, setDeleteFromWordPress] = useState(true);

    const handleSubmit = async () => {
        if (!category) return;

        let result;
        if (deleteFromWordPress) {
            result = await execute<string>(
                () => categoryService.deleteInWordPress(category.id),
                {
                    successMessage: "Category deleted from WordPress successfully",
                    showSuccessToast: true
                }
            );
        } else {
            result = await execute<string>(
                () => categoryService.deleteCategory(category.id),
                {
                    successMessage: "Category deleted successfully",
                    showSuccessToast: true
                }
            );
        }

        if (result) {
            onOpenChange(false);
            setDeleteFromWordPress(true);
            onSuccess?.();
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle>Delete Category</DialogTitle>
                    <DialogDescription>
                        This action cannot be undone. This will permanently delete the category
                        {deleteFromWordPress ? " from WordPress and " : " from the system "}
                        remove all associated data.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="bg-muted/50 border rounded-lg p-3">
                        <p className="font-medium text-muted-foreground">{category?.name}</p>
                        {category?.description && (
                            <p className="text-sm text-muted-foreground mt-1">{category.description}</p>
                        )}
                    </div>

                    <div className="flex items-center space-x-2">
                        <Checkbox
                            id="delete-from-wp"
                            checked={deleteFromWordPress}
                            onCheckedChange={(checked) => setDeleteFromWordPress(checked === true)}
                            disabled={isLoading}
                        />
                        <Label htmlFor="delete-from-wp" className="text-sm">
                            Also delete from WordPress
                        </Label>
                    </div>
                    <p className="text-xs text-muted-foreground">
                        If unchecked, the category will only be removed from this system but remain in WordPress.
                    </p>
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
                        onClick={handleSubmit}
                        disabled={isLoading}
                    >
                        {isLoading ? "Deleting..." : "Delete Category"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}