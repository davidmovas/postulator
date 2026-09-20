import { useSyncExternalStore } from "react";

const listeners = new Set<() => void>();

let open = false;

function publish(next: boolean): void {
    open = next;
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

function snapshot(): boolean {
    return open;
}

export function usePaletteOpen(): boolean {
    return useSyncExternalStore(subscribe, snapshot, snapshot);
}

export function openPalette(): void {
    if (!open) {
        publish(true);
    }
}

export function closePalette(): void {
    if (open) {
        publish(false);
    }
}
