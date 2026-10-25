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

const inFlight: { hang: boolean; signals: AbortSignal[] } = { hang: false, signals: [] };

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
    { pageId: "p3", path: "/shop/cups/", name: "Mugs", label: "Mugs", parent: "/shop/", parentEntityId: "e1", clash: "name_in_batch" },
    { pageId: "p4", path: "/shop/plates/", name: "Plates", label: "Plates", parent: "/shop/", existingEntityId: "e7" },
];

function mutation<Request, Answer>(record: (request: Request) => void, answer: (request: Request) => Answer) {
    return {
        mutate: (input: { request: Request; signal?: AbortSignal }, options?: { onSuccess?: (answered: Answer) => void }) => {
            record(input.request);
            if (inFlight.hang && input.signal !== undefined) {
                inFlight.signals.push(input.signal);
                return;
            }
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
    useGraph: () => ({ data: { entities: [{ id: "e1", name: "Shop" }], edges: [] }, isPending: false, error: null }),
    useApplyProposals: () =>
        mutation<{ siteId: string; entities: ProposedEntity[] }, { entities: unknown[]; edges: unknown[]; mapped: number; skipped: number }>(
            (request) => calls.applied.push(request),
            () => ({ entities: [{}], edges: [{}, {}], mapped: 1, skipped: 0 }),
        ),
}));

const { ProposeEntitiesDrawer } = await import("./propose-entities.js");
const { forgetProposal } = await import("./proposal-memory.js");

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
    inFlight.hang = false;
    inFlight.signals.length = 0;
    for (const siteId of ["s1", "s2"]) {
        forgetProposal(siteId, "pages");
        forgetProposal(siteId, "keywords");
        forgetProposal(siteId, "seed");
    }
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

        fireEvent.click(keepBox("Travel Mugs", "/shop/mugs/travel/"));
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
        expect(rowOf("p2").style.paddingLeft).not.toBe(rowOf("p1").style.paddingLeft);
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
        expect(keepBox("Blue Mug", "/shop/mugs/blue/").checked).toBe(false);
        fireEvent.click(keepBox("Blue Mug", "/shop/mugs/blue/"));
        expect(within(rowOf("p2")).getByText(said.orphan)).toBeTruthy();
        fireEvent.click(screen.getByRole("button", { name: said.create(2) }));

        expect(calls.applied).toStrictEqual([]);
        expect(screen.getByText(said.confirmBody(1, 1))).toBeTruthy();
        fireEvent.click(screen.getByRole("button", { name: said.confirmApply }));

        expect(calls.applied).toStrictEqual([
            { siteId: "s1", entities: [{ ...fromSite[1], parent: "", parentEntityId: "" }, fromSite[3]], source: "site" },
        ]);
        expect(closed).toStrictEqual([false]);
    });
});

function drawer(siteId: string, start: Parameters<typeof ProposeEntitiesDrawer>[0]["start"]) {
    return render(<ProposeEntitiesDrawer open={true} onOpenChange={() => {}} siteId={siteId} start={start} />);
}

const fromPages = { kind: "pick" as const, source: "pages" as const, index, onReview: () => {} };

function pickBoth(): void {
    fireEvent.click(screen.getByRole("checkbox", { name: "mugs/" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "travel/" }));
}

describe("what the entity proposal drawer keeps", () => {
    it("keeps the picked pages and the tree's search after the drawer is unmounted", async () => {
        const first = drawer("s1", fromPages);
        pickBoth();
        fireEvent.change(screen.getByLabelText(copy.pages.pick.search), { target: { value: "mug" } });
        first.unmount();

        drawer("s1", fromPages);

        expect(screen.getByLabelText<HTMLInputElement>(copy.pages.pick.search).value).toBe("mug");
        fireEvent.change(screen.getByLabelText(copy.pages.pick.search), { target: { value: "" } });
        await waitFor(() => {
            expect(screen.getByRole<HTMLInputElement>("checkbox", { name: "mugs/" }).checked).toBe(true);
        });
        expect(screen.getByRole<HTMLInputElement>("checkbox", { name: "travel/" }).checked).toBe(true);
        expect(screen.getByRole("button", { name: said.preview(2) }).hasAttribute("disabled")).toBe(false);
    });

    it("keeps the proposals and their ticks without asking the model again", async () => {
        const first = drawer("s1", fromPages);
        pickBoth();
        fireEvent.click(screen.getByRole("button", { name: said.preview(2) }));
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(2))).toBeTruthy();
        });
        fireEvent.click(keepBox("Travel Mugs", "/shop/mugs/travel/"));
        first.unmount();

        drawer("s1", fromPages);

        expect(screen.getByText(said.reviewTitle(2))).toBeTruthy();
        expect(keepBox("Travel Mugs", "/shop/mugs/travel/").checked).toBe(false);
        expect(screen.getByRole("button", { name: said.create(1) })).toBeTruthy();
        expect(calls.previewed).toHaveLength(1);
    });

    it("starts blank once the proposals are accepted", async () => {
        const first = drawer("s1", fromPages);
        pickBoth();
        fireEvent.click(screen.getByRole("button", { name: said.preview(2) }));
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(2))).toBeTruthy();
        });
        fireEvent.click(screen.getByRole("button", { name: said.create(2) }));
        await waitFor(() => {
            expect(screen.getByText(said.applied(1, 1, 2, 0))).toBeTruthy();
        });
        first.unmount();

        drawer("s1", fromPages);

        expect(screen.getByRole("button", { name: said.preview(0) }).hasAttribute("disabled")).toBe(true);
        expect(screen.getByRole<HTMLInputElement>("checkbox", { name: "mugs/" }).checked).toBe(false);
    });

    it("starts over when the person asks", async () => {
        const first = drawer("s1", fromPages);
        pickBoth();
        fireEvent.click(screen.getByRole("button", { name: said.preview(2) }));
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(2))).toBeTruthy();
        });

        fireEvent.click(screen.getByRole("button", { name: said.reset }));

        expect(screen.getByRole("button", { name: said.preview(0) })).toBeTruthy();
        expect(screen.getByRole<HTMLInputElement>("checkbox", { name: "mugs/" }).checked).toBe(false);
        expect(screen.queryByRole("button", { name: said.reset })).toBeNull();
        first.unmount();
        drawer("s1", fromPages);
        expect(screen.getByRole("button", { name: said.preview(0) })).toBeTruthy();
    });

    it("keeps each site's pick and each opener's apart", () => {
        const first = drawer("s1", fromPages);
        pickBoth();
        first.unmount();

        const other = drawer("s2", fromPages);
        expect(screen.getByRole("button", { name: said.preview(0) })).toBeTruthy();
        other.unmount();

        const keywords = drawer("s1", { ...fromPages, source: "keywords" });
        expect(screen.getByLabelText<HTMLTextAreaElement>(said.keywords).value).toBe("");
        keywords.unmount();

        drawer("s1", fromPages);
        expect(screen.getByRole("button", { name: said.preview(2) })).toBeTruthy();
    });

    it("keeps a findings seed's proposals and asks again for a different seed", async () => {
        const first = drawer("s1", { kind: "seed", seed: { siteId: "s1", pageIds: ["p1", "p2", "p3", "p4"] } });
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(4))).toBeTruthy();
        });
        fireEvent.click(keepBox("Blue Mug", "/shop/mugs/blue/"));
        first.unmount();

        const again = drawer("s1", { kind: "seed", seed: { siteId: "s1", pageIds: ["p4", "p3", "p2", "p1"] } });
        expect(calls.previewed).toHaveLength(1);
        expect(keepBox("Blue Mug", "/shop/mugs/blue/").checked).toBe(false);
        again.unmount();

        drawer("s1", { kind: "seed", seed: { siteId: "s1", pageIds: ["p1", "p2"] } });
        expect(calls.previewed).toHaveLength(2);
        await waitFor(() => {
            expect(keepBox("Blue Mug", "/shop/mugs/blue/").checked).toBe(true);
        });
    });

    it("unticks a parent with its subtree and shows a partial parent", async () => {
        drawer("s1", fromPages);
        pickBoth();
        fireEvent.click(screen.getByRole("button", { name: said.preview(2) }));
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(2))).toBeTruthy();
        });

        fireEvent.click(keepBox("Mugs", "/shop/mugs/"));
        expect(keepBox("Travel Mugs", "/shop/mugs/travel/").checked).toBe(false);
        expect(screen.getByText(said.tickedOf(0, 2))).toBeTruthy();

        fireEvent.click(keepBox("Travel Mugs", "/shop/mugs/travel/"));
        expect(rowOf("mugs").dataset.proposalState).toBe("some");
        expect(keepBox("Mugs", "/shop/mugs/").getAttribute("aria-checked")).toBe("mixed");

        fireEvent.click(keepBox("Mugs", "/shop/mugs/"));
        expect(screen.getByText(said.tickedOf(2, 2))).toBeTruthy();
    });

    it("writes a proposal under the name the person gave it, and keeps the edit across an unmount", async () => {
        const first = drawer("s1", fromPages);
        pickBoth();
        fireEvent.click(screen.getByRole("button", { name: said.preview(2) }));
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(2))).toBeTruthy();
        });
        fireEvent.click(within(rowOf("travel")).getByText("Travel Mugs"));
        fireEvent.change(screen.getByLabelText(said.nameField), { target: { value: "Travel cups" } });
        expect(within(rowOf("travel")).getByText(said.edited)).toBeTruthy();
        first.unmount();

        drawer("s1", fromPages);
        expect(screen.getByLabelText<HTMLInputElement>(said.nameField).value).toBe("Travel cups");
        fireEvent.click(screen.getByRole("button", { name: said.create(2) }));

        expect(calls.applied).toStrictEqual([
            {
                siteId: "s1",
                entities: [proposals[0], { ...proposals[1], name: "Travel cups", label: "Travel cups", parent: "/shop/mugs/", parentEntityId: "" }],
            },
        ]);
    });

    it("refuses an empty name and a name its parent already holds", async () => {
        drawer("s1", fromPages);
        pickBoth();
        fireEvent.click(screen.getByRole("button", { name: said.preview(2) }));
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(2))).toBeTruthy();
        });
        fireEvent.click(within(rowOf("travel")).getByText("Travel Mugs"));

        fireEvent.change(screen.getByLabelText(said.nameField), { target: { value: " " } });
        expect(within(rowOf("travel")).getAllByText(said.nameNeeded).length).toBeGreaterThan(0);
        expect(screen.getByRole("button", { name: said.create(1) })).toBeTruthy();

        fireEvent.change(screen.getByLabelText(said.nameField), { target: { value: "Travel Mugs" } });
        expect(screen.getByRole("button", { name: said.create(2) })).toBeTruthy();
    });

    it("keeps the ancestors of a search match in view", async () => {
        drawer("s1", fromPages);
        pickBoth();
        fireEvent.click(screen.getByRole("button", { name: said.preview(2) }));
        await waitFor(() => {
            expect(screen.getByText(said.reviewTitle(2))).toBeTruthy();
        });
        fireEvent.click(screen.getByRole("button", { name: said.collapseAll }));
        expect(document.querySelector('[data-proposal-page="travel"]')).toBeNull();

        fireEvent.change(screen.getByLabelText(said.searchProposals), { target: { value: "travel" } });
        expect(rowOf("mugs")).toBeTruthy();
        expect(rowOf("travel")).toBeTruthy();

        fireEvent.change(screen.getByLabelText(said.searchProposals), { target: { value: "nothing like it" } });
        expect(screen.getByText(said.noMatchProposals)).toBeTruthy();
    });

    it("returns to the pick with its pages when leaving cuts a preview short", () => {
        inFlight.hang = true;
        const first = drawer("s1", fromPages);
        pickBoth();
        fireEvent.click(screen.getByRole("button", { name: said.preview(2) }));
        first.unmount();

        expect(inFlight.signals.map((signal) => signal.aborted)).toStrictEqual([true]);
        drawer("s1", fromPages);
        expect(screen.getByRole<HTMLInputElement>("checkbox", { name: "mugs/" }).checked).toBe(true);
        expect(screen.getByRole("button", { name: said.preview(2) })).toBeTruthy();
    });
});
