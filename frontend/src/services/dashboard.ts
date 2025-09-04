"use client";
import type { DashboardData, DashboardTotals } from "@/types/dashboard";
import { GetDashboard } from "@/wailsjs/wailsjs/go/main/App";

// Local copy of base response to keep UI strictly typed
type BaseResponse = {
  success: boolean;
  message?: string;
  error?: string;
  data?: unknown;
};

function toNumber(v: unknown, fallback = 0): number {
  return typeof v === "number" && Number.isFinite(v) ? v : fallback;
}

function parseDashboardData(data: unknown): DashboardData {
  const obj = (data && typeof data === "object") ? (data as Record<string, unknown>) : {};
  const totalsObj = (obj["totals"] && typeof obj["totals"] === "object") ? obj["totals"] as Record<string, unknown> : {};
  const totals: DashboardTotals = {
    sites: toNumber(totalsObj["sites"]),
    sites_active: toNumber(totalsObj["sites_active"]),
    topics: toNumber(totalsObj["topics"]),
    articles: toNumber(totalsObj["articles"]),
    jobs_running: toNumber(totalsObj["jobs_running"]),
    jobs_pending: toNumber(totalsObj["jobs_pending"]),
  };
  const last_run_at = typeof obj["last_run_at"] === "string" ? obj["last_run_at"] as string : undefined;
  return { totals, last_run_at };
}

export async function getDashboard(): Promise<DashboardData> {
  const res = (await GetDashboard()) as BaseResponse;
  if (!res?.success) {
    // Return safe defaults on failure
    return { totals: { sites: 0, sites_active: 0, topics: 0, articles: 0, jobs_running: 0, jobs_pending: 0 }, last_run_at: undefined };
  }
  return parseDashboardData(res.data);
}
