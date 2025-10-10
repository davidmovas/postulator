'use client';

import { useState, useEffect } from 'react';
import { checkSiteHealth, deleteSite, listSites, Site } from "@/services/site";
import { SitesTable } from "@/components/tables/SitesTable";
import { useErrorHandling } from "@/lib/error-handling";

export default function SitesPage() {
    const [sites, setSites] = useState<Site[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const { withErrorHandling, showSuccess } = useErrorHandling();

    const loadSites = async () => {
        try {
            const data = await listSites();
            setSites(data);
            return data;
        } catch (error) {
            // Ошибка уже обработана в listSites через unwrapResponse
            // Можно добавить дополнительную логику если нужно
            console.error('Failed to load sites:', error);
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
                // Check sites sequentially to avoid overwhelming the server
                for (const site of sites) {
                    await checkSiteHealth(site.id);
                }
                // Reload all sites after checking
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
        // TODO: Open edit dialog/modal
        console.log('Edit site:', site);
        // Example: setEditingSite(site); setIsEditDialogOpen(true);
    };

    // Delete handler
    const handleDelete = async (siteId: number) => {
        const result = await withErrorHandling(
            async () => {
                await deleteSite(siteId);
                // Remove from local state
                setSites((prev) => prev.filter((site) => site.id !== siteId));
            },
            {
                successMessage: 'Site deleted successfully',
                showSuccess: true,
            }
        );
    };

    // Create handler
    const handleCreate = () => {
        // TODO: Open create dialog/modal
        console.log('Create new site');
        // Example: setIsCreateDialogOpen(true);
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
        </div>
    );
}