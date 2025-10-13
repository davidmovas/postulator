import {
  CountUnusedTopics,
  CreateTopic,
  DeleteTopic,
  GetAvailableTopic,
  GetTopic,
  GetTopicsBySite,
  ImportAndAssignToSite,
  ImportTopics,
  ListTopics,
  MarkTopicAsUsed,
  UpdateTopic,
} from "@/wailsjs/wailsjs/go/app/App";
import { dto } from "@/wailsjs/wailsjs/go/models";
import { unwrapOne, unwrapString, unwrapMany } from "./utils";
import type { TopicStrategy } from "@/constants/topics";

// UI types
export interface Topic {
  id: number;
  title: string;
  createdAt: string;
}

export interface ImportResult {
  totalRead: number;
  totalAdded: number;
  totalSkipped: number;
  added: string[];
  skipped: string[];
  errors: string[];
}

export function mapTopic(x: dto.Topic): Topic {
  return { id: x.id, title: x.title, createdAt: x.createdAt };
}

export function mapImportResult(x: dto.ImportResult): ImportResult {
  return {
    totalRead: x.totalRead,
    totalAdded: x.totalAdded,
    totalSkipped: x.totalSkipped,
    added: x.added,
    skipped: x.skipped,
    errors: x.errors,
  };
}

export interface PaginatedTopics {
  success: boolean;
  items: Topic[];
  total: number;
  limit: number;
  offset: number;
  hasMore: boolean;
  error?: { code: string; message: string; context?: Record<string, any> };
}

export async function listTopics(limit: number, offset: number): Promise<PaginatedTopics> {
  const res = await ListTopics(limit, offset);
  const pr = dto.PaginatedResponse__Postulator_internal_dto_Topic_.createFrom(res as any);
  return {
    success: pr.success,
    items: (pr.items || []).map(mapTopic),
    total: pr.total,
    limit: pr.limit,
    offset: pr.offset,
    hasMore: pr.hasMore,
    error: pr.error ? { code: pr.error.code, message: pr.error.message, context: pr.error.context } : undefined,
  };
}

export async function getTopic(id: number): Promise<Topic> {
  const res = await GetTopic(id);
  return mapTopic(unwrapOne<dto.Topic>(res));
}

export async function createTopic(title: string): Promise<string> {
  const payload = new dto.Topic({ title });
  const res = await CreateTopic(payload);
  return unwrapString(res);
}

export async function updateTopic(id: number, title: string): Promise<string> {
  const payload = new dto.Topic({ id, title });
  const res = await UpdateTopic(payload);
  return unwrapString(res);
}

export async function deleteTopic(id: number): Promise<string> {
  const res = await DeleteTopic(id);
  return unwrapString(res);
}

export async function getTopicsBySite(siteId: number): Promise<Topic[]> {
  const res = await GetTopicsBySite(siteId);
  return unwrapMany<dto.Topic>(res).map(mapTopic);
}

export async function getAvailableTopic(siteId: number, strategy: TopicStrategy): Promise<Topic> {
  const res = await GetAvailableTopic(siteId, strategy);
  return mapTopic(unwrapOne<dto.Topic>(res));
}

export async function importTopics(filePath: string): Promise<ImportResult> {
  const res = await ImportTopics(filePath);
  const dtoRes = unwrapOne<dto.ImportResult>(res as any); // generated type is Response__ImportResult_
  return mapImportResult(dtoRes);
}

export async function importAndAssignToSite(
    filePath: string,
    siteId: number,
    categoryID: number,
    strategy: string
): Promise<ImportResult> {
  const res = await ImportAndAssignToSite(filePath, siteId, categoryID, strategy);
  const dtoRes = unwrapOne<dto.ImportResult>(res as any);
  return mapImportResult(dtoRes);
}

export async function countUnusedTopics(siteId: number): Promise<number> {
  const res = await CountUnusedTopics(siteId);
  return unwrapOne<number>(res as any);
}

export async function markTopicAsUsed(topicId: number, siteId: number): Promise<string> {
  const res = await MarkTopicAsUsed(topicId, siteId);
  return unwrapString(res);
}
