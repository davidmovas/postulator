import { describe, expect, it } from "vitest";

import type { ProposedEntity } from "../../../data/types.js";
import type { ParentChoice, ProposalEdit, ProposalEdits } from "./proposal-tree.js";
import {
    cascaded,
    changed,
    clashesOf,
    entityKey,
    everyTickable,
    keywordLinesOf,
    nameOf,
    orphaned,
    originalParent,
    parentChoices,
    proposalTree,
    pruned,
    sentProposals,
    subtree,
    tickState,
    visibleRows,
} from "./proposal-tree.js";

const site: ProposedEntity[] = [
    { pageId: "p0", path: "/shop/", name: "Shop", parent: "/", parentEntityId: "" },
    { pageId: "p1", path: "/shop/mugs/", name: "Mugs", parent: "/shop/" },
    { pageId: "p2", path: "/shop/mugs/blue/", name: "Blue Mug", parent: "/shop/mugs/" },
    { pageId: "p3", path: "/shop/mugs/red/", name: "Red Mug", parent: "/shop/mugs/" },
    { pageId: "p4", path: "/shop/cups/", name: "Mugs", parent: "/shop/", clash: "name_in_batch" },
    { pageId: "p5", path: "/journal/", name: "Journal", parent: "/blog/", parentEntityId: "e9" },
    { pageId: "p6", path: "/shop/plates/", name: "Plates", parent: "/shop/", clash: "name_carried" },
];

const words: ProposedEntity[] = [
    { name: "Trail shoes", keywords: [{ text: "trail shoes", volume: 900 }] },
    { name: "Waterproof trail shoes", keywords: [{ text: "waterproof trail shoes" }], parent: "Trail shoes" },
    { name: "Road shoes", keywords: [{ text: "road shoes" }], parent: "Running" },
];

const none: ProposalEdits = new Map();

function edits(entries: [number, ProposalEdit][]): ProposalEdits {
    return new Map(entries);
}

const named = (id: string): string | undefined => (id === "e9" ? "Blog" : undefined);

describe("where a proposal sits", () => {
    it.each<[number, ParentChoice]>([
        [0, { kind: "ref", ref: "/" }],
        [1, { kind: "batch", at: 0 }],
        [2, { kind: "batch", at: 1 }],
        [5, { kind: "entity", id: "e9" }],
    ])("reads the parent of site proposal %i from its path or its entity", (at, choice) => {
        expect(originalParent(site, at)).toStrictEqual(choice);
    });

    it.each<[number, ParentChoice]>([
        [0, { kind: "root" }],
        [1, { kind: "batch", at: 0 }],
        [2, { kind: "ref", ref: "Running" }],
    ])("reads the parent of keyword proposal %i by name when it has no page", (at, choice) => {
        expect(originalParent(words, at)).toStrictEqual(choice);
    });

    it("builds the tree by parent and keeps the batch's order", () => {
        const tree = proposalTree(site, none);
        expect(tree.roots).toStrictEqual([0, 5]);
        expect(tree.children.get(0)).toStrictEqual([1, 4, 6]);
        expect(tree.children.get(1)).toStrictEqual([2, 3]);
        expect(subtree(tree, 0)).toStrictEqual([0, 1, 2, 3, 4, 6]);
    });

    it("lifts a proposal that would sit under itself to the top", () => {
        const looped: ProposedEntity[] = [
            { path: "/a/", name: "A", parent: "/b/" },
            { path: "/b/", name: "B", parent: "/a/" },
        ];
        expect(proposalTree(looped, none).roots).toStrictEqual([0, 1]);
    });

    it("moves a proposal and its subtree when its parent is edited", () => {
        const tree = proposalTree(site, edits([[1, { parent: { kind: "batch", at: 5 } }]]));
        expect(tree.children.get(5)).toStrictEqual([1]);
        expect(subtree(tree, 5)).toStrictEqual([5, 1, 2, 3]);
    });

    it("offers as a parent every proposal outside the row's own subtree", () => {
        const tree = proposalTree(site, none);
        expect(parentChoices(tree, site.length, 1)).toStrictEqual([0, 4, 5, 6]);
        expect(parentChoices(tree, site.length, 0)).toStrictEqual([5]);
    });
});

describe("what blocks a proposal", () => {
    it("keeps the site's clashes, and an edit settles the ones it answers", () => {
        expect([...clashesOf(site, none)]).toStrictEqual([
            [4, "name_in_batch"],
            [6, "name_carried"],
        ]);
        expect([...clashesOf(site, edits([[4, { name: "Cups" }]]))]).toStrictEqual([[6, "name_carried"]]);
        expect([...clashesOf(site, edits([[6, { name: "Dishes" }]]))]).toStrictEqual([[4, "name_in_batch"]]);
    });

    it.each([
        { label: "an empty name", edit: [2, { name: "  " }] as [number, ProposalEdit], clash: [2, "no_name"] },
        {
            label: "a name taken under the same parent",
            edit: [3, { name: "blue mug" }] as [number, ProposalEdit],
            clash: [3, "name_in_batch"],
        },
    ])("refuses $label", ({ edit, clash }) => {
        expect(clashesOf(site, edits([edit]))).toContainEqual(clash);
    });

    it("lets the same name stand under two parents", () => {
        const clashes = clashesOf(site, edits([[3, { name: "Blue Mug", parent: { kind: "root" } }]]));
        expect(clashes.has(3)).toBe(false);
    });

    it("folds only ASCII letters when it compares names, as the store does", () => {
        expect(entityKey("  CAFÉ ")).toBe("cafÉ");
        expect(entityKey("Café")).not.toBe(entityKey("CAFÉ"));
    });

    it("drops a tick the clashes no longer allow", () => {
        const clashes = clashesOf(site, none);
        expect([...pruned(new Set([1, 4, 6]), clashes)]).toStrictEqual([1]);
        expect([...everyTickable(site.length, clashes)]).toStrictEqual([0, 1, 2, 3, 5]);
    });
});

describe("ticking a subtree", () => {
    const tree = proposalTree(site, none);
    const clashes = clashesOf(site, none);

    it("ticks and unticks a node with its whole subtree, never a blocked row", () => {
        const on = cascaded(tree, new Set(), clashes, 0);
        expect([...on].sort()).toStrictEqual([0, 1, 2, 3]);
        expect([...cascaded(tree, on, clashes, 1)].sort()).toStrictEqual([0]);
    });

    it.each([
        { kept: [0, 1, 2, 3], at: 0, state: "on" },
        { kept: [], at: 0, state: "off" },
        { kept: [2], at: 1, state: "some" },
        { kept: [0, 1, 2], at: 0, state: "some" },
        { kept: [0, 1, 2], at: 2, state: "on" },
    ])("shows $state for $at with $kept ticked", ({ kept, at, state }) => {
        expect(tickState(tree, new Set(kept), clashes, at)).toBe(state);
    });

    it("ticks the whole subtree when the node is partial", () => {
        expect([...cascaded(tree, new Set([2]), clashes, 1)].sort()).toStrictEqual([1, 2, 3]);
    });

    it("says a ticked row whose parent is not ticked goes to the top level", () => {
        expect(orphaned(tree, new Set([2]), 2)).toBe(true);
        expect(orphaned(tree, new Set([1, 2]), 2)).toBe(false);
        expect(orphaned(tree, new Set([0]), 0)).toBe(false);
    });
});

describe("what is sent to be written", () => {
    it("sends the ticked proposals as the model gave them when nothing changed", () => {
        expect(sentProposals(site, none, new Set([0, 1, 2]), { source: "site", entityName: named })).toStrictEqual([
            site[0],
            site[1],
            site[2],
        ]);
    });

    it("never sends a blocked row", () => {
        expect(sentProposals(site, none, new Set([4, 6]), { source: "site", entityName: named })).toStrictEqual([]);
    });

    it("sends a renamed site proposal under its new name with the stale verdicts dropped", () => {
        const withJoin: ProposedEntity[] = [{ ...site[0], existingEntityId: "e1" } as ProposedEntity];
        expect(sentProposals(withJoin, edits([[0, { name: " Store " }]]), new Set([0]), { source: "site", entityName: named })).toStrictEqual([
            { pageId: "p0", path: "/shop/", name: "Store", label: "Store", parent: "/", parentEntityId: "" },
        ]);
    });

    it("writes a ticked row whose parent is not sent at the top level", () => {
        expect(sentProposals(site, none, new Set([2]), { source: "site", entityName: named })).toStrictEqual([
            { ...site[2], parent: "", parentEntityId: "" },
        ]);
    });

    it.each<{ choice: ParentChoice; source: "site" | "model"; placed: Pick<ProposedEntity, "parent" | "parentEntityId"> }>([
        { choice: { kind: "root" }, source: "site", placed: { parent: "", parentEntityId: "" } },
        { choice: { kind: "entity", id: "e9" }, source: "site", placed: { parent: "", parentEntityId: "e9" } },
        { choice: { kind: "entity", id: "e9" }, source: "model", placed: { parent: "Blog", parentEntityId: "e9" } },
        { choice: { kind: "batch", at: 5 }, source: "site", placed: { parent: "/journal/", parentEntityId: "" } },
    ])("sends a parent moved to $choice.kind as apply resolves it from the $source", ({ choice, source, placed }) => {
        const sent = sentProposals(site, edits([[3, { parent: choice }]]), new Set([3, 5]), { source, entityName: named });
        expect(sent.find((held) => held.pageId === "p3")).toMatchObject({ name: "Red Mug", ...placed });
    });

    it("names a keyword parent by its edited name and keeps the volumes of kept keywords", () => {
        const changes = edits([
            [0, { name: "Trail running shoes", keywords: ["trail shoes", "trail runners"] }],
        ]);
        expect(sentProposals(words, changes, new Set([0, 1]), { source: "model", entityName: named })).toStrictEqual([
            {
                name: "Trail running shoes",
                label: "Trail running shoes",
                keywords: [{ text: "trail shoes", volume: 900 }, { text: "trail runners" }],
                parent: "",
                parentEntityId: "",
            },
            { ...words[1], parent: "Trail running shoes", parentEntityId: "" },
        ]);
    });

    it("knows an edit that puts back what the model said changes nothing", () => {
        const back = edits([[1, { name: "Mugs", parent: { kind: "batch", at: 0 } }]]);
        expect(changed(site, back, 1)).toBe(false);
        expect(nameOf(site, back, 1)).toBe("Mugs");
        expect(sentProposals(site, back, new Set([0, 1]), { source: "site", entityName: named })[1]).toBe(site[1]);
    });

    it("reads keyword lines once each, without blanks", () => {
        expect(keywordLinesOf(" a \n\nb\na\r\nc ")).toStrictEqual(["a", "b", "c"]);
    });
});

describe("the rows the tree shows", () => {
    const tree = proposalTree(site, none);
    const view = { query: "", filter: "all" as const, collapsed: new Set<number>() };

    it("shows every row in tree order with its depth", () => {
        expect(visibleRows(site, none, tree, new Set(), view).map((row) => [row.at, row.depth])).toStrictEqual([
            [0, 0],
            [1, 1],
            [2, 2],
            [3, 2],
            [4, 1],
            [6, 1],
            [5, 0],
        ]);
    });

    it("hides what a collapsed node holds", () => {
        const rows = visibleRows(site, none, tree, new Set(), { ...view, collapsed: new Set([1]) });
        expect(rows.map((row) => row.at)).toStrictEqual([0, 1, 4, 6, 5]);
        expect(rows.find((row) => row.at === 1)).toMatchObject({ branch: true, open: false });
    });

    it("keeps the ancestors of a match visible and opens them", () => {
        const rows = visibleRows(site, none, tree, new Set(), { ...view, query: "red", collapsed: new Set([0, 1]) });
        expect(rows.map((row) => row.at)).toStrictEqual([0, 1, 3]);
    });

    it("matches a path as well as a name, and an edited name", () => {
        expect(visibleRows(site, none, tree, new Set(), { ...view, query: "/journal" }).map((row) => row.at)).toStrictEqual([5]);
        const renamed = edits([[5, { name: "Field notes" }]]);
        expect(
            visibleRows(site, renamed, proposalTree(site, renamed), new Set(), { ...view, query: "field" }).map((row) => row.at),
        ).toStrictEqual([5]);
    });

    it.each([
        { filter: "ticked" as const, rows: [0, 1, 2] },
        { filter: "unticked" as const, rows: [0, 1, 3, 5] },
        { filter: "clash" as const, rows: [0, 4, 6] },
    ])("filters to the $filter rows with their ancestors", ({ filter, rows }) => {
        expect(visibleRows(site, none, tree, new Set([2]), { ...view, filter }).map((row) => row.at)).toStrictEqual(rows);
    });
});
