"use client";
import * as React from "react";
import { cn } from "@/lib/utils";
import type { SiteStatus } from "@/types/site";

export interface SiteStatusBadgeProps {
  status: SiteStatus;
  className?: string;
}

export function SiteStatusBadge({ status, className }: SiteStatusBadgeProps) {
  const color =
    status === "connected"
      ? "bg-emerald-500/15 text-emerald-400 border-emerald-500/30"
      : status === "error"
      ? "bg-rose-500/15 text-rose-400 border-rose-500/30"
      : status === "pending"
      ? "bg-amber-500/15 text-amber-400 border-amber-500/30"
      : "bg-zinc-500/15 text-zinc-400 border-zinc-500/30";

  const label =
    status === "connected"
      ? "Connected"
      : status === "error"
      ? "Error"
      : status === "pending"
      ? "Pending"
      : "Disabled";

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-medium",
        color,
        className,
      )}
    >
      <span className="size-1.5 rounded-full bg-current opacity-80" />
      {label}
    </span>
  );
}

export default SiteStatusBadge;
