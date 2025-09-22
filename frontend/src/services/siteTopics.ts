"use client";
import { dto } from "@/wailsjs/wailsjs/go/models";
import {
  GetSiteTopics,
  CreateSiteTopic,
  UpdateSiteTopic,
  DeleteSiteTopic,
  DeleteSiteTopicBySiteAndTopic,
} from "@/wailsjs/wailsjs/go/bindings/Binder";

// Minimal types for site-topic row
export interface SiteTopicLink {
  id: number;
  site_id: number;
  site_name?: string;
  topic_id: number;
  topic_title?: string;
  priority: number;
  usage_count: number;
  last_used_at?: string;
  round_robin_pos: number;
}

function mapLink(r: dto.SiteTopicResponse): SiteTopicLink {
  return {
    id: r.id,
    site_id: r.site_id,
    site_name: r.site_name,
    topic_id: r.topic_id,
    topic_title: r.topic_title,
    priority: r.priority,
    usage_count: r.usage_count,
    last_used_at: typeof r.last_used_at === "string" ? (r.last_used_at as unknown as string) : undefined,
    round_robin_pos: r.round_robin_pos,
  };
}

export async function getSiteTopics(siteId: number, page: number, limit: number) {
  const req = new dto.PaginationRequest({ page, limit });
  const res = await GetSiteTopics(siteId, req);
  return {
    items: (res.site_topics ?? []).map(mapLink),
    pagination: res.pagination,
  };
}

export async function createSiteTopic(values: { site_id: number; topic_id: number; priority: number; }) {
  const req = new dto.CreateSiteTopicRequest(values);
  const res = await CreateSiteTopic(req);
  return mapLink(res);
}

export async function updateSiteTopic(values: { id: number; site_id: number; topic_id: number; priority: number; }) {
  const req = new dto.UpdateSiteTopicRequest(values);
  const res = await UpdateSiteTopic(req);
  return mapLink(res);
}

export async function deleteSiteTopic(id: number) {
  await DeleteSiteTopic(id);
}

export async function deleteSiteTopicBySiteAndTopic(site_id: number, topic_id: number) {
  await DeleteSiteTopicBySiteAndTopic(site_id, topic_id);
}

