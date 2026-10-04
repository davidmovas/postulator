import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { Envelope, EventType } from "../generated/events.js";

type Listener = (envelope: Envelope<EventType>) => void;

const listeners = new Map<string, Listener>();

vi.mock("../lib/events.js", () => ({
    on: (type: string, handler: Listener) => {
        listeners.set(type, handler);
        return () => {
            listeners.delete(type);
        };
    },
}));

const { EventBridge, usageCoalesceMs } = await import("./bridge.js");
const { keys } = await import("./keys.js");

const seeded = {
    lastWeek: keys.models.spendOver(7),
    lastMonth: keys.models.spendOver(30),
    thisRun: keys.models.spendOfRun("r1"),
    otherRun: keys.models.spendOfRun("r2"),
    calls: keys.models.calls({}, 25),
} as const;

type Seeded = keyof typeof seeded;

function fresh(): QueryClient {
    const client = new QueryClient({
        defaultOptions: { queries: { retry: false, gcTime: Number.POSITIVE_INFINITY } },
    });
    client.setQueryData(keys.settings.lock(), { locked: false, protected: false });
    for (const key of Object.values(seeded)) {
        client.setQueryData(key, { held: true });
    }
    return client;
}

function bridged(): QueryClient {
    const client = fresh();
    render(
        <QueryClientProvider client={client}>
            <EventBridge />
        </QueryClientProvider>,
    );
    return client;
}

function fire<T extends EventType>(type: T, payload: Envelope<T>["payload"], runId = ""): void {
    const listener = listeners.get(type);
    if (listener === undefined) {
        throw new Error(`nothing listens to ${type}`);
    }
    listener({ type, seq: 1, at: "2026-10-03T10:00:00Z", runId, payload } as Envelope<EventType>);
}

function stale(client: QueryClient): Seeded[] {
    return (Object.keys(seeded) as Seeded[]).filter(
        (name) => client.getQueryState(seeded[name])?.isInvalidated === true,
    );
}

const usage = {
    runId: "r1",
    itemId: "i1",
    provider: "openai",
    model: "gpt-5.6-terra",
    tier: "flex",
    promptTokens: 1200,
    completionTokens: 300,
    reasoningTokens: 120,
    cacheWriteTokens: 0,
    usd: 0.004,
};

beforeEach(() => {
    vi.useFakeTimers();
});

afterEach(() => {
    vi.useRealTimers();
    listeners.clear();
});

describe("the spend panel follows the ledger", () => {
    it("waits for a burst of calls to settle before it refetches", () => {
        const client = bridged();

        fire("llm.usage", usage, "r1");
        vi.advanceTimersByTime(usageCoalesceMs - 1);

        expect(stale(client)).toEqual([]);
    });

    it("refreshes every range, the recent calls and the run that spent", () => {
        const client = bridged();

        fire("llm.usage", usage, "r1");
        vi.advanceTimersByTime(usageCoalesceMs);

        expect(stale(client)).toEqual(["lastWeek", "lastMonth", "thisRun", "calls"]);
    });

    it("leaves every run report alone for a call no run made", () => {
        const client = bridged();

        fire("llm.usage", { ...usage, runId: "", itemId: "" });
        vi.advanceTimersByTime(usageCoalesceMs);

        expect(stale(client)).toEqual(["lastWeek", "lastMonth", "calls"]);
    });

    it("counts a failed call that spent no tokens once the item fails", () => {
        const client = bridged();

        fire("item.failed", { runId: "r1", itemId: "i1", code: "EXTERNAL", message: "server error" }, "r1");
        vi.advanceTimersByTime(usageCoalesceMs);

        expect(stale(client)).toEqual(["lastWeek", "lastMonth", "thisRun", "calls"]);
    });

    it("counts a failed call that spent no tokens once the item waits for a person", () => {
        const client = bridged();

        fire("item.needs_human", { runId: "r1", itemId: "i1", reason: "needs_human", message: "out of credit" }, "r1");
        vi.advanceTimersByTime(usageCoalesceMs);

        expect(stale(client)).toEqual(["lastWeek", "lastMonth", "thisRun", "calls"]);
    });

    it("counts a failed call that spent no tokens once the run fails", () => {
        const client = bridged();

        fire("run.failed", { runId: "r1", code: "NEEDS_HUMAN", message: "out of credit" }, "r1");
        vi.advanceTimersByTime(usageCoalesceMs);

        expect(stale(client)).toEqual(["lastWeek", "lastMonth", "thisRun", "calls"]);
    });

    it("counts an assistant answer once the turn is over", () => {
        const client = bridged();

        fire("agent.done", {
            conversationId: "c1",
            messageId: "m1",
            text: "",
            code: "NEEDS_HUMAN",
            error: "out of credit",
            inputTokens: 0,
            cachedInputTokens: 0,
            outputTokens: 0,
            calls: 0,
            usd: 0,
        });
        vi.advanceTimersByTime(usageCoalesceMs);

        expect(stale(client)).toEqual(["lastWeek", "lastMonth", "calls"]);
    });
});
