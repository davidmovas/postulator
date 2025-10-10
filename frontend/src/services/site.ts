import {
    AssignTopicToSite, CheckHealth,
    CreateSite,
    DeleteSite,
    GetSite,
    GetSiteCategories,
    GetSiteTopics,
    GetTopicsBySite,
    ListSites,
    SetSitePassword,
    SyncCategories,
    UnassignTopicFromSite,
    UpdateSite,
} from "@/wailsjs/wailsjs/go/app/App";
import { dto } from "@/wailsjs/wailsjs/go/models";
import { unwrapMany, unwrapString } from "./utils";
import { Topic } from "@/services/topic";
import { unwrapArrayResponse, unwrapResponse } from "@/lib/error-handling";

export interface Site {
  id: number;
  name: string;
  url: string;
  wpUsername: string;
  status: string;
  lastHealthCheck?: string;
  healthStatus: string;
  createdAt: string;
  updatedAt: string;
}

export interface Category {
  id: number;
  siteId: number;
  wpCategoryId: number;
  name: string;
  slug?: string;
  count: number;
  createdAt: string;
}

export interface SiteTopic {
  id: number;
  siteId: number;
  topicId: number;
  categoryId: number;
  strategy: string;
  createdAt: string;
}

// Input types for create/update
export type SiteCreateInput = Pick<Site, "name" | "url" | "wpUsername">;
export type SiteUpdateInput = Partial<Pick<Site, "name" | "url" | "wpUsername" | "status">> & { id: number };


function mapSite(x: dto.Site): Site {
  return {
    id: x.id,
    name: x.name,
    url: x.url,
    wpUsername: x.wpUsername,
    status: x.status,
    lastHealthCheck: x.lastHealthCheck,
    healthStatus: x.healthStatus,
    createdAt: x.createdAt,
    updatedAt: x.updatedAt,
  };
}

function mapCategory(x: dto.Category): Category {
  return {
    id: x.id,
    siteId: x.siteId,
    wpCategoryId: x.wpCategoryId,
    name: x.name,
    slug: x.slug,
    count: x.count,
    createdAt: x.createdAt,
  };
}

function mapTopic(x: dto.Topic): Topic {
  return {
    id: x.id,
    title: x.title,
    createdAt: x.createdAt,
  };
}

function mapSiteTopic(x: dto.SiteTopic): SiteTopic {
  return {
    id: x.id,
    siteId: x.siteId,
    topicId: x.topicId,
    categoryId: x.categoryId,
    strategy: x.strategy,
    createdAt: x.createdAt,
  };
}

export async function listSites(): Promise<Site[]> {
    const response = await ListSites();
    const sites = unwrapArrayResponse<dto.Site>(response);
    return sites.map(mapSite);
}

export async function getSite(id: number): Promise<Site> {
    const response = await GetSite(id);
    const site = unwrapResponse<dto.Site>(response);
    return mapSite(site);
}

export async function createSite(input: SiteCreateInput): Promise<void> {
    const payload = new dto.Site({
        name: input.name,
        url: input.url,
        wpUsername: input.wpUsername,
    });
    const response = await CreateSite(payload);
    unwrapResponse<string>(response);
}

export async function updateSite(input: SiteUpdateInput): Promise<void> {
    const payload = new dto.Site({
        id: input.id,
        name: input.name,
        url: input.url,
        wpUsername: input.wpUsername,
        status: input.status,
    });
    const response = await UpdateSite(payload);
    unwrapResponse<string>(response);
}

export async function deleteSite(id: number): Promise<void> {
    const response = await DeleteSite(id);
    unwrapResponse<string>(response);
}

export async function setSitePassword(siteId: number, password: string): Promise<void> {
    const response = await SetSitePassword(siteId, password);
    unwrapResponse<string>(response);
}

export async function syncCategories(siteId: number): Promise<string> {
  const res = await SyncCategories(siteId);
  return unwrapString(res);
}

export async function getSiteCategories(siteId: number): Promise<Category[]> {
  const res = await GetSiteCategories(siteId);
  return unwrapMany<dto.Category>(res).map(mapCategory);
}

export async function getSiteTopics(siteId: number): Promise<SiteTopic[]> {
  const res = await GetSiteTopics(siteId);
  return unwrapMany<dto.SiteTopic>(res).map(mapSiteTopic);
}

export async function getTopicsBySite(siteId: number): Promise<Topic[]> {
  const res = await GetTopicsBySite(siteId);
  return unwrapMany<dto.Topic>(res).map(mapTopic);
}

export async function checkSiteHealth(siteId: number): Promise<string> {
    const res = await CheckHealth(siteId);
    return unwrapString(res);
}

export async function assignTopicToSite(
  siteId: number,
  topicId: number,
  priority: number,
  note: string
): Promise<string> {
  const res = await AssignTopicToSite(siteId, topicId, priority, note);
  return unwrapString(res);
}

export async function unassignTopicFromSite(siteId: number, topicId: number): Promise<string> {
  const res = await UnassignTopicFromSite(siteId, topicId);
  return unwrapString(res);
}
