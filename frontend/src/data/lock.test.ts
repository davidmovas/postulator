import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { afterEach, describe, expect, test } from "vitest";

import { keys } from "./keys.js";
import { isLockKey, markLocked, markUnlocked } from "./lock.js";
import type { LockState } from "./types.js";

const unlocked: LockState = { locked: false, protected: true };

let stops: (() => void)[] = [];

function fresh(): QueryClient {
    const client = new QueryClient({
        defaultOptions: { queries: { retry: false, gcTime: Number.POSITIVE_INFINITY } },
    });
    client.setQueryData(keys.settings.lock(), unlocked);
    client.setQueryData(keys.sites.lists(), { items: ["a site"] });
    client.setQueryData(keys.agent.conversationsAll(), { items: ["a conversation"] });
    return client;
}

function watchLock(client: QueryClient): QueryObserver<LockState, Error, LockState> {
    const observer = new QueryObserver<LockState, Error, LockState>(client, {
        queryKey: keys.settings.lock(),
        queryFn: () => unlocked,
        enabled: false,
    });
    stops.push(observer.subscribe(() => undefined));
    return observer;
}

afterEach(() => {
    for (const stop of stops) {
        stop();
    }
    stops = [];
});

describe("the lock key is the one query a lock never removes", () => {
    test("only the lock query answers to isLockKey", () => {
        expect(isLockKey(keys.settings.lock())).toBe(true);
        expect(isLockKey(keys.settings.values())).toBe(false);
        expect(isLockKey(keys.sites.lists())).toBe(false);
        expect(isLockKey([...keys.settings.lock(), "more"])).toBe(false);
    });

    test("a watching observer is told the workspace locked", () => {
        const client = fresh();
        const observer = watchLock(client);
        expect(observer.getCurrentResult().data).toStrictEqual(unlocked);

        markLocked(client);

        expect(observer.getCurrentResult().data).toStrictEqual({ locked: true, protected: true });
        expect(client.getQueryData(keys.settings.lock())).toStrictEqual({ locked: true, protected: true });
    });

    test("every other query is dropped by the lock", () => {
        const client = fresh();
        watchLock(client);

        markLocked(client);

        expect(client.getQueryData(keys.sites.lists())).toBeUndefined();
        expect(client.getQueryData(keys.agent.conversationsAll())).toBeUndefined();
        expect(client.getQueryCache().findAll()).toHaveLength(1);
    });

    test("the same observer is told the workspace unlocked again", () => {
        const client = fresh();
        const observer = watchLock(client);
        markLocked(client);

        markUnlocked(client, { locked: false, protected: false });

        expect(observer.getCurrentResult().data).toStrictEqual({ locked: false, protected: false });
    });

    test("locking twice leaves the observer on the locked state", () => {
        const client = fresh();
        const observer = watchLock(client);

        markLocked(client);
        markLocked(client);

        expect(observer.getCurrentResult().data).toStrictEqual({ locked: true, protected: true });
        expect(client.getQueryCache().findAll()).toHaveLength(1);
    });
});
