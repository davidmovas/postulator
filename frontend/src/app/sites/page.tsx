'use client';

import { useEffect, useState } from 'react';
import { checkSiteHealth, deleteSite, listSites, Site } from "@/services/site";
import { SitesTable } from "@/components/tables/SitesTable";
import { useErrorHandling } from "@/lib/error-handling";
import { EditSiteModal } from "@/components/modals/EditSiteModal";
import { CreateSiteModal } from "@/components/sites/CreateSiteModal";

export default function SitesPage() {
    const [sites, setSites] = useState<Site[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    const [isCreateOpen, setIsCreateOpen] = useState(false);

    const [isEditOpen, setIsEditOpen] = useState(false);
    const [editingSite, setEditingSite] = useState<Site | null>(null);

    const { withErrorHandling, showSuccess } = useErrorHandling();

    const loadSites = async () => {
        setIsLoading(true);
        try {
            const data = await listSites();
            setSites(data);
            return data;
        } catch (error) {
            console.error('Failed to load sites:', error);
        } finally {
            setIsLoading(false);
        }
    };

    // Refresh handler
    const handleRefresh = async () => {
        await withErrorHandling(
            async () => {
                const data = await listSites();
                setSites(data);
                return data;
            },
            {
                successMessage: 'Sites list updated',
                showSuccess: true,
            }
        );
    };

    // Health check single site
    const handleHealthCheck = async (siteId: number) => {
        await withErrorHandling(
            async () => {
                await checkSiteHealth(siteId);
                await loadSites();
            },
            {
                successMessage: `Site health checked successfully`,
                showSuccess: true,
            }
        );
    };

    // Health check all sites
    const handleHealthCheckAll = async () => {
        await withErrorHandling(
            async () => {
                for (const site of sites) {
                    await checkSiteHealth(site.id);
                }
                await loadSites();
            },
            {
                successMessage: 'All sites checked successfully',
                showSuccess: true,
            }
        );
    };

    useEffect(() => {
        loadSites();
    }, []);

    // Edit handler
    const handleEdit = (site: Site) => {
        setEditingSite(site);
        setIsEditOpen(true);
    };

    // Delete handler
    const handleDelete = async (siteId: number) => {
        await withErrorHandling(
            async () => {
                await deleteSite(siteId);
                setSites((prev) => prev.filter((site) => site.id !== siteId));
            },
            {
                successMessage: 'Site deleted successfully',
                showSuccess: true,
            }
        );
    };

    // Helpers
    // Create modal open
    const handleCreate = () => {
        setIsCreateOpen(true);
    };

    return (
        <div className="p-4 md:p-6 lg:p-8">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">Sites Management</h1>
                <p className="mt-2 text-muted-foreground">
                    View and manage all your WordPress sites
                </p>
            </div>

            <SitesTable
                sites={sites}
                isLoading={isLoading}
                onRefresh={handleRefresh}
                onHealthCheck={handleHealthCheck}
                onHealthCheckAll={handleHealthCheckAll}
                onEdit={handleEdit}
                onDelete={handleDelete}
                onCreate={handleCreate}
            />

            <CreateSiteModal
                open={isCreateOpen}
                onOpenChange={setIsCreateOpen}
                onCreated={async () => { await loadSites(); }}
            />

            <EditSiteModal
                open={isEditOpen}
                onOpenChange={setIsEditOpen}
                site={editingSite}
                onSaved={async () => { await loadSites(); }}
            />
        </div>
    );
}