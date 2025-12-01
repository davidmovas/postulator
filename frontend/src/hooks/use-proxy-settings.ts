"use client";

import { useState, useEffect, useCallback } from "react";
import { proxyService } from "@/services/proxy";
import { ProxySettings, ProxyState, ProxyNode, TorDetectionResult, ProxyHealth, IPComparison } from "@/models/proxy";
import { useToast } from "@/components/ui/use-toast";

export function useProxySettings() {
    const [settings, setSettings] = useState<ProxySettings | null>(null);
    const [state, setState] = useState<ProxyState | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [isSaving, setIsSaving] = useState(false);
    const [isTesting, setIsTesting] = useState(false);
    const { toast } = useToast();

    const loadSettings = useCallback(async () => {
        try {
            const [settingsData, stateData] = await Promise.all([
                proxyService.getSettings(),
                proxyService.getState(),
            ]);
            setSettings(settingsData);
            setState(stateData);
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to load proxy settings",
                variant: "destructive",
            });
        } finally {
            setIsLoading(false);
        }
    }, [toast]);

    useEffect(() => {
        loadSettings();
    }, [loadSettings]);

    const updateSettings = useCallback(async (newSettings: ProxySettings) => {
        setIsSaving(true);
        try {
            await proxyService.updateSettings(newSettings);
            setSettings(newSettings);
            const newState = await proxyService.getState();
            setState(newState);
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to update proxy settings",
                variant: "destructive",
            });
        } finally {
            setIsSaving(false);
        }
    }, [toast]);

    const testNode = useCallback(async (node: ProxyNode): Promise<ProxyHealth | null> => {
        setIsTesting(true);
        try {
            const health = await proxyService.testNode(node);
            return health;
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to test proxy node",
                variant: "destructive",
            });
            return null;
        } finally {
            setIsTesting(false);
        }
    }, [toast]);

    const testAllNodes = useCallback(async (): Promise<ProxyHealth[]> => {
        setIsTesting(true);
        try {
            const health = await proxyService.testAllNodes();
            const newState = await proxyService.getState();
            setState(newState);
            return health;
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to test proxy nodes",
                variant: "destructive",
            });
            return [];
        } finally {
            setIsTesting(false);
        }
    }, [toast]);

    const detectTor = useCallback(async (): Promise<TorDetectionResult | null> => {
        try {
            const result = await proxyService.detectTor();
            return result;
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to detect Tor",
                variant: "destructive",
            });
            return null;
        }
    }, [toast]);

    const addTorNode = useCallback(async (): Promise<ProxyNode | null> => {
        try {
            const node = await proxyService.addTorNode();
            return node;
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to add Tor node",
                variant: "destructive",
            });
            return null;
        }
    }, [toast]);

    const enable = useCallback(async () => {
        try {
            await proxyService.enable();
            await loadSettings();
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to enable proxy",
                variant: "destructive",
            });
        }
    }, [loadSettings, toast]);

    const disable = useCallback(async () => {
        try {
            await proxyService.disable();
            await loadSettings();
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to disable proxy",
                variant: "destructive",
            });
        }
    }, [loadSettings, toast]);

    const refreshState = useCallback(async () => {
        try {
            const newState = await proxyService.getState();
            setState(newState);
        } catch (error) {
            console.error("Failed to refresh proxy state:", error);
        }
    }, []);

    const compareIPs = useCallback(async (): Promise<IPComparison | null> => {
        try {
            const result = await proxyService.compareIPs();
            return result;
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to compare IPs",
                variant: "destructive",
            });
            return null;
        }
    }, [toast]);

    return {
        settings,
        state,
        isLoading,
        isSaving,
        isTesting,
        updateSettings,
        testNode,
        testAllNodes,
        detectTor,
        addTorNode,
        enable,
        disable,
        refreshState,
        compareIPs,
        reload: loadSettings,
    };
}
