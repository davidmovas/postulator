"use client";
import { dto } from "@/wailsjs/wailsjs/go/models";
import {
  GetPromptSites,
  GetSitePrompt,
  CreateSitePrompt,
  UpdateSitePrompt,
  DeleteSitePrompt,
  DeleteSitePromptBySite,
  ActivateSitePrompt,
  DeactivateSitePrompt,
} from "@/wailsjs/wailsjs/go/bindings/Binder";

interface ErrorResponse {
  code: string;
  message: string;
  details?: string;
  fields?: Record<string, string>;
  technical?: string;
}

export class StructuredError extends Error {
  code: string;
  details?: string;
  fields?: Record<string, string>;
  technical?: string;
  constructor(errorResponse: ErrorResponse) {
    super(errorResponse.message);
    this.name = "StructuredError";
    this.code = errorResponse.code;
    this.details = errorResponse.details;
    this.fields = errorResponse.fields;
    this.technical = errorResponse.technical;
  }
}

function parseError(error: unknown): Error {
  if (error instanceof Error) {
    try {
      const parsed = JSON.parse(error.message);
      if ((parsed as ErrorResponse).code && (parsed as ErrorResponse).message) {
        return new StructuredError(parsed as ErrorResponse);
      }
    } catch {}
  }
  return error instanceof Error ? error : new Error(String(error));
}

export async function getPromptSites(prompt_id: number, page: number, limit: number) {
  const req = new dto.PaginationRequest({ page, limit });
  const res = await GetPromptSites(prompt_id, req);
  return res;
}

export async function getSitePrompt(site_id: number) {
  try {
    const res = await GetSitePrompt(site_id);
    return res; // dto.SitePromptResponse
  } catch (e) {
    // backend may return error if none; treat as undefined
    return undefined;
  }
}

export async function createSitePrompt(site_id: number, prompt_id: number, is_active: boolean) {
  const req = new dto.CreateSitePromptRequest({ site_id, prompt_id, is_active });
  try {
    return await CreateSitePrompt(req);
  } catch (e) {
    throw parseError(e);
  }
}

export async function updateSitePrompt(id: number, site_id: number, prompt_id: number, is_active: boolean) {
  const req = new dto.UpdateSitePromptRequest({ id, site_id, prompt_id, is_active });
  try {
    return await UpdateSitePrompt(req);
  } catch (e) {
    throw parseError(e);
  }
}

export async function deleteSitePrompt(id: number) {
  try {
    await DeleteSitePrompt(id);
  } catch (e) {
    throw parseError(e);
  }
}

export async function deleteSitePromptBySite(site_id: number) {
  try {
    await DeleteSitePromptBySite(site_id);
  } catch (e) {
    throw parseError(e);
  }
}

export async function setSitePromptActive(id: number, active: boolean) {
  try {
    if (active) await ActivateSitePrompt(id);
    else await DeactivateSitePrompt(id);
  } catch (e) {
    throw parseError(e);
  }
}
