import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";

import { isCancellation } from "./call.js";
import type { Code, Reaction } from "./errors.js";
import { codes, failure, react, retryAfterOf } from "./errors.js";
import { markLocked } from "./lock.js";
import type { ToastTone } from "./toasts.js";
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

export type QuietMeta = { quiet: readonly Code[] };

export type Announcement =
    | { kind: "none" }
    | { kind: "locked" }
    | { kind: "toast"; tone: ToastTone; message: string; afterMs: number | null };

const silence: ReadonlySet<Code> = new Set<Code>();

export function quietMeta(expected: readonly Code[]): QuietMeta {
    return { quiet: expected };
}

export function quietOf(meta: Record<string, unknown> | undefined): ReadonlySet<Code> {
    const declared = meta?.["quiet"];
    if (!Array.isArray(declared)) {
        return silence;
    }
    const expected = new Set<Code>();
    for (const held of declared) {
        if (typeof held === "string" && (codes as readonly string[]).includes(held)) {
            expected.add(held as Code);
        }
    }
    return expected;
}

function toastOf(reaction: Reaction): { tone: ToastTone; message: string; afterMs: number | null } | null {
    switch (reaction.kind) {
        case "throttle":
            return { tone: "warning", message: reaction.message, afterMs: reaction.afterMs };
        case "refetch":
            return { tone: "info", message: reaction.message, afterMs: null };
        case "credentials":
        case "budget":
        case "review":
        case "external":
            return { tone: "warning", message: reaction.message, afterMs: null };
        case "fatal":
            return { tone: "danger", message: reaction.message, afterMs: null };
        default:
            return null;
    }
}

export function announcementOf(thrown: unknown, expected: ReadonlySet<Code>): Announcement {
    const reaction = react(thrown);
    if (reaction.kind === "unlock") {
        return { kind: "locked" };
    }
    const toast = toastOf(reaction);
    if (toast === null || expected.has(failure(thrown).code)) {
        return { kind: "none" };
    }
    return { kind: "toast", ...toast };
}

function announce(client: QueryClient | null, thrown: unknown, meta: Record<string, unknown> | undefined): void {
    const announcement = announcementOf(thrown, quietOf(meta));
    if (announcement.kind === "locked") {
        if (client !== null) {
            markLocked(client);
        }
        return;
    }
    if (announcement.kind === "toast") {
        pushToast(announcement.tone, announcement.message, announcement.afterMs);
    }
}

export function createQueryClient(): QueryClient {
    let client: QueryClient | null = null;
    const queryCache = new QueryCache({
        onError: (thrown, query) => {
            announce(client, thrown, query.meta);
        },
    });
    const mutationCache = new MutationCache({
        onError: (thrown, _variables, _context, mutation) => {
            announce(client, thrown, mutation.meta);
        },
    });
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
