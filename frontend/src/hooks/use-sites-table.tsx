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
import { formatDateTime } from "@/lib/time";
import { useApiCall } from "@/hooks/use-api-call";
import { siteService } from "@/services/sites";
import { healthcheckService } from "@/services/healthcheck";
import { Button } from "@/components/ui/button";
import {
    RiCheckboxBlankCircleLine,
    RiCheckboxCircleFill, RiCheckboxCircleLine,
    RiLockPasswordLine,
    RiPencilLine,
    RiPulseLine
} from "@remixicon/react";
import { useContextModal } from "@/context/modal-context";
import SiteStatusBadge from "@/components/sites/site-status-badge";
import HealthIndicator from "@/components/sites/site-health-badge";

export function useSitesTable() {
    const [sites, setSites] = useState<Site[]>([]);
    const { editSiteModal, passwordModal, deleteSiteModal } = useContextModal();
    const { execute, isLoading } = useApiCall();

    const updateSiteStatus = useCallback((siteId: number, newStatus: string) => {
        setSites(prev => prev.map(site =>
            site.id === siteId ? { ...site, healthStatus: newStatus } : site
        ));
    }, []);

    const handleCheckHealth = useCallback(async (siteId: number) => {
        await execute<string>(
            () => healthcheckService.checkHealth(siteId),
            {
                onSuccess: () => {
                    updateSiteStatus(siteId, "healthy");
                    loadSites();
                },
                onError: () => {
                    updateSiteStatus(siteId, "unhealthy");
                },
                showSuccessToast: false,
            }
        );
    }, [execute, updateSiteStatus]);

    const handleToggleAutoHealthCheck = async (site: Site) => {
        await execute<void>(
            () => siteService.updateSite({
                id: site.id,
                name: site.name,
                url: site.url,
                wpUsername: site.wpUsername,
                status: site.status,
                autoHealthCheck: !site.autoHealthCheck
            }),
            {
                successMessage: `Auto health check ${!site.autoHealthCheck ? "enabled" : "disabled"} successfully`,
                showSuccessToast: true,
                onSuccess: loadSites
            }
        );
    };

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
            accessorKey: "autoHealthCheck",
            header: "Auto Check",
            cell: ({ row }) => {
                const autoHealthCheck = row.getValue("autoHealthCheck") as boolean;

                return (
                    <div className="pl-6">
                        {autoHealthCheck ? (
                            <RiCheckboxCircleFill className="h-5 w-5 text-green-500" />
                        ) : (
                            <RiCheckboxBlankCircleLine className="h-5 w-5 text-gray-400" />
                        )}
                    </div>
                );
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
                    deleteSiteModal.open(site);
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

                            <DropdownMenuItem onClick={() => handleToggleAutoHealthCheck(site)}>
                                <RiCheckboxCircleLine className="mr-2 h-4 w-4" />
                                <span>{site.autoHealthCheck ? "Disable" : "Enable"} Auto Check</span>
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
    ], [handleCheckHealth, handleToggleAutoHealthCheck]);

    return {
        sites,
        setSites,
        columns,
        isLoading,
        loadSites,
        handleCheckHealth
    };
}