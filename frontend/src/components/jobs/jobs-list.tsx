"use client";

import { Job } from "@/models/jobs";
import { JobCard } from "./job-card";

interface JobsListProps {
    jobs: Job[];
    onEdit: (job: Job) => void;
    onDelete: (job: Job) => void;
    onPause: (job: Job) => void;
    onResume: (job: Job) => void;
    onExecute: (job: Job) => void;
    isLoading?: boolean;
}

export function JobsList({
    jobs,
    onEdit,
    onDelete,
    onPause,
    onResume,
    onExecute,
    isLoading = false
}: JobsListProps) {
    if (isLoading) {
        return (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {[...Array(6)].map((_, i) => (
                    <div key={i} className="animate-pulse">
                        <div className="h-48 bg-muted rounded-lg"></div>
                    </div>
                ))}
            </div>
        );
    }

    if (jobs.length === 0) {
        return (
            <div className="text-center py-12">
                <div className="text-muted-foreground">
                    No jobs found. Create your first job to get started.
                </div>
            </div>
        );
    }

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {jobs.map((job) => (
                <JobCard
                    key={job.id}
                    job={job}
                    onEdit={onEdit}
                    onDelete={onDelete}
                    onPause={onPause}
                    onResume={onResume}
                    onExecute={onExecute}
                />
            ))}
        </div>
    );
}