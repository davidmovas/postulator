'use client';

import { useEffect, useState } from 'react';
import { checkSiteHealth, deleteSite, listSites, Site } from "@/services/site";
import { SitesTable } from "@/components/sites/SitesTable";
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
                try {
                    await checkSiteHealth(siteId);
                } finally {
                    // Always refresh the list regardless of the check result
                    await loadSites();
                }
            },
            {
                successMessage: `Site health checked successfully`,
                showSuccess: true,
            }
        );
    };

    // Health check all sites
    const handleHealthCheckAll = async () => {
        // We handle all errors internally to avoid bubbling to Next.js overlay.
        await withErrorHandling(
            async () => {
                let successCount = 0;
                let failCount = 0;

                const tasks = sites.map(async (site) => {
                    try {
                        await checkSiteHealth(site.id);
                        successCount += 1;
                    } catch (_) {
                        // Swallow per-site error to prevent Next.js error overlay.
                        // We'll show a friendly summary toast after processing all sites.
                        failCount += 1;
                    } finally {
                        // Refresh after each site's response (success or error)
                        await loadSites();
                    }
                });

                // Wait for all to settle without throwing
                await Promise.all(tasks);

                // Final refresh to ensure consistency
                await loadSites();

                // Show a single user-friendly toast instead of console errors
                if (failCount === 0) {
                    showSuccess(`Checked ${successCount} site(s) successfully`);
                } else if (successCount === 0) {
                    // All failed
                    showError(`Failed to check ${failCount} site(s)`);
                } else {
                    showError(`Checked ${successCount} succeeded, ${failCount} failed`);
                }
            },
            {
                // Disable automatic success toast here since we show a custom summary toast above
                showSuccess: false,
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
        <div className="p-4 md:p-6 lg:p-8 space-y-6">
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