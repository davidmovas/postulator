"use client";

import { ProxyNode, ProxyHealth, ProxyMode } from "@/models/proxy";
import {
    Timeline,
    TimelineItem,
    TimelineIndicator,
    TimelineSeparator,
    TimelineContent,
    TimelineHeader,
} from "@/components/ui/timeline";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    RiComputerLine,
    RiGlobalLine,
    RiCheckLine,
    RiCloseLine,
    RiLoader4Line,
    RiArrowRightSLine,
    RiArrowDownSLine,
    RiDeleteBinLine,
    RiEyeLine,
    RiEyeOffLine,
    RiTimeLine,
    RiRefreshLine,
} from "@remixicon/react";
import { cn } from "@/lib/utils";
import { useState, useCallback, useRef, useEffect } from "react";

interface ProxyChainViewProps {
    nodes: ProxyNode[];
    healthMap: Record<string, ProxyHealth>;
    mode: ProxyMode;
    onNodeUpdate: (index: number, node: ProxyNode) => void;
    onNodeDelete: (index: number) => void;
    onNodeTest: (node: ProxyNode) => void;
    isTesting: boolean;
}

function getNodeStatus(health?: ProxyHealth) {
    if (!health) return "pending";
    return health.status;
}

function NodeCard({
    node,
    index,
    health,
    isLast,
    mode,
    onUpdate,
    onDelete,
    onTest,
    isTesting,
    canDelete,
}: {
    node: ProxyNode;
    index: number;
    health?: ProxyHealth;
    isLast: boolean;
    mode: ProxyMode;
    onUpdate: (node: ProxyNode) => void;
    onDelete: () => void;
    onTest: () => void;
    isTesting: boolean;
    canDelete: boolean;
}) {
    const [isOpen, setIsOpen] = useState(false);
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

    const status = getNodeStatus(health);
    const isConnected = status === "connected";
    const isError = status === "error";

    const getTypeLabel = (type: string) => {
        switch (type) {
            case "tor": return "Tor";
            case "socks5": return "SOCKS5";
            case "http": return "HTTP";
            default: return type;
        }
    };

    const getStatusIcon = () => {
        if (isTesting) return <RiLoader4Line className="h-3 w-3 animate-spin" />;
        if (isConnected) return <RiCheckLine className="h-3 w-3" />;
        if (isError) return <RiCloseLine className="h-3 w-3" />;
        return <span className="text-xs font-bold">{index + 1}</span>;
    };

    return (
        <TimelineItem isLast={isLast}>
            <TimelineIndicator
                isCompleted={isConnected && node.enabled}
                isError={isError && node.enabled}
                isActive={!isConnected && !isError && node.enabled}
            >
                {getStatusIcon()}
            </TimelineIndicator>
            <TimelineSeparator isCompleted={isConnected && node.enabled} isLast={isLast} />
            <TimelineContent>
                <Collapsible open={isOpen} onOpenChange={setIsOpen}>
                    <div className={cn(
                        "border rounded-lg transition-all",
                        !node.enabled && "opacity-50",
                        isOpen ? "bg-muted/30" : "hover:bg-muted/20"
                    )}>
                        <CollapsibleTrigger asChild>
                            <div className="flex items-center justify-between p-3 cursor-pointer">
                                <div className="flex items-center gap-3">
                                    <div className="flex flex-col">
                                        <div className="flex items-center gap-2">
                                            <Badge variant="outline" className="text-xs">
                                                {getTypeLabel(node.type)}
                                            </Badge>
                                            <span className="font-medium text-sm">
                                                {node.host}:{node.port}
                                            </span>
                                        </div>
                                        {health && isConnected && (
                                            <div className="flex items-center gap-3 text-xs text-muted-foreground mt-1">
                                                <span className="flex items-center gap-1">
                                                    <RiGlobalLine className="h-3 w-3" />
                                                    {health.externalIp}
                                                </span>
                                                <span className="flex items-center gap-1">
                                                    <RiTimeLine className="h-3 w-3" />
                                                    {health.latencyMs}ms
                                                </span>
                                            </div>
                                        )}
                                        {health && isError && health.error && (
                                            <span className="text-xs text-destructive mt-1">
                                                {health.error}
                                            </span>
                                        )}
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
                                        onClick={(e) => e.stopPropagation()}
                                    />
                                    <RiArrowDownSLine className={cn(
                                        "h-4 w-4 text-muted-foreground transition-transform",
                                        isOpen && "rotate-180"
                                    )} />
                                </div>
                            </div>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                            <div className="px-3 pb-3 pt-0 space-y-3 border-t">
                                <div className="grid grid-cols-2 gap-3 pt-3">
                                    <div className="space-y-1.5">
                                        <Label className="text-xs">Type</Label>
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
                                            <SelectTrigger className="h-8">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="socks5">SOCKS5</SelectItem>
                                                <SelectItem value="http">HTTP</SelectItem>
                                                <SelectItem value="tor">Tor</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                    <div className="space-y-1.5">
                                        <Label className="text-xs">Host</Label>
                                        <Input
                                            value={localNode.host}
                                            onChange={(e) => handleChange({ host: e.target.value })}
                                            placeholder="127.0.0.1"
                                            className="h-8"
                                        />
                                    </div>
                                    <div className="space-y-1.5">
                                        <Label className="text-xs">Port</Label>
                                        <Input
                                            type="number"
                                            value={localNode.port}
                                            onChange={(e) => handleChange({ port: parseInt(e.target.value) || 0 })}
                                            min={1}
                                            max={65535}
                                            className="h-8"
                                        />
                                    </div>
                                    <div className="space-y-1.5">
                                        <Label className="text-xs">Username</Label>
                                        <Input
                                            value={localNode.username || ""}
                                            onChange={(e) => handleChange({ username: e.target.value })}
                                            placeholder="optional"
                                            className="h-8"
                                        />
                                    </div>
                                    <div className="col-span-2 space-y-1.5">
                                        <Label className="text-xs">Password</Label>
                                        <div className="relative">
                                            <Input
                                                type={showPassword ? "text" : "password"}
                                                value={localNode.password || ""}
                                                onChange={(e) => handleChange({ password: e.target.value })}
                                                placeholder="optional"
                                                className="h-8 pr-8"
                                            />
                                            <Button
                                                type="button"
                                                variant="ghost"
                                                size="icon"
                                                className="absolute right-0 top-0 h-8 w-8"
                                                onClick={() => setShowPassword(!showPassword)}
                                            >
                                                {showPassword ? (
                                                    <RiEyeOffLine className="h-3 w-3" />
                                                ) : (
                                                    <RiEyeLine className="h-3 w-3" />
                                                )}
                                            </Button>
                                        </div>
                                    </div>
                                </div>
                                <div className="flex items-center justify-end gap-2 pt-2">
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={onTest}
                                        disabled={isTesting}
                                    >
                                        {isTesting ? (
                                            <RiLoader4Line className="h-3 w-3 mr-1 animate-spin" />
                                        ) : (
                                            <RiRefreshLine className="h-3 w-3 mr-1" />
                                        )}
                                        Test
                                    </Button>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={onDelete}
                                        disabled={!canDelete}
                                        className="text-destructive hover:text-destructive"
                                    >
                                        <RiDeleteBinLine className="h-3 w-3 mr-1" />
                                        Remove
                                    </Button>
                                </div>
                            </div>
                        </CollapsibleContent>
                    </div>
                </Collapsible>
            </TimelineContent>
        </TimelineItem>
    );
}

export function ProxyChainView({
    nodes,
    healthMap,
    mode,
    onNodeUpdate,
    onNodeDelete,
    onNodeTest,
    isTesting,
}: ProxyChainViewProps) {
    const sortedNodes = [...nodes].sort((a, b) => a.order - b.order);

    return (
        <div className="space-y-4">
            {/* Chain visualization header */}
            <div className="flex items-center gap-2 text-xs text-muted-foreground px-1">
                <RiComputerLine className="h-4 w-4" />
                <span>App</span>
                {sortedNodes.length > 0 && (
                    <>
                        <RiArrowRightSLine className="h-4 w-4" />
                        {mode === "chain" ? (
                            <span className="text-primary">Chain through {sortedNodes.filter(n => n.enabled).length} node{sortedNodes.filter(n => n.enabled).length !== 1 ? "s" : ""}</span>
                        ) : (
                            <span>Single proxy</span>
                        )}
                        <RiArrowRightSLine className="h-4 w-4" />
                        <RiGlobalLine className="h-4 w-4" />
                        <span>Target</span>
                    </>
                )}
            </div>

            {/* Nodes timeline */}
            <Timeline>
                {sortedNodes.map((node, index) => (
                    <NodeCard
                        key={node.id}
                        node={node}
                        index={index}
                        health={healthMap[node.id]}
                        isLast={index === sortedNodes.length - 1}
                        mode={mode}
                        onUpdate={(n) => onNodeUpdate(index, n)}
                        onDelete={() => onNodeDelete(index)}
                        onTest={() => onNodeTest(node)}
                        isTesting={isTesting}
                        canDelete={sortedNodes.length > 1 || !node.enabled}
                    />
                ))}
            </Timeline>

            {sortedNodes.length === 0 && (
                <div className="text-center py-6 text-muted-foreground text-sm">
                    No proxy nodes configured. Add a Tor or custom proxy to get started.
                </div>
            )}
        </div>
    );
}
