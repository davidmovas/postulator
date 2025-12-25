import { dto } from "@/wailsjs/wailsjs/go/models";

// =========================================================================
// Linking Types
// =========================================================================

export type PlanStatus = "draft" | "suggesting" | "ready" | "applying" | "applied" | "failed";
export type LinkStatus = "planned" | "approved" | "rejected" | "applying" | "applied" | "failed";
export type LinkSource = "ai" | "manual";

export interface LinkPlan {
    id: number;
    sitemapId: number;
    siteId: number;
    name: string;
    status: PlanStatus;
    providerId?: number;
    promptId?: number;
    error?: string;
    createdAt: string;
    updatedAt: string;
}

export interface PlannedLink {
    id: number;
    planId: number;
    sourceNodeId: number;
    targetNodeId: number;
    anchorText?: string;
    anchorContext?: string;
    status: LinkStatus;
    source: LinkSource;
    position?: number;
    confidence?: number;
    error?: string;
    appliedAt?: string;
    createdAt: string;
    updatedAt: string;
}

export interface GraphNode {
    nodeId: number;
    title: string;
    slug: string;
    path: string;
    hasContent: boolean;
    outgoingLinkCount: number;
    incomingLinkCount: number;
}

export interface GraphEdge {
    id: number;
    sourceNodeId: number;
    targetNodeId: number;
    anchorText?: string;
    status: LinkStatus;
    source: LinkSource;
    confidence?: number;
}

export interface LinkGraph {
    nodes: GraphNode[];
    edges: GraphEdge[];
}

// =========================================================================
// Input Types
// =========================================================================

export interface CreateLinkPlanInput {
    sitemapId: number;
    siteId: number;
    name: string;
}

export interface AddLinkInput {
    planId: number;
    sourceNodeId: number;
    targetNodeId: number;
}

export interface UpdateLinkInput {
    id: number;
    anchorText?: string;
    anchorContext?: string;
}

export interface SuggestLinksInput {
    planId: number;
    providerId: number;
    promptId?: number;
    nodeIds?: number[];
    feedback?: string;
    maxIncoming?: number;
    maxOutgoing?: number;
}

export interface ApplyLinksInput {
    planId: number;
    linkIds: number[];
    providerId: number;
    promptId: number;
}

export interface ApplyLinksResult {
    totalLinks: number;
    appliedLinks: number;
    failedLinks: number;
}

// =========================================================================
// Apply Links Event Types (PascalCase - as sent from Go)
// =========================================================================

export interface ApplyStartedEvent {
    TaskID: string;
    TotalLinks: number;
    TotalPages: number;
}

export interface ApplyProgressEvent {
    TaskID: string;
    ProcessedPages: number;
    TotalPages: number;
    AppliedLinks: number;
    FailedLinks: number;
    CurrentPage?: PageInfo;
}

export interface PageInfo {
    NodeID: number;
    Title: string;
    Path: string;
}

export interface ApplyCompletedEvent {
    TaskID: string;
    TotalLinks: number;
    AppliedLinks: number;
    FailedLinks: number;
    DurationMs: number;
}

export interface ApplyFailedEvent {
    TaskID: string;
    Error: string;
}

export interface PageProcessingEvent {
    TaskID: string;
    NodeID: number;
    Title: string;
    LinkCount: number;
}

export interface PageCompletedEvent {
    TaskID: string;
    NodeID: number;
    Title: string;
    AppliedLinks: number;
    FailedLinks: number;
}

export interface PageFailedEvent {
    TaskID: string;
    NodeID: number;
    Title: string;
    Error: string;
}

// =========================================================================
// Suggest Links Event Types (PascalCase - as sent from Go)
// =========================================================================

export interface SuggestStartedEvent {
    TaskID: string;
    TotalNodes: number;
    TotalBatches: number;
}

export interface SuggestProgressEvent {
    TaskID: string;
    CurrentBatch: number;
    TotalBatches: number;
    ProcessedNodes: number;
    TotalNodes: number;
    LinksCreated: number;
    CurrentBatchSize: number;
}

export interface SuggestCompletedEvent {
    TaskID: string;
    TotalNodes: number;
    LinksCreated: number;
    DurationMs: number;
}

export interface SuggestFailedEvent {
    TaskID: string;
    Error: string;
}

export interface SuggestCancelledEvent {
    TaskID: string;
    ProcessedNodes: number;
    LinksCreated: number;
}

export interface ApplyCancelledEvent {
    TaskID: string;
    ProcessedPages: number;
    AppliedLinks: number;
}

// =========================================================================
// Mappers
// =========================================================================

export function mapLinkPlan(x: dto.LinkPlan): LinkPlan {
    return {
        id: x.id,
        sitemapId: x.sitemapId,
        siteId: x.siteId,
        name: x.name,
        status: x.status as PlanStatus,
        providerId: x.providerId || undefined,
        promptId: x.promptId || undefined,
        error: x.error || undefined,
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}

export function mapPlannedLink(x: dto.PlannedLink): PlannedLink {
    return {
        id: x.id,
        planId: x.planId,
        sourceNodeId: x.sourceNodeId,
        targetNodeId: x.targetNodeId,
        anchorText: x.anchorText || undefined,
        anchorContext: x.anchorContext || undefined,
        status: x.status as LinkStatus,
        source: x.source as LinkSource,
        position: x.position || undefined,
        confidence: x.confidence || undefined,
        error: x.error || undefined,
        appliedAt: x.appliedAt || undefined,
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}

export function mapGraphNode(x: dto.GraphNode): GraphNode {
    return {
        nodeId: x.nodeId,
        title: x.title,
        slug: x.slug,
        path: x.path,
        hasContent: x.hasContent,
        outgoingLinkCount: x.outgoingLinkCount,
        incomingLinkCount: x.incomingLinkCount,
    };
}

export function mapGraphEdge(x: dto.GraphEdge): GraphEdge {
    return {
        id: x.id,
        sourceNodeId: x.sourceNodeId,
        targetNodeId: x.targetNodeId,
        anchorText: x.anchorText || undefined,
        status: x.status as LinkStatus,
        source: x.source as LinkSource,
        confidence: x.confidence || undefined,
    };
}

export function mapLinkGraph(x: dto.LinkGraph): LinkGraph {
    return {
        nodes: (x.nodes || []).map(mapGraphNode),
        edges: (x.edges || []).map(mapGraphEdge),
    };
}
