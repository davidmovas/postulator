import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { ModelCall, ModelCallFilter } from "../../../data/types.js";
import type { List } from "../../../lib/paging.js";

const said = copy.settings.models.spend;

interface Held {
    pages: List<ModelCall>[] | undefined;
    hasNextPage: boolean;
    fetchingNext: boolean;
    asked: { filter: ModelCallFilter | undefined; limit: number | undefined }[];
    forward: number;
}

const held: Held = { pages: undefined, hasNextPage: false, fetchingNext: false, asked: [], forward: 0 };

vi.mock("../../../data/hooks/models.js", () => ({
    useModelCalls: (filter?: ModelCallFilter, limit?: number) => {
        held.asked.push({ filter, limit });
        return {
            data: held.pages === undefined ? undefined : { pages: held.pages, pageParams: [] },
            isPending: held.pages === undefined,
            error: null,
            hasNextPage: held.hasNextPage,
            isFetchingNextPage: held.fetchingNext,
            fetchNextPage: () => {
                held.forward += 1;
                return Promise.resolve();
            },
            refetch: () => Promise.resolve(),
        };
    },
}));

const { RecentCalls, recentPageSize } = await import("./calls.js");

function call(overrides: Partial<ModelCall>): ModelCall {
    return {
        id: "call-1",
        createdAt: "2026-10-03T20:55:00Z",
        purpose: "run",
        step: "generate_body",
        provider: "openai",
        model: "gpt-5.6-terra",
        tier: "flex",
        input: 12_400,
        cachedInput: 9_800,
        cacheWrite: 0,
        output: 3_100,
        reasoning: 1_200,
        total: 15_500,
        usd: 0.031,
        latencyMs: 41_000,
        status: "ok",
        errorCode: "",
        runId: "r1",
        itemId: "i1",
        conversationId: "",
        ...overrides,
    };
}

function listed(items: ModelCall[], hasMore = false): List<ModelCall> {
    return { items, hasMore, nextCursor: hasMore ? "next" : undefined };
}

function rows(): HTMLElement[] {
    return within(screen.getByRole("table", { name: said.recent.title })).getAllByRole("row").slice(1);
}

beforeEach(() => {
    held.pages = [
        listed([
            call({}),
            call({
                id: "call-2",
                purpose: "chat",
                step: "chat",
                tier: "default",
                runId: "",
                itemId: "",
                conversationId: "c1",
                input: 21_000,
                cachedInput: 18_000,
                output: 400,
                reasoning: 0,
                usd: 0.004,
            }),
        ]),
    ];
    held.hasNextPage = false;
    held.fetchingNext = false;
    held.asked = [];
    held.forward = 0;
});

describe("the recent model calls", () => {
    it("asks for the newest calls a page at a time", () => {
        render(<RecentCalls />);

        expect(held.asked.at(-1)?.limit).toBe(recentPageSize);
    });

    it("says what each call was for, on which model and tier, and what it read, wrote and cost", () => {
        render(<RecentCalls />);
        const [writer, chat] = rows();
        if (writer === undefined || chat === undefined) {
            throw new Error("two rows expected");
        }

        expect(within(writer).getByText("Page runs").parentElement?.textContent).toBe("Page runs · Write the body");
        expect(within(writer).getByText("gpt-5.6-terra")).toBeDefined();
        expect(within(writer).getByText("Flex")).toBeDefined();
        expect(within(writer).getByText(said.recent.flow("12.4K", "3,100"))).toBeDefined();
        expect(within(writer).getByText("$0.03")).toBeDefined();

        expect(within(chat).getByText("Assistant").parentElement?.textContent).toBe("Assistant");
        expect(within(chat).getByText("Standard")).toBeDefined();
        expect(within(chat).getByText("$0.0040")).toBeDefined();
    });

    it("tells how much of a call was cached and how much was reasoning", () => {
        render(<RecentCalls />);
        const [writer] = rows();
        if (writer === undefined) {
            throw new Error("a row expected");
        }

        expect(within(writer).getByText(said.recent.flow("12.4K", "3,100")).getAttribute("title")).toBe(
            said.recent.inside("9,800", "1,200"),
        );
    });

    it("marks a failed call and says why in a sentence, never in a code", () => {
        held.pages = [listed([call({ status: "error", errorCode: "NEEDS_HUMAN", input: 0, cachedInput: 0, output: 0, reasoning: 0, usd: 0 })])];
        render(<RecentCalls />);
        const [failed] = rows();
        if (failed === undefined) {
            throw new Error("a row expected");
        }

        expect(within(failed).getByText(said.recent.failed)).toBeDefined();
        expect(within(failed).getByText(said.failures.NEEDS_HUMAN)).toBeDefined();
        expect(within(failed).queryByText(/NEEDS_HUMAN/)).toBeNull();
    });

    it("draws no failure on a call that answered", () => {
        render(<RecentCalls />);

        expect(screen.queryByText(said.recent.failed)).toBeNull();
    });

    it("pages forward when the owner asks for more", () => {
        held.hasNextPage = true;
        render(<RecentCalls />);

        fireEvent.click(screen.getByRole("button", { name: copy.app.loadMore }));

        expect(held.forward).toBe(1);
    });

    it("lists every page it has read, in order", () => {
        held.pages = [listed([call({ id: "a", model: "gpt-5.6-sol" })], true), listed([call({ id: "b", model: "gpt-5.6-luna" })])];
        render(<RecentCalls />);

        expect(rows().map((row) => within(row).getAllByRole("cell")[2]?.textContent)).toEqual([
            "gpt-5.6-sol",
            "gpt-5.6-luna",
        ]);
    });

    it("offers no more once the ledger has none", () => {
        render(<RecentCalls />);

        expect(screen.queryByRole("button", { name: copy.app.loadMore })).toBeNull();
    });

    it("says no model has been called yet", () => {
        held.pages = [listed([])];
        render(<RecentCalls />);

        expect(screen.getByText(said.recent.empty)).toBeDefined();
        expect(screen.queryByRole("table")).toBeNull();
    });

    it("shows a loading state before the first page", () => {
        held.pages = undefined;
        render(<RecentCalls />);

        expect(screen.getByRole("status", { name: said.recent.loading })).toBeDefined();
    });
});
