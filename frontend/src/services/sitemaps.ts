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
    UpdateNodeContentStatus,
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
    SitemapStatus,
    NodeContentStatus,
    mapSitemap,
    mapSitemapNode,
    mapSitemapWithNodes,
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

    async updateNodeContentStatus(nodeId: number, status: NodeContentStatus): Promise<void> {
        const response = await UpdateNodeContentStatus(nodeId, status);
        unwrapResponse<string>(response);
    },
};
