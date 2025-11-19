"use client";

import React from "react";
import { Badge } from "@/components/ui/badge";
import type { BadgeVariant } from "@/components/ui/badge";
import { JobStatus } from "@/constants/jobs";

type Props = {
  status: JobStatus | string;
  className?: string;
};

export function JobStatusBadge({ status, className }: Props) {
  const s = status as JobStatus;
  type StatusMap = { label: string; variant: BadgeVariant };

    const map: StatusMap = (() => {
        switch (s) {
            case "active":
                return { label: "Active", variant: "success" };
            case "paused":
                return { label: "Paused", variant: "warning" };
            case "completed":
                return { label: "Completed", variant: "info" };
            case "error":
                return { label: "Error", variant: "destructive" };
            default:
                return { label: String(status), variant: "outline" };
        }
    })();

  return (
    <Badge variant={map.variant} className={`text-xs px-2 py-0.5 font-medium capitalize ${className ?? ""}`}>
      {map.label}
    </Badge>
  );
}

export default JobStatusBadge;
