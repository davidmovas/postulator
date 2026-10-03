import { screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { Page } from "../../data/types.js";
import { aPage } from "../../testing/pages.js";
import { renderScreen } from "../../testing/render.js";
import type { EntityIndex } from "./entities.js";
import { defaultQuery } from "./params.js";
import type { TreeRow } from "./tree-model.js";
import type { TreeView } from "./tree-state.js";

const held: { pages: Page[]; page: Page | undefined } = { pages: [], page: undefined };

vi.mock("../../data/hooks/pages.js", async (importOriginal) => ({
    ...(await importOriginal<typeof import("../../data/hooks/pages.js")>()),
    usePages: () => ({
        data: { pages: [{ items: held.pages, hasMore: false }] },
        isPending: false,
        hasNextPage: false,
        isFetchingNextPage: false,
        fetchNextPage: () => Promise.resolve(),
        error: null,
    }),
    usePage: () => ({
        data: held.page === undefined ? undefined : { page: held.page, links: [] },
        isPending: false,
        error: null,
    }),
    useDeletePage: () => ({ mutate: () => {}, isPending: false }),
}));

vi.mock("../../data/hooks/graph.js", async (importOriginal) => ({
    ...(await importOriginal<typeof import("../../data/hooks/graph.js")>()),
    useEntity: () => ({ data: undefined, isPending: false, error: null }),
}));

vi.mock("../../data/hooks/reports.js", async (importOriginal) => ({
    ...(await importOriginal<typeof import("../../data/hooks/reports.js")>()),
    usePageReport: () => ({ data: undefined, isPending: false, error: null }),
}));

vi.mock("../../data/hooks/sites.js", async (importOriginal) => ({
    ...(await importOriginal<typeof import("../../data/hooks/sites.js")>()),
    useSite: () => ({ data: undefined, isPending: false, error: null }),
}));

vi.mock("./details.js", () => ({ PageDetails: () => null }));
vi.mock("./links.js", () => ({ PageLinks: () => null }));
vi.mock("./mapping.js", () => ({ PageMapping: () => null }));
vi.mock("./preview/tab.js", () => ({ PreviewTab: () => null }));
vi.mock("./report.js", () => ({ PageReportPanel: () => null }));
vi.mock("./template.js", () => ({ TemplatePanel: () => null }));

const { PageTable } = await import("./table.js");
const { PageTree } = await import("./tree.js");
const { PageSummary } = await import("./summary.js");
const { PageDrawer } = await import("./drawer.js");

const filed = [
    { entityId: "peptides", name: "Peptides", termId: 12 },
    { entityId: "healing", name: "Healing" },
];

const index: EntityIndex = { entities: [], byId: new Map(), labels: new Map(), complete: true, loading: false };

function filedPage(overrides: Partial<Page> = {}): Page {
    return aPage("p1", "/peptides/healing/", { entityId: "healing", categories: filed, ...overrides });
}

function statesIn(container: HTMLElement): (string | undefined)[] {
    const trail = within(container).getByRole("list", { name: copy.categories.trail });
    return [...trail.querySelectorAll("li")].map((item) => item.dataset["categoryState"]);
}

function rowOf(path: string): HTMLElement {
    const row = screen.getAllByText(path)[0]?.closest('[role="row"]');
    if (row === null || row === undefined) {
        throw new Error(`no row carries ${path}`);
    }
    return row as HTMLElement;
}

beforeEach(() => {
    held.pages = [];
    held.page = undefined;
});

describe("the page table", () => {
    it("gives every page a Categories column with its trail", () => {
        held.pages = [filedPage(), aPage("p2", "/about/")];
        renderScreen(
            <PageTable
                siteId="s1"
                query={defaultQuery}
                index={index}
                selectedId="p1"
                onQueryChange={() => {}}
                onSelect={() => {}}
                onOpen={() => {}}
                onCreate={() => {}}
                onImport={() => {}}
                onSync={() => {}}
                syncing={false}
            />,
        );

        expect(screen.getByText(copy.pages.columns.categories)).toBeDefined();
        expect(statesIn(rowOf("/peptides/healing/"))).toStrictEqual(["onSite", "onPublish"]);
        expect(within(rowOf("/about/")).queryByRole("list", { name: copy.categories.trail })).toBeNull();
    });
});

describe("the page tree", () => {
    it("gives every row a Categories column with its trail", () => {
        const page = filedPage({ categoriesNeedPlugin: true });
        const rows: TreeRow[] = [{ kind: "page", id: page.id, page, depth: 0, childCount: 0, expanded: false }];
        const view: TreeView = {
            armed: true,
            pending: false,
            sizing: false,
            total: 1,
            nodes: 1,
            rows,
            arm: () => {},
            expandAll: () => {},
            collapseAll: () => {},
            toggle: () => {},
            lift: () => {},
        };
        renderScreen(
            <PageTree view={view} index={index} selectedId={null} onSelect={() => {}} onOpen={() => {}} onCreate={() => {}} />,
        );

        expect(screen.getByText(copy.pages.columns.categories)).toBeDefined();
        expect(statesIn(rowOf("/peptides/healing/"))).toStrictEqual(["needsPlugin", "needsPlugin"]);
    });
});

describe("the page summary", () => {
    function summary(): HTMLElement {
        renderScreen(<PageSummary pageId="p1" siteId="s1" search="" onOpen={() => {}} />);
        const section = document.querySelector<HTMLElement>("[data-page-categories]");
        if (section === null) {
            throw new Error("the summary has no categories section");
        }
        return section;
    }

    it("lists the categories that file the page", () => {
        held.page = filedPage();
        const section = summary();
        expect(within(section).getByText(copy.pages.summary.categories)).toBeDefined();
        expect(statesIn(section)).toStrictEqual(["onSite", "onPublish"]);
        expect(section.querySelector("[data-categories-need-plugin]")).toBeNull();
    });

    it("says what to do when the site's plugin cannot file pages", () => {
        held.page = filedPage({ categoriesNeedPlugin: true });
        const section = summary();
        expect(statesIn(section)).toStrictEqual(["needsPlugin", "needsPlugin"]);
        expect(within(section).getByText(copy.pages.summary.categoriesNeedPlugin)).toBeDefined();
    });

    it.each([
        ["a mapped page", "healing", copy.pages.summary.noCategoriesHint],
        ["an unmapped page", null, copy.pages.summary.noCategories],
    ])("says %s is filed under nothing", (_, entityId, words) => {
        held.page = filedPage({ entityId, categories: [] });
        const section = summary();
        expect(within(section).getByText(words)).toBeDefined();
        expect(within(section).queryByRole("list", { name: copy.categories.trail })).toBeNull();
    });
});

describe("the page drawer", () => {
    function drawer(): void {
        renderScreen(
            <PageDrawer
                pageId="p1"
                siteId="s1"
                index={index}
                search=""
                tab="details"
                onTabChange={() => {}}
                onClose={() => {}}
            />,
        );
    }

    it("shows what the page is filed under right beneath its title", () => {
        held.page = filedPage();
        drawer();
        const strip = document.querySelector<HTMLElement>("[data-page-categories]");
        if (strip === null) {
            throw new Error("the drawer has no categories strip");
        }
        expect(within(strip).getByText(copy.pages.detail.filedUnder)).toBeDefined();
        expect(statesIn(strip)).toStrictEqual(["onSite", "onPublish"]);
        expect(strip.getAttribute("title")).toBeNull();
    });

    it("warns beneath the trail when the plugin cannot file the page", () => {
        held.page = filedPage({ categoriesNeedPlugin: true });
        drawer();
        expect(screen.getByText(copy.pages.summary.categoriesNeedPlugin)).toBeDefined();
    });

    it("leaves the strip out for a page no category files", () => {
        held.page = filedPage({ categories: [] });
        drawer();
        expect(document.querySelector("[data-page-categories]")).toBeNull();
    });
});
