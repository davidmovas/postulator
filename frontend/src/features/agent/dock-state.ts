import { useSyncExternalStore } from "react";

import { clampWidth, dockKey, parseChoices, parseWidth } from "./model/dock.js";

export interface DockState {
    open: boolean;
    width: number;
    choices: Readonly<Record<string, string>>;
    prefill: string | null;
    focusSeq: number;
}

const openKey = "postulator.dock.open";
const widthKey = "postulator.dock.width";
const choicesKey = "postulator.dock.conversations";

function readStored(key: string): string | null {
    try {
        return window.localStorage.getItem(key);
    } catch {
        return null;
    }
}

function writeStored(key: string, value: string): void {
    try {
        window.localStorage.setItem(key, value);
    } catch {
        return;
    }
}

const listeners = new Set<() => void>();

let state: DockState = Object.freeze({
    open: readStored(openKey) === "1",
    width: parseWidth(readStored(widthKey)),
    choices: Object.freeze(parseChoices(readStored(choicesKey))),
    prefill: null,
    focusSeq: 0,
});

function publish(next: DockState): void {
    state = Object.freeze(next);
    listeners.forEach((listener) => {
        listener();
    });
}

function subscribe(listener: () => void): () => void {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

function snapshot(): DockState {
    return state;
}

export function useDock(): DockState {
    return useSyncExternalStore(subscribe, snapshot, snapshot);
}

export function openDock(): void {
    if (state.open) {
        return;
    }
    writeStored(openKey, "1");
    publish({ ...state, open: true });
}

export function closeDock(): void {
    if (!state.open) {
        return;
    }
    writeStored(openKey, "0");
    publish({ ...state, open: false });
}

export function toggleDock(): void {
    if (state.open) {
        closeDock();
    } else {
        openDock();
    }
}

export function setDockWidth(width: number): void {
    const clamped = clampWidth(width);
    if (clamped === state.width) {
        return;
    }
    writeStored(widthKey, String(clamped));
    publish({ ...state, width: clamped });
}

export function rememberConversation(siteId: string | null, conversationId: string): void {
    const key = dockKey(siteId);
    if (state.choices[key] === conversationId) {
        return;
    }
    const choices = Object.freeze({ ...state.choices, [key]: conversationId });
    writeStored(choicesKey, JSON.stringify(choices));
    publish({ ...state, choices });
}

export function forgetConversation(conversationId: string): void {
    const kept: Record<string, string> = {};
    let changed = false;
    for (const [key, held] of Object.entries(state.choices)) {
        if (held === conversationId) {
            changed = true;
        } else {
            kept[key] = held;
        }
    }
    if (!changed) {
        return;
    }
    const choices = Object.freeze(kept);
    writeStored(choicesKey, JSON.stringify(choices));
    publish({ ...state, choices });
}

export function askAgent(prefill: string): void {
    writeStored(openKey, "1");
    publish({ ...state, open: true, prefill, focusSeq: state.focusSeq + 1 });
}

export function takePrefill(): string | null {
    const held = state.prefill;
    if (held !== null) {
        publish({ ...state, prefill: null });
    }
    return held;
}
