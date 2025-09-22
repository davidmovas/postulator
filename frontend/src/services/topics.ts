"use client";
import type { Topic } from "@/types/topic";
import { dto } from "@/wailsjs/wailsjs/go/models";
import {
  GetTopics,
  GetTopic,
  CreateTopic,
  UpdateTopic,
  DeleteTopic,
  GetTopicSites,
} from "@/wailsjs/wailsjs/go/bindings/Binder";

// Error helpers (reuse pattern from services/sites.ts)
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

function mapTopic(r: dto.TopicResponse): Topic {
  return {
    id: r.id,
    title: r.title,
    keywords: r.keywords,
    category: r.category,
    tags: r.tags,
    created_at: typeof r.created_at === "string" ? (r.created_at as unknown as string) : undefined,
    updated_at: typeof r.updated_at === "string" ? (r.updated_at as unknown as string) : undefined,
  };
}

export type TopicsPage = {
  items: Topic[];
  page: number;
  limit: number;
  total: number;
  total_pages: number;
};

export async function getTopics(page: number, limit: number): Promise<TopicsPage> {
  const req = new dto.PaginationRequest({ page, limit });
  const res = await GetTopics(req);
  const items = (res.topics ?? []).map(mapTopic);
  const p = res.pagination;
  return {
    items,
    page: p?.page ?? page,
    limit: p?.limit ?? limit,
    total: p?.total ?? items.length,
    total_pages: p?.total_pages ?? 1,
  };
}

export type UpsertTopicValues = {
  title: string;
  keywords?: string;
  category?: string;
  tags?: string;
};

export async function createTopic(values: UpsertTopicValues): Promise<Topic> {
  const req = new dto.CreateTopicRequest({
    title: values.title,
    keywords: values.keywords,
    category: values.category,
    tags: values.tags,
  });
  try {
    const res = await CreateTopic(req);
    return mapTopic(res);
  } catch (e) {
    throw parseError(e);
  }
}

export async function updateTopic(id: number, values: UpsertTopicValues): Promise<Topic> {
  const req = new dto.UpdateTopicRequest({
    id,
    title: values.title,
    keywords: values.keywords,
    category: values.category,
    tags: values.tags,
  });
  try {
    const res = await UpdateTopic(req);
    return mapTopic(res);
  } catch (e) {
    throw parseError(e);
  }
}

export async function deleteTopic(id: number): Promise<void> {
  try {
    await DeleteTopic(id);
  } catch (e) {
    throw parseError(e);
  }
}


// Extra helper for future: get sites by topic (for Topic -> Sites tab)
export async function getTopicSites(topicId: number, page: number, limit: number) {
  const req = new dto.PaginationRequest({ page, limit });
  const res = await GetTopicSites(topicId, req);
  return res;
}
