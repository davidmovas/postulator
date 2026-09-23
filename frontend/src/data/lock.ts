import type { QueryClient } from "@tanstack/react-query";
import { useQuery } from "@tanstack/react-query";

import { dropAllTurns } from "./agent/turn.js";
import { lockState } from "./endpoints/settings.js";
import { cancelCoalesced } from "./invalidate.js";
import { keys } from "./keys.js";
import { catchUpNow, dropAllLogs, liveRunIds } from "./runs/log.js";
import { clearToasts } from "./toasts.js";
import type { LockState } from "./types.js";

const lockedState: LockState = Object.freeze({ locked: true, protected: true });

export function isLockKey(key: readonly unknown[]): boolean {
    const lock = keys.settings.lock();
    return key.length === lock.length && lock.every((part, index) => key[index] === part);
}

export function markLocked(client: QueryClient): void {
    cancelCoalesced();
    client.setQueryData(keys.settings.lock(), lockedState);
    client.removeQueries({ predicate: (query) => !isLockKey(query.queryKey) });
    dropAllLogs();
    dropAllTurns();
    clearToasts();
}

export function markUnlocked(client: QueryClient, state: LockState): void {
    client.setQueryData(keys.settings.lock(), state);
    void client.invalidateQueries();
    for (const runId of liveRunIds()) {
        void catchUpNow(runId);
    }
}

export interface LockGate {
    ready: boolean;
    locked: boolean;
    protectedByPassword: boolean;
}

export function useLockState() {
    return useQuery({
        queryKey: keys.settings.lock(),
        queryFn: ({ signal }) => lockState({}, signal),
        networkMode: "always",
        retry: false,
        staleTime: 0,
    });
}

export function useLockGate(): LockGate {
    const state = useLockState();
    return {
        ready: state.data !== undefined || state.isError,
        locked: state.data?.locked ?? false,
        protectedByPassword: state.data?.protected ?? false,
    };
}
