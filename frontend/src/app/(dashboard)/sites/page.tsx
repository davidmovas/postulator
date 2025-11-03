"use client";

import { useEffect, useState } from "react";
import { siteService } from "@/services/sites";
import { Site } from "@/models/sites";
import { Button } from "@/components/ui/button";
import { PlusIcon, RefreshCwIcon } from "lucide-react";
import { DataTable } from "@/components/table/data-table";
import { columns } from "@/components/sites/sites-columns";
import { tableFilters } from "@/components/sites/sites-filter";

export default function SitesPage() {
    const [data, setData] = useState<Site[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    // Load sites
    const loadSites = async () => {
        try {
            setIsLoading(true);
            const sites = await siteService.listSites();
            setData(sites);
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        loadSites();
    }, []);

    const handleAddSite = () => {
        // TODO: Open add site modal
        console.log("Open add site modal");
    };

    const handleCheckAllHealth = async () => {
        try {
            const message = await siteService.checkAllHealth();

            // Reload sites to get updated health status
            setTimeout(() => {
                loadSites();
            }, 3000);
        } finally {
            console.log("Check all health finished");
        }
    }

    const handleRowSelectionChange = (selectedSites: Site[]) => {
        console.log("Selected sites:", selectedSites);
        // Можно добавить bulk actions
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
                        <RefreshCwIcon className="w-4 h-4" />
                        Check All Health
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
                data={data}
                searchKey="name"
                searchPlaceholder="Search sites..."
                filters={tableFilters}
                toolbarActions={null}
                isLoading={isLoading}
                emptyMessage="No sites found. Create your first site to get started."
                onRowSelectionChange={handleRowSelectionChange}
                showPagination={true}
                defaultPageSize={10}
            />
        </div>
    );
}