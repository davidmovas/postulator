import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import type { Page } from "../../../data/types.js";
import { aPage } from "../../../testing/pages.js";
import type { Selection } from "../../../ui/selection.js";
import { clearSubtree, noSelection, selectAll, selectGroup, selectSubtree, toggle } from "../../../ui/selection.js";
import { aNode } from "../../map/model/fixture.js";
import type { Branches } from "../../map/model/tree.js";
import type { KnownRow, Naming } from "./model.js";
import {
    childrenFrom,
    filterWords,
    groupState,
    headState,
    listLayout,
    nodeRow,
    nodeSize,
    pageRow,
    prefillOf,
    selectedPaths,
    selectionWords,
    typeReasonLine,
    withoutType,
} from "./model.js";

const said = copy.pages;

const rules = new Map([["r7", { number: 7, name: "A product listing below the top level" }]]);

const ruleOf = (ruleId: string): { number: number; name: string } | undefined => rules.get(ruleId);

const naming: Naming = {
    typeName: (typeId) => (typeId === "t-guide" ? "Guide" : "Category"),
    pagePath: (pageId) => `/${pageId}/`,
    ruleName: (ruleId) => (ruleId === "r7" ? "Rule 7" : "a rule"),
    entityName: () => "Mugs",
};

const typeIdOf = (name: string): string | null => (name === "Guide" ? "t-guide" : name === "Hub" ? "t-hub" : null);

function must(state: Selection | null): Selection {
    if (state === null) {
        throw new Error("refused");
    }
    return state;
}

describe("why a page has its type", () => {
    it.each<{ label: string; page: Partial<Page>; line: string }>([
        { label: "a rule", page: { typeSource: "auto", typeReason: "rule", typeRuleId: "r7" }, line: "Rule 7: A product listing below the top level" },
        { label: "a rule since removed", page: { typeSource: "auto", typeReason: "rule", typeRuleId: "r-gone" }, line: said.typeReason.ruleGone },
        { label: "no rule", page: { typeSource: "auto", typeReason: "unmatched" }, line: said.typeReason.unmatched },
        { label: "a person", page: { typeSource: "manual", typeReason: "user" }, line: said.typeReason.user },
        { label: "the agent", page: { typeSource: "manual", typeReason: "ai" }, line: said.typeReason.ai },
        { label: "an import", page: { typeSource: "manual", typeReason: "import" }, line: said.typeReason.import },
        { label: "nothing yet", page: {}, line: said.typeReason.none },
        { label: "an older reason", page: { typeSource: "auto", typeReason: "listing" }, line: copy.pageTypes.reasons.listing },
    ])("says $label", ({ page, line }) => {
        expect(typeReasonLine(aPage("p1", "/x/", page), ruleOf)).toBe(line);
    });
});

describe("the facts a selection keeps of a row", () => {
    it("keeps a list row's type and rule, with no place in the tree", () => {
        expect(pageRow(aPage("p1", "/x/", { pageTypeId: "t-guide", typeRuleId: "r7" }))).toStrictEqual({ typeId: "t-guide", ruleId: "r7" });
        expect(pageRow(aPage("p1", "/x/"))).toStrictEqual({ typeId: null, ruleId: null });
    });

    it("keeps a tree node's place and finds its type by its name", () => {
        const guide = aNode("g", "/g/", { page: { ...aNode("g", "/g/").page, type: "Guide" }, descendants: 4 });
        expect(nodeRow(guide, ["hub"], typeIdOf)).toStrictEqual({ ancestors: ["hub"], typeId: "t-guide" });
        expect(nodeRow(aNode("u", "/u/", { page: { ...aNode("u", "/u/").page, type: "" } }), [], typeIdOf).typeId).toBeNull();
        expect(nodeSize(guide)).toBe(5);
    });

    it("hands a loaded branch's children over, and nothing for one still loading or paged", () => {
        const branches: Branches = new Map([
            ["hub", { nodes: [aNode("a", "/hub/a/", { descendants: 2 }), aNode("b", "/hub/b/")], hasMore: false, pending: false }],
            ["paged", { nodes: [aNode("c", "/c/")], hasMore: true, pending: false }],
        ]);
        const chains = new Map([["hub", ["root"]]]);
        const children = childrenFrom(branches, (pageId) => chains.get(pageId) ?? (pageId === "paged" ? [] : undefined), typeIdOf, undefined);
        expect(children("hub")).toStrictEqual([
            { id: "a", row: { ancestors: ["hub", "root"], typeId: "t-hub" }, size: 3 },
            { id: "b", row: { ancestors: ["hub", "root"], typeId: "t-hub" }, size: 1 },
        ]);
        expect(children("paged")).toBeNull();
        expect(children("nowhere")).toBeNull();
        expect(childrenFrom(branches, (pageId) => chains.get(pageId), typeIdOf, "t-guide")("hub")?.[0]?.size).toBeNull();
    });
});

describe("a selection in words", () => {
    it.each([
        { label: "every page", filter: {}, words: said.describe.everyPage },
        { label: "a type and a status", filter: { pageTypeId: "t-guide", status: "published" }, words: "every page that is a Guide page, is published" },
        {
            label: "text, findings, a branch, a prefix and a search",
            filter: { body: "without", findings: true, under: "shop", pathPrefix: "/shop/", search: "mug" },
            words: "every page that has no text of its own, has open findings, sits under /shop/, has a path starting with /shop/, mentions “mug”",
        },
        { label: "an entity and no entity", filter: { entityId: "e1", includeDescendants: true, unmapped: true }, words: "every page that belongs to Mugs or an entity under it, has no entity" },
    ])("names a filter of $label", ({ filter, words }) => {
        expect(filterWords(filter, naming)).toBe(words);
    });

    it.each<{ label: string; state: () => Selection; words: string }>([
        { label: "a group", state: () => selectGroup(noSelection, "t-guide", 4), words: "every Guide page" },
        { label: "a rule", state: () => ({ ...noSelection, whole: { kind: "rule" as const, ruleId: "r7", seen: 3 } }), words: "every page Rule 7 types" },
        {
            label: "a branch with a page taken out and one beside it",
            state: () => {
                const row = { ancestors: ["shop"] };
                const branch = must(selectSubtree(noSelection, { pageId: "shop", size: 9 }, { ancestors: [] }));
                return must(toggle(must(toggle(branch, "mugs", row)), "about", {}));
            },
            words: "/shop/ and everything under it; 1 page picked by hand, without 1 page taken back out",
        },
        {
            label: "a type's branch with a branch out",
            state: () =>
                clearSubtree(must(selectSubtree(noSelection, { pageId: "shop", typeId: "t-guide" }, { ancestors: [] })), { pageId: "mugs", typeId: "t-guide" }, {
                    ancestors: ["shop"],
                }),
            words: "the Guide pages under /shop/, without what sits under /mugs/",
        },
    ])("names $label", ({ state, words }) => {
        expect(selectionWords(state(), naming)).toBe(words);
    });

    it("hands the agent the count, the words and the paths", () => {
        expect(prefillOf(214, "every Guide page", ["/a/", "/b/"])).toBe(said.bulk.prefill(214, "every Guide page", ["/a/", "/b/"]));
        expect(prefillOf(null, "x", [])).toContain("These selected pages");
    });

    it("lists the paths of the selected rows it has seen, up to a limit", () => {
        const known = new Map<string, KnownRow>([
            ["a", { path: "/a/", row: {} }],
            ["b", { path: "/b/", row: {} }],
            ["c", { path: "/c/", row: {} }],
        ]);
        const state = must(toggle(must(toggle(noSelection, "a", {})), "c", {}));
        expect(selectedPaths(state, known, 20)).toStrictEqual(["/a/", "/c/"]);
        expect(selectedPaths(selectAll(noSelection, {}, 3), known, 2)).toStrictEqual(["/a/", "/b/"]);
    });
});

describe("the boxes of a whole list and a group", () => {
    it.each<{ label: string; state: () => Selection; shown: string }>([
        { label: "nothing", state: () => noSelection, shown: "off" },
        { label: "the list's filter", state: () => selectAll(noSelection, { status: "published" }, 40), shown: "on" },
        { label: "the list's filter less a row", state: () => must(toggle(selectAll(noSelection, { status: "published" }, 40), "p1", {})), shown: "some" },
        { label: "another filter", state: () => selectAll(noSelection, { status: "planned" }, 4), shown: "some" },
        { label: "a picked row", state: () => must(toggle(noSelection, "p1", {})), shown: "some" },
    ])("shows the list's head box for $label as $shown", ({ state, shown }) => {
        expect(headState(state(), { status: "published" })).toBe(shown);
    });

    it.each<{ label: string; state: () => Selection; shown: string }>([
        { label: "the group", state: () => selectGroup(noSelection, "t-guide", 4), shown: "on" },
        { label: "another group", state: () => selectGroup(noSelection, "t-cat", 4), shown: "off" },
        { label: "a row of the type", state: () => must(toggle(noSelection, "p1", { typeId: "t-guide" })), shown: "some" },
        { label: "a branch of the type", state: () => must(selectSubtree(noSelection, { pageId: "shop", typeId: "t-guide" }, {})), shown: "some" },
    ])("shows a group's box for $label as $shown", ({ state, shown }) => {
        expect(groupState(state(), "t-guide")).toBe(shown);
    });

    it("takes a type out of a selection and keeps the rest", () => {
        const mixed = must(toggle(must(toggle(selectGroup(noSelection, "t-guide", 4), "c1", { typeId: "t-cat" })), "g1", { typeId: "t-guide" }));
        const out = withoutType(mixed, "t-guide");
        expect(out.whole).toBeNull();
        expect([...out.ids.keys()]).toStrictEqual(["c1"]);
        expect(groupState(out, "t-guide")).toBe("off");
    });
});

describe("the columns the page list shows at a width", () => {
    it.each([
        { width: 0, columns: ["box", "path", "title", "type", "status", "entity", "created", "drift", "synced"] },
        { width: 1100, columns: ["box", "path", "title", "type", "status", "entity", "created", "drift", "synced"] },
        { width: 844, columns: ["box", "path", "title", "type", "status", "entity", "created", "drift", "synced"] },
        { width: 843, columns: ["box", "path", "title", "type", "status", "entity", "created", "drift"] },
        { width: 640, columns: ["box", "path", "title", "type", "status", "drift"] },
        { width: 420, columns: ["box", "path", "type", "status"] },
        { width: 260, columns: ["box", "path", "type"] },
    ])("shows $columns at $width px", ({ width, columns }) => {
        expect(listLayout(width).columns).toStrictEqual(columns);
    });
});
