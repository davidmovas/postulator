import { useState, useCallback, useEffect } from "react";
import { sitemapService } from "@/services/sitemaps";

// HistoryState returned from Go backend
export interface HistoryState {
    canUndo: boolean;
    canRedo: boolean;
    undoCount: number;
    redoCount: number;
    lastAction?: string;
    actionApplied?: string;
}

interface UseSitemapHistoryOptions {
    sitemapId: number;
}

export function useSitemapHistory(options: UseSitemapHistoryOptions) {
    const { sitemapId } = options;

    const [state, setState] = useState<HistoryState>({
        canUndo: false,
        canRedo: false,
        undoCount: 0,
        redoCount: 0,
    });
    const [isApplying, setIsApplying] = useState(false);

    // Load initial history state
    const loadHistoryState = useCallback(async () => {
        if (!sitemapId) return;
        try {
            const result = await sitemapService.getHistoryState(sitemapId);
            if (result) {
                setState(result);
            }
        } catch {
            // Error handled silently - history state will remain empty
        }
    }, [sitemapId]);

    // Load state on mount and when sitemapId changes
    useEffect(() => {
        loadHistoryState();
    }, [loadHistoryState]);

    // Clear history when component unmounts (editor closes)
    useEffect(() => {
        return () => {
            if (sitemapId) {
                sitemapService.clearHistory(sitemapId).catch(() => {});
            }
        };
    }, [sitemapId]);

    // Undo the last action
    const undo = useCallback(async (): Promise<boolean> => {
        if (!state.canUndo || isApplying || !sitemapId) return false;

        setIsApplying(true);
        try {
            const result = await sitemapService.undo(sitemapId);
            if (result) {
                setState(result);
                return true;
            }
            return false;
        } catch {
            return false;
        } finally {
            setIsApplying(false);
        }
    }, [state.canUndo, isApplying, sitemapId]);

    // Redo the next action
    const redo = useCallback(async (): Promise<boolean> => {
        if (!state.canRedo || isApplying || !sitemapId) return false;

        setIsApplying(true);
        try {
            const result = await sitemapService.redo(sitemapId);
            if (result) {
                setState(result);
                return true;
            }
            return false;
        } catch {
            return false;
        } finally {
            setIsApplying(false);
        }
    }, [state.canRedo, isApplying, sitemapId]);

    // Refresh state after actions that may affect history
    // Call this after operations that modify nodes
    const refreshState = useCallback(async () => {
        await loadHistoryState();
    }, [loadHistoryState]);

    // Clear all history
    const clear = useCallback(async () => {
        if (!sitemapId) return;
        try {
            await sitemapService.clearHistory(sitemapId);
            setState({
                canUndo: false,
                canRedo: false,
                undoCount: 0,
                redoCount: 0,
            });
        } catch {
            // Error handled silently
        }
    }, [sitemapId]);

    return {
        // State
        canUndo: state.canUndo && !isApplying,
        canRedo: state.canRedo && !isApplying,
        isApplying,
        historyLength: state.undoCount,
        futureLength: state.redoCount,
        lastAction: state.lastAction,

        // Actions
        undo,
        redo,
        clear,
        refreshState,
    };
}
