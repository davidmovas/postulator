import { fireEvent, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { CategoryNode } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import type { EntityIndex } from "./entities.js";
import { defaultQuery } from "./params.js";
import type { PagesQuery } from "./params.js";

const held: { categories: CategoryNode[] | null; pending: boolean } = { categories: [], pending: false };

vi.mock("../../data/hooks/pages.js", async (importOriginal) => ({
    ...(await importOriginal<typeof import("../../data/hooks/pages.js")>()),
    useCategoryTree: () => ({
        data: held.categories === null ? undefined : { categories: held.categories },
        isPending: held.pending,
        error: null,
    }),
}));

vi.mock("../../data/hooks/reports.js", async (importOriginal) => ({
    ...(await importOriginal<typeof import("../../data/hooks/reports.js")>()),
    useSiteOverview: () => ({
        data: { pages: { total: 40, unmapped: 3, byStatus: { planned: 10 } } },
        isPending: false,
        error: null,
    }),
}));

const { PageRail } = await import("./rail.js");

const said = copy.pages.categories;

const index: EntityIndex = { entities: [], byId: new Map(), labels: new Map(), complete: true, loading: false };

const tree: CategoryNode[] = [
    { id: "peptides", name: "Peptides", parentId: null, pages: 5, termIds: { category: 12, productCategory: 31 } },
    { id: "tools", name: "Tools", parentId: null, pages: 1, termIds: {} },
    { id: "healing", name: "Healing", parentId: "peptides", pages: 3, termIds: {} },
    { id: "liquid", name: "Liquid", parentId: "healing", pages: 2, termIds: { category: 14 } },
];

function show(query: Partial<PagesQuery> = {}, disabled = false) {
    const onChange = vi.fn();
    renderScreen(
        <PageRail siteId="s1" query={{ ...defaultQuery, ...query }} index={index} disabled={disabled} onChange={onChange} />,
    );
    return onChange;
}

function filter(): HTMLElement {
    const found = document.querySelector<HTMLElement>("[data-category-filter]");
    if (found === null) {
        throw new Error("the rail has no category filter");
    }
    return found;
}

function nodeOf(id: string): HTMLElement {
    const found = filter().querySelector<HTMLElement>(`[data-category-node="${id}"]`);
    if (found === null) {
        throw new Error(`the tree shows no ${id}`);
    }
    return found;
}

function shownIds(): (string | undefined)[] {
    return [...filter().querySelectorAll<HTMLElement>("[data-category-node]")].map((item) => item.dataset["categoryNode"]);
}

beforeEach(() => {
    held.categories = tree;
    held.pending = false;
});

describe("the category tree in the pages rail", () => {
    it("lists the root categories with their page counts and where each stands on WordPress", () => {
        show();
        expect(within(filter()).getByText(said.title)).toBeDefined();
        expect(shownIds()).toStrictEqual(["peptides", "tools"]);

        const peptides = within(nodeOf("peptides"));
        expect(peptides.getByText("5")).toBeDefined();
        expect(peptides.getByText("#12")).toBeDefined();
        const filed = peptides.getByRole("button", { pressed: false });
        expect(filed.getAttribute("title")).toContain(said.onSiteBoth(12, 31));
        expect(filed.getAttribute("title")).toContain(said.pagesUnder(5));

        const tools = nodeOf("tools");
        expect(within(tools).getByText(said.newLabel)).toBeDefined();
        expect(tools.querySelector("[data-category-chip]")?.getAttribute("data-category-chip")).toBe("onPublish");
        expect(within(filter()).getByText(said.legend)).toBeDefined();
    });

    it("opens and shuts a branch", () => {
        show();
        fireEvent.click(screen.getByRole("button", { name: said.expand("Peptides") }));
        expect(shownIds()).toStrictEqual(["peptides", "healing", "tools"]);
        fireEvent.click(screen.getByRole("button", { name: said.expand("Healing") }));
        expect(shownIds()).toStrictEqual(["peptides", "healing", "liquid", "tools"]);
        expect(within(nodeOf("liquid")).getByText("#14")).toBeDefined();
        fireEvent.click(screen.getByRole("button", { name: said.collapse("Peptides") }));
        expect(shownIds()).toStrictEqual(["peptides", "tools"]);
    });

    it("filters the list by a branch and keeps the other filters", () => {
        const onChange = show({ status: "published" });
        fireEvent.click(within(nodeOf("peptides")).getByRole("button", { pressed: false }));
        expect(onChange).toHaveBeenCalledWith({ ...defaultQuery, status: "published", categoryId: "peptides" });
    });

    it("opens the branch down to the category the address filters by, and marks it", () => {
        show({ categoryId: "liquid" });
        expect(shownIds()).toStrictEqual(["peptides", "healing", "liquid", "tools"]);
        expect(within(nodeOf("liquid")).getByRole("button", { pressed: true })).toBeDefined();
        expect(screen.getByRole("button", { name: new RegExp(`^${said.all}`) }).getAttribute("aria-pressed")).toBe("false");
    });

    it("lets go of a category clicked twice, and of every category through All pages", () => {
        const onChange = show({ categoryId: "liquid" });
        fireEvent.click(within(nodeOf("liquid")).getByRole("button", { pressed: true }));
        expect(onChange).toHaveBeenLastCalledWith({ ...defaultQuery, categoryId: "" });
        fireEvent.click(screen.getByRole("button", { name: new RegExp(`^${said.all}`) }));
        expect(onChange).toHaveBeenLastCalledWith({ ...defaultQuery, categoryId: "" });
    });

    it("counts every page on All pages", () => {
        show();
        const all = screen.getByRole("button", { name: new RegExp(`^${said.all}`) });
        expect(all.textContent).toContain("40");
        expect(all.getAttribute("aria-pressed")).toBe("true");
    });

    it("says where categories come from while the site has none", () => {
        held.categories = [];
        show();
        expect(within(filter()).getByText(said.empty)).toBeDefined();
        expect(within(filter()).getByRole("link", { name: said.importSheet }).getAttribute("href")).toBe("/s/s1/import");
        expect(screen.queryByRole("button", { name: new RegExp(`^${said.all}`) })).toBeNull();
    });

    it("says it is loading before the tree arrives", () => {
        held.categories = null;
        held.pending = true;
        show();
        expect(within(filter()).getByText(said.loading)).toBeDefined();
    });

    it("stays shut while the tree view reads the whole site", () => {
        show({}, true);
        expect(within(nodeOf("peptides")).getByRole("button", { pressed: false }).hasAttribute("disabled")).toBe(true);
        expect(screen.getByRole("button", { name: said.expand("Peptides") }).hasAttribute("disabled")).toBe(true);
    });

    it("resets a category filter with the other filters", () => {
        const onChange = show({ categoryId: "tools", sort: { field: "path", desc: true } });
        fireEvent.click(screen.getByRole("button", { name: copy.pages.filters.reset }));
        expect(onChange).toHaveBeenCalledWith({ ...defaultQuery, sort: { field: "path", desc: true } });
    });
});
