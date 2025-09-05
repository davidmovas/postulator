"use client";
import type { DashboardData } from "@/types/dashboard";

// Temporary safe implementation until backend dashboard endpoint is available via Wails bindings.
export async function getDashboard(): Promise<DashboardData> {
  return {
    totals: { sites: 0, sites_active: 0, topics: 0, articles: 0, jobs_running: 0, jobs_pending: 0 },
    last_run_at: undefined,
  };
}
