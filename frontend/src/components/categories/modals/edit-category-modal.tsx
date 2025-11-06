"use client";

import { useState, useEffect } from "react";
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
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useApiCall } from "@/hooks/use-api-call";
import { categoryService } from "@/services/categories";
import { Category, CategoryUpdateInput } from "@/models/categories";

interface EditCategoryModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    category: Category | null;
    onSuccess?: () => void;
}

export function EditCategoryModal({
    open,
    onOpenChange,
    category,
    onSuccess
}: EditCategoryModalProps) {
    const { execute, isLoading } = useApiCall();

    const [formData, setFormData] = useState<CategoryUpdateInput>({
        id: 0,
        siteId: category?.siteId,
        wpCategoryId: category?.wpCategoryId || 0,
        name: "",
        slug: "",
        description: ""
    });

    useEffect(() => {
        if (category) {
            setFormData({
                id: category.id,
                siteId: category?.siteId,
                wpCategoryId: category?.wpCategoryId || 0,
                name: category.name,
                slug: category.slug || "",
                description: category.description || ""
            });
        }
    }, [category]);

    const isFormValid = formData.name?.trim();

    const handleSubmit = async () => {
        if (!isFormValid || !category) return;

        const result = await execute<string>(
            () => categoryService.updateInWordPress(formData),
            {
                successMessage: "Category updated successfully",
                showSuccessToast: true
            }
        );

        if (result) {
            onOpenChange(false);
            onSuccess?.();
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle>Edit Category</DialogTitle>
                    <DialogDescription>
                        Update the category information in WordPress.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="space-y-2">
                        <Label htmlFor="edit-name">Category Name<span className="text-red-600">*</span></Label>
                        <Input
                            id="edit-name"
                            value={formData.name}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                name: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="edit-slug">Slug</Label>
                        <Input
                            id="edit-slug"
                            value={formData.slug}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                slug: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="edit-description">Description</Label>
                        <Textarea
                            id="edit-description"
                            value={formData.description}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                description: e.target.value
                            }))}
                            disabled={isLoading}
                            rows={3}
                        />
                    </div>
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
                        onClick={handleSubmit}
                        disabled={!isFormValid || isLoading}
                    >
                        {isLoading ? "Updating..." : "Update Category"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}