"use client";

import { useState, useEffect } from "react";
import { Job } from "@/models/jobs";
import { jobService } from "@/services/jobs";
import { useApiCall } from "./use-api-call";

export function useJobs(siteId?: number) {
    const [jobs, setJobs] = useState<Job[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const { execute } = useApiCall();

    const loadJobs = async () => {
        setIsLoading(true);
        try {
            const data = await jobService.listJobs();
            const filteredJobs = siteId ? data.filter(job => job.siteId === siteId) : data;
            setJobs(filteredJobs);
        } catch (error) {
            console.error("Failed to load jobs:", error);
        } finally {
            setIsLoading(false);
        }
    };

    const handleEditJob = (job: Job) => {
        // TODO: Implement edit modal
        console.log("Edit job:", job);
    };

    const handleDeleteJob = (job: Job) => {
        // TODO: Implement delete confirmation
        console.log("Delete job:", job);
    };

    const handlePauseJob = async (job: Job) => {
        await execute(
            () => jobService.pauseJob(job.id),
            {
                successMessage: "Job paused successfully",
                showSuccessToast: true,
                onSuccess: loadJobs
            }
        );
    };

    const handleResumeJob = async (job: Job) => {
        await execute(
            () => jobService.resumeJob(job.id),
            {
                successMessage: "Job resumed successfully",
                showSuccessToast: true,
                onSuccess: loadJobs
            }
        );
    };

    const handleExecuteJob = async (job: Job) => {
        await execute(
            () => jobService.executeManually(job.id),
            {
                successMessage: "Job executed manually",
                showSuccessToast: true
            }
        );
    };

    useEffect(() => {
        loadJobs();
    }, [siteId]);

    return {
        jobs,
        isLoading,
        loadJobs,
        handleEditJob,
        handleDeleteJob,
        handlePauseJob,
        handleResumeJob,
        handleExecuteJob
    };
}