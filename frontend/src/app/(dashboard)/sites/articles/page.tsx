"use client";

import { useCallback, useEffect, useMemo, useState, Suspense } from "react";
import { useRouter } from "next/navigation";
import { useQueryId } from "@/hooks/use-query-param";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/table/data-table";
import { useArticlesTable } from "@/hooks/use-articles-table";
import { Article } from "@/models/articles";
import {
    RiAddLine,
    RiDeleteBinLine,
    RiRefreshLine,
    RiUpload2Line,
    RiWordpressFill,
} from "@remixicon/react";
import { useContextModal } from "@/context/modal-context";
import { useApiCall } from "@/hooks/use-api-call";
import { articleService } from "@/services/articles";
import { DeleteArticleModal } from "@/components/articles/modals/delete-article-modal";

function SiteArticlesPageContent() {
    const router = useRouter();
    const siteId = useQueryId();

    const [selected, setSelected] = useState<Article[]>([]);
    const [selectionResetKey, setSelectionResetKey] = useState(0);
    const [deletingArticle, setDeletingArticle] = useState<Article | null>(null);
    const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
    const { confirmationModal } = useContextModal();
    const { execute } = useApiCall();

    const handleEditArticle = useCallback((article: Article) => {
        router.push(`/sites/articles/edit?id=${siteId}&articleId=${article.id}`);
    }, [router, siteId]);

    const handleOpenDeleteModal = useCallback((article: Article) => {
        setDeletingArticle(article);
        setIsDeleteModalOpen(true);
    }, []);

    const {
        articles,
        total,
        columns,
        isLoading,
        loadArticles,
        serverSidePagination,
        handleSearchChange,
    } = useArticlesTable(siteId, {
        onEdit: handleEditArticle,
        onDelete: handleOpenDeleteModal,
    });

    const handleDeleteSuccess = useCallback(() => {
        loadArticles();
    }, [loadArticles]);

    useEffect(() => {
        loadArticles();
    }, [loadArticles]);

    const handleBulkDelete = useCallback(() => {
        if (selected.length === 0) return;

        const publishedCount = selected.filter(a => a.wpPostId > 0).length;

        confirmationModal.open({
            title: "Delete Selected Articles",
            description: (
                <div>
                    <p>Are you sure you want to delete {selected.length} selected article(s)?</p>
                    {publishedCount > 0 && (
                        <p className="mt-2 text-amber-600 dark:text-amber-400">
                            {publishedCount} article(s) are published to WordPress. They will only be deleted locally.
                        </p>
                    )}
                </div>
            ),
            confirmText: "Delete",
            variant: "destructive",
            onConfirm: async () => {
                await execute(async () => {
                    await articleService.bulkDeleteArticles(selected.map(a => a.id));
                }, {
                    showSuccessToast: true,
                    successMessage: `Deleted ${selected.length} article(s)`,
                    errorTitle: "Failed to delete articles",
                });
                await loadArticles();
                setSelected([]);
                setSelectionResetKey((k) => k + 1);
            }
        });
    }, [selected, confirmationModal, execute, loadArticles]);

    const handleBulkPublish = useCallback(() => {
        const draftArticles = selected.filter(a => a.status === 'draft' && !a.wpPostId);
        if (draftArticles.length === 0) {
            return;
        }
        confirmationModal.open({
            title: "Publish Selected Articles",
            description: `Are you sure you want to publish ${draftArticles.length} draft article(s) to WordPress?`,
            confirmText: "Publish",
            variant: "default",
            onConfirm: async () => {
                await execute(async () => {
                    return await articleService.bulkPublishToWordPress(draftArticles.map(a => a.id));
                }, {
                    showSuccessToast: true,
                    successMessage: `Published ${draftArticles.length} article(s) to WordPress`,
                    errorTitle: "Failed to publish articles",
                });
                await loadArticles();
                setSelected([]);
                setSelectionResetKey((k) => k + 1);
            }
        });
    }, [selected, confirmationModal, execute, loadArticles]);

    const handleSyncFromWP = useCallback(() => {
        confirmationModal.open({
            title: "Sync with WordPress",
            description: "This will sync all articles with WordPress: import new articles, update existing ones, and remove articles that no longer exist on WordPress.",
            confirmText: "Sync",
            variant: "default",
            onConfirm: async () => {
                await execute(async () => {
                    await articleService.syncFromWordPress(siteId);
                }, {
                    showSuccessToast: true,
                    successMessage: "Articles synced with WordPress",
                    errorTitle: "Failed to sync articles",
                });
                await loadArticles();
            }
        });
    }, [siteId, confirmationModal, execute, loadArticles]);

    const toolbarActions = useMemo(() => (
        <div className="flex items-center gap-2">
            {selected.length > 0 && (
                <>
                    <Button
                        variant="destructive"
                        onClick={handleBulkDelete}
                        className="flex items-center gap-2"
                    >
                        <RiDeleteBinLine className="w-4 h-4" />
                        Delete ({selected.length})
                    </Button>

                    {selected.some(a => a.status === 'draft' && !a.wpPostId) && (
                        <Button
                            onClick={handleBulkPublish}
                            className="flex items-center gap-2 bg-green-600 hover:bg-green-700 text-white"
                        >
                            <RiUpload2Line className="w-4 h-4" />
                            Publish ({selected.filter(a => a.status === 'draft' && !a.wpPostId).length})
                        </Button>
                    )}
                </>
            )}

            <Button
                onClick={() => loadArticles()}
                variant="outline"
                className="flex items-center gap-2"
            >
                <RiRefreshLine className="w-4 h-4" />
                Refresh
            </Button>

            <Button
                onClick={handleSyncFromWP}
                variant="wordpress"
                className="flex items-center gap-2"
            >
                <RiWordpressFill className="w-4 h-4" />
                Sync
            </Button>

            <Link href={`/sites/articles/new?id=${siteId}`}>
                <Button className="flex items-center gap-2">
                    <RiAddLine className="w-4 h-4" />
                    New Article
                </Button>
            </Link>
        </div>
    ), [selected, loadArticles, handleBulkDelete, handleBulkPublish, handleSyncFromWP, siteId]);

    return (
        <div className="p-6 space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-3xl font-bold tracking-tight">Articles</h1>
                <p className="text-muted-foreground mt-1">
                    Manage your site articles • {total} total
                </p>
            </div>

            {/* Table */}
            <DataTable
                columns={columns}
                data={articles}
                toolbarActions={toolbarActions}
                isLoading={isLoading}
                emptyMessage="No articles found. Create or sync articles from WordPress to get started."
                showPagination={true}
                defaultPageSize={25}
                enableViewOption={false}
                onRowSelectionChange={(rows) => setSelected(rows as Article[])}
                rowSelectionResetKey={selectionResetKey}
                serverSidePagination={serverSidePagination}
                searchPlaceholder="Search articles by title..."
                onSearchChange={handleSearchChange}
            />

            {/* Delete Article Modal */}
            <DeleteArticleModal
                open={isDeleteModalOpen}
                onOpenChange={setIsDeleteModalOpen}
                article={deletingArticle}
                onSuccess={handleDeleteSuccess}
            />
        </div>
    );
}

export default function SiteArticlesPage() {
    return (
        <Suspense fallback={
            <div className="p-6 space-y-6">
                <div className="h-8 w-32 bg-muted/30 rounded animate-pulse" />
                <div className="h-64 bg-muted/30 rounded-lg animate-pulse" />
            </div>
        }>
            <SiteArticlesPageContent />
        </Suspense>
    );
}
