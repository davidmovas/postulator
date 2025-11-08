"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/table/data-table";
import { useTopicsTable } from "@/hooks/use-topics-table";
import { Topic } from "@/models/topics";
import { RiAddLine, RiDeleteBinLine, RiRefreshLine, RiUpload2Line } from "@remixicon/react";
import { useContextModal } from "@/context/modal-context";
import { ConfirmationModal } from "@/modals/confirm-modal";
import { CreateTopicsModal } from "@/components/topics/modals/create-topics-modal";
import { EditTopicModal } from "@/components/topics/modals/edit-topic-modal";
import { ImportTopicsModal } from "@/components/topics/modals/import-topics-modal";
import { useApiCall } from "@/hooks/use-api-call";
import { topicService } from "@/services/topics";

export default function SiteTopicsPage() {
    const params = useParams();
    const siteId = parseInt(params.id as string);

    const { topics, columns, isLoading, loadTopics } = useTopicsTable(siteId);
    const [selected, setSelected] = useState<Topic[]>([]);
    const [selectionResetKey, setSelectionResetKey] = useState(0);
    const [isCreateOpen, setIsCreateOpen] = useState(false);
        const [isImportOpen, setIsImportOpen] = useState(false);
    const { confirmationModal, editTopicModal } = useContextModal();
    const { execute } = useApiCall();

    useEffect(() => {
        loadTopics();
    }, [loadTopics]);

    const handleBulkDelete = useCallback(() => {
        if (selected.length === 0) return;
        confirmationModal.open({
            title: "Delete Selected Topics",
            description: `Are you sure you want to delete ${selected.length} selected topic(s)?`,
            confirmText: "Delete",
            variant: "destructive",
            onConfirm: async () => {
                await execute(async () => {
                    await Promise.all(selected.map(t => topicService.deleteTopic(t.id)));
                }, {
                    showSuccessToast: true,
                    successMessage: `Deleted ${selected.length} topic(s)`,
                    errorTitle: "Failed to delete topics",
                });
                await loadTopics();
                setSelected([]);
                setSelectionResetKey((k) => k + 1);
            }
        });
    }, [selected, confirmationModal, execute, loadTopics]);

    const toolbarActions = useMemo(() => (
        <div className="flex items-center gap-2">
            {selected.length > 0 && (
                <Button
                    variant="destructive"
                    onClick={handleBulkDelete}
                    className="flex items-center gap-2"
                >
                    <RiDeleteBinLine className="w-4 h-4" />
                    Delete Selected ({selected.length})
                </Button>
            )}

            <Button
                onClick={() => loadTopics()}
                variant="outline"
                className="flex items-center gap-2"
            >
                <RiRefreshLine className="w-4 h-4" />
                Refresh
            </Button>

            <Button
                onClick={() => setIsImportOpen(true)}
                className="flex items-center gap-2 bg-violet-600 hover:bg-violet-700 text-white"
            >
                <RiUpload2Line className="w-4 h-4" />
            Import
            </Button>

            <Button
                onClick={() => setIsCreateOpen(true)}
                className="flex items-center gap-2"
            >
                <RiAddLine className="w-4 h-4" />
                Add Topic
            </Button>
        </div>
    ), [selected, loadTopics, handleBulkDelete]);

    const handleCreateSuccess = useCallback(() => {
        loadTopics();
    }, [loadTopics]);

    return (
        <div className="p-6 space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-3xl font-bold tracking-tight">Topics</h1>
                <p className="text-muted-foreground mt-1">Manage your site topics here</p>
            </div>

            {/* Table */}
            <DataTable
                columns={columns}
                data={topics}
                searchKey="title"
                searchPlaceholder="Search topics..."
                toolbarActions={toolbarActions}
                isLoading={isLoading}
                emptyMessage="No topics found. Create or import topics to get started."
                showPagination={true}
                defaultPageSize={50}
                enableViewOption={false}
                onRowSelectionChange={(rows) => setSelected(rows as Topic[])}
                rowSelectionResetKey={selectionResetKey}
            />

            {/* Modals */}
            <CreateTopicsModal
                open={isCreateOpen}
                onOpenChange={setIsCreateOpen}
                siteId={siteId}
                onSuccess={handleCreateSuccess}
            />

            <EditTopicModal
                open={editTopicModal.isOpen}
                onOpenChange={editTopicModal.close}
                topic={editTopicModal.topic}
                onSuccess={handleCreateSuccess}
            />

            <ImportTopicsModal
                open={isImportOpen}
                onOpenChange={setIsImportOpen}
                siteId={siteId}
                onSuccess={handleCreateSuccess}
            />

            <ConfirmationModal
                open={confirmationModal.isOpen}
                onOpenChange={confirmationModal.close}
                data={confirmationModal.data}
            />
        </div>
    );
}