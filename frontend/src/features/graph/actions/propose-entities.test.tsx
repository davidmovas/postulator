import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { PreviewFromPagesRequest, ProposedEntity } from "../../../data/types.js";
import { answerSiteMap, resetSiteMap } from "../../../testing/site-map.js";
import { aNode } from "../../map/model/fixture.js";
import { buildGraphIndex } from "../model/index.js";

interface Calls {
    previewed: PreviewFromPagesRequest[];
    keyworded: { siteId: string; keywords: string[]; parentEntityId?: string }[];
    applied: { siteId: string; entities: ProposedEntity[]; source?: string }[];
}

const calls: Calls = { previewed: [], keyworded: [], applied: [] };

const proposals: ProposedEntity[] = [
    { pageId: "mugs", path: "/shop/mugs/", name: "Mugs", keywords: [{ text: "mugs" }], parent: "/shop/" },
    {
        pageId: "travel",
        path: "/shop/mugs/travel/",
        name: "Travel Mugs",
        keywords: [{ text: "travel mugs" }],
        parent: "/shop/mugs/",
    },
];

const fromSite: ProposedEntity[] = [
    { pageId: "p1", path: "/shop/mugs/", name: "Mugs", label: "Mugs", parent: "/shop/", parentEntityId: "e1" },
    {
        pageId: "p2",
        path: "/shop/mugs/blue/",
        name: "Blue Mug",
        label: "Blue Mug",
        parent: "/shop/mugs/",
        facts: [
            { label: "Colour", value: "Blue" },
            { label: "Size", value: "L" },
        ],
    },
    { pageId: "p3", path: "/shop/cups/", name: "Mugs", label: "Mugs", parent: "/shop/", clash: "name_in_batch" },
    { pageId: "p4", path: "/shop/plates/", name: "Plates", label: "Plates", parent: "/shop/", existingEntityId: "e7" },
];

function mutation<Request, Answer>(record: (request: Request) => void, answer: (request: Request) => Answer) {
    return {
        mutate: (input: { request: Request }, options?: { onSuccess?: (answered: Answer) => void }) => {
            record(input.request);
            options?.onSuccess?.(answer(input.request));
        },
        reset: () => {},
        isPending: false,
        error: null,
    };
}

vi.mock("../../../data/hooks/pages.js", async () => {
    const fake = await import("../../../testing/site-map.js");
    return { useSiteMap: fake.fakeSiteMap };
});

vi.mock("../../../data/hooks/graph.js", () => ({
    usePreviewFromPages: () =>
        mutation<PreviewFromPagesRequest, { entities: ProposedEntity[]; pages: number; skipped: number; tokens: number }>(
            (request) => calls.previewed.push(request),
            (request) =>
                request.source === "site"
                    ? { entities: fromSite, pages: 4, skipped: 0, tokens: 0 }
                    : { entities: proposals, pages: 2, skipped: 0, tokens: 12 },
        ),
    useProposeFromKeywords: () =>
        mutation<{ siteId: string; keywords: string[]; parentEntityId?: string }, { entities: ProposedEntity[]; skipped: number; tokens: number }>(
            (request) => calls.keyworded.push(request),
            () => ({ entities: [proposals[1] as ProposedEntity], skipped: 0, tokens: 12 }),
        ),
    useApplyProposals: () =>
        mutation<{ siteId: string; entities: ProposedEntity[] }, { entities: unknown[]; edges: unknown[]; mapped: number; skipped: number }>(
            (request) => calls.applied.push(request),
            () => ({ entities: [{}], edges: [{}, {}], mapped: 1, skipped: 0 }),
        ),
}));

const { ProposeEntitiesDrawer } = await import("./propose-entities.js");

const index = buildGraphIndex([], []);

const said = copy.graph.ai;

function rowOf(pageId: string): HTMLElement {
    const row = document.querySelector<HTMLElement>(`[data-proposal-page="${pageId}"]`);
    if (row === null) {
        throw new Error(`no row for ${pageId}`);
    }
    return row;
}

function keepBox(label: string, path: string): HTMLInputElement {
    return screen.getByRole<HTMLInputElement>("checkbox", { name: said.keepRow(label, path) });
}

beforeEach(() => {
    calls.previewed.length = 0;
    calls.keyworded.length = 0;
    calls.applied.length = 0;
    resetSiteMap();
    answerSiteMap({}, [aNode("shop", "/shop/", { entityId: "e1", children: 1, descendants: 2 }), aNode("mugs", "/shop/mugs/"), aNode("travel", "/shop/mugs/travel/")]);
});

describe("the entity proposal drawer", () => {
    it("previews the picked pages from the model, writes only the kept proposals and reports what landed", async () => {
        render(
            <ProposeEntitiesDrawer
                open={true}
                onOpenChange={() => {}}
                siteId="s1"
                start={{ kind: "pick", source: "pages", index, onReview: () => {} }}
            />,
        );

        const preview = screen.getByRole("button", { name: said.preview(0) });
        expect(preview.hasAttribute("disabled")).toBe(true);

        fireEvent.click(screen.getByRole("checkbox", { name: "mugs/" }));
        fireEvent.click(screen.getByRole("checkbox", { name: "travel/" }));
        fireEvent.click(screen.getByRole("button", { name: said.preview(2) }));

        expect(calls.previewed).toStrictEqual([{ siteId: "s1", pageIds: ["mugs", "travel"] }]);
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(2))).toBeTruthy();
        });

        fireEvent.click(screen.getByRole("checkbox", { name: "Travel Mugs" }));
        fireEvent.click(screen.getByRole("button", { name: said.create(1) }));

        expect(calls.applied).toStrictEqual([{ siteId: "s1", entities: [proposals[0]] }]);
        await waitFor(() => {
            expect(screen.getByText(said.applied(1, 1, 2, 0))).toBeTruthy();
        });
        expect(screen.getByRole("button", { name: said.reviewEdges })).toBeTruthy();
    });

    it("turns the keyword lines into a proposal request", async () => {
        render(
            <ProposeEntitiesDrawer
                open={true}
                onOpenChange={() => {}}
                siteId="s1"
                start={{ kind: "pick", source: "keywords", index, onReview: () => {} }}
            />,
        );

        fireEvent.change(screen.getByLabelText(said.keywords), { target: { value: "trail shoes\nroad shoes\n" } });
        fireEvent.click(screen.getByRole("button", { name: said.preview(2) }));

        expect(calls.keyworded).toStrictEqual([{ siteId: "s1", keywords: ["trail shoes", "road shoes"] }]);
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(1))).toBeTruthy();
        });
    });

    it("opens a seed at its site preview, keeps a clash unticked and writes the ticked rows after the confirmation", async () => {
        const closed: boolean[] = [];
        const seed: PreviewFromPagesRequest = { siteId: "s1", pageIds: ["p1", "p2", "p3", "p4"] };
        render(
            <ProposeEntitiesDrawer
                open={true}
                onOpenChange={(next) => {
                    closed.push(next);
                }}
                siteId="s1"
                start={{ kind: "seed", seed }}
            />,
        );

        expect(calls.previewed).toStrictEqual([{ siteId: "s1", pageIds: ["p1", "p2", "p3", "p4"], source: "site" }]);
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(4))).toBeTruthy();
        });
        expect(screen.queryByRole("button", { name: said.back })).toBeNull();

        const blue = within(rowOf("p2"));
        expect(blue.getByText("Blue Mug")).toBeTruthy();
        expect(blue.getByText("/shop/mugs/blue/")).toBeTruthy();
        expect(blue.getByText("/shop/mugs/")).toBeTruthy();
        expect(blue.getByText(said.facts([{ label: "Colour", value: "Blue" }, { label: "Size", value: "L" }]))).toBeTruthy();

        const clashing = keepBox("Mugs", "/shop/cups/");
        expect(clashing.checked).toBe(false);
        expect(clashing.disabled).toBe(true);
        expect(within(rowOf("p3")).getByText(said.clashes.name_in_batch)).toBeTruthy();
        expect(within(rowOf("p4")).getByText(said.joins)).toBeTruthy();

        fireEvent.click(screen.getByRole("button", { name: said.keepNone }));
        expect(keepBox("Blue Mug", "/shop/mugs/blue/").checked).toBe(false);
        fireEvent.click(screen.getByRole("button", { name: said.keepAll }));
        expect(keepBox("Blue Mug", "/shop/mugs/blue/").checked).toBe(true);
        expect(keepBox("Mugs", "/shop/cups/").checked).toBe(false);

        fireEvent.click(keepBox("Mugs", "/shop/mugs/"));
        fireEvent.click(screen.getByRole("button", { name: said.create(2) }));

        expect(calls.applied).toStrictEqual([]);
        expect(screen.getByText(said.confirmBody(1, 1))).toBeTruthy();
        fireEvent.click(screen.getByRole("button", { name: said.confirmApply }));

        expect(calls.applied).toStrictEqual([{ siteId: "s1", entities: [fromSite[1], fromSite[3]], source: "site" }]);
        expect(closed).toStrictEqual([false]);
    });
});
