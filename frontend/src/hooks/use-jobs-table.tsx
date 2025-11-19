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
import { Edit, MoreHorizontal, Pause, Play, Trash2, Zap } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import JobStatusBadge from "@/components/jobs/job-status-badge";
import { siteService } from "@/services/sites";
import { promptService } from "@/services/prompts";
import { providerService } from "@/services/providers";
import { topicService } from "@/services/topics";
import { useRouter } from "next/navigation";
import { useToast } from "@/components/ui/use-toast";
import { JobStatus } from "@/constants/jobs";

export function useJobsTable(siteId?: number) {
  const [jobs, setJobs] = useState<Job[]>([]);
  const { execute, isLoading } = useApiCall();
  const { confirmationModal } = useContextModal();
  const router = useRouter();
  const { toast } = useToast();
  // Ключ обновления, чтобы форсировать перезагрузку зависимых данных в раскрытых строках
  const [refreshKey, setRefreshKey] = useState(0);
  // Локальное состояние выполнения конкретных джоб вручную (не блокируем всю таблицу)
  const [executingJobs, setExecutingJobs] = useState<Record<number, boolean>>({});
  // Мгновенная защита от повторных кликов до обновления состояния (ре-энтранси гард)
  const executingJobsRef = useRef<Record<number, boolean>>({});

  const loadJobs = useCallback(async () => {
    const data = await execute<Job[]>(() => jobService.listJobs());
    if (data) {
      setJobs(siteId ? data.filter(j => j.siteId === siteId) : data);
      // Сбросить кэш «Next Topic To Use», чтобы при обновлении данные подтягивались заново
      setNextTopicTitles({});
      // Увеличить ключ, чтобы эффекты в раскрытых строках перезапустились
      setRefreshKey((k) => k + 1);
    }
  }, [execute, siteId]);

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
    // Синхронная защита от повторных кликов (до того как React обновит состояние)
    if (executingJobsRef.current[job.id]) return;
    executingJobsRef.current[job.id] = true;
    setExecutingJobs((prev) => ({ ...prev, [job.id]: true }));
    toast({
      title: "Starting job",
      description: `Executing job "${job.name}"...`,
    });

    try {
      const message = await jobService.executeManually(job.id);
      // Успешное завершение: обновляем таблицу и показываем тост
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
      // Снимаем блокировку
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
            return `${s.config?.hour}:${String(s.config?.minute).padStart(2, "0")}`;
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
        const isExecuting = !!executingJobs[job.id];

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
  ], [executingJobs]);

  // Caches for human-readable names and next topic
  const [siteNames, setSiteNames] = useState<Record<number, string>>({});
  const [promptNames, setPromptNames] = useState<Record<number, string>>({});
  const [providerNames, setProviderNames] = useState<Record<number, string>>({});
  const [nextTopicTitles, setNextTopicTitles] = useState<Record<number, string | null>>({});

  const capitalize = (s?: string) => (s ? s.charAt(0).toUpperCase() + s.slice(1) : "");

  const ExpandedRow = ({ job }: { job: Job }) => {
    const [loadingRefs, setLoadingRefs] = useState({ site: false, prompt: false, provider: false, topic: false });

    useEffect(() => {
      const loadRefs = async () => {
        try {
          if (job.siteId && siteNames[job.siteId] === undefined && !loadingRefs.site) {
            setLoadingRefs(prev => ({ ...prev, site: true }));
            try {
              const site = await siteService.getSite(job.siteId);
              setSiteNames(prev => ({ ...prev, [job.siteId]: site.name }));
            } catch { /* silent */ }
            finally { setLoadingRefs(prev => ({ ...prev, site: false })); }
          }
          if (job.promptId && promptNames[job.promptId] === undefined && !loadingRefs.prompt) {
            setLoadingRefs(prev => ({ ...prev, prompt: true }));
            try {
              const prompt = await promptService.getPrompt(job.promptId);
              setPromptNames(prev => ({ ...prev, [job.promptId]: prompt.name }));
            } catch { /* silent */ }
            finally { setLoadingRefs(prev => ({ ...prev, prompt: false })); }
          }
          if (job.aiProviderId && providerNames[job.aiProviderId] === undefined && !loadingRefs.provider) {
            setLoadingRefs(prev => ({ ...prev, provider: true }));
            try {
              const provider = await providerService.getProvider(job.aiProviderId);
              setProviderNames(prev => ({ ...prev, [job.aiProviderId]: provider.name }));
            } catch { /* silent */ }
            finally { setLoadingRefs(prev => ({ ...prev, provider: false })); }
          }
          if (nextTopicTitles[job.id] === undefined && !loadingRefs.topic) {
            setLoadingRefs(prev => ({ ...prev, topic: true }));
            try {
              const topic = await topicService.getNextTopicForJob(job.id);
              setNextTopicTitles(prev => ({ ...prev, [job.id]: topic?.title || null }));
            } catch {
              setNextTopicTitles(prev => ({ ...prev, [job.id]: null }));
            } finally { setLoadingRefs(prev => ({ ...prev, topic: false })); }
          }
        } catch { /* noop */ }
      };
      loadRefs();
    // Добавили refreshKey: при обновлении данных таблицы принудительно перезапрашиваем next topic
    }, [job.id, job.siteId, job.promptId, job.aiProviderId, refreshKey]);

    const schedule = job.schedule;

    const InfoItem = ({label, children}:{label:string; children: React.ReactNode}) => (
      <div className="min-w-[180px] max-w-full">
        <div className="text-muted-foreground text-xs">{label}</div>
        <div className="font-medium mt-1 break-words">{children}</div>
      </div>
    );

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
                      <InfoItem label="Weekdays">{schedule.config.weekdays.join(", ")}</InfoItem>
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
            <InfoItem label="Topic">{capitalize(job.topicStrategy)}</InfoItem>
            <InfoItem label="Category">{capitalize(job.categoryStrategy)}</InfoItem>
            <InfoItem label="Requires Validation">{job.requiresValidation ? "Yes" : "No"}</InfoItem>
            <InfoItem label="Jitter">{job.jitterEnabled ? `${job.jitterMinutes} min` : "Disabled"}</InfoItem>
          </div>
        </div>

        {/* References */}
        <div className="space-y-2">
          <div className="text-xs uppercase tracking-wide text-muted-foreground">References</div>
          <div className="flex flex-wrap gap-6">
            <InfoItem label="Site">{siteNames[job.siteId] ?? ""}</InfoItem>
            <InfoItem label="Prompt">{promptNames[job.promptId] ?? ""}</InfoItem>
            <InfoItem label="Provider">{providerNames[job.aiProviderId] ?? ""}</InfoItem>
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
        {nextTopicTitles[job.id] !== undefined && nextTopicTitles[job.id] !== null && (
          <div className="space-y-2">
            <div className="text-xs uppercase tracking-wide text-muted-foreground">Next Topic To Use</div>
            <Badge className="text-xs px-2 py-0.5 font-medium">
              {nextTopicTitles[job.id]}
            </Badge>
          </div>
        )}
      </div>
    );
  };

  const renderExpandedRow = (job: Job) => {
    return <ExpandedRow job={job} />;
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
