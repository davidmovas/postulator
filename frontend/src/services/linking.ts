import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    CreatePlan,
    GetPlan,
    GetPlanBySitemap,
    GetActivePlan,
    GetOrCreateActivePlan,
    ListPlans,
    DeletePlan,
    AddLink,
    RemoveLink,
    UpdateLink,
    GetLinks,
    GetLinksByNode,
    ApproveLink,
    RejectLink,
    SuggestLinks,
    CancelSuggest,
    ApplyLinks,
    CancelApply,
    GetLinkGraph,
} from "@/wailsjs/wailsjs/go/handlers/LinkingHandler";
import {
    LinkPlan,
    PlannedLink,
    LinkGraph,
    CreateLinkPlanInput,
    AddLinkInput,
    UpdateLinkInput,
    SuggestLinksInput,
    ApplyLinksInput,
    ApplyLinksResult,
    mapLinkPlan,
    mapPlannedLink,
    mapLinkGraph,
} from "@/models/linking";
import { unwrapResponse, unwrapArrayResponse } from "@/lib/api-utils";

export const linkingService = {
    // Plan operations
    async createPlan(input: CreateLinkPlanInput): Promise<LinkPlan> {
        const payload = new dto.CreateLinkPlanRequest({
            sitemapId: input.sitemapId,
            siteId: input.siteId,
            name: input.name,
        });
        const response = await CreatePlan(payload);
        const data = unwrapResponse<dto.LinkPlan>(response);
        return mapLinkPlan(data);
    },

    async getPlan(id: number): Promise<LinkPlan> {
        const response = await GetPlan(id);
        const data = unwrapResponse<dto.LinkPlan>(response);
        return mapLinkPlan(data);
    },

    async getPlanBySitemap(sitemapId: number): Promise<LinkPlan | null> {
        const response = await GetPlanBySitemap(sitemapId);
        const data = unwrapResponse<dto.LinkPlan | null>(response);
        return data ? mapLinkPlan(data) : null;
    },

    async getActivePlan(sitemapId: number): Promise<LinkPlan | null> {
        const response = await GetActivePlan(sitemapId);
        const data = unwrapResponse<dto.LinkPlan | null>(response);
        return data ? mapLinkPlan(data) : null;
    },

    async getOrCreateActivePlan(sitemapId: number, siteId: number): Promise<LinkPlan> {
        const response = await GetOrCreateActivePlan(sitemapId, siteId);
        const data = unwrapResponse<dto.LinkPlan>(response);
        return mapLinkPlan(data);
    },

    async listPlans(siteId: number): Promise<LinkPlan[]> {
        const response = await ListPlans(siteId);
        const data = unwrapArrayResponse<dto.LinkPlan>(response);
        return data.map(mapLinkPlan);
    },

    async deletePlan(id: number): Promise<void> {
        const response = await DeletePlan(id);
        unwrapResponse<boolean>(response);
    },

    // Link operations
    async addLink(input: AddLinkInput): Promise<PlannedLink> {
        const payload = new dto.AddLinkRequest({
            planId: input.planId,
            sourceNodeId: input.sourceNodeId,
            targetNodeId: input.targetNodeId,
        });
        const response = await AddLink(payload);
        const data = unwrapResponse<dto.PlannedLink>(response);
        return mapPlannedLink(data);
    },

    async removeLink(linkId: number): Promise<void> {
        const response = await RemoveLink(linkId);
        unwrapResponse<boolean>(response);
    },

    async updateLink(input: UpdateLinkInput): Promise<PlannedLink> {
        const payload = new dto.UpdateLinkRequest({
            id: input.id,
            anchorText: input.anchorText,
            anchorContext: input.anchorContext,
        });
        const response = await UpdateLink(payload);
        const data = unwrapResponse<dto.PlannedLink>(response);
        return mapPlannedLink(data);
    },

    async getLinks(planId: number): Promise<PlannedLink[]> {
        const response = await GetLinks(planId);
        const data = unwrapArrayResponse<dto.PlannedLink>(response);
        return data.map(mapPlannedLink);
    },

    async getLinksByNode(planId: number, nodeId: number): Promise<PlannedLink[]> {
        const response = await GetLinksByNode(planId, nodeId);
        const data = unwrapArrayResponse<dto.PlannedLink>(response);
        return data.map(mapPlannedLink);
    },

    async approveLink(linkId: number): Promise<void> {
        const response = await ApproveLink(linkId);
        unwrapResponse<boolean>(response);
    },

    async rejectLink(linkId: number): Promise<void> {
        const response = await RejectLink(linkId);
        unwrapResponse<boolean>(response);
    },

    // AI suggestions
    async suggestLinks(input: SuggestLinksInput): Promise<void> {
        const payload = new dto.SuggestLinksRequest({
            planId: input.planId,
            providerId: input.providerId,
            promptId: input.promptId,
            nodeIds: input.nodeIds,
            feedback: input.feedback || "",
            maxIncoming: input.maxIncoming || 0,
            maxOutgoing: input.maxOutgoing || 0,
        });
        const response = await SuggestLinks(payload);
        unwrapResponse<boolean>(response);
    },

    async cancelSuggest(planId: number): Promise<boolean> {
        const response = await CancelSuggest(planId);
        return unwrapResponse<boolean>(response);
    },

    // Apply links to WordPress
    async applyLinks(input: ApplyLinksInput): Promise<ApplyLinksResult> {
        const payload = new dto.ApplyLinksRequest({
            planId: input.planId,
            linkIds: input.linkIds,
            providerId: input.providerId,
        });
        const response = await ApplyLinks(payload);
        const data = unwrapResponse<dto.ApplyLinksResult>(response);
        return {
            totalLinks: data.totalLinks,
            appliedLinks: data.appliedLinks,
            failedLinks: data.failedLinks,
        };
    },

    async cancelApply(planId: number): Promise<boolean> {
        const response = await CancelApply(planId);
        return unwrapResponse<boolean>(response);
    },

    // Graph visualization
    async getLinkGraph(planId: number): Promise<LinkGraph> {
        const response = await GetLinkGraph(planId);
        const data = unwrapResponse<dto.LinkGraph>(response);
        return mapLinkGraph(data);
    },
};
