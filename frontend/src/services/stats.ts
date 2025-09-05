"use client";
import { dto } from "@/wailsjs/wailsjs/go/models";
import {
  GetTopicStats,
  GetSiteUsageHistory,
  GetTopicUsageHistory,
  CheckStrategyAvailability,
  SelectTopicForSite,
} from "@/wailsjs/wailsjs/go/bindings/Binder";

export async function getTopicStats(site_id: number) {
  const res = await GetTopicStats(site_id);
  return res;
}

export async function getSiteUsageHistory(site_id: number, page: number, limit: number) {
  const req = new dto.PaginationRequest({ page, limit });
  const res = await GetSiteUsageHistory(site_id, req);
  return res;
}

export async function getTopicUsageHistory(site_id: number, topic_id: number, page: number, limit: number) {
  const req = new dto.PaginationRequest({ page, limit });
  const res = await GetTopicUsageHistory(site_id, topic_id, req);
  return res;
}

export async function checkStrategyAvailability(site_id: number, strategy: string) {
  const res = await CheckStrategyAvailability(site_id, strategy);
  return res;
}

export async function selectNextTopicForSite(site_id: number, strategy?: string) {
  const req = new dto.TopicSelectionRequest({ site_id, strategy });
  const res = await SelectTopicForSite(req);
  return res;
}
