import { useSyncExternalStore } from "react";

import type { Selection } from "../../../ui/selection.js";
import { noSelection } from "../../../ui/selection.js";

export const pagesViews = ["list", "tree", "types"] as const;

export type PagesView = (typeof pagesViews)[number];

export const groupModes = ["list", "tree"] as const;

export type GroupMode = (typeof groupModes)[number];

export interface PagesMemory {
    view: PagesView;
    selections: Readonly<Record<PagesView, Selection>>;
    openType: string | null;
    groupMode: GroupMode;
}

export const blankPages: PagesMemory = Object.freeze({
    view: "list",
    selections: Object.freeze({ list: noSelection, tree: noSelection, types: noSelection }),
    openType: null,
    groupMode: "list",
});

const memories = new Map<string, PagesMemory>();
const listeners = new Set<() => void>();

function announce(): void {
    for (const listener of listeners) {
        listener();
    }
}

function subscribe(listener: () => void): () => void {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

export function pagesMemoryOf(siteId: string): PagesMemory {
    return memories.get(siteId) ?? blankPages;
}

export function rememberPages(siteId: string, patch: Partial<Omit<PagesMemory, "selections">>): void {
    memories.set(siteId, Object.freeze({ ...pagesMemoryOf(siteId), ...patch }));
    announce();
}

export function rememberSelection(siteId: string, view: PagesView, selection: Selection): void {
    const held = pagesMemoryOf(siteId);
    if (held.selections[view] === selection) {
        return;
    }
    memories.set(siteId, Object.freeze({ ...held, selections: Object.freeze({ ...held.selections, [view]: selection }) }));
    announce();
}

export function forgetPages(siteId: string): void {
    if (memories.delete(siteId)) {
        announce();
    }
}

export function usePagesMemory(siteId: string): PagesMemory {
    return useSyncExternalStore(subscribe, () => pagesMemoryOf(siteId));
}
