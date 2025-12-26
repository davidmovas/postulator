import { dto } from "@/wailsjs/wailsjs/go/models";

// =========================================================================
// Sitemap Types
// =========================================================================

export type SitemapSource = "manual" | "imported" | "generated" | "scanned";
export type SitemapStatus = "draft" | "active" | "archived";
export type NodeSource = "manual" | "imported" | "generated" | "scanned";
export type NodeContentType = "page" | "post" | "none";

export type NodeDesignStatus = "draft" | "ready" | "approved";
export type NodeGenerationStatus = "none" | "queued" | "generating" | "generated" | "failed";
export type NodePublishStatus = "none" | "publishing" | "draft" | "pending" | "published" | "failed";

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
    designStatus: NodeDesignStatus;
    generationStatus: NodeGenerationStatus;
    publishStatus: NodePublishStatus;
    isModifiedLocally: boolean;
    lastError?: string;
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
    position?: number;
    source?: NodeSource;
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
    position?: number;
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
        designStatus: x.designStatus as NodeDesignStatus,
        generationStatus: x.generationStatus as NodeGenerationStatus,
        publishStatus: x.publishStatus as NodePublishStatus,
        isModifiedLocally: x.isModifiedLocally || false,
        lastError: x.lastError || undefined,
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

export interface ChangePublishStatusInput {
    siteId: number;
    nodeId: number;
    newStatus: NodePublishStatus;
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

// =========================================================================
// History Types
// =========================================================================

export interface HistoryState {
    canUndo: boolean;
    canRedo: boolean;
    undoCount: number;
    redoCount: number;
    lastAction?: string;
    actionApplied?: string;
}

export function mapHistoryState(x: dto.HistoryState): HistoryState {
    return {
        canUndo: x.canUndo,
        canRedo: x.canRedo,
        undoCount: x.undoCount,
        redoCount: x.redoCount,
        lastAction: x.lastAction,
        actionApplied: x.actionApplied,
    };
}

// =========================================================================
// Page Generation Types
// =========================================================================

export type PageGenerationTaskStatus = "pending" | "running" | "paused" | "completed" | "failed" | "cancelled";
export type PageGenerationNodeStatus = "pending" | "generating" | "publishing" | "completed" | "failed" | "skipped";
export type PublishAs = "draft" | "pending" | "publish";

export type WritingStyle = "professional" | "casual" | "formal" | "friendly" | "technical";
export type ContentTone = "informative" | "persuasive" | "educational" | "engaging" | "authoritative";
export type AutoLinkMode = "none" | "before" | "after";
export type LinkingPhase = "none" | "suggesting" | "applying" | "completed";

export interface ContentSettings {
    wordCount: string; // e.g. "1000" or "800-1200"
    writingStyle: WritingStyle;
    contentTone: ContentTone;
    customInstructions?: string;
    useWebSearch?: boolean; // Enable web search for AI generation
    includeLinks?: boolean; // Include approved links from linking plan
    autoLinkMode?: AutoLinkMode; // Automatic link suggestion mode
    autoLinkProviderId?: number; // Provider for link suggestion (defaults to content provider)
    autoLinkSuggestPromptId?: number; // Prompt for link suggestion (link_suggest category)
    autoLinkApplyPromptId?: number; // Prompt for link insertion (link_apply category)
    maxIncomingLinks?: number; // Max incoming links per page (0 = no limit)
    maxOutgoingLinks?: number; // Max outgoing links per page (0 = no limit)
}

export interface StartPageGenerationInput {
    sitemapId: number;
    nodeIds?: number[];
    providerId: number;
    promptId?: number;
    publishAs: PublishAs;
    placeholders?: Record<string, string>;
    maxConcurrency?: number;
    contentSettings?: ContentSettings;
}

export interface GenerationNodeInfo {
    nodeId: number;
    title: string;
    path: string;
    status: PageGenerationNodeStatus;
    articleId?: number;
    wpPageId?: number;
    wpUrl?: string;
    error?: string;
    startedAt?: string;
    completedAt?: string;
}

export interface GenerationTask {
    id: string;
    sitemapId: number;
    siteId: number;
    totalNodes: number;
    processedNodes: number;
    failedNodes: number;
    skippedNodes: number;
    status: PageGenerationTaskStatus;
    startedAt: string;
    completedAt?: string;
    error?: string;
    nodes?: GenerationNodeInfo[];
    // Linking phase tracking
    linkingPhase?: LinkingPhase;
    linksCreated?: number;
    linksApplied?: number;
    linksFailed?: number;
}

export interface DefaultPrompt {
    name: string;
    systemPrompt: string;
    userPrompt: string;
    placeholders: string[];
}

export function mapGenerationTask(x: dto.GenerationTaskResponse): GenerationTask {
    return {
        id: x.id,
        sitemapId: x.sitemapId,
        siteId: x.siteId,
        totalNodes: x.totalNodes,
        processedNodes: x.processedNodes,
        failedNodes: x.failedNodes,
        skippedNodes: x.skippedNodes,
        status: x.status as PageGenerationTaskStatus,
        startedAt: x.startedAt,
        completedAt: x.completedAt || undefined,
        error: x.error || undefined,
        nodes: x.nodes?.map((n) => ({
            nodeId: n.nodeId,
            title: n.title,
            path: n.path,
            status: n.status as PageGenerationNodeStatus,
            articleId: n.articleId || undefined,
            wpPageId: n.wpPageId || undefined,
            wpUrl: n.wpUrl || undefined,
            error: n.error || undefined,
            startedAt: n.startedAt || undefined,
            completedAt: n.completedAt || undefined,
        })),
        // Linking phase tracking
        linkingPhase: (x.linkingPhase as LinkingPhase) || undefined,
        linksCreated: x.linksCreated || undefined,
        linksApplied: x.linksApplied || undefined,
        linksFailed: x.linksFailed || undefined,
    };
}

export function mapDefaultPrompt(x: dto.DefaultPromptResponse): DefaultPrompt {
    return {
        name: x.name,
        systemPrompt: x.systemPrompt,
        userPrompt: x.userPrompt,
        placeholders: x.placeholders || [],
    };
}
