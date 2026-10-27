import { fireEvent, screen, within } from "@testing-library/react";
import { Route, Routes } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { readToasts, clearToasts } from "../../data/toasts.js";
import type { Page } from "../../data/types.js";
import { aPage } from "../../testing/pages.js";
import { aPageType } from "../../testing/pagetypes.js";
import { renderScreen } from "../../testing/render.js";
import { answerSiteMap, resetSiteMap, siteMapFake } from "../../testing/site-map.js";
import { aNode, aRef } from "../map/model/fixture.js";
import { forgetPages } from "./views/memory.js";

const said = copy.pages;

interface Calls {
    typed: unknown[];
    templated: unknown[];
    resolved: unknown[];
    counted: unknown[];
    listed: unknown[];
    asked: string[];
    run: (readonly string[])[];
    paths: (readonly (readonly [string, string])[])[];
}

const calls: Calls = { typed: [], templated: [], resolved: [], counted: [], listed: [], asked: [], run: [], paths: [] };

const answers: { typeError: Error | null; count: number } = { typeError: null, count: 3 };

const rows: Page[] = [
    aPage("p1", "/peptides/", { title: "Peptides", status: "published", pageTypeId: "t-landing", typeSource: "auto", typeReason: "rule", typeRuleId: "r7" }),
    aPage("p2", "/bpc-157/", { title: "BPC-157", status: "published", pageTypeId: "t-cat", typeSource: "manual", typeReason: "user" }),
    aPage("p3", "/lab-guides/", { title: "Lab guides", status: "published", pageTypeId: "t-guide", typeSource: "auto", typeReason: "unmatched" }),
];

const types = [
    aPageType("t-landing", "landing", { name: "Landing", pages: 1 }),
    aPageType("t-cat", "category", { name: "Category", pages: 1, templateId: "tpl-hub" }),
    aPageType("t-guide", "guide", { name: "Guide", pages: 7 }),
];

function writer(record: unknown[], answered: unknown, errorOf: () => Error | null = () => null) {
    return () => ({
        mutate: (request: unknown, options?: { onSuccess?: (answer: unknown) => void; onError?: (thrown: unknown) => void }) => {
            record.push(request);
            const failed = errorOf();
            if (failed === null) {
                options?.onSuccess?.(answered);
            } else {
                options?.onError?.(failed);
            }
        },
        reset: () => undefined,
        isPending: false,
        error: null,
    });
}

vi.mock("../../data/hooks/pages.js", async () => {
    const fake = await import("../../testing/site-map.js");
    return {
        useSiteMap: fake.fakeSiteMap,
        usePages: (filter: unknown) => {
            calls.listed.push(filter);
            return { data: { pages: [{ items: rows, hasMore: false }] }, isPending: false, hasNextPage: false, isFetchingNextPage: false, fetchNextPage: () => undefined };
        },
        usePageCount: (request: unknown) => {
            if (request !== null) {
                calls.counted.push(request);
            }
            return { data: request === null ? undefined : { pages: answers.count } };
        },
        useSelectionCount: (request: unknown) => ({ data: request === null ? undefined : { pageIds: ["x1", "x2", "x3", "x4"] }, error: null }),
        usePagePaths: () => new Map(),
        useSetPageTypes: writer(calls.typed, { typed: 2, refused: 1 }, () => answers.typeError),
        useSetPageTemplates: writer(calls.templated, { changed: 2 }),
        useResolveSelection: writer(calls.resolved, { pageIds: ["p1", "p2"], paths: { p1: "/peptides/", p2: "/bpc-157/" } }),
    };
});

vi.mock("../../data/hooks/reports.js", () => ({
    useSiteOverview: () => ({ data: { pages: { total: 3, byStatus: {}, unmapped: 0 } } }),
}));

vi.mock("../../data/hooks/sync.js", () => ({
    useSyncSite: () => ({ mutate: () => undefined, isPending: false }),
}));

vi.mock("../../data/hooks/graph.js", () => ({
    useEntities: () => ({ data: { pages: [{ items: [], hasMore: false }] }, hasNextPage: false, isFetchingNextPage: false, fetchNextPage: () => undefined, isPending: false }),
}));

vi.mock("../../data/hooks/templates.js", () => ({
    usePageTypes: () => ({ data: { types, fields: [] }, isPending: false }),
    useTemplates: (filter: { scope?: string }) => ({
        data: { pages: [{ items: filter.scope === "global" ? [{ id: "tpl-hub", name: "Hub page" }] : [], hasMore: false }] },
    }),
}));

vi.mock("../../data/hooks/classification.js", () => ({
    useRules: () => ({ data: { rules: [{ id: "r7", number: 7, name: "A product listing below the top level" }], fallback: null } }),
}));

vi.mock("./summary.js", () => ({ PageSummary: () => null }));
vi.mock("./drawer.js", () => ({ PageDrawer: () => null }));
vi.mock("./create.js", () => ({ PlanPageDialog: () => null }));

vi.mock("../runs/start.js", () => ({
    StartRunDrawer: ({ open, preselect, paths }: { open: boolean; preselect: readonly string[]; paths?: ReadonlyMap<string, string> }) => {
        if (open) {
            calls.run.push(preselect);
            calls.paths.push(paths === undefined ? [] : [...paths]);
        }
        return open ? <div data-run-drawer={preselect.join(",")} /> : null;
    },
}));

vi.mock("../agent/index.js", () => ({
    askAgent: (prefill: string) => {
        calls.asked.push(prefill);
    },
}));

const { PagesScreen } = await import("./screen.js");

beforeAll(() => {
    Element.prototype.hasPointerCapture = () => false;
    Element.prototype.releasePointerCapture = () => undefined;
});

beforeEach(() => {
    for (const record of [calls.typed, calls.templated, calls.resolved, calls.counted, calls.listed, calls.asked, calls.run, calls.paths]) {
        record.length = 0;
    }
    answers.typeError = null;
    answers.count = 3;
    forgetPages("s1");
    clearToasts();
    resetSiteMap();
});

function show(at = "/s/s1/pages") {
    return renderScreen(
        <Routes>
            <Route path="/s/:siteId/pages" element={<PagesScreen />} />
            <Route path="/s/:siteId/pages/:pageId" element={<PagesScreen />} />
            <Route path="/s/:siteId/runs/:runId" element={<span>run</span>} />
        </Routes>,
        at,
    );
}

function rowOf(pageId: string): HTMLElement {
    const held = document.querySelector<HTMLElement>(`[data-page-row][data-page-id="${pageId}"]`);
    if (held === null) {
        throw new Error(`no row ${pageId}`);
    }
    return held;
}

function bar() {
    return within(screen.getByRole("region", { name: said.bulk.label }));
}

function choose(menu: string, item: string): void {
    fireEvent.pointerDown(bar().getByRole("button", { name: menu }), { button: 0, ctrlKey: false, pointerType: "mouse" });
    fireEvent.click(screen.getByRole("menuitem", { name: item }));
}

function toasted(): string[] {
    return readToasts().map((toast) => toast.message);
}

describe("the pages as a list", () => {
    it("shows each page's type with the reason it has it", () => {
        show();

        expect(rowOf("p1").querySelector('[data-page-type="t-landing"]')?.textContent).toBe("Landing");
        expect(rowOf("p1").querySelector(`[title="${said.typeReason.rule(7, "A product listing below the top level")}"]`)).not.toBeNull();
        expect(rowOf("p2").querySelector(`[title="${said.typeReason.user}"]`)).not.toBeNull();
        expect(rowOf("p3").querySelector(`[title="${said.typeReason.unmatched}"]`)).not.toBeNull();
    });

    it("asks for the type, the text, the findings and the search the person filters by", () => {
        show("/s/s1/pages?type=t-guide&text=without&findings=1&q=mug");

        expect(calls.listed.at(-1)).toStrictEqual({ siteId: "s1", pageTypeId: "t-guide", body: "without", findings: true, search: "mug" });
        expect(calls.counted.at(-1)).toStrictEqual({ siteId: "s1", pageTypeId: "t-guide", body: "without", findings: true, search: "mug" });
    });

    it("selects rows by their boxes and a shift-range, and sets their type once the person confirms", () => {
        show();

        fireEvent.click(within(rowOf("p1")).getByRole("checkbox", { name: said.select.row("/peptides/") }));
        fireEvent.click(within(rowOf("p3")).getByRole("checkbox", { name: said.select.row("/lab-guides/") }), { shiftKey: true });
        expect(bar().getByText(said.bulk.selected(3))).toBeDefined();
        fireEvent.click(within(rowOf("p2")).getByRole("checkbox", { name: said.select.row("/bpc-157/") }));
        expect(bar().getByText(said.bulk.selected(2))).toBeDefined();

        choose(said.bulk.setType, "Guide");
        expect(calls.typed).toHaveLength(0);
        expect(screen.getByText(said.bulk.typeBody(2, "Guide"))).toBeDefined();
        fireEvent.click(screen.getByRole("button", { name: said.bulk.confirm }));

        expect(calls.typed).toStrictEqual([{ selection: { siteId: "s1", pageIds: ["p1", "p3"] }, pageTypeId: "t-guide" }]);
        expect(toasted()).toContain(said.bulk.typed(2, 1, "Guide"));
        expect(screen.queryByRole("region", { name: said.bulk.label })).toBeNull();
    });

    it("selects every page the filter matches by the count the person saw, and hands it back to the rules", () => {
        answers.count = 2140;
        show("/s/s1/pages?status=published");

        fireEvent.click(screen.getByRole("checkbox", { name: said.select.allMatching(2140) }));
        expect(bar().getByText(said.bulk.selected(2140))).toBeDefined();
        fireEvent.click(within(rowOf("p2")).getByRole("checkbox", { name: said.select.row("/bpc-157/") }));
        expect(bar().getByText(said.bulk.selected(2139))).toBeDefined();

        choose(said.bulk.setType, said.bulk.backToRules);
        fireEvent.click(screen.getByRole("button", { name: said.bulk.confirm }));

        expect(calls.typed).toStrictEqual([
            { selection: { siteId: "s1", filter: { status: "published" }, seen: 2140, except: ["p2"] }, pageTypeId: null },
        ]);
    });

    it("says the pages changed and drops what the filter matched when the count moved", () => {
        answers.count = 40;
        answers.typeError = new Error("moved", { cause: { code: "CONFLICT", message: "moved", details: { reason: "selection_moved" } } });
        show();

        fireEvent.click(screen.getByRole("checkbox", { name: said.select.allMatching(40) }));
        choose(said.bulk.setType, "Guide");
        fireEvent.click(screen.getByRole("button", { name: said.bulk.confirm }));

        expect(toasted()).toContain(said.bulk.moved);
        expect(screen.queryByRole("region", { name: said.bulk.label })).toBeNull();
    });

    it("sets a template, or makes the pages follow their type", () => {
        show();
        fireEvent.click(within(rowOf("p1")).getByRole("checkbox", { name: said.select.row("/peptides/") }));

        choose(said.bulk.template, "Hub page");
        fireEvent.click(screen.getByRole("button", { name: said.bulk.confirm }));
        fireEvent.click(within(rowOf("p1")).getByRole("checkbox", { name: said.select.row("/peptides/") }));
        choose(said.bulk.template, said.bulk.followType);
        fireEvent.click(screen.getByRole("button", { name: said.bulk.confirm }));

        expect(calls.templated).toStrictEqual([
            { selection: { siteId: "s1", pageIds: ["p1"] }, templateId: "tpl-hub" },
            { selection: { siteId: "s1", pageIds: ["p1"] }, templateId: null },
        ]);
        expect(toasted()).toContain(said.bulk.templated(2, "Hub page"));
    });

    it("starts a run over the resolved pages, and asks the agent about the selection with its paths", () => {
        show();
        fireEvent.click(within(rowOf("p1")).getByRole("checkbox", { name: said.select.row("/peptides/") }));
        fireEvent.click(within(rowOf("p2")).getByRole("checkbox", { name: said.select.row("/bpc-157/") }));

        fireEvent.click(bar().getByRole("button", { name: said.bulk.ask }));
        expect(calls.asked).toStrictEqual([said.bulk.prefill(2, said.describe.picked(2), ["/peptides/", "/bpc-157/"])]);

        fireEvent.click(bar().getByRole("button", { name: said.bulk.run }));
        expect(calls.resolved).toStrictEqual([{ siteId: "s1", pageIds: ["p1", "p2"] }]);
        expect(calls.run.at(-1)).toStrictEqual(["p1", "p2"]);
        expect(calls.paths.at(-1)).toStrictEqual([
            ["p1", "/peptides/"],
            ["p2", "/bpc-157/"],
        ]);
    });
});

describe("the pages as a tree", () => {
    const tree = (): void => {
        answerSiteMap({}, [
            aNode("hub", "/peptides/", { page: aRef("hub", "/peptides/", { type: "Landing" }), children: 2, descendants: 3 }),
            aNode("about", "/about/", { page: aRef("about", "/about/", { type: "Guide" }) }),
        ]);
        answerSiteMap({ parentId: "hub" }, [
            aNode("bpc", "/bpc-157/", { page: aRef("bpc", "/bpc-157/", { type: "Category" }), children: 1, descendants: 1 }),
            aNode("tb", "/tb-500/", { page: aRef("tb", "/tb-500/", { type: "Category" }) }),
        ]);
    };

    it("is remembered as the site's view, and ticks a page with everything under it", () => {
        tree();
        const first = show();
        fireEvent.click(screen.getByRole("radio", { name: said.views.tree }));
        first.unmount();
        show();

        fireEvent.click(screen.getByRole("checkbox", { name: said.select.branch("/peptides/") }));
        expect(bar().getByText(said.bulk.selected(4))).toBeDefined();
        fireEvent.click(screen.getByRole("button", { name: copy.map.expand("/peptides/") }));
        fireEvent.click(screen.getByRole("checkbox", { name: said.select.branch("/bpc-157/") }));
        expect(bar().getByText(said.bulk.selected(2))).toBeDefined();
        expect(document.querySelector('[data-map-node="hub"]')?.getAttribute("data-map-tick")).toBe("some");

        choose(said.bulk.setType, "Guide");
        fireEvent.click(screen.getByRole("button", { name: said.bulk.confirm }));
        expect(calls.typed).toStrictEqual([
            { selection: { siteId: "s1", subtrees: [{ pageId: "hub" }], exceptSubtrees: [{ pageId: "bpc" }] }, pageTypeId: "t-guide" },
        ]);
    });
});

describe("the pages by type", () => {
    it("lists one group per type with its pages and template, selects a whole group and shows its pages as a list or a tree", () => {
        answerSiteMap({ within: "t-guide" }, [aNode("lab", "/lab-guides/", { page: aRef("lab", "/lab-guides/", { type: "Guide" }) })]);
        show();
        fireEvent.click(screen.getByRole("radio", { name: said.views.types }));

        const groups = [...document.querySelectorAll("[data-type-group]")].map((held) => held.getAttribute("data-type-group"));
        expect(groups).toStrictEqual(["t-cat", "t-landing", "t-guide"]);
        expect(within(document.querySelector<HTMLElement>('[data-type-group="t-cat"]') ?? document.body).getByText(said.byType.template("Hub page"))).toBeDefined();
        expect(screen.getByRole("link", { name: said.byType.fieldsOf("Guide") }).getAttribute("href")).toBe("/settings/types?site=s1&type=t-guide");

        fireEvent.click(screen.getByRole("checkbox", { name: said.byType.group("Guide") }));
        expect(bar().getByText(said.bulk.selected(7))).toBeDefined();

        fireEvent.click(screen.getByRole("button", { name: said.byType.open("Guide") }));
        expect(calls.listed.at(-1)).toStrictEqual({ siteId: "s1", pageTypeId: "t-guide" });
        fireEvent.click(within(screen.getByRole("radiogroup", { name: said.byType.mode })).getByRole("radio", { name: said.views.tree }));
        expect(siteMapFake.asked.some((filter) => filter.within === "t-guide")).toBe(true);
        expect(document.querySelector('[data-map-node="lab"]')?.getAttribute("data-map-tick")).toBe("on");

        choose(said.bulk.setType, "Category");
        fireEvent.click(screen.getByRole("button", { name: said.bulk.confirm }));
        expect(calls.typed).toStrictEqual([{ selection: { siteId: "s1", filter: { pageTypeId: "t-guide" }, seen: 7 }, pageTypeId: "t-cat" }]);
    });
});
