"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal, Pencil, Play, Pause, Trash2 } from "lucide-react";
import { Job } from "@/services/job";

export interface JobRowActionsProps {
  job: Job;
  disabled?: boolean;
  onEdit: (job: Job) => void;
  onRun: (jobId: number) => void | Promise<void>;
  onPause: (jobId: number) => void | Promise<void>;
  onResume: (jobId: number) => void | Promise<void>;
  onRequestDelete: (jobId: number) => void;
}

export function JobRowActions({
  job,
  disabled = false,
  onEdit,
  onRun,
  onPause,
  onResume,
  onRequestDelete,
}: JobRowActionsProps) {
  const isPaused = job.status?.toLowerCase() === "paused";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" disabled={disabled}>
          <MoreHorizontal className="h-4 w-4" />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-44">
        <DropdownMenuLabel>Actions</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => onEdit(job)}>
          <Pencil className="h-4 w-4 mr-2" /> Edit
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onRun(job.id)}>
          <Play className="h-4 w-4 mr-2" /> Run now
        </DropdownMenuItem>
        {isPaused ? (
          <DropdownMenuItem onClick={() => onResume(job.id)}>
            <Play className="h-4 w-4 mr-2" /> Resume
          </DropdownMenuItem>
        ) : (
          <DropdownMenuItem onClick={() => onPause(job.id)}>
            <Pause className="h-4 w-4 mr-2" /> Pause
          </DropdownMenuItem>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={() => onRequestDelete(job.id)}>
          <Trash2 className="h-4 w-4 mr-2" /> Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export default JobRowActions;
