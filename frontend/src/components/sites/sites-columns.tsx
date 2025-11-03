"use client";

import { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal, ExternalLink, Play, Trash2, Wrench, RefreshCw } from "lucide-react";
import Link from "next/link";
import { Site } from "@/models/sites";

// Health status indicator
const HealthIndicator = ({ status }: { status: string }) => {
    const getHealthConfig = (health: string) => {
        switch (health) {
            case "healthy":
                return { color: "bg-green-500", label: "Healthy" };
            case "unhealthy":
                return { color: "bg-red-500", label: "Unhealthy" };
            case "unknown":
                return { color: "bg-gray-500", label: "Unknown" };
            default:
                return { color: "bg-gray-500", label: "Unknown" };
        }
    };

    const config = getHealthConfig(status);

    return (
        <div className="flex items-center gap-2">
            <div className={`w-3 h-3 rounded-full ${config.color}`} />
            <span className="text-sm">{config.label}</span>
        </div>
    );
};

// Status badge
const StatusBadge = ({ status }: { status: string }) => {
    const getStatusConfig = (status: string) => {
        switch (status) {
            case "active":
                return { variant: "default" as const, label: "Active" };
            case "inactive":
                return { variant: "secondary" as const, label: "Inactive" };
            case "error":
                return { variant: "destructive" as const, label: "Error" };
            default:
                return { variant: "outline" as const, label: status };
        }
    };

    const config = getStatusConfig(status);

    return <Badge variant={config.variant}>{config.label}</Badge>;
};

export const columns: ColumnDef<Site>[] = [
    {
        accessorKey: "name",
        header: "Name",
        cell: ({ row }) => {
            const site = row.original;
            return (
                <Link
                    href={`/sites/${site.id}`}
                    className="font-medium text-blue-600 hover:text-blue-800 hover:underline"
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
                    className="text-blue-600 hover:text-blue-800 hover:underline flex items-center gap-1"
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
            return <StatusBadge status={status} />;
        },
        filterFn: (row, id, value) => {
            return value.includes(row.getValue(id));
        },
    },
    {
        accessorKey: "healthStatus",
        header: "Health",
        cell: ({ row }) => {
            const healthStatus = row.getValue("healthStatus") as string;
            return <HealthIndicator status={healthStatus} />;
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
            return new Date(date).toLocaleDateString('en-US', {
                year: 'numeric',
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        },
    },
    {
        id: "articles",
        header: "Articles",
        cell: ({ row }) => {
            const site = row.original;
            // TODO: Replace with actual articles count from your API
            const articlesCount = 0;

            return (
                <Link
                    href={`/sites/${site.id}?tab=articles`}
                    className="text-blue-600 hover:text-blue-800 hover:underline font-medium"
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
            // TODO: Replace with actual jobs count from your API
            const jobsCount = 0;

            return (
                <Link
                    href={`/sites/${site.id}?tab=jobs`}
                    className="text-blue-600 hover:text-blue-800 hover:underline font-medium"
                >
                    {jobsCount}
                </Link>
            );
        },
    },
    {
        id: "actions",
        cell: ({ row }) => {
            const site = row.original;

            const handleEdit = () => {
                // TODO: Open edit modal
                console.log('Edit site:', site.id);
            };

            const handleDelete = () => {
                // TODO: Open delete confirmation
                console.log('Delete site:', site.id);
            };

            const handleCheckHealth = async () => {
                // TODO: Call checkHealth API
                console.log('Check health for site:', site.id);
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
                        <DropdownMenuLabel>Actions</DropdownMenuLabel>
                        <DropdownMenuItem onClick={handleEdit}>
                            <Wrench className="mr-2 h-4 w-4" />
                            <span>Edit</span>
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={openWordPress}>
                            <ExternalLink className="mr-2 h-4 w-4" />
                            <span>Open WordPress</span>
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={handleCheckHealth}>
                            <RefreshCw className="mr-2 h-4 w-4" />
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
];