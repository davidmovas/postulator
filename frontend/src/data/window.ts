import { Events, Window } from "@wailsio/runtime";
import { useSyncExternalStore } from "react";

const maximiseEvents = [
    "common:WindowMaximise",
    "common:WindowUnMaximise",
    "common:WindowRestore",
    "windows:WindowMaximise",
    "windows:WindowUnMaximise",
    "windows:WindowRestore",
] as const;

const listeners = new Set<() => void>();
const sources: (() => void)[] = [];

let maximised = false;

function publish(next: boolean): void {
    if (next === maximised) {
        return;
    }
    maximised = next;
    listeners.forEach((listener) => {
        listener();
    });
}

function refresh(): void {
    void Window.IsMaximised()
        .then(publish)
        .catch(() => undefined);
}

function watch(): void {
    for (const name of maximiseEvents) {
        sources.push(Events.On(name, refresh));
    }
    const onResize = (): void => {
        refresh();
    };
    window.addEventListener("resize", onResize);
    window.addEventListener("focus", onResize);
    sources.push(() => {
        window.removeEventListener("resize", onResize);
        window.removeEventListener("focus", onResize);
    });
    refresh();
}

function unwatch(): void {
    while (sources.length > 0) {
        sources.pop()?.();
    }
}

function subscribe(listener: () => void): () => void {
    if (listeners.size === 0) {
        watch();
    }
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
        if (listeners.size === 0) {
            unwatch();
        }
    };
}

function snapshot(): boolean {
    return maximised;
}

export function useMaximised(): boolean {
    return useSyncExternalStore(subscribe, snapshot, snapshot);
}

export function minimise(): void {
    void Window.Minimise().catch(() => undefined);
}

export function toggleMaximise(): void {
    void Window.ToggleMaximise()
        .then(refresh)
        .catch(() => undefined);
}

export function close(): void {
    void Window.Close().catch(() => undefined);
}
