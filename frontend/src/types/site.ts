export type SiteStatus = "connected" | "error" | "pending" | "disabled";
export type Strategy = "unique" | "round_robin" | "random";

export interface Site {
  id: number;
  name: string;
  url: string;
  is_active: boolean;
  status: SiteStatus;
  last_check_at?: string; // ISO string
  username?: string;
  password?: string;
  strategy: Strategy;
}
