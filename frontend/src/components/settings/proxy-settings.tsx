"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { SettingsSection } from "./settings-section";
import { useProxySettings } from "@/hooks/use-proxy-settings";
import { ProxyNode, ProxySettings as ProxySettingsType, ProxyHealth, IPComparison, createDefaultProxyNode, hasTorNode } from "@/models/proxy";
import {
    RiShieldLine,
    RiAddLine,
    RiDeleteBinLine,
    RiRefreshLine,
    RiCheckLine,
    RiCloseLine,
    RiLoader4Line,
    RiGlobalLine,
    RiTimeLine,
    RiDragMoveLine,
    RiEyeLine,
    RiEyeOffLine,
    RiAlertLine,
    RiSpyLine,
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

function ProxyNodeCard({
    node,
    health,
    onUpdate,
    onDelete,
    onTest,
    isTesting,
    isOnly,
}: {
    node: ProxyNode;
    health?: ProxyHealth;
    onUpdate: (node: ProxyNode) => void;
    onDelete: () => void;
    onTest: () => void;
    isTesting: boolean;
    isOnly: boolean;
}) {
    const [showPassword, setShowPassword] = useState(false);
    const [localNode, setLocalNode] = useState(node);
    const debounceRef = useRef<NodeJS.Timeout | null>(null);

    useEffect(() => {
        setLocalNode(node);
    }, [node]);

    const handleChange = useCallback((updates: Partial<ProxyNode>) => {
        const updated = { ...localNode, ...updates };
        setLocalNode(updated);

        if (debounceRef.current) {
            clearTimeout(debounceRef.current);
        }
        debounceRef.current = setTimeout(() => {
            onUpdate(updated);
        }, 500);
    }, [localNode, onUpdate]);

    const getTypeLabel = (type: string) => {
        switch (type) {
            case "tor": return "Tor";
            case "socks5": return "SOCKS5";
            case "http": return "HTTP";
            default: return type;
        }
    };

    return (
        <div className={cn(
            "border rounded-lg p-4 space-y-4",
            !node.enabled && "opacity-60"
        )}>
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <RiDragMoveLine className="h-4 w-4 text-muted-foreground cursor-move" />
                    <div className="flex items-center gap-2">
                        <Badge variant="outline">{getTypeLabel(node.type)}</Badge>
                        {health && getStatusBadge(health.status, false)}
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <Switch
                        checked={localNode.enabled}
                        onCheckedChange={(checked) => {
                            const updated = { ...localNode, enabled: checked };
                            setLocalNode(updated);
                            onUpdate(updated);
                        }}
                    />
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={onTest}
                        disabled={isTesting}
                    >
                        {isTesting ? (
                            <RiLoader4Line className="h-4 w-4 animate-spin" />
                        ) : (
                            <RiRefreshLine className="h-4 w-4" />
                        )}
                    </Button>
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={onDelete}
                        disabled={isOnly}
                        className="text-destructive hover:text-destructive"
                    >
                        <RiDeleteBinLine className="h-4 w-4" />
                    </Button>
                </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                    <Label>Type</Label>
                    <Select
                        value={localNode.type}
                        onValueChange={(value) => {
                            const updated = {
                                ...localNode,
                                type: value as ProxyNode["type"],
                                port: value === "tor" ? 9050 : localNode.port,
                            };
                            setLocalNode(updated);
                            onUpdate(updated);
                        }}
                    >
                        <SelectTrigger>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="socks5">SOCKS5</SelectItem>
                            <SelectItem value="http">HTTP</SelectItem>
                            <SelectItem value="tor">Tor</SelectItem>
                        </SelectContent>
                    </Select>
                </div>

                <div className="space-y-2">
                    <Label>Host</Label>
                    <Input
                        value={localNode.host}
                        onChange={(e) => handleChange({ host: e.target.value })}
                        placeholder="127.0.0.1"
                    />
                </div>

                <div className="space-y-2">
                    <Label>Port</Label>
                    <Input
                        type="number"
                        value={localNode.port}
                        onChange={(e) => handleChange({ port: parseInt(e.target.value) || 0 })}
                        min={1}
                        max={65535}
                    />
                </div>

                <div className="space-y-2">
                    <Label>Username (optional)</Label>
                    <Input
                        value={localNode.username || ""}
                        onChange={(e) => handleChange({ username: e.target.value })}
                        placeholder="username"
                    />
                </div>

                <div className="col-span-2 space-y-2">
                    <Label>Password (optional)</Label>
                    <div className="relative">
                        <Input
                            type={showPassword ? "text" : "password"}
                            value={localNode.password || ""}
                            onChange={(e) => handleChange({ password: e.target.value })}
                            placeholder="password"
                            className="pr-10"
                        />
                        <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="absolute right-0 top-0 h-full"
                            onClick={() => setShowPassword(!showPassword)}
                        >
                            {showPassword ? (
                                <RiEyeOffLine className="h-4 w-4" />
                            ) : (
                                <RiEyeLine className="h-4 w-4" />
                            )}
                        </Button>
                    </div>
                </div>
            </div>

            {health && health.status === "connected" && (
                <div className="flex items-center gap-4 text-sm text-muted-foreground pt-2 border-t">
                    <div className="flex items-center gap-1">
                        <RiGlobalLine className="h-4 w-4" />
                        <span>IP: {health.externalIp || "Unknown"}</span>
                    </div>
                    <div className="flex items-center gap-1">
                        <RiTimeLine className="h-4 w-4" />
                        <span>{health.latencyMs}ms</span>
                    </div>
                </div>
            )}

            {health && health.status === "error" && health.error && (
                <div className="flex items-center gap-2 text-sm text-destructive pt-2 border-t">
                    <RiAlertLine className="h-4 w-4" />
                    <span>{health.error}</span>
                </div>
            )}
        </div>
    );
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

    useEffect(() => {
        if (settings) {
            setFormData(settings);
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

        if (immediate) {
            updateSettings(newData);
        } else {
            debounceRef.current = setTimeout(() => {
                updateSettings(newData);
            }, 500);
        }
    }, [formData, updateSettings]);

    const handleNodeUpdate = useCallback((index: number, node: ProxyNode) => {
        const newNodes = [...formData.nodes];
        newNodes[index] = node;
        handleChange({ nodes: newNodes }, true);
    }, [formData.nodes, handleChange]);

    const handleNodeDelete = useCallback((index: number) => {
        const newNodes = formData.nodes.filter((_, i) => i !== index);
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

                {state && state.status === "connected" && state.externalIp && (
                    <div className="flex items-center gap-4 p-3 bg-green-500/10 rounded-lg border border-green-500/20">
                        <RiCheckLine className="h-5 w-5 text-green-500" />
                        <div className="flex-1">
                            <p className="text-sm font-medium">Connected via proxy</p>
                            <p className="text-sm text-muted-foreground">
                                External IP: {state.externalIp} | Latency: {state.latencyMs}ms
                            </p>
                        </div>
                    </div>
                )}

                {state && state.status === "error" && state.lastError && (
                    <div className="flex items-center gap-4 p-3 bg-destructive/10 rounded-lg border border-destructive/20">
                        <RiCloseLine className="h-5 w-5 text-destructive" />
                        <div className="flex-1">
                            <p className="text-sm font-medium text-destructive">Connection Error</p>
                            <p className="text-sm text-muted-foreground">{state.lastError}</p>
                        </div>
                    </div>
                )}

                {formData.enabled && (
                    <div className="space-y-3 p-4 border rounded-lg bg-muted/30">
                        <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                                <RiSpyLine className="h-5 w-5 text-muted-foreground" />
                                <span className="font-medium">Anonymity Test</span>
                            </div>
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={handleCompareIPs}
                                disabled={isComparingIPs}
                            >
                                {isComparingIPs ? (
                                    <RiLoader4Line className="h-4 w-4 mr-2 animate-spin" />
                                ) : (
                                    <RiRefreshLine className="h-4 w-4 mr-2" />
                                )}
                                Check IPs
                            </Button>
                        </div>

                        {ipComparison && (
                            <div className="space-y-2">
                                <div className="grid grid-cols-2 gap-4 text-sm">
                                    <div className="space-y-1">
                                        <p className="text-muted-foreground">Your Real IP:</p>
                                        <p className="font-mono font-medium">
                                            {ipComparison.directError ? (
                                                <span className="text-destructive">{ipComparison.directError}</span>
                                            ) : (
                                                ipComparison.directIp || "—"
                                            )}
                                        </p>
                                    </div>
                                    <div className="space-y-1">
                                        <p className="text-muted-foreground">Proxy IP:</p>
                                        <p className="font-mono font-medium">
                                            {ipComparison.proxyError ? (
                                                <span className="text-destructive">{ipComparison.proxyError}</span>
                                            ) : (
                                                ipComparison.proxyIp || "—"
                                            )}
                                        </p>
                                    </div>
                                </div>

                                {ipComparison.directIp && ipComparison.proxyIp && (
                                    <div className={cn(
                                        "flex items-center gap-2 p-2 rounded-md text-sm",
                                        ipComparison.isAnonymous
                                            ? "bg-green-500/10 text-green-600"
                                            : "bg-destructive/10 text-destructive"
                                    )}>
                                        {ipComparison.isAnonymous ? (
                                            <>
                                                <RiCheckLine className="h-4 w-4" />
                                                <span>Anonymous - IPs are different, proxy is working!</span>
                                            </>
                                        ) : (
                                            <>
                                                <RiAlertLine className="h-4 w-4" />
                                                <span>Not anonymous - IPs are the same, proxy may not be working</span>
                                            </>
                                        )}
                                    </div>
                                )}
                            </div>
                        )}
                    </div>
                )}


                {formData.enabled && (
                    <>
                        <div className="flex items-center justify-between">
                            <div className="space-y-1">
                                <Label className="font-medium">Connection Mode</Label>
                                <p className="text-sm text-muted-foreground">
                                    How to use multiple proxies
                                </p>
                            </div>
                            <Select
                                value={formData.mode}
                                onValueChange={(value) => handleChange({ mode: value as "single" | "chain" }, true)}
                            >
                                <SelectTrigger className="w-40">
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="single">Single</SelectItem>
                                    <SelectItem value="chain">Chain</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>

                        <div className="space-y-4 border-t pt-4">
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
                                            <RiLoader4Line className="h-4 w-4 mr-2 animate-spin" />
                                        ) : (
                                            <RiAddLine className="h-4 w-4 mr-2" />
                                        )}
                                        Add Tor
                                    </Button>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={handleAddNode}
                                    >
                                        <RiAddLine className="h-4 w-4 mr-2" />
                                        Add Proxy
                                    </Button>
                                    {formData.nodes.length > 0 && (
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={handleTestAll}
                                            disabled={isTesting}
                                        >
                                            {isTesting ? (
                                                <RiLoader4Line className="h-4 w-4 mr-2 animate-spin" />
                                            ) : (
                                                <RiRefreshLine className="h-4 w-4 mr-2" />
                                            )}
                                            Test All
                                        </Button>
                                    )}
                                </div>
                            </div>

                            {formData.nodes.length === 0 ? (
                                <div className="text-center py-8 border rounded-lg border-dashed">
                                    <RiShieldLine className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
                                    <p className="text-sm text-muted-foreground">
                                        No proxy nodes configured. Add a Tor or custom proxy to get started.
                                    </p>
                                </div>
                            ) : (
                                <div className="space-y-4">
                                    {formData.nodes.map((node, index) => (
                                        <ProxyNodeCard
                                            key={node.id}
                                            node={node}
                                            health={nodeHealthMap[node.id]}
                                            onUpdate={(n) => handleNodeUpdate(index, n)}
                                            onDelete={() => handleNodeDelete(index)}
                                            onTest={() => handleTestNode(node)}
                                            isTesting={isTesting}
                                            isOnly={formData.nodes.length === 1}
                                        />
                                    ))}
                                </div>
                            )}
                        </div>

                        {formData.mode === "single" && formData.nodes.length > 1 && (
                            <div className="space-y-4 border-t pt-4">
                                <div className="flex items-center justify-between">
                                    <div className="space-y-1">
                                        <Label className="font-medium">Auto-Rotation</Label>
                                        <p className="text-sm text-muted-foreground">
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
                                        <div className="space-y-1">
                                            <Label className="font-normal">Rotation Interval</Label>
                                            <p className="text-sm text-muted-foreground">
                                                How often to switch proxies
                                            </p>
                                        </div>
                                        <div className="flex items-center gap-3">
                                            <Input
                                                type="number"
                                                min={10}
                                                value={formData.rotationInterval}
                                                onChange={(e) => handleChange({
                                                    rotationInterval: parseInt(e.target.value) || 60
                                                })}
                                                className="w-20 text-right"
                                            />
                                            <span className="text-sm text-muted-foreground w-8">sec</span>
                                        </div>
                                    </div>
                                )}
                            </div>
                        )}

                        <div className="space-y-4 border-t pt-4">
                            <h4 className="font-medium">Health Monitoring</h4>

                            <div className="flex items-center justify-between">
                                <div className="space-y-1">
                                    <Label className="font-normal">Auto Health Check</Label>
                                    <p className="text-sm text-muted-foreground">
                                        Periodically check proxy connectivity
                                    </p>
                                </div>
                                <Switch
                                    checked={formData.healthCheckEnabled}
                                    onCheckedChange={(checked) => handleChange({ healthCheckEnabled: checked }, true)}
                                />
                            </div>

                            {formData.healthCheckEnabled && (
                                <div className="flex items-center justify-between">
                                    <div className="space-y-1">
                                        <Label className="font-normal">Check Interval</Label>
                                        <p className="text-sm text-muted-foreground">
                                            How often to verify proxy health
                                        </p>
                                    </div>
                                    <div className="flex items-center gap-3">
                                        <Input
                                            type="number"
                                            min={10}
                                            value={formData.healthCheckInterval}
                                            onChange={(e) => handleChange({
                                                healthCheckInterval: parseInt(e.target.value) || 60
                                            })}
                                            className="w-20 text-right"
                                        />
                                        <span className="text-sm text-muted-foreground w-8">sec</span>
                                    </div>
                                </div>
                            )}

                            <div className="flex items-center justify-between">
                                <div className="space-y-1">
                                    <Label className="font-normal">Notify on Failure</Label>
                                    <p className="text-sm text-muted-foreground">
                                        Show notification when proxy connection fails
                                    </p>
                                </div>
                                <Switch
                                    checked={formData.notifyOnFailure}
                                    onCheckedChange={(checked) => handleChange({ notifyOnFailure: checked }, true)}
                                />
                            </div>

                            <div className="flex items-center justify-between">
                                <div className="space-y-1">
                                    <Label className="font-normal">Notify on Recovery</Label>
                                    <p className="text-sm text-muted-foreground">
                                        Show notification when proxy reconnects
                                    </p>
                                </div>
                                <Switch
                                    checked={formData.notifyOnRecover}
                                    onCheckedChange={(checked) => handleChange({ notifyOnRecover: checked }, true)}
                                />
                            </div>
                        </div>
                    </>
                )}
            </div>
        </SettingsSection>
    );
}
