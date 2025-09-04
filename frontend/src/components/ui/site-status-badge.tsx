"use client";
import { Badge } from "@/components/ui/badge";
import type { SiteStatus } from "@/types/site";

export function SiteStatusBadge({ status }: { status: SiteStatus }) {
  const label =
    status === "connected"
      ? "Connected"
      : status === "error"
      ? "Error"
      : status === "pending"
      ? "Pending"
      : "Disabled";

  const variant: "default" | "secondary" | "destructive" | "outline" =
    status === "connected"
      ? "default"
      : status === "error"
      ? "destructive"
      : status === "pending"
      ? "secondary"
      : "outline";

  return <Badge variant={variant}>{label}</Badge>;
}

export default SiteStatusBadge;
