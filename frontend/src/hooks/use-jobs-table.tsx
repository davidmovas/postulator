"use client";

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { Job } from "@/models/jobs";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useApiCall } from "@/hooks/use-api-call";
import { jobService } from "@/services/jobs";
import { formatDateTime } from "@/lib/time";
import { useContextModal } from "@/context/modal-context";
import { AlertCircle, Edit, MoreHorizontal, Pause, Play, Trash2, Zap, RefreshCw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import JobStatusBadge from "@/components/jobs/job-status-badge";
import { siteService } from "@/services/sites";
import { promptService } from "@/services/prompts";
import { providerService } from "@/services/providers";
import { topicService } from "@/services/topics";
import { useRouter } from "next/navigation";
import { useToast } from "@/components/ui/use-toast";
import { JobStatus } from "@/constants/jobs";
import { TOPIC_STRATEGY_REUSE_WITH_VARIATION } from "@/constants/topics";
import { JobTopicsStatus } from "@/models/topics";

export function useJobsTable(siteId?: number) {
    const [jobs, setJobs] = useState<Job[]>([]);
    const [remainingTopics, setRemainingTopics] = useState<Record<number, number>>({});
    const { execute, isLoading } = useApiCall();
    const { confirmationModal } = useContextModal();

    const router = useRouter();
    const { toast } = useToast();
    const [refreshKey, setRefreshKey] = useState(0);

    const [executingJobs, setExecutingJobs] = useState<Record<number, boolean>>({});
    const executingJobsRef = useRef<Record<number, boolean>>({});

    const loadJobs = useCallback(async () => {
        const data = await execute<Job[]>(() => jobService.listJobs());
        if (data) {
            const filteredJobs = siteId ? data.filter(j => j.siteId === siteId) : data;
            setJobs(filteredJobs);

            loadAllRemainingTopics(filteredJobs);

            setRefreshKey((k) => k + 1);
        }
    }, [execute, siteId]);

    const loadAllRemainingTopics = useCallback(async (jobsToLoad: Job[]) => {
        const counts: Record<number, number> = {};

        for (const job of jobsToLoad) {
            if (job.topicStrategy === TOPIC_STRATEGY_REUSE_WITH_VARIATION) {
                counts[job.id] = -1;
                continue;
            }

            try {
                const topicsStatus: JobTopicsStatus = await topicService.getJobRemainingTopics(job.id);
                counts[job.id] = topicsStatus.count;
            } catch (error) {
                counts[job.id] = 0;
            }
        }

        setRemainingTopics(counts);
    }, []);

    const handlePause = async (job: Job) => {
        await execute<void>(() => jobService.pauseJob(job.id), {
            successMessage: "Job paused successfully",
            showSuccessToast: true,
            onSuccess: loadJobs,
        });
    };

    const handleResume = async (job: Job) => {
        await execute<void>(() => jobService.resumeJob(job.id), {
            successMessage: "Job resumed successfully",
            showSuccessToast: true,
            onSuccess: loadJobs,
        });
    };

    const handleExecute = async (job: Job) => {
        if (executingJobsRef.current[job.id]) return;
        executingJobsRef.current[job.id] = true;
        setExecutingJobs((prev) => ({ ...prev, [job.id]: true }));
        toast({
            title: "Starting job",
            description: `Executing job "${job.name}"...`,
        });

        try {
            const message = await jobService.executeManually(job.id);
            toast({
                title: "Job finished",
                description: message || "Execution completed",
            });
            await loadJobs();
        } catch (e: any) {
            toast({
                title: "Execution error",
                description: e?.message || "Failed to execute job",
                variant: "destructive",
            });
        } finally {
            delete executingJobsRef.current[job.id];
            setExecutingJobs((prev) => ({ ...prev, [job.id]: false }));
        }
    };

    const handleDelete = (job: Job) => {
        confirmationModal.open({
            title: "Delete Job",
            description: (
                <div>
                    Are you sure you want to delete job <span className="font-semibold">{job.name}</span>? This action cannot be undone.
                </div>
            ),
            confirmText: "Delete",
            cancelText: "Cancel",
            variant: "destructive",
            onConfirm: async () => {
                await jobService.deleteJob(job.id);
                await loadJobs();
            },
        });
    };

    const columns: ColumnDef<Job>[] = useMemo(() => [
        {
            accessorKey: "name",
            header: "Name",
            cell: ({ row }) => {
                const job = row.original;
                return <div className="font-medium">{job.name}</div>;
            },
        },
        {
            accessorKey: "status",
            header: "Status",
            cell: ({ row }) => {
                const job = row.original;
                return <JobStatusBadge status={job.status as JobStatus} />;
            },
            filterFn: (row, id, value) => {
                return (value as string[]).includes(row.getValue(id));
            },
        },
        {
            id: "schedule",
            header: "Schedule",
            cell: ({ row }) => {
                const j = row.original;
                if (!j.schedule) return <span className="text-muted-foreground">Manual</span>;
                const s = j.schedule;
                switch (s.type) {
                    case "manual":
                        return "Manual";
                    case "once":
                        return `${formatDateTime(s.config?.executeAt || s.config?.execute_at)}`;
                    case "interval":
                        return `Every ${s.config?.value} ${s.config?.unit}`;
                    case "daily":
                        return `Daily @ ${s.config?.hour}:${String(s.config?.minute).padStart(2, "0")}`;
                    default:
                        return "—";
                }
            },
        },
        {
            id: "lastRun",
            header: "Last Run",
            cell: ({ row }) => {
                const dt = row.original.state?.lastRunAt;
                return dt ? formatDateTime(dt) : "Never";
            },
        },
        {
            id: "nextRun",
            header: "Next Run",
            cell: ({ row }) => {
                const dt = row.original.state?.nextRunAt;
                return dt ? formatDateTime(dt) : "—";
            },
        },
        {
            id: "topicsLeft",
            header: "Topics Left",
            cell: ({ row }) => {
                const job = row.original;
                const count = remainingTopics[job.id];

                if (job.topicStrategy === TOPIC_STRATEGY_REUSE_WITH_VARIATION) {
                    return (
                        <RefreshCw className="w-4 h-4" />
                    );
                }

                if (count === undefined) {
                    return <span className="text-muted-foreground">Loading...</span>;
                }

                let textColor = "";
                if (count === 0) {
                    textColor = "text-red-500";
                } else if (count <= 5) {
                    textColor = "text-amber-500";
                }

                return (
                    <span className={`font-medium ${textColor}`}>
                        {count}
                    </span>
                );
            },
        },
        {
            id: "executionsTotal",
            header: "Executions",
            cell: ({ row }) => {
                const total = row.original.state?.totalExecutions ?? 0;
                return <span className="font-semibold">{total}</span>;
            },
        },
        {
            id: "executionsFailed",
            header: "Failed",
            cell: ({ row }) => {
                const failed = row.original.state?.failedExecutions ?? 0;
                return <span className={failed > 0 ? "font-semibold text-red-500" : "font-semibold"}>{failed}</span>;
            },
        },
        {
            id: "actions",
            header: "Actions",
            cell: ({ row }) => {
                const job = row.original;
                const isActive = job.status === "active";
                const isExecuting = executingJobs[job.id];

                const handleEdit = () => {
                    router.push(`/jobs/${job.id}/edit`);
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
                            {isActive ? (
                                <DropdownMenuItem onClick={() => handlePause(job)}>
                                    <Pause className="mr-2 h-4 w-4" />
                                    <span>Pause</span>
                                </DropdownMenuItem>
                            ) : (
                                <DropdownMenuItem onClick={() => handleResume(job)}>
                                    <Play className="mr-2 h-4 w-4" />
                                    <span>Resume</span>
                                </DropdownMenuItem>
                            )}

                            <DropdownMenuItem onClick={() => !isExecuting && handleExecute(job)} disabled={isExecuting}>
                                <Zap className={`mr-2 h-4 w-4 ${isExecuting ? "opacity-50" : ""}`} />
                                <span>{isExecuting ? "Running..." : "Run Now"}</span>
                            </DropdownMenuItem>

                            <DropdownMenuItem onClick={handleEdit}>
                                <Edit className="mr-2 h-4 w-4" />
                                <span>Edit</span>
                            </DropdownMenuItem>

                            <DropdownMenuSeparator />

                            <DropdownMenuItem onClick={() => handleDelete(job)} className="text-destructive focus:text-destructive">
                                <Trash2 className="mr-2 h-4 w-4" />
                                <span>Delete</span>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                );
            },
        },
    ], [remainingTopics, executingJobs, router]);

    const capitalize = (s?: string) => (s ? s.charAt(0).toUpperCase() + s.slice(1) : "");

    const ExpandedRow = ({ job }: { job: Job }) => {
        const [siteName, setSiteName] = useState<string>("");
        const [promptName, setPromptName] = useState<string>("");
        const [providerName, setProviderName] = useState<string>("");
        const [nextTopicTitle, setNextTopicTitle] = useState<string | null>(null);
        const [isLoading, setIsLoading] = useState({
            site: false,
            prompt: false,
            provider: false,
            topic: false
        });

        useEffect(() => {
            const loadData = async () => {
                try {
                    // Загружаем site name
                    if (job.siteId) {
                        setIsLoading(prev => ({ ...prev, site: true }));
                        try {
                            const site = await siteService.getSite(job.siteId);
                            setSiteName(site.name);
                        } catch (error) {
                            console.error(`Failed to load site ${job.siteId}:`, error);
                            setSiteName("Error loading site");
                        } finally {
                            setIsLoading(prev => ({ ...prev, site: false }));
                        }
                    }

                    // Загружаем prompt name
                    if (job.promptId) {
                        setIsLoading(prev => ({ ...prev, prompt: true }));
                        try {
                            const prompt = await promptService.getPrompt(job.promptId);
                            setPromptName(prompt.name);
                        } catch (error) {
                            console.error(`Failed to load prompt ${job.promptId}:`, error);
                            setPromptName("Error loading prompt");
                        } finally {
                            setIsLoading(prev => ({ ...prev, prompt: false }));
                        }
                    }

                    // Загружаем provider name
                    if (job.aiProviderId) {
                        setIsLoading(prev => ({ ...prev, provider: true }));
                        try {
                            const provider = await providerService.getProvider(job.aiProviderId);
                            setProviderName(provider.name);
                        } catch (error) {
                            console.error(`Failed to load provider ${job.aiProviderId}:`, error);
                            setProviderName("Error loading provider");
                        } finally {
                            setIsLoading(prev => ({ ...prev, provider: false }));
                        }
                    }

                    // Для НЕ-Reuse стратегий загружаем следующий топик
                    if (job.topicStrategy !== TOPIC_STRATEGY_REUSE_WITH_VARIATION) {
                        setIsLoading(prev => ({ ...prev, topic: true }));
                        try {
                            const topic = await topicService.getNextTopicForJob(job.id);
                            setNextTopicTitle(topic?.title || null);
                        } catch (error) {
                            console.error(`Failed to load next topic for job ${job.id}:`, error);
                            setNextTopicTitle(null);
                        } finally {
                            setIsLoading(prev => ({ ...prev, topic: false }));
                        }
                    }
                } catch (error) {
                    console.error("Error loading expanded row data:", error);
                }
            };

            loadData();
        }, [job.id, job.siteId, job.promptId, job.aiProviderId, job.topicStrategy]); // Зависимости только от ID

        const schedule = job.schedule;

        const InfoItem = ({ label, children }: { label: string; children: React.ReactNode }) => (
            <div className="min-w-[180px] max-w-full">
                <div className="text-muted-foreground text-xs">{label}</div>
                <div className="font-medium mt-1 break-words">{children}</div>
            </div>
        );

        const getLoadingText = (isLoading: boolean) => isLoading ? "Loading..." : "—";

        return (
            <div className="text-sm space-y-4 p-4 bg-muted/30 rounded-md">
                {/* Scheduling */}
                <div className="space-y-2">
                    <div className="text-xs uppercase tracking-wide text-muted-foreground">Scheduling</div>
                    <div className="flex flex-wrap gap-6">
                        {!schedule ? (
                            <InfoItem label="Mode">Manual</InfoItem>
                        ) : (
                            <>
                                <InfoItem label="Type">{capitalize(schedule.type)}</InfoItem>
                                {schedule.type === "once" && (
                                    <InfoItem label="Execute At">{formatDateTime(schedule.config?.executeAt || schedule.config?.execute_at)}</InfoItem>
                                )}
                                {schedule.type === "interval" && (
                                    <>
                                        <InfoItem label="Every">{schedule.config?.value} {schedule.config?.unit}</InfoItem>
                                        {(schedule.config?.startAt || schedule.config?.start_at) && (
                                            <InfoItem label="Start At">{formatDateTime(schedule.config?.startAt || schedule.config?.start_at)}</InfoItem>
                                        )}
                                    </>
                                )}
                                {schedule.type === "daily" && (
                                    <>
                                        <InfoItem label="Time">{schedule.config?.hour}:{String(schedule.config?.minute).padStart(2, "0")}</InfoItem>
                                        {Array.isArray(schedule.config?.weekdays) && (
                                            <InfoItem label="Weekdays">
                                                {formatWeekdays(schedule.config.weekdays)}
                                            </InfoItem>
                                        )}
                                    </>
                                )}
                            </>
                        )}
                    </div>
                </div>

                {/* Strategies */}
                <div className="space-y-2">
                    <div className="text-xs uppercase tracking-wide text-muted-foreground">Strategies</div>
                    <div className="flex flex-wrap gap-6">
                        <InfoItem label="Topic">{job.topicStrategy === TOPIC_STRATEGY_REUSE_WITH_VARIATION ? "Reuse" : "Unique"}</InfoItem>
                        <InfoItem label="Category">{capitalize(job.categoryStrategy)}</InfoItem>
                        <InfoItem label="Requires Validation">{job.requiresValidation ? "Yes" : "No"}</InfoItem>
                        <InfoItem label="Jitter">{job.jitterEnabled ? `${job.jitterMinutes} min` : "Disabled"}</InfoItem>
                    </div>
                </div>

                {/* References */}
                <div className="space-y-2">
                    <div className="text-xs uppercase tracking-wide text-muted-foreground">References</div>
                    <div className="flex flex-wrap gap-6">
                        <InfoItem label="Site">
                            {job.siteId ? (isLoading.site ? getLoadingText(true) : siteName) : "—"}
                        </InfoItem>
                        <InfoItem label="Prompt">
                            {job.promptId ? (isLoading.prompt ? getLoadingText(true) : promptName) : "—"}
                        </InfoItem>
                        <InfoItem label="Provider">
                            {job.aiProviderId ? (isLoading.provider ? getLoadingText(true) : providerName) : "—"}
                        </InfoItem>
                    </div>
                </div>

                {/* Stats */}
                <div className="space-y-2">
                    <div className="text-xs uppercase tracking-wide text-muted-foreground">Stats</div>
                    <div className="flex flex-wrap gap-6">
                        <InfoItem label="Last Run">{job.state?.lastRunAt ? formatDateTime(job.state.lastRunAt) : "Never"}</InfoItem>
                        <InfoItem label="Next Run">{job.state?.nextRunAt ? formatDateTime(job.state.nextRunAt) : "—"}</InfoItem>
                    </div>
                </div>

                {/* Next Topic */}
                <div className="space-y-2">
                    <div className="text-xs uppercase tracking-wide text-muted-foreground">Next Topic To Use</div>
                    {job.topicStrategy === TOPIC_STRATEGY_REUSE_WITH_VARIATION ? (
                        <Badge variant="outline" className="bg-blue-50 text-blue-700 border-blue-200 flex items-center gap-1">
                            Will generate topic variation from random provided topics
                        </Badge>
                    ) : isLoading.topic ? (
                        <span className="text-muted-foreground">Loading next topic...</span>
                    ) : nextTopicTitle ? (
                        <Badge variant="default" className="text-xs px-2 py-0.5 font-medium">
                            {nextTopicTitle}
                        </Badge>
                    ) : (
                        <Badge variant="destructive" className="text-xs px-2 py-0.5 font-medium flex items-center gap-1">
                            <AlertCircle className="h-4 w-4" />
                            No topics available - job paused
                        </Badge>
                    )}
                </div>
            </div>
        );
    };

    const renderExpandedRow = (job: Job) => {
        return <ExpandedRow key={job.id} job={job} />;
    };

    return {
        jobs,
        setJobs,
        columns,
        isLoading,
        loadJobs,
        renderExpandedRow,
    };
}

function formatWeekdays(weekdays: number[]): string {
    const dayMap: Record<number, string> = {
        1: "Mon",
        2: "Tue",
        3: "Wed",
        4: "Thu",
        5: "Fri",
        6: "Sat",
        7: "Sun"
    };

    const sortedDays = [...weekdays].sort((a, b) => a - b);
    const dayNames = sortedDays.map(day => dayMap[day]).filter(Boolean);
    return dayNames.join(", ");
}