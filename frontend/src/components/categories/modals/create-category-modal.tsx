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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useApiCall } from "@/hooks/use-api-call";
import { categoryService } from "@/services/categories";
import { CategoryCreateInput } from "@/models/categories";

interface CreateCategoryModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: number;
    onSuccess?: () => void;
}

export function CreateCategoryModal({
    open,
    onOpenChange,
    siteId,
    onSuccess
}: CreateCategoryModalProps) {
    const { execute, isLoading } = useApiCall();

    const [formData, setFormData] = useState<CategoryCreateInput>({
        siteId: siteId,
        name: "",
        slug: "",
        description: ""
    });

    const resetForm = () => {
        setFormData({
            siteId,
            name: "",
            slug: "",
            description: ""
        });
    };

    const isFormValid = formData.name.trim();

    const handleSubmit = async () => {
        if (!isFormValid) return;

        const result = await execute<string>(
            () => categoryService.createInWordPress(formData),
            {
                successMessage: "Category created successfully",
                showSuccessToast: true,
            }
        );

        if (result) {

            onOpenChange(false);
            resetForm();
            onSuccess?.();
        }
    };

    const handleOpenChange = (newOpen: boolean) => {
        if (!newOpen) {
            resetForm();
        }
        onOpenChange(newOpen);
    };

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle>Create New Category</DialogTitle>
                    <DialogDescription>
                        Create a new category in WordPress. The category will be synced with your site.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="space-y-2">
                        <Label htmlFor="name">Category Name<span className="text-red-600">*</span></Label>
                        <Input
                            id="name"
                            placeholder="Technology"
                            value={formData.name}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                name: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="slug">Slug</Label>
                        <Input
                            id="slug"
                            placeholder="technology"
                            value={formData.slug}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                slug: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                        <p className="text-xs text-muted-foreground">
                            WordPress will auto-generate slug if left empty
                        </p>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="description">Description</Label>
                        <Textarea
                            id="description"
                            placeholder="Articles about technology and innovation"
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
                        onClick={() => handleOpenChange(false)}
                        disabled={isLoading}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        disabled={!isFormValid || isLoading}
                    >
                        {isLoading ? "Creating..." : "Create Category"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}