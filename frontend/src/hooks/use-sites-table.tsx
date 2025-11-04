"use client";

import { useState, useMemo, useCallback } from "react";
import { ColumnDef } from "@tanstack/react-table";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal, ExternalLink, Trash2, Wrench, RefreshCw } from "lucide-react";
import Link from "next/link";
import { Site } from "@/models/sites";
import SiteStatusBadge from "@/components/sites/SiteStatusBadge";
import HealthIndicator from "@/components/sites/SiteHealthBadge";
import { formatDateTime } from "@/lib/time";
import { useApiCall } from "@/hooks/use-api-call";
import { siteService } from "@/services/sites";
import { Button } from "@/components/ui/button";
import { RiPencilLine, RiPulseLine } from "@remixicon/react";

export function useSitesTable() {
    const [sites, setSites] = useState<Site[]>([]);
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
                    // Временно ставим healthy, в реальности нужно обновить данные
                    updateSiteStatus(siteId, "healthy");
                    // Через секунду обновим весь список для получения актуальных данных
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
                    console.log('Edit site:', site.id);
                };

                const handleDelete = () => {
                    console.log('Delete site:', site.id);
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

                            <DropdownMenuItem onClick={openWordPress}>
                                <ExternalLink className="mr-2 h-4 w-4" />
                                <span>Open Admin Panel</span>
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