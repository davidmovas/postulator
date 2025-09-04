export interface DashboardTotals {
  sites: number;
  sites_active: number;
  topics: number;
  articles: number;
  jobs_running: number;
  jobs_pending: number;
}

export interface DashboardData {
  totals: DashboardTotals;
  last_run_at?: string; // ISO string of last scheduler run
}
