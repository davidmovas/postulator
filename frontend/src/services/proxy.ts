import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    GetSettings,
    UpdateSettings,
    GetState,
    TestNode,
    TestAllNodes,
    DetectTor,
    Enable,
    Disable,
    AddTorNode,
    GetDefaultTorNode,
    GetTorBrowserNode,
    CompareIPs,
} from "@/wailsjs/wailsjs/go/handlers/ProxyHandler";
import {
    ProxySettings,
    ProxyState,
    ProxyHealth,
    ProxyNode,
    TorDetectionResult,
    IPComparison,
    mapProxySettings,
    mapProxyState,
    mapProxyHealth,
    mapProxyNode,
    mapTorDetectionResult,
    mapIPComparison,
    toDtoProxySettings,
    toDtoProxyNode,
} from "@/models/proxy";
import { unwrapResponse } from "@/lib/api-utils";

export const proxyService = {
    async getSettings(): Promise<ProxySettings> {
        const response = await GetSettings();
        const settings = unwrapResponse<dto.ProxySettings>(response);
        return mapProxySettings(settings);
    },

    async updateSettings(settings: ProxySettings): Promise<string> {
        const payload = toDtoProxySettings(settings);
        const response = await UpdateSettings(payload);
        return unwrapResponse<string>(response);
    },

    async getState(): Promise<ProxyState> {
        const response = await GetState();
        const state = unwrapResponse<dto.ProxyState>(response);
        return mapProxyState(state);
    },

    async testNode(node: ProxyNode): Promise<ProxyHealth> {
        const payload = toDtoProxyNode(node);
        const response = await TestNode(payload);
        const health = unwrapResponse<dto.ProxyHealth>(response);
        return mapProxyHealth(health);
    },

    async testAllNodes(): Promise<ProxyHealth[]> {
        const response = await TestAllNodes();
        const healthList = unwrapResponse<dto.ProxyHealth[]>(response);
        return healthList.map(mapProxyHealth);
    },

    async detectTor(): Promise<TorDetectionResult> {
        const response = await DetectTor();
        const result = unwrapResponse<dto.TorDetectionResult>(response);
        return mapTorDetectionResult(result);
    },

    async enable(): Promise<string> {
        const response = await Enable();
        return unwrapResponse<string>(response);
    },

    async disable(): Promise<string> {
        const response = await Disable();
        return unwrapResponse<string>(response);
    },

    async addTorNode(): Promise<ProxyNode> {
        const response = await AddTorNode();
        const node = unwrapResponse<dto.ProxyNode>(response);
        return mapProxyNode(node);
    },

    async getDefaultTorNode(): Promise<ProxyNode> {
        const response = await GetDefaultTorNode();
        const node = unwrapResponse<dto.ProxyNode>(response);
        return mapProxyNode(node);
    },

    async getTorBrowserNode(): Promise<ProxyNode> {
        const response = await GetTorBrowserNode();
        const node = unwrapResponse<dto.ProxyNode>(response);
        return mapProxyNode(node);
    },

    async compareIPs(): Promise<IPComparison> {
        const response = await CompareIPs();
        const comparison = unwrapResponse<dto.IPComparison>(response);
        return mapIPComparison(comparison);
    },
};
