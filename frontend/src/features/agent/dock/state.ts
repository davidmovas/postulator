import { useSyncExternalStore } from "react";

export type DockMode = "narrow" | "wide";

export const narrowDockWidth = 392;
export const wideDockFraction = 0.5;
export const minimumDockWidth = 320;
export const maximumDockWidth = 900;
export const reservedMainWidth = 360;

export const globalDockKey = "global";

export function dockKey(siteId: string | null): string {
    return siteId === null || siteId === "" ? globalDockKey : siteId;
}

export function clampWidth(width: number, windowWidth: number): number {
    const ceiling = Math.max(minimumDockWidth, Math.min(maximumDockWidth, windowWidth - reservedMainWidth));
    return Math.min(Math.max(Math.round(width), minimumDockWidth), ceiling);
}

export function widthOf(mode: DockMode, custom: number | null, windowWidth: number): number {
    if (custom !== null) {
        return clampWidth(custom, windowWidth);
    }
    return clampWidth(mode === "wide" ? windowWidth * wideDockFraction : narrowDockWidth, windowWidth);
}

export function parseMode(raw: string | null): DockMode {
    return raw === "wide" ? "wide" : "narrow";
}

export function parseCustom(raw: string | null): number | null {
    const parsed = raw === null ? Number.NaN : Number.parseInt(raw, 10);
    return Number.isNaN(parsed) ? null : parsed;
}

export function chooseConversation(
    remembered: string | undefined,
    available: readonly { id: string }[],
): string | null {
    if (remembered !== undefined && available.some((held) => held.id === remembered)) {
        return remembered;
    }
    return available[0]?.id ?? null;
}

export function parseChoices(raw: string | null): Record<string, string> {
    if (raw === null) {
        return {};
    }
    let decoded: unknown;
    try {
        decoded = JSON.parse(raw);
    } catch {
        return {};
    }
    if (typeof decoded !== "object" || decoded === null || Array.isArray(decoded)) {
        return {};
    }
    const out: Record<string, string> = {};
    for (const [key, value] of Object.entries(decoded)) {
        if (typeof value === "string") {
            out[key] = value;
        }
    }
    return out;
}

export interface DockState {
    open: boolean;
    mode: DockMode;
    custom: number | null;
    choices: Readonly<Record<string, string>>;
    prefill: string | null;
    focusSeq: number;
}

const openKey = "postulator.dock.open";
const modeKey = "postulator.dock.mode";
const customKey = "postulator.dock.width";
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
    mode: parseMode(readStored(modeKey)),
    custom: parseCustom(readStored(customKey)),
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

export function setDockMode(mode: DockMode): void {
    if (state.mode === mode && state.custom === null) {
        return;
    }
    writeStored(modeKey, mode);
    writeStored(customKey, "");
    publish({ ...state, mode, custom: null });
}

export function setDockWidth(width: number): void {
    const rounded = Math.round(width);
    if (state.custom === rounded) {
        return;
    }
    writeStored(customKey, String(rounded));
    publish({ ...state, custom: rounded });
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
