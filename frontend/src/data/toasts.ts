import { useSyncExternalStore } from "react";

export type ToastTone = "info" | "warning" | "danger";

export interface ToastAction {
    label: string;
    to: string;
}

export interface Toast {
    id: number;
    tone: ToastTone;
    message: string;
    afterMs: number | null;
    action: ToastAction | null;
}

export const holdToastMs = 6000;

const listeners = new Set<() => void>();
const timers = new Map<number, ReturnType<typeof setTimeout>>();

let toasts: readonly Toast[] = Object.freeze([]);
let nextId = 1;

function publish(next: readonly Toast[]): void {
    toasts = Object.freeze(next);
    listeners.forEach((listener) => {
        listener();
    });
}

export function pushToast(
    tone: ToastTone,
    message: string,
    afterMs: number | null = null,
    action: ToastAction | null = null,
): number {
    const id = nextId;
    nextId += 1;
    publish([...toasts, { id, tone, message, afterMs, action }]);
    if (action === null) {
        timers.set(
            id,
            setTimeout(() => {
                dismissToast(id);
            }, holdToastMs),
        );
    }
    return id;
}

export function dismissToast(id: number): void {
    const timer = timers.get(id);
    if (timer !== undefined) {
        clearTimeout(timer);
        timers.delete(id);
    }
    publish(toasts.filter((held) => held.id !== id));
}

export function clearToasts(): void {
    timers.forEach((timer) => {
        clearTimeout(timer);
    });
    timers.clear();
    publish([]);
}

export function readToasts(): readonly Toast[] {
    return toasts;
}

function subscribe(listener: () => void): () => void {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

function snapshot(): readonly Toast[] {
    return toasts;
}

export function useToasts(): readonly Toast[] {
    return useSyncExternalStore(subscribe, snapshot, snapshot);
}
