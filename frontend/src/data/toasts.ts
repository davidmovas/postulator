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

const listeners = new Set<() => void>();

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
    return id;
}

export function dismissToast(id: number): void {
    publish(toasts.filter((held) => held.id !== id));
}

export function clearToasts(): void {
    publish([]);
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
