"use client";
import * as React from "react";
import { getDashboard } from "@/services/dashboard";
import type { DashboardData } from "@/types/dashboard";
import { ClientTime } from "@/components/common/ClientTime";

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border p-4 bg-background/50">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-2xl font-semibold tabular-nums">{value}</div>
    </div>
  );
}

export function DashboardOverview() {
  const [data, setData] = React.useState<DashboardData>({
    totals: { sites: 0, sites_active: 0, topics: 0, articles: 0, jobs_running: 0, jobs_pending: 0 },
    last_run_at: undefined,
  });
  const [loading, setLoading] = React.useState<boolean>(true);

  const refresh = React.useCallback(async () => {
    setLoading(true);
    try {
      const d = await getDashboard();
      setData(d);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    // Initial load
    void refresh();
  }, [refresh]);

  const { totals, last_run_at } = data;

  return (
    <div className="p-4 md:p-6 lg:p-8">
      <div className="mb-4">
        <h2 className="mt-1 text-2xl font-semibold tracking-tight">Dashboard</h2>
        <p className="mt-2 text-muted-foreground">Overview of your content system.</p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-3">
        <StatCard label="Sites" value={totals.sites} />
        <StatCard label="Active Sites" value={totals.sites_active} />
        <StatCard label="Topics" value={totals.topics} />
        <StatCard label="Articles" value={totals.articles} />
        <StatCard label="Running Jobs" value={totals.jobs_running} />
        <StatCard label="Pending Jobs" value={totals.jobs_pending} />
      </div>

      <div className="mt-6 text-sm text-muted-foreground">
        Last scheduler run: {last_run_at ? (
          <>
            <time suppressHydrationWarning dateTime={last_run_at} className="sr-only">{last_run_at}</time>
            <ClientTime iso={last_run_at} />
          </>
        ) : "—"}
        {loading && <span className="ml-2 inline-flex items-center gap-2"><span className="size-2 animate-pulse rounded-full bg-foreground/40"/> Loading…</span>}
      </div>
    </div>
  );
}

export default DashboardOverview;
