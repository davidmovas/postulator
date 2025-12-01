"use client";

import { useState, useMemo, useCallback, useRef } from "react";
import { ColumnDef } from "@tanstack/react-table";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal, Trash2, ExternalLink, Upload, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { RiPencilLine } from "@remixicon/react";
import {
    Article,
    ArticleListFilter,
    ArticleListResult,
    articleStatusLabels,
    articleStatusColors,
    articleSourceLabels,
} from "@/models/articles";
import { articleService } from "@/services/articles";
import { useApiCall } from "@/hooks/use-api-call";
import { formatSmartDate } from "@/lib/time";

const DEFAULT_PAGE_SIZE = 25;

interface UseArticlesTableOptions {
    onEdit?: (article: Article) => void;
    onDelete?: (article: Article) => void;
}

export function useArticlesTable(siteId: number, options?: UseArticlesTableOptions) {
    const [articles, setArticles] = useState<Article[]>([]);
    const [total, setTotal] = useState(0);
    const [pageIndex, setPageIndex] = useState(0);
    const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
    const [search, setSearch] = useState<string | undefined>(undefined);
    const { execute, isLoading } = useApiCall();

    // Use ref to avoid stale closure issues
    const filterRef = useRef({ pageIndex, pageSize, search });
    filterRef.current = { pageIndex, pageSize, search };

    const loadArticles = useCallback(async (overrides?: { pageIndex?: number; pageSize?: number; search?: string }) => {
        const currentPageIndex = overrides?.pageIndex ?? filterRef.current.pageIndex;
        const currentPageSize = overrides?.pageSize ?? filterRef.current.pageSize;
        // Check if 'search' key exists in overrides object, not just if it's undefined
        const currentSearch = overrides && 'search' in overrides ? overrides.search : filterRef.current.search;

        const filter: ArticleListFilter = {
            siteId,
            limit: currentPageSize,
            offset: currentPageIndex * currentPageSize,
            sortBy: 'created_at',
            sortOrder: 'desc',
            search: currentSearch || undefined,
        };

        const result = await execute<ArticleListResult>(
            () => articleService.listArticles(filter),
            { errorTitle: "Failed to load articles" }
        );
        if (result) {
            setArticles(result.articles);
            setTotal(result.total);
        }
    }, [siteId, execute]);

    const handlePublishToWP = useCallback(async (article: Article) => {
        await execute(
            () => articleService.publishToWordPress(article.id),
            {
                showSuccessToast: true,
                successMessage: "Article published to WordPress",
                errorTitle: "Failed to publish article",
            }
        );
        await loadArticles();
    }, [execute, loadArticles]);

    const handleUpdateInWP = useCallback(async (article: Article) => {
        await execute(
            () => articleService.updateInWordPress(article.id),
            {
                showSuccessToast: true,
                successMessage: "Article updated in WordPress",
                errorTitle: "Failed to update article in WordPress",
            }
        );
        await loadArticles();
    }, [execute, loadArticles]);

    const columns: ColumnDef<Article>[] = useMemo(() => [
        {
            id: "select",
            header: ({ table }) => {
                const isAllSelected = table.getIsAllPageRowsSelected();
                return (
                    <input
                        type="checkbox"
                        aria-label="Select all"
                        checked={isAllSelected}
                        onChange={(e) => table.toggleAllPageRowsSelected(e.currentTarget.checked)}
                    />
                );
            },
            cell: ({ row }) => (
                <input
                    type="checkbox"
                    aria-label="Select row"
                    checked={row.getIsSelected()}
                    onChange={(e) => row.toggleSelected(e.currentTarget.checked)}
                />
            ),
            enableSorting: false,
            enableHiding: false,
            size: 32,
        },
        {
            accessorKey: "title",
            header: "Title",
            cell: ({ row }) => {
                const article = row.original;
                return (
                    <div className="max-w-md">
                        <div className="font-medium line-clamp-2">{article.title}</div>
                        {article.isEdited && article.originalTitle !== article.title && (
                            <div className="text-xs text-muted-foreground mt-1 line-clamp-1">
                                Original: {article.originalTitle}
                            </div>
                        )}
                    </div>
                );
            },
        },
        {
            accessorKey: "status",
            header: "Status",
            cell: ({ row }) => {
                const status = row.original.status;
                return (
                    <Badge variant="secondary" className={articleStatusColors[status]}>
                        {articleStatusLabels[status]}
                    </Badge>
                );
            },
        },
        {
            accessorKey: "source",
            header: "Source",
            cell: ({ row }) => {
                const source = row.original.source;
                return (
                    <span className="text-muted-foreground text-sm">
                        {articleSourceLabels[source]}
                    </span>
                );
            },
        },
        {
            accessorKey: "wordCount",
            header: "Words",
            cell: ({ row }) => {
                const wordCount = row.original.wordCount;
                return (
                    <span className="text-muted-foreground">
                        {wordCount ? wordCount.toLocaleString() : '-'}
                    </span>
                );
            },
        },
        {
            accessorKey: "createdAt",
            header: "Created",
            cell: ({ row }) => {
                const created = row.original.createdAt;
                return (
                    <span className="text-muted-foreground text-sm">
                        {formatSmartDate(created)}
                    </span>
                );
            },
        },
        {
            id: "actions",
            header: "Actions",
            cell: ({ row }) => {
                const article = row.original;
                const isPublished = article.wpPostId > 0;

                return (
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" className="h-8 w-8 p-0">
                                <span className="sr-only">Open menu</span>
                                <MoreHorizontal className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>

                        <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => options?.onEdit?.(article)}>
                                <RiPencilLine className="mr-2 h-4 w-4" />
                                <span>Edit</span>
                            </DropdownMenuItem>

                            {!isPublished && article.status === 'draft' && (
                                <DropdownMenuItem onClick={() => handlePublishToWP(article)}>
                                    <Upload className="mr-2 h-4 w-4" />
                                    <span>Publish to WordPress</span>
                                </DropdownMenuItem>
                            )}

                            {isPublished && (
                                <>
                                    <DropdownMenuItem onClick={() => handleUpdateInWP(article)}>
                                        <RefreshCw className="mr-2 h-4 w-4" />
                                        <span>Sync</span>
                                    </DropdownMenuItem>

                                    <DropdownMenuItem
                                        onClick={() => window.open(article.wpPostUrl, '_blank')}
                                    >
                                        <ExternalLink className="mr-2 h-4 w-4" />
                                        <span>View</span>
                                    </DropdownMenuItem>
                                </>
                            )}

                            <DropdownMenuSeparator />

                            <DropdownMenuItem
                                onClick={() => options?.onDelete?.(article)}
                                className="text-red-600"
                            >
                                <Trash2 className="mr-2 h-4 w-4" />
                                <span>Delete</span>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                );
            },
        },
    ], [handlePublishToWP, handleUpdateInWP, options]);

    const pageCount = Math.ceil(total / pageSize);

    const handlePaginationChange = useCallback((pagination: { pageIndex: number; pageSize: number }) => {
        setPageIndex(pagination.pageIndex);
        setPageSize(pagination.pageSize);
        loadArticles({
            pageIndex: pagination.pageIndex,
            pageSize: pagination.pageSize,
        });
    }, [loadArticles]);

    const handleSearchChange = useCallback((searchValue: string) => {
        const newSearch = searchValue || undefined;
        setSearch(newSearch);
        setPageIndex(0); // Reset to first page on search
        loadArticles({
            pageIndex: 0,
            search: newSearch,
        });
    }, [loadArticles]);

    const serverSidePagination = useMemo(() => ({
        pageIndex,
        pageSize,
        pageCount,
        totalCount: total,
        onPaginationChange: handlePaginationChange,
    }), [pageIndex, pageSize, pageCount, total, handlePaginationChange]);

    return {
        articles,
        total,
        columns,
        isLoading,
        loadArticles,
        serverSidePagination,
        handleSearchChange,
    };
}
