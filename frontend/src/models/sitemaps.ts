import { dto } from "@/wailsjs/wailsjs/go/models";

// =========================================================================
// Sitemap Types
// =========================================================================

export type SitemapSource = "manual" | "imported" | "generated" | "scanned";
export type SitemapStatus = "draft" | "active" | "archived";
export type NodeSource = "manual" | "imported" | "generated" | "scanned";
export type NodeContentType = "page" | "post" | "none";
export type NodeContentStatus = "none" | "pending" | "draft" | "published";

export interface Sitemap {
    id: number;
    siteId: number;
    name: string;
    description?: string;
    source: SitemapSource;
    status: SitemapStatus;
    createdAt: string;
    updatedAt: string;
}

export interface SitemapNode {
    id: number;
    sitemapId: number;
    parentId?: number;
    title: string;
    slug: string;
    description?: string;
    isRoot: boolean;
    depth: number;
    position: number;
    path: string;
    contentType: NodeContentType;
    articleId?: number;
    wpPageId?: number;
    wpUrl?: string;
    source: NodeSource;
    isSynced: boolean;
    lastSyncedAt?: string;
    contentStatus: NodeContentStatus;
    positionX?: number;
    positionY?: number;
    keywords: string[];
    children?: SitemapNode[];
    createdAt: string;
    updatedAt: string;
}

export interface SitemapWithNodes {
    sitemap: Sitemap;
    nodes: SitemapNode[];
}

// =========================================================================
// Input Types
// =========================================================================

export interface CreateSitemapInput {
    siteId: number;
    name: string;
    description?: string;
    source: SitemapSource;
    siteUrl: string;
}

export interface UpdateSitemapInput {
    id: number;
    name: string;
    description?: string;
    status: SitemapStatus;
}

export interface CreateNodeInput {
    sitemapId: number;
    parentId?: number;
    title: string;
    slug: string;
    description?: string;
    position: number;
    source: NodeSource;
    keywords?: string[];
}

export interface UpdateNodeInput {
    id: number;
    title: string;
    slug: string;
    description?: string;
    keywords?: string[];
}

export interface MoveNodeInput {
    nodeId: number;
    newParentId?: number;
    position: number;
}

export interface UpdateNodePositionsInput {
    nodeId: number;
    positionX: number;
    positionY: number;
}

// =========================================================================
// Mappers
// =========================================================================

export function mapSitemap(x: dto.Sitemap): Sitemap {
    return {
        id: x.id,
        siteId: x.siteId,
        name: x.name,
        description: x.description || undefined,
        source: x.source as SitemapSource,
        status: x.status as SitemapStatus,
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}

export function mapSitemapNode(x: dto.SitemapNode): SitemapNode {
    return {
        id: x.id,
        sitemapId: x.sitemapId,
        parentId: x.parentId || undefined,
        title: x.title,
        slug: x.slug,
        description: x.description || undefined,
        isRoot: x.isRoot || false,
        depth: x.depth,
        position: x.position,
        path: x.path,
        contentType: x.contentType as NodeContentType,
        articleId: x.articleId || undefined,
        wpPageId: x.wpPageId || undefined,
        wpUrl: x.wpUrl || undefined,
        source: x.source as NodeSource,
        isSynced: x.isSynced,
        lastSyncedAt: x.lastSyncedAt || undefined,
        contentStatus: x.contentStatus as NodeContentStatus,
        positionX: x.positionX || undefined,
        positionY: x.positionY || undefined,
        keywords: x.keywords || [],
        children: x.children?.map(mapSitemapNode) || undefined,
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}

export function mapSitemapWithNodes(x: dto.SitemapWithNodes): SitemapWithNodes {
    return {
        sitemap: mapSitemap(x.sitemap!),
        nodes: (x.nodes || []).map(mapSitemapNode),
    };
}
