"use client";

import { useEffect, useState } from "react";
import { proxyService } from "@/services/proxy";
import { ProxyState } from "@/models/proxy";
import { RiShieldLine, RiShieldCheckLine, RiLoader4Line } from "@remixicon/react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

export function ProxyStatusIndicator() {
    const [state, setState] = useState<ProxyState | null>(null);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        const loadState = async () => {
            try {
                const proxyState = await proxyService.getState();
                setState(proxyState);
            } catch (error) {
                console.error("Failed to load proxy state:", error);
            } finally {
                setIsLoading(false);
            }
        };

        loadState();

        const interval = setInterval(loadState, 30000);
        return () => clearInterval(interval);
    }, []);

    if (isLoading) {
        return null;
    }

    if (!state || state.status === "disabled") {
        return null;
    }

    const getStatusInfo = () => {
        switch (state.status) {
            case "connected":
                return {
                    icon: RiShieldCheckLine,
                    color: "text-green-500",
                    bgColor: "bg-green-500/10",
                    label: "Proxy Active",
                    description: state.externalIp
                        ? `Connected via ${state.externalIp} (${state.latencyMs}ms)`
                        : "Connected",
                };
            case "connecting":
                return {
                    icon: RiLoader4Line,
                    color: "text-yellow-500",
                    bgColor: "bg-yellow-500/10",
                    label: "Connecting",
                    description: "Establishing proxy connection...",
                    animate: true,
                };
            case "error":
                return {
                    icon: RiShieldLine,
                    color: "text-red-500",
                    bgColor: "bg-red-500/10",
                    label: "Proxy Error",
                    description: state.lastError || "Connection failed",
                };
            case "tor_not_found":
                return {
                    icon: RiShieldLine,
                    color: "text-red-500",
                    bgColor: "bg-red-500/10",
                    label: "Tor Not Found",
                    description: "Please start Tor Browser or Tor service",
                };
            default:
                return null;
        }
    };

    const info = getStatusInfo();
    if (!info) return null;

    const Icon = info.icon;

    return (
        <TooltipProvider>
            <Tooltip>
                <TooltipTrigger asChild>
                    <div
                        className={cn(
                            "flex items-center gap-2 px-2 py-1 rounded-md cursor-default",
                            info.bgColor
                        )}
                    >
                        <Icon
                            className={cn(
                                "h-4 w-4",
                                info.color,
                                info.animate && "animate-spin"
                            )}
                        />
                        <span className={cn("text-xs font-medium", info.color)}>
                            {info.label}
                        </span>
                    </div>
                </TooltipTrigger>
                <TooltipContent side="bottom">
                    <p>{info.description}</p>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
    );
}
