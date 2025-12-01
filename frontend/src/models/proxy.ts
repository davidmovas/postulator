import { dto } from "@/wailsjs/wailsjs/go/models";

export type ProxyType = "http" | "socks5" | "tor";
export type ProxyStatus = "disabled" | "connecting" | "connected" | "error" | "tor_not_found";
export type ProxyMode = "single" | "chain";

export interface ProxyNode {
    id: string;
    type: ProxyType;
    host: string;
    port: number;
    username?: string;
    password?: string;
    enabled: boolean;
    order: number;
}

export interface ProxySettings {
    enabled: boolean;
    mode: ProxyMode;
    nodes: ProxyNode[];
    rotationEnabled: boolean;
    rotationInterval: number;
    healthCheckEnabled: boolean;
    healthCheckInterval: number;
    notifyOnFailure: boolean;
    notifyOnRecover: boolean;
    currentNodeId?: string;
}

export interface ProxyHealth {
    nodeId: string;
    status: ProxyStatus;
    latencyMs: number;
    lastChecked: number;
    error?: string;
    externalIp?: string;
}

export interface ProxyState {
    status: ProxyStatus;
    activeNodeId?: string;
    externalIp?: string;
    latencyMs: number;
    nodesHealth: ProxyHealth[];
    lastError?: string;
    lastCheckedAt: number;
}

export interface TorDetectionResult {
    found: boolean;
    port: number;
    serviceType: string;
}

export function mapProxyNode(x: dto.ProxyNode): ProxyNode {
    return {
        id: x.id,
        type: x.type as ProxyType,
        host: x.host,
        port: x.port,
        username: x.username,
        password: x.password,
        enabled: x.enabled,
        order: x.order,
    };
}

export function mapProxySettings(x: dto.ProxySettings): ProxySettings {
    return {
        enabled: x.enabled,
        mode: x.mode as ProxyMode,
        nodes: x.nodes?.map(mapProxyNode) || [],
        rotationEnabled: x.rotation_enabled,
        rotationInterval: x.rotation_interval,
        healthCheckEnabled: x.health_check_enabled,
        healthCheckInterval: x.health_check_interval,
        notifyOnFailure: x.notify_on_failure,
        notifyOnRecover: x.notify_on_recover,
        currentNodeId: x.current_node_id,
    };
}

export function mapProxyHealth(x: dto.ProxyHealth): ProxyHealth {
    return {
        nodeId: x.node_id,
        status: x.status as ProxyStatus,
        latencyMs: x.latency_ms,
        lastChecked: x.last_checked,
        error: x.error,
        externalIp: x.external_ip,
    };
}

export function mapProxyState(x: dto.ProxyState): ProxyState {
    return {
        status: x.status as ProxyStatus,
        activeNodeId: x.active_node_id,
        externalIp: x.external_ip,
        latencyMs: x.latency_ms,
        nodesHealth: x.nodes_health?.map(mapProxyHealth) || [],
        lastError: x.last_error,
        lastCheckedAt: x.last_checked_at,
    };
}

export function mapTorDetectionResult(x: dto.TorDetectionResult): TorDetectionResult {
    return {
        found: x.found,
        port: x.port,
        serviceType: x.service_type,
    };
}

export function toDtoProxyNode(node: ProxyNode): dto.ProxyNode {
    return new dto.ProxyNode({
        id: node.id,
        type: node.type,
        host: node.host,
        port: node.port,
        username: node.username || "",
        password: node.password || "",
        enabled: node.enabled,
        order: node.order,
    });
}

export function toDtoProxySettings(settings: ProxySettings): dto.ProxySettings {
    return new dto.ProxySettings({
        enabled: settings.enabled,
        mode: settings.mode,
        nodes: settings.nodes.map(toDtoProxyNode),
        rotation_enabled: settings.rotationEnabled,
        rotation_interval: settings.rotationInterval,
        health_check_enabled: settings.healthCheckEnabled,
        health_check_interval: settings.healthCheckInterval,
        notify_on_failure: settings.notifyOnFailure,
        notify_on_recover: settings.notifyOnRecover,
        current_node_id: settings.currentNodeId || "",
    });
}

export function createDefaultProxyNode(type: ProxyType = "socks5"): ProxyNode {
    return {
        id: crypto.randomUUID(),
        type,
        host: "127.0.0.1",
        port: type === "tor" ? 9050 : 1080,
        enabled: true,
        order: 0,
    };
}

export function createTorNode(port: number = 9050): ProxyNode {
    return {
        id: "tor",
        type: "tor",
        host: "127.0.0.1",
        port,
        enabled: true,
        order: 0,
    };
}

export function hasTorNode(nodes: ProxyNode[]): boolean {
    return nodes.some(node => node.type === "tor");
}

export interface IPComparison {
    directIp: string;
    directError?: string;
    proxyIp: string;
    proxyError?: string;
    isAnonymous: boolean;
}

export function mapIPComparison(x: dto.IPComparison): IPComparison {
    return {
        directIp: x.direct_ip,
        directError: x.direct_error,
        proxyIp: x.proxy_ip,
        proxyError: x.proxy_error,
        isAnonymous: x.is_anonymous,
    };
}
