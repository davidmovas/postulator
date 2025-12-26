import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    CreateSitemap,
    GetSitemap,
    GetSitemapWithNodes,
    ListSitemaps,
    UpdateSitemap,
    DeleteSitemap,
    DuplicateSitemap,
    SetSitemapStatus,
    CreateNode,
    GetNode,
    GetNodes,
    GetNodesTree,
    UpdateNode,
    DeleteNode,
    MoveNode,
    UpdateNodePositions,
    SetNodeKeywords,
    AddNodeKeyword,
    RemoveNodeKeyword,
    DistributeKeywords,
    LinkNodeToArticle,
    LinkNodeToPage,
    UnlinkNodeContent,
    UpdateNodeDesignStatus,
    UpdateNodeGenerationStatus,
    UpdateNodePublishStatus,
    GetSupportedImportFormats,
    ImportNodes,
    ScanSite,
    ScanIntoSitemap,
    SyncNodesFromWP,
    UpdateNodesToWP,
    ResetNode,
    ChangePublishStatus,
    GenerateSitemapStructure,
    CancelSitemapGeneration,
    StartPageGeneration,
    PausePageGeneration,
    ResumePageGeneration,
    CancelPageGeneration,
    GetPageGenerationTask,
    ListActivePageGenerationTasks,
    GetDefaultPagePrompt,
    Undo,
    Redo,
    GetHistoryState,
    ClearHistory,
} from "@/wailsjs/wailsjs/go/handlers/SitemapsHandler";
import {
    Sitemap,
    SitemapNode,
    SitemapWithNodes,
    CreateSitemapInput,
    UpdateSitemapInput,
    CreateNodeInput,
    UpdateNodeInput,
    MoveNodeInput,
    UpdateNodePositionsInput,
    ImportNodesInput,
    ImportNodesResult,
    ScanSiteInput,
    ScanIntoSitemapInput,
    ScanSiteResult,
    SyncNodesInput,
    SyncNodesResult,
    ChangePublishStatusInput,
    SitemapStatus,
    NodeDesignStatus,
    NodeGenerationStatus,
    NodePublishStatus,
    GenerateSitemapStructureInput,
    GenerateSitemapStructureResult,
    HistoryState,
    StartPageGenerationInput,
    GenerationTask,
    DefaultPrompt,
    mapSitemap,
    mapSitemapNode,
    mapSitemapWithNodes,
    mapImportNodesResult,
    mapScanSiteResult,
    mapSyncNodesResult,
    mapGenerateSitemapStructureResult,
    mapHistoryState,
    mapGenerationTask,
    mapDefaultPrompt,
} from "@/models/sitemaps";
import { unwrapArrayResponse, unwrapResponse } from "@/lib/api-utils";

export const sitemapService = {
    async createSitemap(input: CreateSitemapInput): Promise<Sitemap> {
        const payload = new dto.CreateSitemapRequest({
            siteId: input.siteId,
            name: input.name,
            description: input.description,
            source: input.source,
            siteUrl: input.siteUrl,
        });

        const response = await CreateSitemap(payload);
        const data = unwrapResponse<dto.Sitemap>(response);
        return mapSitemap(data);
    },

    async getSitemap(id: number): Promise<Sitemap> {
        const response = await GetSitemap(id);
        const data = unwrapResponse<dto.Sitemap>(response);
        return mapSitemap(data);
    },

    async getSitemapWithNodes(id: number): Promise<SitemapWithNodes> {
        const response = await GetSitemapWithNodes(id);
        const data = unwrapResponse<dto.SitemapWithNodes>(response);
        return mapSitemapWithNodes(data);
    },

    async listSitemaps(siteId: number): Promise<Sitemap[]> {
        const response = await ListSitemaps(siteId);
        const data = unwrapArrayResponse<dto.Sitemap>(response);
        return data.map(mapSitemap);
    },

    async updateSitemap(input: UpdateSitemapInput): Promise<void> {
        const payload = new dto.UpdateSitemapRequest({
            id: input.id,
            name: input.name,
            description: input.description,
            status: input.status,
        });

        const response = await UpdateSitemap(payload);
        unwrapResponse<string>(response);
    },

    async deleteSitemap(id: number): Promise<void> {
        const response = await DeleteSitemap(id);
        unwrapResponse<string>(response);
    },

    async duplicateSitemap(id: number, newName: string): Promise<Sitemap> {
        const payload = new dto.DuplicateSitemapRequest({
            id: id,
            newName: newName,
        });

        const response = await DuplicateSitemap(payload);
        const data = unwrapResponse<dto.Sitemap>(response);
        return mapSitemap(data);
    },

    async setSitemapStatus(id: number, status: SitemapStatus): Promise<void> {
        const response = await SetSitemapStatus(id, status);
        unwrapResponse<string>(response);
    },

    async createNode(input: CreateNodeInput): Promise<SitemapNode> {
        const payload = new dto.CreateNodeRequest({
            sitemapId: input.sitemapId,
            parentId: input.parentId,
            title: input.title,
            slug: input.slug,
            description: input.description,
            position: input.position,
            source: input.source,
            keywords: input.keywords,
        });

        const response = await CreateNode(payload);
        const data = unwrapResponse<dto.SitemapNode>(response);
        return mapSitemapNode(data);
    },

    async getNode(id: number): Promise<SitemapNode> {
        const response = await GetNode(id);
        const data = unwrapResponse<dto.SitemapNode>(response);
        return mapSitemapNode(data);
    },

    async getNodes(sitemapId: number): Promise<SitemapNode[]> {
        const response = await GetNodes(sitemapId);
        const data = unwrapArrayResponse<dto.SitemapNode>(response);
        return data.map(mapSitemapNode);
    },

    async getNodesTree(sitemapId: number): Promise<SitemapNode[]> {
        const response = await GetNodesTree(sitemapId);
        const data = unwrapArrayResponse<dto.SitemapNode>(response);
        return data.map(mapSitemapNode);
    },

    async updateNode(input: UpdateNodeInput): Promise<void> {
        const payload = new dto.UpdateNodeRequest({
            id: input.id,
            title: input.title,
            slug: input.slug,
            description: input.description,
            keywords: input.keywords,
        });

        const response = await UpdateNode(payload);
        unwrapResponse<string>(response);
    },

    async deleteNode(id: number): Promise<void> {
        const response = await DeleteNode(id);
        unwrapResponse<string>(response);
    },

    async moveNode(input: MoveNodeInput): Promise<void> {
        const payload = new dto.MoveNodeRequest({
            nodeId: input.nodeId,
            newParentId: input.newParentId,
            position: input.position,
        });

        const response = await MoveNode(payload);
        unwrapResponse<string>(response);
    },

    async updateNodePositions(input: UpdateNodePositionsInput): Promise<void> {
        const payload = new dto.UpdateNodePositionsRequest({
            nodeId: input.nodeId,
            positionX: input.positionX,
            positionY: input.positionY,
        });

        const response = await UpdateNodePositions(payload);
        unwrapResponse<string>(response);
    },

    async setNodeKeywords(nodeId: number, keywords: string[]): Promise<void> {
        const payload = new dto.SetNodeKeywordsRequest({
            nodeId: nodeId,
            keywords: keywords,
        });

        const response = await SetNodeKeywords(payload);
        unwrapResponse<string>(response);
    },

    async addNodeKeyword(nodeId: number, keyword: string): Promise<void> {
        const response = await AddNodeKeyword(nodeId, keyword);
        unwrapResponse<string>(response);
    },

    async removeNodeKeyword(nodeId: number, keyword: string): Promise<void> {
        const response = await RemoveNodeKeyword(nodeId, keyword);
        unwrapResponse<string>(response);
    },

    async distributeKeywords(
        sitemapId: number,
        keywords: string[],
        strategy: "even" | "bypath"
    ): Promise<void> {
        const payload = new dto.DistributeKeywordsRequest({
            sitemapId: sitemapId,
            keywords: keywords,
            strategy: strategy,
        });

        const response = await DistributeKeywords(payload);
        unwrapResponse<string>(response);
    },

    async linkNodeToArticle(nodeId: number, articleId: number): Promise<void> {
        const payload = new dto.LinkNodeToArticleRequest({
            nodeId: nodeId,
            articleId: articleId,
        });

        const response = await LinkNodeToArticle(payload);
        unwrapResponse<string>(response);
    },

    async linkNodeToPage(nodeId: number, wpPageId: number, wpUrl: string): Promise<void> {
        const payload = new dto.LinkNodeToPageRequest({
            nodeId: nodeId,
            wpPageId: wpPageId,
            wpUrl: wpUrl,
        });

        const response = await LinkNodeToPage(payload);
        unwrapResponse<string>(response);
    },

    async unlinkNodeContent(nodeId: number): Promise<void> {
        const response = await UnlinkNodeContent(nodeId);
        unwrapResponse<string>(response);
    },

    async updateNodeDesignStatus(nodeId: number, status: NodeDesignStatus): Promise<void> {
        const response = await UpdateNodeDesignStatus(nodeId, status);
        unwrapResponse<string>(response);
    },

    async updateNodeGenerationStatus(nodeId: number, status: NodeGenerationStatus): Promise<void> {
        const response = await UpdateNodeGenerationStatus(nodeId, status);
        unwrapResponse<string>(response);
    },

    async updateNodePublishStatus(nodeId: number, status: NodePublishStatus): Promise<void> {
        const response = await UpdateNodePublishStatus(nodeId, status);
        unwrapResponse<string>(response);
    },

    async getSupportedImportFormats(): Promise<string[]> {
        const response = await GetSupportedImportFormats();
        const data = unwrapResponse<dto.SupportedFormatsResponse>(response);
        return data.formats || [];
    },

    async importNodes(input: ImportNodesInput): Promise<ImportNodesResult> {
        const payload = new dto.ImportNodesRequest({
            sitemapId: input.sitemapId,
            parentNodeId: input.parentNodeId,
            filename: input.filename,
            fileDataBase64: input.fileDataBase64,
        });

        const response = await ImportNodes(payload);
        const data = unwrapResponse<dto.ImportNodesResponse>(response);
        return mapImportNodesResult(data);
    },

    async scanSite(input: ScanSiteInput): Promise<ScanSiteResult> {
        const payload = new dto.ScanSiteRequest({
            siteId: input.siteId,
            sitemapName: input.sitemapName,
            titleSource: input.titleSource,
            contentFilter: input.contentFilter,
            includeDrafts: input.includeDrafts,
            maxDepth: input.maxDepth,
        });

        const response = await ScanSite(payload);
        const data = unwrapResponse<dto.ScanSiteResponse>(response);
        return mapScanSiteResult(data);
    },

    async scanIntoSitemap(input: ScanIntoSitemapInput): Promise<ScanSiteResult> {
        const payload = new dto.ScanIntoSitemapRequest({
            sitemapId: input.sitemapId,
            parentNodeId: input.parentNodeId,
            titleSource: input.titleSource,
            contentFilter: input.contentFilter,
            includeDrafts: input.includeDrafts,
            maxDepth: input.maxDepth,
        });

        const response = await ScanIntoSitemap(payload);
        const data = unwrapResponse<dto.ScanSiteResponse>(response);
        return mapScanSiteResult(data);
    },

    async syncNodesFromWP(input: SyncNodesInput): Promise<SyncNodesResult> {
        const payload = new dto.SyncNodesRequest({
            siteId: input.siteId,
            nodeIds: input.nodeIds,
        });

        const response = await SyncNodesFromWP(payload);
        const data = unwrapResponse<dto.SyncNodesResponse>(response);
        return mapSyncNodesResult(data);
    },

    async updateNodesToWP(input: SyncNodesInput): Promise<SyncNodesResult> {
        const payload = new dto.UpdateNodesToWPRequest({
            siteId: input.siteId,
            nodeIds: input.nodeIds,
        });

        const response = await UpdateNodesToWP(payload);
        const data = unwrapResponse<dto.SyncNodesResponse>(response);
        return mapSyncNodesResult(data);
    },

    async resetNode(nodeId: number): Promise<void> {
        const response = await ResetNode(nodeId);
        unwrapResponse<string>(response);
    },

    async changePublishStatus(input: ChangePublishStatusInput): Promise<void> {
        const payload = new dto.ChangePublishStatusRequest({
            siteId: input.siteId,
            nodeId: input.nodeId,
            newStatus: input.newStatus,
        });

        const response = await ChangePublishStatus(payload);
        unwrapResponse<string>(response);
    },

    async generateSitemapStructure(
        input: GenerateSitemapStructureInput
    ): Promise<GenerateSitemapStructureResult> {
        const payload = new dto.GenerateSitemapStructureRequest({
            sitemapId: input.sitemapId,
            siteId: input.siteId,
            name: input.name,
            promptId: input.promptId,
            placeholders: input.placeholders,
            titles: input.titles.map(
                (t) =>
                    new dto.TitleInput({
                        title: t.title,
                        keywords: t.keywords,
                    })
            ),
            parentNodeIds: input.parentNodeIds,
            maxDepth: input.maxDepth,
            includeExistingTree: input.includeExistingTree,
            providerId: input.providerId,
        });

        const response = await GenerateSitemapStructure(payload);
        const data = unwrapResponse<dto.GenerateSitemapStructureResponse>(response);
        return mapGenerateSitemapStructureResult(data);
    },

    async cancelSitemapGeneration(): Promise<void> {
        const response = await CancelSitemapGeneration();
        unwrapResponse<string>(response);
    },

    // History operations
    async undo(sitemapId: number): Promise<HistoryState | null> {
        const response = await Undo(sitemapId);
        const data = unwrapResponse<dto.HistoryState>(response);
        return mapHistoryState(data);
    },

    async redo(sitemapId: number): Promise<HistoryState | null> {
        const response = await Redo(sitemapId);
        const data = unwrapResponse<dto.HistoryState>(response);
        return mapHistoryState(data);
    },

    async getHistoryState(sitemapId: number): Promise<HistoryState | null> {
        const response = await GetHistoryState(sitemapId);
        const data = unwrapResponse<dto.HistoryState>(response);
        return mapHistoryState(data);
    },

    async clearHistory(sitemapId: number): Promise<void> {
        const response = await ClearHistory(sitemapId);
        unwrapResponse<string>(response);
    },

    // Page Generation operations
    async startPageGeneration(input: StartPageGenerationInput): Promise<GenerationTask> {
        const payload = new dto.StartPageGenerationRequest({
            sitemapId: input.sitemapId,
            nodeIds: input.nodeIds,
            providerId: input.providerId,
            promptId: input.promptId,
            publishAs: input.publishAs,
            placeholders: input.placeholders,
            maxConcurrency: input.maxConcurrency,
            contentSettings: input.contentSettings ? new dto.ContentSettingsDTO({
                wordCount: input.contentSettings.wordCount,
                writingStyle: input.contentSettings.writingStyle,
                contentTone: input.contentSettings.contentTone,
                customInstructions: input.contentSettings.customInstructions || "",
                useWebSearch: input.contentSettings.useWebSearch || false,
                includeLinks: input.contentSettings.includeLinks || false,
                autoLinkMode: input.contentSettings.autoLinkMode || "none",
                autoLinkProviderId: input.contentSettings.autoLinkProviderId,
                autoLinkSuggestPromptId: input.contentSettings.autoLinkSuggestPromptId,
                autoLinkApplyPromptId: input.contentSettings.autoLinkApplyPromptId,
                maxIncomingLinks: input.contentSettings.maxIncomingLinks || 0,
                maxOutgoingLinks: input.contentSettings.maxOutgoingLinks || 0,
            }) : undefined,
        });

        const response = await StartPageGeneration(payload);
        const data = unwrapResponse<dto.GenerationTaskResponse>(response);
        return mapGenerationTask(data);
    },

    async pausePageGeneration(taskId: string): Promise<void> {
        const response = await PausePageGeneration(taskId);
        unwrapResponse<string>(response);
    },

    async resumePageGeneration(taskId: string): Promise<void> {
        const response = await ResumePageGeneration(taskId);
        unwrapResponse<string>(response);
    },

    async cancelPageGeneration(taskId: string): Promise<void> {
        const response = await CancelPageGeneration(taskId);
        unwrapResponse<string>(response);
    },

    async getPageGenerationTask(taskId: string): Promise<GenerationTask> {
        const response = await GetPageGenerationTask(taskId);
        const data = unwrapResponse<dto.GenerationTaskResponse>(response);
        return mapGenerationTask(data);
    },

    async listActivePageGenerationTasks(): Promise<GenerationTask[]> {
        const response = await ListActivePageGenerationTasks();
        const data = unwrapArrayResponse<dto.GenerationTaskResponse>(response);
        return data.map(mapGenerationTask);
    },

    async getDefaultPagePrompt(): Promise<DefaultPrompt> {
        const response = await GetDefaultPagePrompt();
        const data = unwrapResponse<dto.DefaultPromptResponse>(response);
        return mapDefaultPrompt(data);
    },
};
