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

describe("the page model follows the observation", () => {
    const modelled = {
        model: keys.pages.model("p1"),
        pack: keys.pages.pack("p1", ""),
        summary: keys.pages.summary("s1"),
        otherSummary: keys.pages.summary("s2"),
    } as const;

    type Modelled = keyof typeof modelled;

    function seeded(): QueryClient {
        const client = bridged();
        for (const key of Object.values(modelled)) {
            client.setQueryData(key, { held: true });
        }
        client.setQueryData(keys.runs.detail("r1"), { run: { siteId: "s1" } });
        return client;
    }

    function refreshed(client: QueryClient): Modelled[] {
        return (Object.keys(modelled) as Modelled[]).filter(
            (name) => client.getQueryState(modelled[name])?.isInvalidated === true,
        );
    }

    it("refreshes every page model, what the writer sees and the site's summary when the site's pages change", () => {
        const client = seeded();

        fire("pages.changed", { siteId: "s1" });
        vi.advanceTimersByTime(usageCoalesceMs);

        expect(refreshed(client)).toEqual(["model", "pack", "summary"]);
    });

    it("refreshes them when a run on the site completes", () => {
        const client = seeded();

        fire("run.completed", { runId: "r1", succeeded: 1, failed: 0 }, "r1");

        expect(refreshed(client)).toEqual(["model", "pack", "summary"]);
    });

    it("refreshes the site's card proposals, and only that site's, when its graph changes or a run on it completes", () => {
        const client = seeded();
        client.setQueryData(keys.graph.cards("s1"), { held: true });
        client.setQueryData(keys.graph.cards("s2"), { held: true });
        const stale = (siteId: string): boolean => client.getQueryState(keys.graph.cards(siteId))?.isInvalidated === true;

        fire("graph.changed", { siteId: "s1" });
        vi.advanceTimersByTime(usageCoalesceMs);
        expect([stale("s1"), stale("s2")]).toEqual([true, false]);

        client.setQueryData(keys.graph.cards("s1"), { held: true });
        fire("run.completed", { runId: "r1", succeeded: 1, failed: 0 }, "r1");
        expect([stale("s1"), stale("s2")]).toEqual([true, false]);
    });

    it.each([
        ["settings.changed", {}],
        ["templates.changed", {}],
        ["graph.changed", { siteId: "s1" }],
    ] as const)("refreshes what the writer sees, and only that of the model, on %s", (type, payload) => {
        const client = seeded();

        fire(type, payload);
        vi.advanceTimersByTime(usageCoalesceMs);

        expect(refreshed(client)).toEqual(["pack"]);
    });
});

describe("a plan's card follows its plan", () => {
    function planned(): QueryClient {
        const client = bridged();
        client.setQueryData(keys.agent.plans("c1"), { plans: [] });
        client.setQueryData(keys.agent.plans("c2"), { plans: [] });
        return client;
    }

    function staleOf(client: QueryClient): boolean[] {
        return ["c1", "c2"].map(
            (conversationId) => client.getQueryState(keys.agent.plans(conversationId))?.isInvalidated === true,
        );
    }

    it("reads the plans of the conversation again when one is proposed there", () => {
        const client = planned();

        fire("agent.plan.proposed", { conversationId: "c1", planId: "p1", title: "Link the kits", steps: 3 });

        expect(staleOf(client)).toEqual([true, false]);
    });

    it("reads them again once a burst of step updates settles", () => {
        const client = planned();

        const moved = { conversationId: "c1", planId: "p1", status: "running", stepId: "s1" };
        fire("agent.plan.updated", { ...moved, stepStatus: "running" });
        fire("agent.plan.updated", { ...moved, stepStatus: "done" });
        expect(staleOf(client)).toEqual([false, false]);

        vi.advanceTimersByTime(usageCoalesceMs);
        expect(staleOf(client)).toEqual([true, false]);
    });
});
