'use client';

import React, { useState, useMemo } from 'react';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import {
    RefreshCw,
    Plus,
    ArrowUpDown,
    Database,
} from 'lucide-react';
import { Site } from "@/services/site";
import { BrowserOpenURL } from '@/wailsjs/wailsjs/runtime';
import { useToast } from '@/components/ui/use-toast';
import { useErrorHandling } from '@/lib/error-handling';
import { setSitePassword } from '@/services/site';
import { SitesToolbar } from '@/components/sites/SitesToolbar';
import { SiteRowActions } from '@/components/sites/SiteRowActions';
import { SetSitePasswordDialog } from '@/components/sites/SetSitePasswordDialog';

// Types for sorting
type SortField = keyof Site;
type SortDirection = 'asc' | 'desc' | null;

interface SitesTableProps {
    sites: Site[];
    isLoading?: boolean;
    onRefresh: () => Promise<void>;
    onHealthCheck: (siteId: number) => Promise<void>;
    onHealthCheckAll: () => Promise<void>;
    onEdit: (site: Site) => void;
    onDelete: (siteId: number) => Promise<void>;
    onCreate: () => void;
}

export function SitesTable({
    sites,
    isLoading = false,
    onRefresh,
    onHealthCheck,
    onHealthCheckAll,
    onEdit,
    onDelete,
    onCreate,
}: SitesTableProps) {
    // States for filtering and sorting
    const [searchQuery, setSearchQuery] = useState('');
    const [sortField, setSortField] = useState<SortField | null>(null);
    const [sortDirection, setSortDirection] = useState<SortDirection>(null);
    const [loadingActions, setLoadingActions] = useState<Record<number, boolean>>({});
    const [isRefreshing, setIsRefreshing] = useState(false);
    const { toast } = useToast();
    const { withErrorHandling } = useErrorHandling();

    // Modals state
    const [deleteTarget, setDeleteTarget] = useState<Site | null>(null);
    const [isDeleting, setIsDeleting] = useState(false);

    const [passwordTarget, setPasswordTarget] = useState<Site | null>(null);
    const [isSettingPassword, setIsSettingPassword] = useState(false);

    const openInDefault = (url: string) => {
        try {
            BrowserOpenURL(url);
        } catch (e) {
            console.error('Failed to open URL in default browser', e);
        }
    };

    const openInTor = (url: string) => {
        try {
            // Attempt Tor Browser custom protocol if registered
            // Tor Browser sometimes registers "torbrowser://" scheme on Windows installations
            BrowserOpenURL(`torbrowser:${url}`);
        } catch (e) {
            console.error('Failed to open URL in Tor', e);
            // Fallback to default
            try { BrowserOpenURL(url); } catch {}
        }
    };

    const copyLink = async (url: string) => {
        try {
            await navigator.clipboard.writeText(url);
            toast({ title: 'Copied', description: 'URL copied to clipboard' });
        } catch (e) {
            console.error('Failed to copy URL', e);
        }
    };

    // Format date helper
    const formatDate = (dateString: string) => {
        if (!dateString) return 'N/A';
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', {
            day: '2-digit',
            month: 'short',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
        });
    };

    // Badge variants based on status
    const getStatusVariant = (status: string): 'default' | 'secondary' | 'destructive' | 'outline' => {
        switch (status.toLowerCase()) {
            case 'active':
                return 'default';
            case 'inactive':
                return 'secondary';
            case 'error':
                return 'destructive';
            default:
                return 'outline';
        }
    };

    const getHealthStatusVariant = (status: string): 'default' | 'secondary' | 'destructive' | 'outline' => {
        switch (status.toLowerCase()) {
            case 'healthy':
                return 'default';
            case 'warning':
                return 'secondary';
            case 'unhealthy':
                return 'destructive';
            default:
                return 'outline';
        }
    };

    // Sort handler
    const handleSort = (field: SortField) => {
        if (sortField === field) {
            // Cycle: asc -> desc -> null
            if (sortDirection === 'asc') {
                setSortDirection('desc');
            } else if (sortDirection === 'desc') {
                setSortDirection(null);
                setSortField(null);
            }
        } else {
            setSortField(field);
            setSortDirection('asc');
        }
    };

    // Filter and sort data
    const filteredAndSortedSites = useMemo(() => {
        let result = [...sites];

        // Filter
        if (searchQuery) {
            const query = searchQuery.toLowerCase();
            result = result.filter(
                (site) =>
                    site.name.toLowerCase().includes(query) ||
                    site.url.toLowerCase().includes(query) ||
                    site.wpUsername.toLowerCase().includes(query)
            );
        }

        // Sort
        if (sortField && sortDirection) {
            result.sort((a, b) => {
                const aValue = a[sortField];
                const bValue = b[sortField];

                // Handle different data types
                if (typeof aValue === 'string' && typeof bValue === 'string') {
                    return sortDirection === 'asc'
                        ? aValue.localeCompare(bValue)
                        : bValue.localeCompare(aValue);
                }

                if (typeof aValue === 'number' && typeof bValue === 'number') {
                    return sortDirection === 'asc' ? aValue - bValue : bValue - aValue;
                }

                return 0;
            });
        }

        return result;
    }, [sites, searchQuery, sortField, sortDirection]);

    // Action handlers with loading states
    const handleRefresh = async () => {
        setIsRefreshing(true);
        try {
            await onRefresh();
        } finally {
            setIsRefreshing(false);
        }
    };

    const handleHealthCheck = async (siteId: number) => {
        setLoadingActions((prev) => ({ ...prev, [siteId]: true }));
        try {
            await onHealthCheck(siteId);
        } finally {
            setLoadingActions((prev) => ({ ...prev, [siteId]: false }));
        }
    };

    const handleDelete = async (siteId: number) => {
        const target = sites.find(s => s.id === siteId) || null;
        setDeleteTarget(target);
    };

    const confirmDelete = async () => {
        if (!deleteTarget) return;
        const siteId = deleteTarget.id;
        setIsDeleting(true);
        setLoadingActions((prev) => ({ ...prev, [siteId]: true }));
        try {
            await onDelete(siteId);
            setDeleteTarget(null);
        } finally {
            setIsDeleting(false);
            setLoadingActions((prev) => ({ ...prev, [siteId]: false }));
        }
    };

    // Sortable header component
    const SortableHeader = ({ field, children }: { field: SortField; children: React.ReactNode }) => (
        <TableHead>
            <button
                className="flex items-center gap-2 hover:text-foreground transition-colors"
                onClick={() => handleSort(field)}
            >
                {children}
                <ArrowUpDown className={`h-4 w-4 ${sortField === field ? 'text-foreground' : 'text-muted-foreground/50'}`} />
            </button>
        </TableHead>
    );

    return (
        <div className="space-y-4">
            {/* Control panel */}
            <SitesToolbar
                searchQuery={searchQuery}
                onSearchChange={setSearchQuery}
                isLoading={isLoading}
                totalSites={sites.length}
                onCreate={onCreate}
                onRefresh={handleRefresh}
                isRefreshing={isRefreshing}
                onHealthCheckAll={onHealthCheckAll}
            />

            {/* Results info */}
            {sites.length > 0 && (
                <div className="text-sm text-muted-foreground">
                    Showing {filteredAndSortedSites.length} of {sites.length} sites
                </div>
            )}

            {/* Table */}
            <div className="w-full overflow-x-auto rounded-lg border">
                <Table className="min-w-[800px]">
                    <TableHeader>
                        <TableRow>
                            <SortableHeader field="name">Name</SortableHeader>
                            <SortableHeader field="url">URL</SortableHeader>
                            <SortableHeader field="wpUsername">User</SortableHeader>
                            <SortableHeader field="status">Status</SortableHeader>
                            <SortableHeader field="healthStatus">Health</SortableHeader>
                            <SortableHeader field="lastHealthCheck">Last Check</SortableHeader>
                            <TableHead className="w-[70px]">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading && sites.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={9} className="text-center py-12">
                                    <RefreshCw className="h-8 w-8 animate-spin mx-auto mb-2 text-muted-foreground" />
                                    <p className="text-muted-foreground">Loading sites...</p>
                                </TableCell>
                            </TableRow>
                        ) : filteredAndSortedSites.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={9} className="text-center py-12">
                                    <Database className="h-12 w-12 mx-auto mb-4 text-muted-foreground/50" />
                                    <h3 className="font-semibold mb-1">No sites found</h3>
                                    <p className="text-sm text-muted-foreground mb-4">
                                        {searchQuery
                                            ? 'Try adjusting your search query'
                                            : 'Get started by adding your first site'}
                                    </p>
                                    {!searchQuery && (
                                        <Button onClick={onCreate} size="sm">
                                            <Plus className="h-4 w-4 mr-2" />
                                            Add Your First Site
                                        </Button>
                                    )}
                                </TableCell>
                            </TableRow>
                        ) : (
                            filteredAndSortedSites.map((site) => (
                                <TableRow key={site.id}>
                                    <TableCell className="font-medium">{site.name}</TableCell>
                                    <TableCell className="max-w-[260px] sm:max-w-none break-words">
                                        <button
                                            onClick={() => BrowserOpenURL(site.url)}
                                            className="text-primary hover:underline text-left break-all"
                                            title="Open in default browser"
                                        >
                                            {site.url}
                                        </button>
                                    </TableCell>
                                    <TableCell>{site.wpUsername}</TableCell>
                                    <TableCell>
                                        <Badge variant={getStatusVariant(site.status)}>{site.status}</Badge>
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant={getHealthStatusVariant(site.healthStatus)}>
                                            {site.healthStatus}
                                        </Badge>
                                    </TableCell>
                                    <TableCell className="text-muted-foreground text-sm">
                                        {site.lastHealthCheck ? formatDate(site.lastHealthCheck) : 'Never checked'}
                                    </TableCell>
                                    <TableCell>
                                        <SiteRowActions
                                            site={site}
                                            disabled={loadingActions[site.id]}
                                            onOpenDefault={openInDefault}
                                            onOpenTor={openInTor}
                                            onCopyUrl={copyLink}
                                            onCheckHealth={handleHealthCheck}
                                            onEdit={onEdit}
                                            onRequestPassword={(s) => setPasswordTarget(s)}
                                            onRequestDelete={(id) => handleDelete(id)}
                                        />
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            {/* Delete confirmation */}
            <ConfirmDialog
                open={!!deleteTarget}
                onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
                title="Delete site?"
                description={deleteTarget ? (
                  <span>Are you sure you want to delete <b>{deleteTarget.name}</b>? This action cannot be undone.</span>
                ) : undefined}
                confirmText="Delete"
                cancelText="Cancel"
                variant="destructive"
                onConfirm={confirmDelete}
                loading={isDeleting}
            />

            {/* Set password modal */}
            <SetSitePasswordDialog
                open={!!passwordTarget}
                site={passwordTarget}
                onOpenChange={(open) => { if (!open) { setPasswordTarget(null); } }}
                loading={isSettingPassword}
                onSubmit={async (pw) => {
                    if (!passwordTarget) return;
                    setIsSettingPassword(true);
                    await withErrorHandling(async () => {
                        await setSitePassword(passwordTarget.id, pw);
                    }, { successMessage: 'Password updated', showSuccess: true });
                    setIsSettingPassword(false);
                    setPasswordTarget(null);
                }}
            />
        </div>
    );
}