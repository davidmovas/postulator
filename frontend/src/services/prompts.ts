"use client";
import type { Prompt } from "@/types/prompt";
import { dto } from "@/wailsjs/wailsjs/go/models";
import {
  GetPrompts,
  GetDefaultPrompt,
  GetPrompt,
  CreatePrompt,
  UpdatePrompt,
  DeletePrompt,
  SetDefaultPrompt,
  GetPromptSites,
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

function mapPrompt(r: dto.PromptResponse): Prompt {
  return {
    id: r.id,
    name: r.name,
    system: r.system,
    user: r.user,
    is_default: r.is_default,
    is_active: r.is_active,
    created_at: typeof r.created_at === "string" ? (r.created_at as unknown as string) : undefined,
    updated_at: typeof r.updated_at === "string" ? (r.updated_at as unknown as string) : undefined,
  };
}

export type PromptsPage = {
  items: Prompt[];
  page: number;
  limit: number;
  total: number;
  total_pages: number;
};

export async function getPrompts(page: number, limit: number): Promise<PromptsPage> {
  const req = new dto.PaginationRequest({ page, limit });
  const res = await GetPrompts(req);
  const items = (res.prompts ?? []).map(mapPrompt);
  const p = res.pagination;
  return {
    items,
    page: p?.page ?? page,
    limit: p?.limit ?? limit,
    total: p?.total ?? items.length,
    total_pages: p?.total_pages ?? 1,
  };
}

export async function getDefaultPrompt(): Promise<Prompt | undefined> {
  try {
    const res = await GetDefaultPrompt();
    if (!res || typeof res.id !== "number") return undefined;
    return mapPrompt(res);
  } catch (e) {
    // If backend returns not found, silently ignore
    return undefined;
  }
}

export type CreatePromptValues = {
  name: string;
  system: string;
  user: string;
  is_default: boolean;
  is_active: boolean;
};

export async function createPrompt(values: CreatePromptValues): Promise<Prompt> {
  const req = new dto.CreatePromptRequest({
    name: values.name,
    system: values.system,
    user: values.user,
    is_default: values.is_default,
    is_active: values.is_active,
  });
  try {
    const res = await CreatePrompt(req);
    return mapPrompt(res);
  } catch (e) {
    throw parseError(e);
  }
}

export type UpdatePromptValues = {
  id: number;
  name: string;
  // Provide both fields; backend allows updating one at a time via Type
  system: string;
  user: string;
  is_default: boolean;
  is_active: boolean;
};

export async function updatePrompt(values: UpdatePromptValues) {
  // Update name/flags using one of the updates; then update both contents
  try {
    // 1. Update system
    let req = new dto.UpdatePromptRequest({
      id: values.id,
      name: values.name,
      type: "system",
      content: values.system,
      is_default: values.is_default,
      is_active: values.is_active,
    });
    await UpdatePrompt(req);
    // 2. Update user
    req = new dto.UpdatePromptRequest({
      id: values.id,
      name: values.name,
      type: "user",
      content: values.user,
      is_default: values.is_default,
      is_active: values.is_active,
    });
    const res = await UpdatePrompt(req);
    return mapPrompt(res);
  } catch (e) {
    throw parseError(e);
  }
}

export async function deletePrompt(id: number): Promise<void> {
  try {
    await DeletePrompt(id);
  } catch (e) {
    throw parseError(e);
  }
}

export async function makeDefaultPrompt(id: number): Promise<void> {
  // Backend validates type: oneof "system"|"user"; we use "system" as default scope
  const req = new dto.SetDefaultPromptRequest({ id, type: "system" });
  try {
    await SetDefaultPrompt(req);
  } catch (e) {
    throw parseError(e);
  }
}

export async function countSitesUsingPrompt(promptId: number): Promise<number> {
  const req = new dto.PaginationRequest({ page: 1, limit: 1 });
  const res = await GetPromptSites(promptId, req);
  const total = res?.pagination?.total ?? (res?.site_prompts?.length ?? 0);
  return total;
}
