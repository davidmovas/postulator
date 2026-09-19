import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";

import { isCancellation } from "./call.js";
import type { Code } from "./errors.js";
import { failure, react, retryAfterOf } from "./errors.js";
import { markLocked } from "./lock.js";
import { pushToast } from "./toasts.js";

const maxRateLimitedRetries = 5;
const maxTransientRetries = 2;
const maxRetryDelayMs = 15_000;
const baseRetryDelayMs = 500;
const defaultGcTimeMs = 300_000;

export const infiniteGcTimeMs = 120_000;

const neverRetried: ReadonlySet<Code> = new Set<Code>([
    "INVALID",
    "NOT_FOUND",
    "UNAUTHORIZED",
    "LOCKED",
    "CONFLICT",
    "BUDGET_EXCEEDED",
    "NEEDS_HUMAN",
    "CANCELLED",
]);

export function shouldRetry(failureCount: number, thrown: unknown): boolean {
    if (isCancellation(thrown)) {
        return false;
    }
    const reported = failure(thrown);
    if (neverRetried.has(reported.code)) {
        return false;
    }
    if (reported.code === "RATE_LIMITED") {
        return failureCount < maxRateLimitedRetries;
    }
    return failureCount < maxTransientRetries;
}

export function retryDelay(attempt: number, thrown: unknown): number {
    const reported = failure(thrown);
    if (reported.code === "RATE_LIMITED") {
        return retryAfterOf(reported);
    }
    return Math.min(baseRetryDelayMs * 2 ** attempt, maxRetryDelayMs);
}

function announce(client: QueryClient | null, thrown: unknown): void {
    const reaction = react(thrown);
    switch (reaction.kind) {
        case "silent":
        case "field":
            return;
        case "unlock":
            if (client !== null) {
                markLocked(client);
            }
            return;
        case "throttle":
            pushToast("warning", reaction.message, reaction.afterMs);
            return;
        case "refetch":
        case "form":
            pushToast("info", reaction.message);
            return;
        case "credentials":
        case "budget":
        case "review":
        case "external":
            pushToast("warning", reaction.message);
            return;
        case "fatal":
            pushToast("danger", reaction.message);
            return;
        default:
            return;
    }
}

export function createQueryClient(): QueryClient {
    let client: QueryClient | null = null;
    const handle = (thrown: unknown): void => {
        announce(client, thrown);
    };
    const queryCache = new QueryCache({ onError: handle });
    const mutationCache = new MutationCache({ onError: handle });
    client = new QueryClient({
        queryCache,
        mutationCache,
        defaultOptions: {
            queries: {
                networkMode: "always",
                staleTime: 0,
                gcTime: defaultGcTimeMs,
                refetchOnWindowFocus: true,
                refetchOnReconnect: false,
                refetchOnMount: true,
                retry: shouldRetry,
                retryDelay,
            },
            mutations: {
                networkMode: "always",
                retry: false,
            },
        },
    });
    return client;
}
