"use client";

import { Button } from "@/components/ui/button";
import { JobsList } from "@/components/jobs/jobs-list";
import { useJobs } from "@/hooks/use-jobs";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";

export default function JobsPage() {
    const router = useRouter();

    const {
        jobs,
        isLoading,
        loadJobs,
        handleEditJob,
        handleDeleteJob,
        handlePauseJob,
        handleResumeJob,
        handleExecuteJob
    } = useJobs();

    return (
        <div className="p-6 space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">All Jobs</h1>
                    <p className="text-muted-foreground mt-2">
                        Manage all automated content generation jobs
                    </p>
                </div>

                <div className="flex items-center gap-3">
                    <Button
                        variant="outline"
                        onClick={loadJobs}
                        disabled={isLoading}
                    >
                        Refresh
                    </Button>

                    <Button onClick={() => router.push("/jobs/new")}>
                        <Plus className="h-4 w-4 mr-2" />
                        Create Job
                    </Button>
                </div>
            </div>

            <div className="flex items-center gap-4 py-4">
                <div className="text-sm text-muted-foreground">
                    {jobs.length} job{jobs.length !== 1 ? 's' : ''} configured
                </div>
                <div className="flex-1 border-t" />
            </div>

            <JobsList
                jobs={jobs}
                onEdit={handleEditJob}
                onDelete={handleDeleteJob}
                onPause={handlePauseJob}
                onResume={handleResumeJob}
                onExecute={handleExecuteJob}
                isLoading={isLoading}
            />
        </div>
    );
}