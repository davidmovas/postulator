"use client";
import type { Site, SiteStatus } from "@/types/site";
import { dto } from "@/wailsjs/wailsjs/go/models";
import {
  GetSites,
  CreateSite,
  UpdateSite,
  DeleteSite,
  ActivateSite,
  DeactivateSite,
  TestSiteConnection,
} from "@/wailsjs/wailsjs/go/bindings/Binder";

// Error response structure matching backend dto.ErrorResponse
interface ErrorResponse {
  code: string;
  message: string;
  details?: string;
  fields?: Record<string, string>;
  technical?: string;
}

// Custom error class for structured errors
export class StructuredError extends Error {
  code: string;
  details?: string;
  fields?: Record<string, string>;
  technical?: string;

  constructor(errorResponse: ErrorResponse) {
    super(errorResponse.message);
    this.name = 'StructuredError';
    this.code = errorResponse.code;
    this.details = errorResponse.details;
    this.fields = errorResponse.fields;
    this.technical = errorResponse.technical;
  }
}

// Parse error from backend - try to extract JSON error structure
function parseError(error: unknown): Error {
  if (error instanceof Error) {
    try {
      // Try to parse error message as JSON
      const parsed = JSON.parse(error.message);
      if (parsed.code && parsed.message) {
        return new StructuredError(parsed as ErrorResponse);
      }
    } catch {
      // If not JSON, check if message contains user-friendly error text
      if (error.message.includes('already exists') || 
          error.message.includes('not found') ||
          error.message.includes('Failed to') ||
          error.message.includes('Invalid')) {
        // It's already a user-friendly message
        return new Error(error.message);
      }
    }
  }
  
  // Fallback to original error
  return error instanceof Error ? error : new Error(String(error));
}

export type SitesPage = {
  items: Site[];
  page: number;
  limit: number;
  total: number;
  total_pages: number;
};

function toStatus(v: unknown): SiteStatus {
  const s = typeof v === "string" ? (v as string) : "";
  if (s === "connected" || s === "error" || s === "pending" || s === "disabled") return s;
  // Fallback mapping from possible backend values
  const low = s.toLowerCase();
  if (low.includes("connect")) return "connected";
  if (low.includes("error") || low.includes("fail")) return "error";
  if (low.includes("pending") || low.includes("checking") || low.includes("unknown")) return "pending";
  return "disabled";
}

function mapSite(r: dto.SiteResponse): Site {
  return {
    id: r.id,
    name: r.name,
    url: r.url,
    is_active: r.is_active,
    status: toStatus(r.status),
    last_check_at: typeof r.last_check === "string" ? (r.last_check as unknown as string) : undefined,
    username: r.username,
    password: r.password,
    strategy: (r.strategy as unknown as Site["strategy"]) ?? "round_robin",
  };
}

export async function getSites(page: number, limit: number): Promise<SitesPage> {
  const req = new dto.PaginationRequest({ page, limit });
  const res = await GetSites(req);
  const items = (res.sites ?? []).map(mapSite);
  const p = res.pagination;
  return {
    items,
    page: p?.page ?? page,
    limit: p?.limit ?? limit,
    total: p?.total ?? items.length,
    total_pages: p?.total_pages ?? 1,
  };
}

export type UpsertSiteValues = {
  name: string;
  url: string;
  username: string;
  password: string;
  is_active: boolean;
  strategy?: string;
};

export async function createSite(values: UpsertSiteValues): Promise<Site> {
  const req = new dto.CreateSiteRequest({
    name: values.name,
    url: values.url,
    username: values.username,
    password: values.password,
    is_active: values.is_active,
    strategy: values.strategy,
  });
  
  try {
    const res = await CreateSite(req);
    return mapSite(res);
  } catch (error) {
    throw parseError(error);
  }
}

export async function updateSite(id: number, values: UpsertSiteValues): Promise<Site> {
  const req = new dto.UpdateSiteRequest({
    id,
    name: values.name,
    url: values.url,
    username: values.username,
    password: values.password,
    is_active: values.is_active,
    strategy: values.strategy,
  });
  
  try {
    const res = await UpdateSite(req);
    return mapSite(res);
  } catch (error) {
    throw parseError(error);
  }
}

export async function deleteSite(id: number): Promise<void> {
  try {
    await DeleteSite(id);
  } catch (error) {
    throw parseError(error);
  }
}

export async function setSiteActive(id: number, active: boolean): Promise<void> {
  try {
    if (active) await ActivateSite(id);
    else await DeactivateSite(id);
  } catch (error) {
    throw parseError(error);
  }
}

export type TestResult = {
  success: boolean;
  status: string;
  message: string;
  details?: string;
  timestamp?: string;
};

export async function testConnection(id: number): Promise<TestResult> {
  const req = new dto.TestSiteConnectionRequest({ site_id: id });
  const res = await TestSiteConnection(req);
  return {
    success: res.success,
    status: res.status,
    message: res.message,
    details: res.details,
    timestamp: typeof res.timestamp === "string" ? (res.timestamp as unknown as string) : undefined,
  };
}
