"use client";

import { useEffect } from "react";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/table/data-table";
import { useCategoriesTable } from "@/hooks/use-categories-table";
import { useContextModal } from "@/context/modal-context";
import { CreateCategoryModal } from "@/components/categories/modals/create-category-modal";
import { EditCategoryModal } from "@/components/categories/modals/edit-category-modal";
import { DeleteCategoryModal } from "@/components/categories/modals/delete-category-modal";
import { RiAddLine, RiRefreshLine, RiWordpressFill, RiWordpressLine } from "@remixicon/react";

export default function SiteCategoriesPage() {
    const params = useParams();
    const siteId = parseInt(params.id as string);

    const {
        categories,
        columns,
        isLoading,
        loadCategories,
        handleSyncFromWordPress
    } = useCategoriesTable(siteId);

    const {
        createCategoryModal,
        editCategoryModal,
        deleteCategoryModal
    } = useContextModal();

    useEffect(() => {
        loadCategories();
    }, [loadCategories]);

    const handleRefresh = async () => {
        await loadCategories();
    };

    const handleAddCategory = () => {
        createCategoryModal.open(siteId);
    };

    const handleSuccess = () => {
        loadCategories();
    };

    return (
        <div className="p-6 space-y-6">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">Categories</h1>
                    <p className="text-muted-foreground mt-1">
                        Manage your WordPress categories and their content
                    </p>
                </div>

                <div className="flex items-center gap-2 mt-4 sm:mt-0">
                    <Button
                        onClick={handleRefresh}
                        variant="outline"
                        className="flex items-center gap-2"
                    >
                        <RiRefreshLine className="w-4 h-4" />
                        Refresh
                    </Button>

                    <Button
                        onClick={handleSyncFromWordPress}
                        variant="wordpress"
                        className="flex items-center gap-2"
                    >
                        <RiWordpressFill className="w-4 h-4" />
                        Sync from WordPress
                    </Button>

                    <Button
                        onClick={handleAddCategory}
                        className="flex items-center gap-2"
                    >
                        <RiAddLine className="w-4 h-4" />
                        Add Category
                    </Button>
                </div>
            </div>

            {/* Table */}
            <DataTable
                columns={columns}
                data={categories}
                searchKey="name"
                searchPlaceholder="Search categories..."
                toolbarActions={null}
                isLoading={isLoading}
                emptyMessage="No categories found. Create your first category or sync from WordPress."
                showPagination={false}
                enableViewOption={false}
            />

            {/* Modals */}
            <CreateCategoryModal
                open={createCategoryModal.isOpen}
                onOpenChange={createCategoryModal.close}
                siteId={createCategoryModal.siteId || siteId}
                onSuccess={handleSuccess}
            />

            <EditCategoryModal
                open={editCategoryModal.isOpen}
                onOpenChange={editCategoryModal.close}
                category={editCategoryModal.category}
                onSuccess={handleSuccess}
            />

            <DeleteCategoryModal
                open={deleteCategoryModal.isOpen}
                onOpenChange={deleteCategoryModal.close}
                category={deleteCategoryModal.category}
                onSuccess={handleSuccess}
            />
        </div>
    );
}