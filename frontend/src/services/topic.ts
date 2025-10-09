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
import { unwrapMany, unwrapOne, unwrapString } from "./utils";

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

export async function listTopics(): Promise<Topic[]> {
  const res = await ListTopics();
  return unwrapMany<dto.Topic>(res).map(mapTopic);
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

export async function getAvailableTopic(siteId: number, search: string): Promise<Topic> {
  const res = await GetAvailableTopic(siteId, search);
  return mapTopic(unwrapOne<dto.Topic>(res));
}

export async function importTopics(csvContent: string): Promise<ImportResult> {
  const res = await ImportTopics(csvContent);
  const dtoRes = unwrapOne<dto.ImportResult>(res as any); // generated type is Response__ImportResult_
  return mapImportResult(dtoRes);
}

export async function importAndAssignToSite(
  csvContent: string,
  siteId: number,
  priority: number,
  note: string
): Promise<ImportResult> {
  const res = await ImportAndAssignToSite(csvContent, siteId, priority, note);
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
