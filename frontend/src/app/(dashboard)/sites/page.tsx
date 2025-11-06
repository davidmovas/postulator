"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { PlusIcon } from "lucide-react";
import { DataTable } from "@/components/table/data-table";
import { RiPulseLine, RiRefreshLine } from "@remixicon/react";
import { useApiCall } from "@/hooks/use-api-call";
import { useSitesTable } from "@/hooks/use-sites-table";
import { siteService } from "@/services/sites";
import { Site } from "@/models/sites";
import { useContextModal } from "@/context/modal-context";
import { CreateSiteModal } from "@/components/sites/modals/create-site-modal";
import { EditSiteModal } from "@/components/sites/modals/edit-site-modal";
import { ChangePasswordModal } from "@/components/sites/modals/reset-site-password-modal";
import { ConfirmationModal } from "@/modals/confirm-modal";
import { DeleteSiteModal } from "@/components/sites/modals/delete-site-modal";

export default function SitesPage() {
    const { sites, columns, isLoading, loadSites } = useSitesTable();
    const { execute } = useApiCall();
    const { createSiteModal, editSiteModal, passwordModal, deleteSiteModal } = useContextModal();

    useEffect(() => {
        loadSites();
    }, [loadSites]);

    const handleRefresh = async () => {
        await loadSites();
    };

    const handleAddSite = () => {
        createSiteModal.open();
    };


    const handleCheckAllHealth = async () => {
        await execute<string>(
            () => siteService.checkAllHealth(),
            {
                successMessage: "All sites health checked successfully",
                showSuccessToast: true,
                onSuccess: () => {
                    loadSites();
                }
            }
        );
    };

    const handleSuccess = () => {
        loadSites();
    };

    const handleRowSelectionChange = (selectedSites: any[]) => {
        console.log("Selected sites:", selectedSites);
    };

    return (
        <div className="p-6 space-y-6">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">Sites</h1>
                    <p className="text-muted-foreground mt-1">
                        Manage your WordPress sites and monitor their health
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
                        onClick={handleCheckAllHealth}
                        variant="outline"
                        className="flex items-center gap-2"
                    >
                        <RiPulseLine className="w-4 h-4" />
                        Check All
                    </Button>

                    <Button
                        onClick={handleAddSite}
                        className="flex items-center gap-2"
                    >
                        <PlusIcon className="w-4 h-4" />
                        Add Site
                    </Button>
                </div>
            </div>

            {/* Table */}
            <DataTable
                columns={columns}
                data={sites}
                searchKey="name"
                searchPlaceholder="Search sites..."
                toolbarActions={null}
                isLoading={isLoading}
                emptyMessage="No sites found. Create your first site to get started."
                onRowSelectionChange={handleRowSelectionChange}
                showPagination={true}
                defaultPageSize={25}
            />

            {/* Modals */}
            <CreateSiteModal
                open={createSiteModal.isOpen}
                onOpenChange={createSiteModal.close}
                onSuccess={handleSuccess}
            />

            <EditSiteModal
                open={editSiteModal.isOpen}
                onOpenChange={editSiteModal.close}
                site={editSiteModal.site}
                onSuccess={handleSuccess}
            />

            <ChangePasswordModal
                open={passwordModal.isOpen}
                onOpenChange={passwordModal.close}
                siteId={passwordModal.site?.id || 0}
                onSuccess={handleSuccess}
            />

            <DeleteSiteModal
                open={deleteSiteModal.isOpen}
                onOpenChange={deleteSiteModal.close}
                site={deleteSiteModal.site}
                onSuccess={loadSites}
            />
        </div>
    );
}