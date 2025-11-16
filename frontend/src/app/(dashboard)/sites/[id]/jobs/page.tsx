"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Plus, ArrowLeft } from "lucide-react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { DataTable } from "@/components/table/data-table";
import { useJobsTable } from "@/hooks/use-jobs-table";
import { RiRefreshLine } from "@remixicon/react";

export default function SiteJobsPage() {
    const params = useParams();
    const siteId = parseInt(params.id as string);
    const router = useRouter();

    const { jobs, isLoading, loadJobs, columns, renderExpandedRow } = useJobsTable(siteId);

    useEffect(() => {
        loadJobs();
    }, [siteId]);

    return (
        <div className="p-6 space-y-6">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <div>
                        <h1 className="text-3xl font-bold tracking-tight">Site Jobs</h1>
                        <p className="text-muted-foreground mt-2">
                            Manage automated content generation for this site
                        </p>
                    </div>
                </div>

                <div className="flex items-center gap-3">
                    <Button
                        variant="outline"
                        onClick={loadJobs}
                        disabled={isLoading}
                    >
                        <RiRefreshLine className="w-4 h-4" />
                        Refresh
                    </Button>

                    <Button onClick={() => router.push(`/sites/${siteId}/jobs/new`)}>
                        <Plus className="h-4 w-4 mr-2" />
                        Create Job
                    </Button>
                </div>
            </div>

            <div className="flex items-center gap-4 py-4">
                <div className="text-sm text-muted-foreground">
                    {jobs.length} job{jobs.length !== 1 ? 's' : ''} for this site
                </div>
                <div className="flex-1 border-t" />
            </div>

            <DataTable
                columns={columns}
                data={jobs}
                searchKey="name"
                searchPlaceholder="Search jobs..."
                isLoading={isLoading}
                emptyMessage="No jobs found"
                enableRowExpand
                expandOnRowClick
                renderExpandedRow={(row) => renderExpandedRow(row)}
            />

        </div>
    );
}