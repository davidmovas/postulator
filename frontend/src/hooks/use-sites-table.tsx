"use client";

import { useState, useMemo, useCallback } from "react";
import { ColumnDef } from "@tanstack/react-table";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal, ExternalLink, Trash2 } from "lucide-react";
import Link from "next/link";
import { Site } from "@/models/sites";
import SiteStatusBadge from "@/components/sites/SiteStatusBadge";
import HealthIndicator from "@/components/sites/SiteHealthBadge";
import { formatDateTime } from "@/lib/time";
import { useApiCall } from "@/hooks/use-api-call";
import { siteService } from "@/services/sites";
import { Button } from "@/components/ui/button";
import { RiLockPasswordLine, RiPencilLine, RiPulseLine } from "@remixicon/react";
import { useContextModal } from "@/context/modal-context";

export function useSitesTable() {
    const [sites, setSites] = useState<Site[]>([]);
    const { editSiteModal, passwordModal, confirmationModal } = useContextModal();
    const { execute, isLoading } = useApiCall();

    const updateSiteStatus = useCallback((siteId: number, newStatus: string) => {
        setSites(prev => prev.map(site =>
            site.id === siteId ? { ...site, healthStatus: newStatus } : site
        ));
    }, []);

    const handleCheckHealth = useCallback(async (siteId: number) => {
        await execute<string>(
            () => siteService.checkHealth(siteId),
            {
                onSuccess: () => {
                    updateSiteStatus(siteId, "unknown");
                    setTimeout(() => {
                        loadSites();
                    }, 1000);
                },
                onError: () => {
                    updateSiteStatus(siteId, "unhealthy");
                },
                showSuccessToast: false,
            }
        );
    }, [execute, updateSiteStatus]);

    const loadSites = useCallback(async () => {
        const sitesData = await execute<Site[]>(() => siteService.listSites());
        if (sitesData) {
            setSites(sitesData);
        }
    }, [execute]);

    const columns: ColumnDef<Site>[] = useMemo(() => [
        {
            accessorKey: "name",
            header: "Name",
            cell: ({ row }) => {
                const site = row.original;
                return (
                    <Link
                        href={`/sites/${site.id}`}
                className="font-medium"
                    >
                    {site.name}
                    </Link>
            );
            },
        },
        {
            accessorKey: "url",
            header: "URL",
            cell: ({ row }) => {
                const url = row.getValue("url") as string;
                return (
                    <a
                        href={url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-blue-500 hover:text-blue-700 hover:underline flex items-center gap-1"
                    >
                    {url.replace(/^https?:\/\//, '')}
                    <ExternalLink className="w-3 h-3" />
                    </a>
            );
            },
        },
        {
            accessorKey: "status",
            header: "Status",
            cell: ({ row }) => {
                const status = row.getValue("status") as string;
                return <SiteStatusBadge status={status} />;
            },
            filterFn: (row, id, value) => {
                return value.includes(row.getValue(id));
            },
        },
        {
            accessorKey: "healthStatus",
            header: "Health",
            cell: ({ row }) => {
                const site = row.original;
                const healthStatus = row.getValue("healthStatus") as string;

                return (
                    <div
                        className="cursor-pointer hover:opacity-80 transition-opacity"
                onClick={() => handleCheckHealth(site.id)}
                title="Click to check health"
                >
                <HealthIndicator status={healthStatus} />
                </div>
            );
            },
            filterFn: (row, id, value) => {
                return value.includes(row.getValue(id));
            },
        },
        {
            accessorKey: "lastHealthCheck",
            header: "Last Check",
            cell: ({ row }) => {
                const date = row.getValue("lastHealthCheck") as string;
                const formattedDate = formatDateTime(date);
                return formattedDate || 'Never';
            },
        },
        {
            id: "articles",
            header: "Articles",
            cell: ({ row }) => {
                const site = row.original;
                const articlesCount = 0;

                return (
                    <Link
                        href={`/sites/${site.id}?tab=articles`}
                className="text-blue-500 hover:text-blue-700 hover:underline font-medium"
                    >
                    {articlesCount}
                    </Link>
            );
            },
        },
        {
            id: "jobs",
            header: "Jobs",
            cell: ({ row }) => {
                const site = row.original;
                const jobsCount = 0;

                return (
                    <Link
                        href={`/sites/${site.id}?tab=jobs`}
                className="text-blue-500 hover:text-blue-700 hover:underline font-medium"
                    >
                    {jobsCount}
                    </Link>
            );
            },
        },
        {
            id: "actions",
            header: "Actions",
            cell: ({ row }) => {
                const site = row.original;

                const handleEdit = () => {
                    editSiteModal.open(site);
                };

                const handleSetPassword = () => {
                    passwordModal.open(site);
                }

                const handleDelete = () => {
                    confirmationModal.open({
                        title: "Delete Site",
                        description: (
                            <div className="space-y-2">
                                <p>Are you sure you want to delete this site?</p>
                                <p className="text-muted-foreground font-medium bg-muted/50 px-3 py-2 rounded-md border">
                                    {site.name}
                                </p>
                                <p className="text-sm text-muted-foreground">
                                    This action cannot be undone.
                                </p>
                            </div>
                        ),
                        confirmText: "Delete",
                        cancelText: "Cancel",
                        variant: "destructive",
                        onConfirm: async () => {
                            await execute<void>(
                                () => siteService.deleteSite(site.id),
                                {
                                    onSuccess: () => {
                                        loadSites();
                                    },
                                    showSuccessToast: true,
                                    errorTitle: "Failed to delete site"
                                }
                            );
                        }
                    });
                };

                const handleCheckHealthAction = async () => {
                    await handleCheckHealth(site.id);
                };

                const openWordPress = () => {
                    window.open(site.url + '/wp-admin', '_blank');
                };

                return (
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" className="h-8 w-8 p-0">
                                <span className="sr-only">Open menu</span>
                                <MoreHorizontal className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>

                        <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={handleEdit}>
                                 <RiPencilLine className="mr-2 h-4 w-4" />
                                     <span>Edit</span>
                            </DropdownMenuItem>

                            <DropdownMenuItem onClick={handleSetPassword}>
                                <RiLockPasswordLine className="mr-2 h-4 w-4" />
                                <span>Set Password</span>
                            </DropdownMenuItem>

                            <DropdownMenuItem onClick={openWordPress}>
                                <ExternalLink className="mr-2 h-4 w-4" />
                                <span>Admin Panel</span>
                            </DropdownMenuItem>

                            <DropdownMenuItem onClick={handleCheckHealthAction}>
                                <RiPulseLine className="mr-2 h-4 w-4" />
                                <span>Check Health</span>
                            </DropdownMenuItem>

                            <DropdownMenuSeparator />

                            <DropdownMenuItem onClick={handleDelete} className="text-red-600">
                                <Trash2 className="mr-2 h-4 w-4" />
                                <span>Delete</span>
                                </DropdownMenuItem>
                         </DropdownMenuContent>
                    </DropdownMenu>
            );
            },
        },
    ], [handleCheckHealth]);

    return {
        sites,
        setSites,
        columns,
        isLoading,
        loadSites,
        handleCheckHealth
    };
}