import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { Adoption, AdoptParentsResponse } from "../../../data/types.js";
import type { AdoptionScope } from "./adopt-parents.js";

const said = copy.graph.adopt;

interface Applied {
    siteId: string;
    under: string;
    hash: string;
}

interface Held {
    pages: AdoptParentsResponse[];
    hasNextPage: boolean;
    asked: { siteId: string; under: string }[];
    applied: Applied[];
    fetchedMore: number;
    refusal: unknown;
}

const held: Held = { pages: [], hasNextPage: false, asked: [], applied: [], fetchedMore: 0, refusal: null };

vi.mock("../../../data/hooks/graph.js", () => ({
    useAdoptionPreview: (siteId: string, under: string) => {
        held.asked.push({ siteId, under });
        return {
            data: { pages: held.pages },
            isPending: false,
            isFetching: false,
            isFetchingNextPage: false,
            error: null,
            hasNextPage: held.hasNextPage,
            fetchNextPage: () => {
                held.fetchedMore += 1;
            },
        };
    },
    useAdoptParents: () => ({
        mutate: (request: Applied, options?: { onSuccess?: () => void; onError?: (thrown: unknown) => void }) => {
            held.applied.push(request);
            if (held.refusal === null) {
                options?.onSuccess?.();
                return;
            }
            options?.onError?.(held.refusal);
        },
        reset: () => undefined,
        isPending: false,
        error: null,
    }),
}));

const { AdoptParentsDrawer } = await import("./adopt-parents.js");

function step(entityId: string, overrides: Partial<Adoption> = {}): Adoption {
    return {
        entityId,
        name: `Entity ${entityId}`,
        path: `/${entityId}/`,
        from: [{ id: "old", name: "Old parent" }],
        to: { id: "new", name: "Starter kits" },
        toPath: "/kits/starter/",
        ...overrides,
    };
}

function answer(items: readonly Adoption[], overrides: Partial<AdoptParentsResponse> = {}): AdoptParentsResponse {
    return {
        steps: { items: [...items], hasMore: false },
        hash: "h1",
        adopting: items.filter((item) => (item.skip ?? "") === "").length,
        skipped: items.filter((item) => (item.skip ?? "") !== "").length,
        applied: false,
        ...overrides,
    } as AdoptParentsResponse;
}

function drawer(under: AdoptionScope | null = null, onOpenChange: (open: boolean) => void = () => undefined) {
    const client = new QueryClient();
    return render(
        <QueryClientProvider client={client}>
            <AdoptParentsDrawer open={true} siteId="s1" under={under} onOpenChange={onOpenChange} />
        </QueryClientProvider>,
    );
}

beforeEach(() => {
    held.pages = [];
    held.hasNextPage = false;
    held.asked = [];
    held.applied = [];
    held.fetchedMore = 0;
    held.refusal = null;
});

describe("taking the page tree's parents", () => {
    it("shows what moves from which parent to which, and what is left with why", () => {
        held.pages = [
            answer([step("kits"), step("loop", { skip: "loop" }), step("top", { to: null, toPath: "", from: [] })]),
        ];
        drawer();

        expect(screen.getByText(said.counts(2, 1))).toBeDefined();
        expect(screen.getAllByText("Old parent")).toHaveLength(2);
        expect(screen.getAllByText("Starter kits")).toHaveLength(2);
        expect(screen.getByText(said.top)).toBeDefined();
        expect(screen.getByText(said.noParent)).toBeDefined();
        expect(screen.getByText(`${said.skipped}: ${said.skips.loop}`)).toBeDefined();
    });

    it("previews a subtree when it is opened under a page", () => {
        held.pages = [answer([step("kits")])];
        drawer({ pageId: "p-kits", path: "/kits/" });

        expect(held.asked.at(-1)).toStrictEqual({ siteId: "s1", under: "p-kits" });
        expect(screen.getByText(said.under("/kits/"))).toBeDefined();
    });

    it("pages through the preview", () => {
        held.pages = [answer([step("kits")])];
        held.hasNextPage = true;
        drawer();

        fireEvent.click(screen.getByRole("button", { name: said.more }));
        expect(held.fetchedMore).toBe(1);
    });

    it("applies under the hash of the preview and closes", () => {
        held.pages = [answer([step("kits"), step("mugs")])];
        const closed: boolean[] = [];
        drawer({ pageId: "p-kits", path: "/kits/" }, (open) => {
            closed.push(open);
        });

        fireEvent.click(screen.getByRole("button", { name: said.apply(2) }));

        expect(held.applied).toStrictEqual([{ siteId: "s1", under: "p-kits", hash: "h1" }]);
        expect(closed).toStrictEqual([false]);
    });

    it("says the model moved when the apply comes back in conflict, and stays open", () => {
        held.pages = [answer([step("kits")])];
        held.refusal = Object.assign(new Error("moved"), {
            cause: { code: "CONFLICT", message: "the page tree or the graph moved since the preview" },
        });
        const closed: boolean[] = [];
        drawer(null, (open) => {
            closed.push(open);
        });

        fireEvent.click(screen.getByRole("button", { name: said.apply(1) }));

        expect(screen.getByText(said.moved)).toBeDefined();
        expect(closed).toStrictEqual([]);
    });

    it("says so when the graph already agrees with the page tree", () => {
        held.pages = [answer([])];
        drawer();

        expect(screen.getByText(said.none)).toBeDefined();
        expect((screen.getByRole("button", { name: said.apply(0) }) as HTMLButtonElement).disabled).toBe(true);
    });
});
