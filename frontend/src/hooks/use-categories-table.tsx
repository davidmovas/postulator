"use client";

import { useState, useMemo, useCallback } from "react";
import { ColumnDef } from "@tanstack/react-table";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal, Trash2 } from "lucide-react";
import { Category } from "@/models/categories";
import { formatDateTime } from "@/lib/time";
import { useApiCall } from "@/hooks/use-api-call";
import { categoryService } from "@/services/categories";
import { Button } from "@/components/ui/button";
import { RiPencilLine, RiRefreshLine, RiWordpressLine } from "@remixicon/react";
import { useContextModal } from "@/context/modal-context";

export function useCategoriesTable(siteId: number) {
    const [categories, setCategories] = useState<Category[]>([]);
    const { editCategoryModal, deleteCategoryModal, confirmationModal } = useContextModal();
    const { execute, isLoading } = useApiCall();

    const loadCategories = useCallback(async () => {
        const categoriesData = await execute<Category[]>(
            () => categoryService.listSiteCategories(siteId),
            {
                errorTitle: "Failed to load categories"
            }
        );
        if (categoriesData) {
            setCategories(categoriesData);
        }
    }, [siteId, execute]);

    const handleSyncFromWordPress = useCallback(async () => {
        await execute<void>(
            () => categoryService.syncFromWordPress(siteId),
            {
                successMessage: "Categories synced from WordPress",
                showSuccessToast: true,
                onSuccess: () => {
                    loadCategories();
                }
            }
        );
    }, [siteId, execute, loadCategories]);

    const columns: ColumnDef<Category>[] = useMemo(() => [
        {
            accessorKey: "name",
            header: "Name",
            cell: ({ row }) => {
                const category = row.original;
                return (
                    <div className="font-medium">
                        {category.name}
                        </div>
                );
            },
        },
        {
            accessorKey: "slug",
            header: "Slug",
            cell: ({ row }) => {
                const slug = row.getValue("slug") as string;
                return <span className="text-muted-foreground">{slug || "-"}</span>;
            },
        },
        {
            accessorKey: "description",
            header: "Description",
            cell: ({ row }) => {
                const description = row.getValue("description") as string;
                return <span className="text-muted-foreground">{description || "-"}</span>;
            },
        },
        {
            accessorKey: "count",
            header: "Posts",
            cell: ({ row }) => {
                const count = row.getValue("count") as number;
                return <span className="font-medium">{count}</span>;
            },
        },
        {
            accessorKey: "wpCategoryId",
            header: "WordPress ID",
            cell: ({ row }) => {
                const wpId = row.getValue("wpCategoryId") as number;
                return <span className="text-muted-foreground">{wpId}</span>;
            },
        },
        {
            accessorKey: "createdAt",
            header: "Created",
            cell: ({ row }) => {
                const date = row.getValue("createdAt") as string;
                return formatDateTime(date) || "-";
            },
        },
        {
            id: "actions",
            header: "Actions",
            cell: ({ row }) => {
                const category = row.original;

                const handleEdit = () => {
                    editCategoryModal.open(category);
                };

                const handleDelete = () => {
                    deleteCategoryModal.open(category);
                };

                return (
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                    <Button variant="ghost" className="h-8 w-8 p-0">
                <span className="sr-only">Open menu</span>
                <MoreHorizontal className="h-4 w-4" />
                    </Button>
                    </DropdownMenuTrigger>

                    <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={handleEdit}>
                <RiPencilLine className="mr-2 h-4 w-4" />
                <span>Edit</span>
                </DropdownMenuItem>

                <DropdownMenuSeparator />

                <DropdownMenuItem onClick={handleDelete} className="text-red-600">
                <Trash2 className="mr-2 h-4 w-4" />
                    <span>Delete</span>
                    </DropdownMenuItem>
                    </DropdownMenuContent>
                    </DropdownMenu>
            );
            },
        },
    ], []);

    return {
        categories,
        setCategories,
        columns,
        isLoading,
        loadCategories,
        handleSyncFromWordPress
    };
}