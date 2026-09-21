import { useCallback, useSyncExternalStore } from "react";

import type { Viewport } from "../../canvas/viewport.js";
import type { FoldState, SortOrder } from "./model/fold.js";

export interface GraphSession {
    fold: FoldState | null;
    order: SortOrder;
    showRelated: boolean;
    minimap: boolean;
    queue: boolean;
}

const emptySession: GraphSession = Object.freeze({
    fold: null,
    order: "score",
    showRelated: false,
    minimap: true,
    queue: false,
});

const sessions = new Map<string, GraphSession>();
const sessionListeners = new Set<() => void>();

function sessionOf(siteId: string): GraphSession {
    return sessions.get(siteId) ?? emptySession;
}

function subscribeSessions(listener: () => void): () => void {
    sessionListeners.add(listener);
    return () => {
        sessionListeners.delete(listener);
    };
}

export function patchSession(siteId: string, patch: Partial<GraphSession>): void {
    sessions.set(siteId, Object.freeze({ ...sessionOf(siteId), ...patch }));
    for (const listener of sessionListeners) {
        listener();
    }
}

export function useGraphSession(siteId: string): [GraphSession, (patch: Partial<GraphSession>) => void] {
    const session = useSyncExternalStore(subscribeSessions, () => sessionOf(siteId));
    const patch = useCallback(
        (next: Partial<GraphSession>) => {
            patchSession(siteId, next);
        },
        [siteId],
    );
    return [session, patch];
}

const viewports = new Map<string, Viewport>();
const viewportListeners = new Set<() => void>();

export function viewportOf(siteId: string): Viewport | null {
    return viewports.get(siteId) ?? null;
}

export function setViewport(siteId: string, view: Viewport): void {
    viewports.set(siteId, view);
    for (const listener of viewportListeners) {
        listener();
    }
}

function subscribeViewports(listener: () => void): () => void {
    viewportListeners.add(listener);
    return () => {
        viewportListeners.delete(listener);
    };
}

export function useViewport(siteId: string): Viewport | null {
    return useSyncExternalStore(subscribeViewports, () => viewportOf(siteId));
}

export function useZoomPercent(siteId: string): number {
    return useSyncExternalStore(subscribeViewports, () => Math.round((viewportOf(siteId)?.k ?? 1) * 100));
}
