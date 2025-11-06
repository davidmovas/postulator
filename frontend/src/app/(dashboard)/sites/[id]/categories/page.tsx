"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/table/data-table";
import { useCategoriesTable } from "@/hooks/use-categories-table";
import { useContextModal } from "@/context/modal-context";
import { CreateCategoryModal } from "@/components/categories/modals/create-category-modal";
import { EditCategoryModal } from "@/components/categories/modals/edit-category-modal";
import { DeleteCategoryModal } from "@/components/categories/modals/delete-category-modal";
import { RiAddLine, RiRefreshLine, RiWordpressFill } from "@remixicon/react";
import { Statistics } from "@/models/categories";
import { toGoDateFormat } from "@/lib/time";
import { categoryService } from "@/services/categories";
import { useApiCall } from "@/hooks/use-api-call";

export default function SiteCategoriesPage() {
    const params = useParams();
    const siteId = parseInt(params.id as string);

    const [siteStats, setSiteStats] = useState<Statistics[]>([]);
    const [categoryStats, setCategoryStats] = useState<Statistics[]>([]);
    const [selectedCategoryId, setSelectedCategoryId] = useState<number>();
    const { execute } = useApiCall();

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

    const loadStatistics = useCallback(async (from?: string, to?: string, categoryId?: number) => {
        try {
            const toDate = to ? new Date(to) : new Date();
            const fromDate = from ? new Date(from) : new Date();
            fromDate.setDate(toDate.getDate() - 30);

            const fromStr = from || toGoDateFormat(fromDate);
            const toStr = to || toGoDateFormat(toDate);

            if (categoryId) {
                const result = await execute<Statistics[]>(
                    () => categoryService.getStatistics(categoryId, fromStr, toStr),
                    {
                        errorTitle: "Failed to load category statistics",
                    }
                );
                setCategoryStats(result || []);
            } else {
                const result = await execute<Statistics[]>(
                    () => categoryService.getSiteStatistics(siteId, fromStr, toStr),
                    {
                        errorTitle: "Failed to load site categories statistics",
                    }
                );
                setSiteStats(result || []);
            }
        } catch (error) {
            setCategoryStats([]);
            setSiteStats([]);
        }
    }, [siteId, execute]);

    const handleStatsUpdate = useCallback((from: string, to: string, categoryId?: number) => {
        loadStatistics(from, to, categoryId);
        setSelectedCategoryId(categoryId);
    }, [loadStatistics]);

    useEffect(() => {
        loadCategories();
        loadStatistics();
    }, [loadCategories, loadStatistics]);

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

            {/* Statistics Section */}
            {/*<div className="space-y-4">
                <h2 className="text-2xl font-bold tracking-tight">Statistics</h2>
                <CategoryStats
                    siteId={siteId}
                    selectedCategoryId={selectedCategoryId}
                    siteStats={siteStats}
                    categories={categories}
                    categoryStats={categoryStats}
                    totalCategories={categories.length}
                    activeCategories={categories.filter(cat => cat.count > 0).length}
                    onStatsUpdate={handleStatsUpdate}
                />
            </div>*/}


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