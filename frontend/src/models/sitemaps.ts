import { dto } from "@/wailsjs/wailsjs/go/models";

// =========================================================================
// Sitemap Types
// =========================================================================

export type SitemapSource = "manual" | "imported" | "generated" | "scanned";
export type SitemapStatus = "draft" | "active" | "archived";
export type NodeSource = "manual" | "imported" | "generated" | "scanned";
export type NodeContentType = "page" | "post" | "none";
export type NodeContentStatus = "none" | "ai_draft" | "pending" | "draft" | "published";

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
    wpTitle?: string;
    wpSlug?: string;
    isModified: boolean;
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

export interface ImportNodesInput {
    sitemapId: number;
    parentNodeId?: number;
    filename: string;
    fileDataBase64: string;
}

export interface ImportError {
    row?: number;
    column?: string;
    message: string;
}

export interface ImportNodesResult {
    totalRows: number;
    nodesCreated: number;
    nodesSkipped: number;
    errors: ImportError[];
    processingTime: string;
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
        wpTitle: x.wpTitle || undefined,
        wpSlug: x.wpSlug || undefined,
        isModified: x.isModified || false,
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

export function mapImportNodesResult(x: dto.ImportNodesResponse): ImportNodesResult {
    return {
        totalRows: x.totalRows,
        nodesCreated: x.nodesCreated,
        nodesSkipped: x.nodesSkipped,
        errors: (x.errors || []).map((e) => ({
            row: e.row,
            column: e.column,
            message: e.message,
        })),
        processingTime: x.processingTime,
    };
}

// =========================================================================
// Scanner Types
// =========================================================================

export type TitleSource = "title" | "h1";
export type ContentFilter = "all" | "pages" | "posts";

export interface ScanSiteInput {
    siteId: number;
    sitemapName: string;
    titleSource: TitleSource;
    contentFilter: ContentFilter;
    includeDrafts: boolean;
    maxDepth: number;
}

export interface ScanIntoSitemapInput {
    sitemapId: number;
    parentNodeId?: number;
    titleSource: TitleSource;
    contentFilter: ContentFilter;
    includeDrafts: boolean;
    maxDepth: number;
}

export interface ScanError {
    wpId?: number;
    type?: string;
    title?: string;
    message: string;
}

export interface ScanSiteResult {
    sitemapId: number;
    pagesScanned: number;
    postsScanned: number;
    nodesCreated: number;
    nodesSkipped: number;
    totalDuration: string;
    errors: ScanError[];
}

export function mapScanSiteResult(x: dto.ScanSiteResponse): ScanSiteResult {
    return {
        sitemapId: x.sitemapId,
        pagesScanned: x.pagesScanned,
        postsScanned: x.postsScanned,
        nodesCreated: x.nodesCreated,
        nodesSkipped: x.nodesSkipped,
        totalDuration: x.totalDuration,
        errors: (x.errors || []).map((e) => ({
            wpId: e.wpId,
            type: e.type,
            title: e.title,
            message: e.message,
        })),
    };
}

// =========================================================================
// Sync Types
// =========================================================================

export interface SyncNodesInput {
    siteId: number;
    nodeIds: number[];
}

export interface SyncNodeResult {
    nodeId: number;
    success: boolean;
    error?: string;
}

export interface SyncNodesResult {
    results: SyncNodeResult[];
}

export function mapSyncNodesResult(x: dto.SyncNodesResponse): SyncNodesResult {
    return {
        results: (x.results || []).map((r) => ({
            nodeId: r.nodeId,
            success: r.success,
            error: r.error || undefined,
        })),
    };
}

// =========================================================================
// AI Generation Types
// =========================================================================

export interface TitleInput {
    title: string;
    keywords?: string[];
}

export interface GenerateSitemapStructureInput {
    sitemapId?: number;
    siteId?: number;
    name?: string;
    promptId: number;
    placeholders?: Record<string, string>;
    titles: TitleInput[];
    parentNodeIds?: number[];
    maxDepth?: number;
    includeExistingTree?: boolean;
    providerId: number;
}

export interface GenerateSitemapStructureResult {
    sitemapId: number;
    nodesCreated: number;
    durationMs: number;
}

export function mapGenerateSitemapStructureResult(
    x: dto.GenerateSitemapStructureResponse
): GenerateSitemapStructureResult {
    return {
        sitemapId: x.sitemapId,
        nodesCreated: x.nodesCreated,
        durationMs: x.durationMs,
    };
}
