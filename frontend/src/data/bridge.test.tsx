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

const settleMs = 250;

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

const filing = {
    tree: keys.pages.categories("s1"),
    otherTree: keys.pages.categories("s2"),
    pageDetail: keys.pages.detail("p1"),
    pageTree: keys.pages.tree("s1"),
    pageList: keys.pages.list({ siteId: "s1" }, null, 200),
    otherPageList: keys.pages.list({ siteId: "s2" }, null, 200),
    entity: keys.graph.entity("e1"),
    entityList: keys.graph.entities({ siteId: "s1" }, null, 50),
    graph: keys.graph.full("s1"),
    otherGraph: keys.graph.full("s2"),
} as const;

type Filed = keyof typeof filing;

const followsSiteOne: Filed[] = ["tree", "pageDetail", "pageTree", "pageList", "entity", "entityList", "graph"];

function bridgedFiling(): QueryClient {
    const client = bridged();
    for (const key of Object.values(filing)) {
        client.setQueryData(key, { held: true });
    }
    client.setQueryData(keys.runs.detail("r1"), { run: { id: "r1", siteId: "s1" } });
    return client;
}

function staleFiling(client: QueryClient): Filed[] {
    return (Object.keys(filing) as Filed[]).filter((name) => client.getQueryState(filing[name])?.isInvalidated === true);
}

describe("the category tree and every label of a page's categories follow the site", () => {
    it.each<[string, () => void]>([
        ["the pages change", () => {
            fire("pages.changed", { siteId: "s1" });
        }],
        ["the graph changes", () => {
            fire("graph.changed", { siteId: "s1" });
        }],
        ["a sync stores the site's terms", () => {
            fire("sites.changed", { siteId: "s1" });
        }],
        ["a run that published pages completes", () => {
            fire("run.completed", { runId: "r1", succeeded: 3, failed: 0 }, "r1");
        }],
    ])("refreshes the tree, the pages and the entities of that site when %s", (_, happen) => {
        const client = bridgedFiling();

        happen();
        vi.advanceTimersByTime(settleMs);

        expect(staleFiling(client)).toEqual(followsSiteOne);
    });

    it("refreshes every site's tree after a run whose site it never saw", () => {
        const client = bridgedFiling();

        fire("run.completed", { runId: "r9", succeeded: 1, failed: 0 }, "r9");

        expect(staleFiling(client)).toEqual(Object.keys(filing));
    });

    it("waits for a burst of page changes to settle", () => {
        const client = bridgedFiling();

        fire("pages.changed", { siteId: "s1" });
        vi.advanceTimersByTime(settleMs - 1);

        expect(staleFiling(client)).toEqual([]);
    });
});
