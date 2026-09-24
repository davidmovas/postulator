import { describe, expect, it } from "vitest";

import type { Page, PageTreeNode } from "../../data/types.js";
import { branchOf, indexTree, narrowTree, requiredParents, tick } from "./start-selection.js";

function aPage(id: string, path: string, overrides: Partial<Page> = {}): Page {
    return {
        id,
        siteId: "s1",
        path,
        slug: "",
        parentPageId: null,
        wpType: "page",
        wpId: null,
        title: path,
        h1: "",
        metaTitle: "",
        metaDescription: "",
        canonical: "",
        primaryKeyword: "",
        keywords: [],
        status: "planned",
        entityId: `entity-${id}`,
        templateId: null,
        contentHash: "",
        observed: { link: "", slug: "", status: "", title: "", h1: "" },
        mismatches: null,
        wpModifiedAt: null,
        lastSyncedAt: null,
        drift: false,
        createdAt: "2026-09-23T10:00:00Z",
        updatedAt: "2026-09-23T10:00:00Z",
        ...overrides,
    };
}

function node(page: Page, children: PageTreeNode[] = []): PageTreeNode {
    return { page, children };
}

const bikes = aPage("bikes", "/bikes/", { wpId: 10, status: "published" });
const cargo = aPage("cargo", "/bikes/cargo/");
const max = aPage("max", "/bikes/cargo/max/");
const mini = aPage("mini", "/bikes/cargo/mini/", { entityId: null });
const guides = aPage("guides", "/guides/", { title: "Riding guides" });

const roots: PageTreeNode[] = [node(bikes, [node(cargo, [node(max), node(mini)])]), node(guides)];
const index = indexTree(roots);

describe("requiredParents", () => {
    it("adds every parent between a chosen page and the first one on the site", () => {
        const required = requiredParents(new Set(["max"]), index);
        expect([...required.entries()]).toStrictEqual([["cargo", "/bikes/cargo/max/"]]);
    });

    it("adds nothing for a parent already chosen or already on the site", () => {
        expect(requiredParents(new Set(["max", "cargo"]), index).size).toBe(0);
        expect(requiredParents(new Set(["cargo"]), index).size).toBe(0);
    });
});

describe("branchOf", () => {
    it("takes a page and everything below it that can be written", () => {
        expect(branchOf("cargo", index)).toStrictEqual(["cargo", "max"]);
    });
});

describe("tick", () => {
    it("reads chosen, required, partly chosen and untouched", () => {
        const selected = new Set(["max"]);
        const required = requiredParents(selected, index);
        expect(tick("max", selected, required, index)).toBe("on");
        expect(tick("cargo", selected, required, index)).toBe("on");
        expect(tick("bikes", selected, required, index)).toBe("some");
        expect(tick("guides", selected, required, index)).toBe("off");
    });
});

describe("narrowTree", () => {
    it("keeps a match with the parents that lead to it", () => {
        const kept = narrowTree(roots, "max", "all");
        expect(kept.map((root) => root.page.id)).toStrictEqual(["bikes"]);
        expect(kept[0]?.children?.[0]?.children?.map((child) => child.page.id)).toStrictEqual(["max"]);
    });

    it("finds a page by its title as well as its path", () => {
        expect(narrowTree(roots, "riding", "all").map((root) => root.page.id)).toStrictEqual(["guides"]);
    });

    it("shows only the pages not on the site, or only those that are", () => {
        const missing = narrowTree(roots, "", "missing");
        expect(missing.map((root) => root.page.id)).toStrictEqual(["bikes", "guides"]);
        expect(missing[0]?.children?.map((child) => child.page.id)).toStrictEqual(["cargo"]);
        expect(narrowTree(roots, "", "present").map((root) => root.page.id)).toStrictEqual(["bikes"]);
        expect(narrowTree(roots, "", "present")[0]?.children).toStrictEqual([]);
    });

    it("answers the whole tree when nothing narrows it", () => {
        expect(narrowTree(roots, "  ", "all")).toBe(roots);
    });
});
