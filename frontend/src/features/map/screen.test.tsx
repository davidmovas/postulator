import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { SiteMapFilter, SiteMapNode, SiteMapView } from "../../data/types.js";
import { aNode, aRef } from "./model/fixture.js";

interface Answer {
    view: SiteMapView;
    more: boolean;
}

const held: { answers: Map<string, Answer>; asked: SiteMapFilter[] } = { answers: new Map(), asked: [] };

const stable = new Map<string, unknown>();

function keyOf(filter: SiteMapFilter): string {
    return JSON.stringify({ parentId: filter.parentId ?? "", search: filter.search ?? "", pageTypeId: filter.pageTypeId ?? "" });
}

function view(nodes: readonly SiteMapNode[], extra: Partial<SiteMapView> = {}): SiteMapView {
    return { trail: [], nodes: { items: [...nodes], hasMore: false }, ...extra };
}

function answer(filter: Partial<SiteMapFilter>, nodes: readonly SiteMapNode[], more = false, extra: Partial<SiteMapView> = {}): void {
    held.answers.set(keyOf({ siteId: "s1", ...filter }), { view: view(nodes, extra), more });
}

vi.mock("../../data/hooks/pages.js", () => ({
    useSiteMap: (filter: SiteMapFilter, enabled = true) => {
        const key = `${keyOf(filter)}:${String(enabled)}`;
        if (enabled) {
            held.asked.push(filter);
        }
        const found = enabled ? held.answers.get(keyOf(filter)) : undefined;
        if (!stable.has(key)) {
            stable.set(key, {
                data: found === undefined ? undefined : { pages: [found.view] },
                isPending: found === undefined,
                isError: false,
                error: null,
                hasNextPage: found?.more ?? false,
                isFetchingNextPage: false,
                fetchNextPage: () => undefined,
            });
        }
        return stable.get(key);
    },
}));

vi.mock("../../data/hooks/templates.js", () => ({
    usePageTypes: () => ({ data: { types: [] }, isPending: false }),
}));

vi.mock("../../data/hooks/sync.js", () => ({
    useSyncSite: () => ({ mutate: () => undefined, isPending: false }),
}));

vi.mock("../pages/index.js", () => ({
    PageDrawer: ({ pageId }: { pageId: string }) => <div data-drawer-page={pageId} />,
    useEntityIndex: () => ({ entities: [], labels: new Map() }),
    readTab: () => "details",
    withTab: (params: URLSearchParams) => params,
}));

const { SiteMapScreen } = await import("./screen.js");

const said = copy.map;

function siteMap(at = "/s/s1/map") {
    return render(
        <MemoryRouter initialEntries={[at]}>
            <Routes>
                <Route path="/s/:siteId/map" element={<SiteMapScreen />} />
                <Route path="/s/:siteId/map/:pageId" element={<SiteMapScreen />} />
            </Routes>
        </MemoryRouter>,
    );
}

const shop = aNode("p-shop", "/shop/", {
    page: aRef("p-shop", "/shop/", { title: "Shop" }),
    twin: aRef("p-cat", "/product-category/shop/", { title: "Shop", type: "Store category" }),
    children: 2,
    descendants: 5,
    findings: 3,
    linksIn: 4,
    linksOut: 7,
});
const about = aNode("p-about", "/about/", { page: aRef("p-about", "/about/", { title: "About" }), body: "ours" as SiteMapNode["body"] });
const mugs = aNode("p-mugs", "/shop/mugs/", { page: aRef("p-mugs", "/shop/mugs/", { title: "Mugs" }) });

beforeEach(() => {
    held.answers = new Map();
    held.asked = [];
    stable.clear();
    answer({}, [shop, about]);
    answer({ parentId: "p-shop" }, [mugs], true);
});

describe("the site map", () => {
    it("reads the roots alone until a node is expanded", () => {
        siteMap();

        expect(screen.getByText("Shop")).toBeDefined();
        expect(screen.getByText("About")).toBeDefined();
        expect(held.asked.every((filter) => filter.parentId === undefined)).toBe(true);
        expect(screen.queryByText("Mugs")).toBeNull();
    });

    it("loads a node's children when it is expanded, with a row for the next page of them", () => {
        siteMap();

        fireEvent.click(screen.getByRole("button", { name: said.expand("/shop/") }));

        expect(held.asked).toContainEqual({ siteId: "s1", parentId: "p-shop" });
        expect(screen.getByText("Mugs")).toBeDefined();
        expect(screen.getByRole("button", { name: said.more })).toBeDefined();
        expect(screen.getByRole("button", { name: said.collapse("/shop/") }).getAttribute("aria-expanded")).toBe("true");
    });

    it("words a node: its type, where its parent came from, its body, its links, its findings and what sits below", () => {
        siteMap();

        const row = within(screen.getByText("Shop").closest("[role=row]") as HTMLElement);
        expect(row.getByText("Hub")).toBeDefined();
        expect(row.getByText(`${copy.pages.model.parentReasons.jsonld_breadcrumb} · ${copy.pages.model.parentMethodNames.href}`)).toBeDefined();
        expect(row.getByText(said.bodyStates.client)).toBeDefined();
        expect(row.getByText(said.links(4, 7))).toBeDefined();
        expect(row.getByRole("link", { name: "3" }).getAttribute("href")).toBe("/s/s1/findings?under=p-shop");
        expect(row.getByText("2").getAttribute("title")).toBe(said.childrenTooltip(2, 5));
    });

    it("shows a hub's store category beside it as one node, and opens the category's own drawer", () => {
        const { container } = siteMap();

        const twin = screen.getByRole("button", { name: said.twin("/product-category/shop/") });
        expect(twin.getAttribute("title")).toBe(said.twinTooltip);
        fireEvent.click(twin);

        expect(container.querySelector("[data-drawer-page]")?.getAttribute("data-drawer-page")).toBe("p-cat");
    });

    it("opens a page's drawer from its title", () => {
        const { container } = siteMap();

        fireEvent.click(within(screen.getByText("About").closest("[role=row]") as HTMLElement).getAllByTitle(said.open)[0] as HTMLElement);

        expect(container.querySelector("[data-drawer-page]")?.getAttribute("data-drawer-page")).toBe("p-about");
    });

    it("roots the tree at a node, with its trail as a breadcrumb", () => {
        answer({ parentId: "p-mugs" }, [aNode("p-blue", "/shop/mugs/blue/", { page: aRef("p-blue", "/shop/mugs/blue/", { title: "Blue mugs" }) })], false, {
            parent: mugs,
            trail: [aRef("p-shop", "/shop/", { title: "Shop" })],
        });
        siteMap("/s/s1/map?under=p-mugs");

        const crumbs = within(screen.getByRole("navigation", { name: said.subtree }));
        expect(crumbs.getByRole("button", { name: said.whole })).toBeDefined();
        expect(crumbs.getByRole("button", { name: "Shop" })).toBeDefined();
        expect(crumbs.getByText("Mugs")).toBeDefined();
        expect(screen.getByText("Blue mugs")).toBeDefined();
    });

    it("lists flat matches of a search, each with its trail", async () => {
        answer({ search: "mug" }, [aNode("p-mugs", "/shop/mugs/", { page: aRef("p-mugs", "/shop/mugs/", { title: "Mugs" }), trail: [aRef("p-shop", "/shop/", { title: "Shop" })] })]);
        siteMap();

        fireEvent.change(screen.getByRole("searchbox", { name: said.search }), { target: { value: "mug" } });

        expect(await screen.findByText(said.trail(["Shop"]))).toBeDefined();
        expect(held.asked).toContainEqual({ siteId: "s1", search: "mug" });
    });

    it("explains an empty map and offers a sync and an import", () => {
        answer({}, []);
        siteMap();

        expect(screen.getByText(said.empty.title)).toBeDefined();
        expect(screen.getByRole("button", { name: said.empty.sync })).toBeDefined();
        expect(screen.getByRole("button", { name: said.empty.import })).toBeDefined();
    });
});
