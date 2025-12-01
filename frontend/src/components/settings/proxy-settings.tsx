"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { SettingsSection } from "./settings-section";
import { ProxyChainView } from "./proxy-chain-view";
import { useProxySettings } from "@/hooks/use-proxy-settings";
import { ProxyNode, ProxySettings as ProxySettingsType, ProxyHealth, IPComparison, createDefaultProxyNode, hasTorNode } from "@/models/proxy";
import {
    RiShieldLine,
    RiAddLine,
    RiRefreshLine,
    RiCheckLine,
    RiCloseLine,
    RiLoader4Line,
    RiAlertLine,
    RiArrowRightLine,
    RiLinkM,
    RiStackLine,
} from "@remixicon/react";
import { cn } from "@/lib/utils";

function getStatusBadge(status: string, showDisabled: boolean = true) {
    switch (status) {
        case "connected":
            return <Badge variant="default" className="bg-green-500/10 text-green-500 border-green-500/20">Connected</Badge>;
        case "connecting":
            return <Badge variant="default" className="bg-yellow-500/10 text-yellow-500 border-yellow-500/20">Connecting</Badge>;
        case "error":
            return <Badge variant="destructive">Error</Badge>;
        case "tor_not_found":
            return <Badge variant="destructive">Tor Not Found</Badge>;
        default:
            return showDisabled ? <Badge variant="secondary">Disabled</Badge> : null;
    }
}

export function ProxySettings() {
    const {
        settings,
        state,
        isLoading,
        isTesting,
        updateSettings,
        testNode,
        testAllNodes,
        detectTor,
        addTorNode,
        compareIPs,
    } = useProxySettings();

    const [formData, setFormData] = useState<ProxySettingsType>({
        enabled: false,
        mode: "single",
        nodes: [],
        rotationEnabled: false,
        rotationInterval: 300,
        healthCheckEnabled: true,
        healthCheckInterval: 60,
        notifyOnFailure: true,
        notifyOnRecover: true,
    });

    const [nodeHealthMap, setNodeHealthMap] = useState<Record<string, ProxyHealth>>({});
    const [isDetectingTor, setIsDetectingTor] = useState(false);
    const [ipComparison, setIpComparison] = useState<IPComparison | null>(null);
    const [isComparingIPs, setIsComparingIPs] = useState(false);
    const debounceRef = useRef<NodeJS.Timeout | null>(null);
    const prevModeRef = useRef<string>(formData.mode);

    useEffect(() => {
        if (settings) {
            setFormData(settings);
            prevModeRef.current = settings.mode;
        }
    }, [settings]);

    useEffect(() => {
        if (state?.nodesHealth) {
            const map: Record<string, ProxyHealth> = {};
            state.nodesHealth.forEach((h) => {
                map[h.nodeId] = h;
            });
            setNodeHealthMap(map);
        }
    }, [state]);

    const handleChange = useCallback((updates: Partial<ProxySettingsType>, immediate: boolean = false) => {
        const newData = { ...formData, ...updates };
        setFormData(newData);

        if (debounceRef.current) {
            clearTimeout(debounceRef.current);
        }

        const doUpdate = async () => {
            await updateSettings(newData);
            // Auto-test when mode changes
            if (updates.mode && updates.mode !== prevModeRef.current && newData.nodes.length > 0) {
                prevModeRef.current = updates.mode;
                setTimeout(() => testAllNodes(), 500);
            }
        };

        if (immediate) {
            doUpdate();
        } else {
            debounceRef.current = setTimeout(doUpdate, 500);
        }
    }, [formData, updateSettings, testAllNodes]);

    const handleNodeUpdate = useCallback((index: number, node: ProxyNode) => {
        const sortedNodes = [...formData.nodes].sort((a, b) => a.order - b.order);
        sortedNodes[index] = node;
        handleChange({ nodes: sortedNodes }, true);
    }, [formData.nodes, handleChange]);

    const handleNodeDelete = useCallback((index: number) => {
        const sortedNodes = [...formData.nodes].sort((a, b) => a.order - b.order);
        const newNodes = sortedNodes.filter((_, i) => i !== index);
        // Reorder remaining nodes
        newNodes.forEach((n, i) => { n.order = i; });
        handleChange({ nodes: newNodes }, true);
    }, [formData.nodes, handleChange]);

    const handleAddNode = useCallback(() => {
        const newNode = createDefaultProxyNode("socks5");
        newNode.order = formData.nodes.length;
        handleChange({ nodes: [...formData.nodes, newNode] }, true);
    }, [formData.nodes, handleChange]);

    const handleAddTor = useCallback(async () => {
        setIsDetectingTor(true);
        try {
            const torResult = await detectTor();
            const node = await addTorNode();
            if (node) {
                if (torResult?.found) {
                    node.port = torResult.port;
                }
                node.order = formData.nodes.length;
                handleChange({ nodes: [...formData.nodes, node] }, true);
            }
        } finally {
            setIsDetectingTor(false);
        }
    }, [detectTor, addTorNode, formData.nodes, handleChange]);

    const handleTestNode = useCallback(async (node: ProxyNode) => {
        const health = await testNode(node);
        if (health) {
            setNodeHealthMap((prev) => ({
                ...prev,
                [node.id]: health,
            }));
        }
    }, [testNode]);

    const handleTestAll = useCallback(async () => {
        const results = await testAllNodes();
        const map: Record<string, ProxyHealth> = {};
        results.forEach((h) => {
            map[h.nodeId] = h;
        });
        setNodeHealthMap(map);
    }, [testAllNodes]);

    const handleCompareIPs = useCallback(async () => {
        setIsComparingIPs(true);
        setIpComparison(null);
        try {
            const result = await compareIPs();
            setIpComparison(result);
        } finally {
            setIsComparingIPs(false);
        }
    }, [compareIPs]);

    const sortedNodes = [...formData.nodes].sort((a, b) => a.order - b.order);
    const enabledNodes = sortedNodes.filter(n => n.enabled);

    if (isLoading && !settings) {
        return (
            <SettingsSection
                title="Proxy"
                icon={<RiShieldLine className="h-5 w-5" />}
            >
                <div className="space-y-4 animate-pulse">
                    <div className="h-4 bg-muted rounded w-3/4"></div>
                    <div className="h-10 bg-muted rounded"></div>
                    <div className="h-10 bg-muted rounded"></div>
                </div>
            </SettingsSection>
        );
    }

    return (
        <SettingsSection
            title="Proxy"
            icon={<RiShieldLine className="h-5 w-5" />}
        >
            <div className="space-y-6">
                {/* Enable toggle */}
                <div className="flex items-center justify-between">
                    <div className="space-y-1">
                        <Label htmlFor="proxy-enabled" className="text-base font-medium">
                            Enable Proxy
                        </Label>
                        <p className="text-sm text-muted-foreground">
                            Route all WordPress requests through a proxy for privacy
                        </p>
                    </div>
                    <div className="flex items-center gap-3">
                        {formData.enabled && state && getStatusBadge(state.status, false)}
                        <Switch
                            id="proxy-enabled"
                            checked={formData.enabled}
                            onCheckedChange={(checked) => handleChange({ enabled: checked }, true)}
                            disabled={isLoading}
                        />
                    </div>
                </div>

                {formData.enabled && (
                    <>
                        {/* Connection status */}
                        {state && state.status === "connected" && state.externalIp && (
                            <div className="flex items-center gap-3 p-3 bg-green-500/10 rounded-lg border border-green-500/20">
                                <RiCheckLine className="h-5 w-5 text-green-500 shrink-0" />
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-medium">Connected via proxy</p>
                                    <p className="text-sm text-muted-foreground truncate">
                                        Exit IP: {state.externalIp} • {state.latencyMs}ms
                                    </p>
                                </div>
                            </div>
                        )}

                        {state && state.status === "error" && state.lastError && (
                            <div className="flex items-center gap-3 p-3 bg-destructive/10 rounded-lg border border-destructive/20">
                                <RiCloseLine className="h-5 w-5 text-destructive shrink-0" />
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-medium text-destructive">Connection Error</p>
                                    <p className="text-sm text-muted-foreground truncate">{state.lastError}</p>
                                </div>
                            </div>
                        )}

                        {/* Mode selection with descriptions */}
                        <div className="space-y-3">
                            <div className="flex items-center justify-between">
                                <Label className="font-medium">Connection Mode</Label>
                                <Select
                                    value={formData.mode}
                                    onValueChange={(value) => handleChange({ mode: value as "single" | "chain" }, true)}
                                >
                                    <SelectTrigger className="w-32">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="single">
                                            <div className="flex items-center gap-2">
                                                <RiStackLine className="h-4 w-4" />
                                                Single
                                            </div>
                                        </SelectItem>
                                        <SelectItem value="chain">
                                            <div className="flex items-center gap-2">
                                                <RiLinkM className="h-4 w-4" />
                                                Chain
                                            </div>
                                        </SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>
                            <p className="text-xs text-muted-foreground">
                                {formData.mode === "single" ? (
                                    <>
                                        <span className="font-medium">Single:</span> Uses first enabled proxy.
                                        {enabledNodes.length > 1 && " Enable rotation to auto-switch between proxies."}
                                    </>
                                ) : (
                                    <>
                                        <span className="font-medium">Chain:</span> Routes traffic through all enabled proxies in sequence.
                                        Exit IP will be from the last node in the list.
                                    </>
                                )}
                            </p>
                        </div>


                        {/* Proxy nodes with timeline */}
                        <div className="space-y-3">
                            <div className="flex items-center justify-between">
                                <h4 className="font-medium">Proxy Nodes</h4>
                                <div className="flex gap-2">
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={handleAddTor}
                                        disabled={isDetectingTor || hasTorNode(formData.nodes)}
                                        title={hasTorNode(formData.nodes) ? "Tor node already exists" : undefined}
                                    >
                                        {isDetectingTor ? (
                                            <RiLoader4Line className="h-4 w-4 mr-1.5 animate-spin" />
                                        ) : (
                                            <RiAddLine className="h-4 w-4 mr-1.5" />
                                        )}
                                        Tor
                                    </Button>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={handleAddNode}
                                    >
                                        <RiAddLine className="h-4 w-4 mr-1.5" />
                                        Proxy
                                    </Button>
                                    {sortedNodes.length > 0 && (
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={handleTestAll}
                                            disabled={isTesting}
                                        >
                                            {isTesting ? (
                                                <RiLoader4Line className="h-4 w-4 mr-1.5 animate-spin" />
                                            ) : (
                                                <RiRefreshLine className="h-4 w-4 mr-1.5" />
                                            )}
                                            Test
                                        </Button>
                                    )}
                                </div>
                            </div>

                            <ProxyChainView
                                nodes={sortedNodes}
                                healthMap={nodeHealthMap}
                                mode={formData.mode}
                                onNodeUpdate={handleNodeUpdate}
                                onNodeDelete={handleNodeDelete}
                                onNodeTest={handleTestNode}
                                isTesting={isTesting}
                            />
                        </div>

                        {/* Anonymity test */}
                        <div className="space-y-3 p-4 border rounded-lg">
                            <div className="flex items-center justify-between">
                                <div>
                                    <h4 className="font-medium">Anonymity Test</h4>
                                    <p className="text-xs text-muted-foreground">
                                        Compare your real IP with proxy exit IP
                                    </p>
                                </div>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={handleCompareIPs}
                                    disabled={isComparingIPs}
                                >
                                    {isComparingIPs ? (
                                        <RiLoader4Line className="h-4 w-4 mr-1.5 animate-spin" />
                                    ) : (
                                        <RiRefreshLine className="h-4 w-4 mr-1.5" />
                                    )}
                                    Check
                                </Button>
                            </div>

                            {ipComparison && (
                                <div className="space-y-3">
                                    <div className="flex items-center gap-4">
                                        <div className="flex-1 p-2 bg-muted/50 rounded text-center">
                                            <p className="text-xs text-muted-foreground mb-1">Real IP</p>
                                            <p className="font-mono text-sm font-medium">
                                                {ipComparison.directError ? (
                                                    <span className="text-destructive text-xs">{ipComparison.directError}</span>
                                                ) : (
                                                    ipComparison.directIp || "—"
                                                )}
                                            </p>
                                        </div>
                                        <RiArrowRightLine className="h-4 w-4 text-muted-foreground shrink-0" />
                                        <div className="flex-1 p-2 bg-muted/50 rounded text-center">
                                            <p className="text-xs text-muted-foreground mb-1">Proxy IP</p>
                                            <p className="font-mono text-sm font-medium">
                                                {ipComparison.proxyError ? (
                                                    <span className="text-destructive text-xs">{ipComparison.proxyError}</span>
                                                ) : (
                                                    ipComparison.proxyIp || "—"
                                                )}
                                            </p>
                                        </div>
                                    </div>

                                    {ipComparison.directIp && ipComparison.proxyIp && (
                                        <div className={cn(
                                            "flex items-center justify-center gap-2 p-2 rounded text-sm",
                                            ipComparison.isAnonymous
                                                ? "bg-green-500/10 text-green-600"
                                                : "bg-destructive/10 text-destructive"
                                        )}>
                                            {ipComparison.isAnonymous ? (
                                                <>
                                                    <RiCheckLine className="h-4 w-4" />
                                                    <span>Anonymous — proxy is working correctly</span>
                                                </>
                                            ) : (
                                                <>
                                                    <RiAlertLine className="h-4 w-4" />
                                                    <span>Not anonymous — IPs match</span>
                                                </>
                                            )}
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>

                        {/* Rotation settings (single mode only) */}
                        {formData.mode === "single" && enabledNodes.length > 1 && (
                            <div className="space-y-3 p-4 border rounded-lg">
                                <div className="flex items-center justify-between">
                                    <div>
                                        <h4 className="font-medium">Auto-Rotation</h4>
                                        <p className="text-xs text-muted-foreground">
                                            Automatically switch between proxies
                                        </p>
                                    </div>
                                    <Switch
                                        checked={formData.rotationEnabled}
                                        onCheckedChange={(checked) => handleChange({ rotationEnabled: checked }, true)}
                                    />
                                </div>

                                {formData.rotationEnabled && (
                                    <div className="flex items-center justify-between">
                                        <Label className="text-sm">Interval</Label>
                                        <div className="flex items-center gap-2">
                                            <Input
                                                type="number"
                                                min={10}
                                                value={formData.rotationInterval}
                                                onChange={(e) => handleChange({
                                                    rotationInterval: parseInt(e.target.value) || 60
                                                })}
                                                className="w-20 h-8 text-right"
                                            />
                                            <span className="text-sm text-muted-foreground">sec</span>
                                        </div>
                                    </div>
                                )}
                            </div>
                        )}

                        {/* Health monitoring settings */}
                        <div className="space-y-3 p-4 border rounded-lg">
                            <div className="flex items-center justify-between">
                                <div>
                                    <h4 className="font-medium">Health Monitoring</h4>
                                    <p className="text-xs text-muted-foreground">
                                        Periodically check proxy connectivity
                                    </p>
                                </div>
                                <Switch
                                    checked={formData.healthCheckEnabled}
                                    onCheckedChange={(checked) => handleChange({ healthCheckEnabled: checked }, true)}
                                />
                            </div>

                            {formData.healthCheckEnabled && (
                                <>
                                    <div className="flex items-center justify-between">
                                        <Label className="text-sm">Check interval</Label>
                                        <div className="flex items-center gap-2">
                                            <Input
                                                type="number"
                                                min={10}
                                                value={formData.healthCheckInterval}
                                                onChange={(e) => handleChange({
                                                    healthCheckInterval: parseInt(e.target.value) || 60
                                                })}
                                                className="w-20 h-8 text-right"
                                            />
                                            <span className="text-sm text-muted-foreground">sec</span>
                                        </div>
                                    </div>

                                    <div className="flex items-center justify-between">
                                        <Label className="text-sm">Notify on failure</Label>
                                        <Switch
                                            checked={formData.notifyOnFailure}
                                            onCheckedChange={(checked) => handleChange({ notifyOnFailure: checked }, true)}
                                        />
                                    </div>

                                    <div className="flex items-center justify-between">
                                        <Label className="text-sm">Notify on recovery</Label>
                                        <Switch
                                            checked={formData.notifyOnRecover}
                                            onCheckedChange={(checked) => handleChange({ notifyOnRecover: checked }, true)}
                                        />
                                    </div>
                                </>
                            )}
                        </div>
                    </>
                )}
            </div>
        </SettingsSection>
    );
}
