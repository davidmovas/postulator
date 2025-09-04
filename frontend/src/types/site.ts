export type SiteStatus = "connected" | "error" | "pending" | "disabled";

export interface Site {
  id: number;
  name: string;
  url: string;
  is_active: boolean;
  status: SiteStatus;
  last_check_at?: string; // ISO string
  username?: string;
  password?: string;
  api_key?: string;
}
