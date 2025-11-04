"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { PlusIcon } from "lucide-react";
import { DataTable } from "@/components/table/data-table";
import { tableFilters } from "@/components/sites/sites-filter";
import { RiPulseLine } from "@remixicon/react";
import { useApiCall } from "@/hooks/use-api-call";
import { useSitesTable } from "@/hooks/use-sites-table";
import { siteService } from "@/services/sites";

export default function SitesPage() {
    const { sites, columns, isLoading, loadSites } = useSitesTable();
    const { execute } = useApiCall();

    useEffect(() => {
        loadSites();
    }, [loadSites]);

    const handleRefresh = async () => {
        await loadSites();
    };

    const handleAddSite = () => {
        console.log("Open add site modal");
    };

    const handleCheckAllHealth = async () => {
        await execute<string>(
            () => siteService.checkAllHealth(),
            {
                successMessage: "All sites health checked successfully",
                showSuccessToast: true,
                onSuccess: () => {
                    setTimeout(() => {
                        loadSites();
                    }, 1000);
                }
            }
        );
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
        </div>
    );
}