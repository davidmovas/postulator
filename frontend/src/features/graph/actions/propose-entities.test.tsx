import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { Page, PageTreeNode, ProposedEntity } from "../../../data/types.js";
import { buildGraphIndex } from "../model/index.js";

interface Calls {
    previewed: { siteId: string; pageIds?: string[] }[];
    keyworded: { siteId: string; keywords: string[]; parentEntityId?: string }[];
    applied: { siteId: string; entities: ProposedEntity[] }[];
}

const calls: Calls = { previewed: [], keyworded: [], applied: [] };

const proposals: ProposedEntity[] = [
    { pageId: "mugs", path: "/shop/mugs/", name: "Mugs", kind: "category", primaryKeyword: "mugs", parent: "/shop/" },
    { pageId: "travel", path: "/shop/mugs/travel/", name: "Travel Mugs", kind: "topic", primaryKeyword: "travel mugs", parent: "/shop/mugs/" },
];

function aPage(id: string, path: string, overrides: Partial<Page> = {}): Page {
    return {
        id,
        siteId: "s1",
        path,
        slug: "",
        parentPageId: null,
        wpType: "page",
        wpId: null,
        title: `Title ${id}`,
        h1: "",
        metaTitle: "",
        metaDescription: "",
        canonical: "",
        primaryKeyword: "",
        keywords: [],
        status: "planned",
        entityId: null,
        templateId: null,
        contentHash: "",
        observed: { link: "", slug: "", status: "", title: "", h1: "" },
        mismatches: [],
        wpModifiedAt: null,
        lastSyncedAt: null,
        drift: false,
        createdAt: "2026-09-24T09:00:00Z",
        updatedAt: "2026-09-24T09:00:00Z",
        ...overrides,
    };
}

const roots: PageTreeNode[] = [
    {
        page: aPage("shop", "/shop/", { entityId: "e1" }),
        children: [
            { page: aPage("mugs", "/shop/mugs/"), children: [{ page: aPage("travel", "/shop/mugs/travel/"), children: [] }] },
        ],
    },
];

function mutation<Request, Answer>(record: (request: Request) => void, answer: Answer) {
    return {
        mutate: (input: { request: Request }, options?: { onSuccess?: (answered: Answer) => void }) => {
            record(input.request);
            options?.onSuccess?.(answer);
        },
        reset: () => {},
        isPending: false,
        error: null,
    };
}

vi.mock("../../../data/hooks/pages.js", () => ({
    usePageTree: () => ({ data: { roots }, isPending: false, error: null }),
}));

vi.mock("../../../data/hooks/graph.js", () => ({
    usePreviewFromPages: () =>
        mutation<{ siteId: string; pageIds?: string[] }, { entities: ProposedEntity[]; pages: number; skipped: number; tokens: number }>(
            (request) => calls.previewed.push(request),
            { entities: proposals, pages: 2, skipped: 0, tokens: 12 },
        ),
    useProposeFromKeywords: () =>
        mutation<{ siteId: string; keywords: string[]; parentEntityId?: string }, { entities: ProposedEntity[]; skipped: number; tokens: number }>(
            (request) => calls.keyworded.push(request),
            { entities: [proposals[1] as ProposedEntity], skipped: 0, tokens: 12 },
        ),
    useApplyProposals: () =>
        mutation<{ siteId: string; entities: ProposedEntity[] }, { entities: unknown[]; edges: unknown[]; mapped: number; skipped: number }>(
            (request) => calls.applied.push(request),
            { entities: [{}], edges: [{}, {}], mapped: 1, skipped: 0 },
        ),
}));

const { ProposeEntitiesDrawer, keywordLines, unmapped } = await import("./propose-entities.js");

const index = buildGraphIndex([], []);

describe("keywordLines", () => {
    it("takes one keyword per line, trimmed and without repeats", () => {
        expect(keywordLines(" trail shoes \n\nroad shoes\r\nTrail Shoes\n")).toStrictEqual(["trail shoes", "road shoes"]);
    });
});

describe("unmapped", () => {
    it("accepts a page without an entity and never the root", () => {
        expect(unmapped(aPage("a", "/a/"))).toBe(true);
        expect(unmapped(aPage("a", "/a/", { entityId: "e1" }))).toBe(false);
        expect(unmapped(aPage("root", "/"))).toBe(false);
    });
});

describe("the entity proposal drawer", () => {
    it("previews the picked pages, writes only the kept proposals and reports what landed", async () => {
        calls.previewed.length = 0;
        calls.applied.length = 0;
        render(<ProposeEntitiesDrawer open={true} onOpenChange={() => {}} siteId="s1" source="pages" index={index} onReview={() => {}} />);

        const preview = screen.getByRole("button", { name: copy.graph.ai.preview(0) });
        expect(preview.hasAttribute("disabled")).toBe(true);

        fireEvent.click(screen.getByRole("checkbox", { name: "mugs/" }));
        fireEvent.click(screen.getByRole("checkbox", { name: "travel/" }));
        fireEvent.click(screen.getByRole("button", { name: copy.graph.ai.preview(2) }));

        expect(calls.previewed).toStrictEqual([{ siteId: "s1", pageIds: ["mugs", "travel"] }]);
        await waitFor(() => {
            expect(screen.getByText(copy.graph.ai.reviewTitle(2))).toBeTruthy();
        });

        fireEvent.click(screen.getByRole("checkbox", { name: "Travel Mugs" }));
        fireEvent.click(screen.getByRole("button", { name: copy.graph.ai.create(1) }));

        expect(calls.applied).toHaveLength(1);
        expect(calls.applied[0]?.entities.map((held) => held.name)).toStrictEqual(["Mugs"]);
        await waitFor(() => {
            expect(screen.getByText(copy.graph.ai.applied(1, 1, 2, 0))).toBeTruthy();
        });
        expect(screen.getByRole("button", { name: copy.graph.ai.reviewEdges })).toBeTruthy();
    });

    it("turns the keyword lines into a proposal request", async () => {
        calls.keyworded.length = 0;
        render(<ProposeEntitiesDrawer open={true} onOpenChange={() => {}} siteId="s1" source="keywords" index={index} onReview={() => {}} />);

        fireEvent.change(screen.getByLabelText(copy.graph.ai.keywords), { target: { value: "trail shoes\nroad shoes\n" } });
        fireEvent.click(screen.getByRole("button", { name: copy.graph.ai.preview(2) }));

        expect(calls.keyworded).toStrictEqual([{ siteId: "s1", keywords: ["trail shoes", "road shoes"] }]);
        await waitFor(() => {
            expect(screen.getByText(copy.graph.ai.reviewTitle(1))).toBeTruthy();
        });
    });
});
