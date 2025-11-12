"use client";

import { useCallback, useMemo, useState } from "react";
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

export function useJobsTable(siteId?: number) {
  const [jobs, setJobs] = useState<Job[]>([]);
  const { execute, isLoading } = useApiCall();
  const { confirmationModal } = useContextModal();

  const loadJobs = useCallback(async () => {
    const data = await execute<Job[]>(() => jobService.listJobs());
    if (data) {
      setJobs(siteId ? data.filter(j => j.siteId === siteId) : data);
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
    await execute<void>(() => jobService.executeManually(job.id), {
      successMessage: "Job executed manually",
      showSuccessToast: true,
    });
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

  // Status icons removed per requirement

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
        return (
          <div className="text-sm">
            <span className="capitalize">{job.status}</span>
          </div>
        );
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
        const typeLabel = s.type ? s.type.charAt(0).toUpperCase() + s.type.slice(1) : "";
        switch (s.type) {
          case "once":
            return `${typeLabel}: ${formatDateTime(s.config?.executeAt)}`;
          case "interval":
            return `${typeLabel}: Every ${s.config?.value} ${s.config?.unit}`;
          case "daily":
            return `${typeLabel}: ${s.config?.hour}:${String(s.config?.minute).padStart(2, "0")}`;
          default:
            return typeLabel;
        }
      },
    },
    {
      accessorKey: "topicStrategy",
      header: "Topic Strategy",
      cell: ({ row }) => {
        const v = row.original.topicStrategy;
        return <span className="font-medium">{v}</span>;
      },
    },
    {
      accessorKey: "categoryStrategy",
      header: "Category Strategy",
      cell: ({ row }) => {
        const v = row.original.categoryStrategy;
        return <span className="font-medium">{v}</span>;
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
      header: "Total Executions",
      cell: ({ row }) => {
        const total = row.original.state?.totalExecutions ?? 0;
        return <span className="font-semibold">{total}</span>;
      },
    },
    {
      id: "executionsFailed",
      header: "Failed Executions",
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

        const handleEdit = () => {
          // Placeholder: navigate to edit in future
          // For now, we do nothing, keeping minimal changes as per scope
          console.log("Edit job", job.id);
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

              <DropdownMenuItem onClick={() => handleExecute(job)}>
                <Zap className="mr-2 h-4 w-4" />
                <span>Run Now</span>
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
  ], []);

  const renderExpandedRow = (job: Job) => {
    const hasSchedule = !!job.schedule;
    return (
      <div className="text-sm space-y-2">
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          <div>
            <div className="text-muted-foreground">Requires Validation</div>
            <div className="font-medium">{job.requiresValidation ? "Yes" : "No"}</div>
          </div>
          <div>
            <div className="text-muted-foreground">Jitter</div>
            <div className="font-medium">{job.jitterEnabled ? `${job.jitterMinutes} min` : "Disabled"}</div>
          </div>
          <div>
            <div className="text-muted-foreground">IDs</div>
            <div className="font-medium">Site #{job.siteId} · Prompt #{job.promptId} · Provider #{job.aiProviderId}</div>
          </div>
          {hasSchedule && (
            <div className="col-span-2 md:col-span-1">
              <div className="text-muted-foreground">Schedule Raw</div>
              <pre className="text-xs bg-muted/50 p-2 rounded overflow-auto">{JSON.stringify(job.schedule?.config ?? {}, null, 2)}</pre>
            </div>
          )}
          <div>
            <div className="text-muted-foreground">Categories</div>
            <div className="font-medium">{job.categories?.length ?? 0}</div>
          </div>
          <div>
            <div className="text-muted-foreground">Topics</div>
            <div className="font-medium">{job.topics?.length ?? 0}</div>
          </div>
        </div>
        {job.state && (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
            <div>
              <div className="text-muted-foreground">Last Run</div>
              <div className="font-medium">{job.state.lastRunAt ? formatDateTime(job.state.lastRunAt) : "Never"}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Next Run</div>
              <div className="font-medium">{job.state.nextRunAt ? formatDateTime(job.state.nextRunAt) : "—"}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Last Category Index</div>
              <div className="font-medium">{job.state.lastCategoryIndex}</div>
            </div>
          </div>
        )}
      </div>
    );
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
