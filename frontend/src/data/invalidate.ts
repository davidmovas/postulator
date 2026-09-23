import type { QueryClient, QueryKey } from "@tanstack/react-query";

import { siteOf } from "./keys.js";

const coalesceDelayMs = 250;

const timers = new Map<string, ReturnType<typeof setTimeout>>();

export function coalesce(tag: string, run: () => void, delayMs: number = coalesceDelayMs): void {
    const held = timers.get(tag);
    if (held !== undefined) {
        clearTimeout(held);
    }
    timers.set(
        tag,
        setTimeout(() => {
            timers.delete(tag);
            run();
        }, delayMs),
    );
}

export function cancelCoalesced(): void {
    timers.forEach((held) => {
        clearTimeout(held);
    });
    timers.clear();
}

function prefixed(key: QueryKey, prefix: readonly unknown[]): boolean {
    if (key.length < prefix.length) {
        return false;
    }
    for (let index = 0; index < prefix.length; index += 1) {
        if (key[index] !== prefix[index]) {
            return false;
        }
    }
    return true;
}

export function invalidate(client: QueryClient, key: readonly unknown[]): void {
    void client.invalidateQueries({ queryKey: key as QueryKey });
}

export function invalidateAll(client: QueryClient, ...list: readonly (readonly unknown[])[]): void {
    for (const key of list) {
        invalidate(client, key);
    }
}

export function invalidateBySite(client: QueryClient, prefix: readonly unknown[], siteId: string): void {
    void client.invalidateQueries({
        predicate: (query) => prefixed(query.queryKey, prefix) && siteOf(query.queryKey) === siteId,
    });
}
