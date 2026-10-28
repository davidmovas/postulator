import { describe, expect, it } from "vitest";

import type { ProposedEntity } from "../../../data/types.js";
import type { ParentChoice, ProposalEdit, ProposalEdits } from "./proposal-tree.js";
import {
    changed,
    clashesOf,
    entityKey,
    everyTickable,
    heldOn,
    keywordLinesOf,
    matchedRows,
    nameOf,
    orphaned,
    originalParent,
    parentChoices,
    proposalTree,
    pruned,
    rangeTicked,
    sentProposals,
    sharedLabel,
    shownReach,
    sourceTypesOf,
    spanTicked,
    subtree,
    tickState,
    tickedAll,
    tickedByType,
    tickedTo,
    treeStep,
    turnsOn,
    typeCounts,
    visibleRows,
} from "./proposal-tree.js";
import type { TreeKey, TreeStep } from "./proposal-tree.js";

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
    const clicked = (kept: ReadonlySet<number>, at: number): ReadonlySet<number> =>
        tickedTo(tree, kept, clashes, at, turnsOn(tree, kept, clashes, at, true), true);

    it("ticks and unticks a node with its whole subtree, never a blocked row", () => {
        const on = clicked(new Set(), 0);
        expect([...on].sort()).toStrictEqual([0, 1, 2, 3]);
        expect([...clicked(on, 1)].sort()).toStrictEqual([0]);
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
        expect([...clicked(new Set([2]), 1)].sort()).toStrictEqual([1, 2, 3]);
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
    const nothing = new Set<number>();

    it("shows every row in tree order with its depth", () => {
        expect(visibleRows(tree, null, nothing).map((row) => [row.at, row.depth, row.matched])).toStrictEqual([
            [0, 0, true],
            [1, 1, true],
            [2, 2, true],
            [3, 2, true],
            [4, 1, true],
            [6, 1, true],
            [5, 0, true],
        ]);
    });

    it("hides what a collapsed node holds", () => {
        const rows = visibleRows(tree, null, new Set([1]));
        expect(rows.map((row) => row.at)).toStrictEqual([0, 1, 4, 6, 5]);
        expect(rows.find((row) => row.at === 1)).toMatchObject({ branch: true, open: false });
    });

    it("keeps the ancestors of a match visible, opens them and marks them as context", () => {
        const rows = visibleRows(tree, matchedRows(site, none, nothing, "red", "all"), new Set([0, 1]));
        expect(rows.map((row) => [row.at, row.matched, row.open])).toStrictEqual([
            [0, false, true],
            [1, false, true],
            [3, true, true],
        ]);
    });

    it("matches a path as well as a name, and an edited name", () => {
        expect([...(matchedRows(site, none, nothing, "/journal", "all") ?? [])]).toStrictEqual([5]);
        const renamed = edits([[5, { name: "Field notes" }]]);
        expect([...(matchedRows(site, renamed, nothing, "field", "all") ?? [])]).toStrictEqual([5]);
    });

    it("narrows nothing without a search or a filter", () => {
        expect(matchedRows(site, none, new Set([2]), "  ", "all")).toBeNull();
    });

    it.each([
        { filter: "ticked" as const, rows: [0, 1, 2] },
        { filter: "unticked" as const, rows: [0, 1, 3, 5] },
        { filter: "clash" as const, rows: [0, 4, 6] },
    ])("filters to the $filter rows with their ancestors", ({ filter, rows }) => {
        expect(visibleRows(tree, matchedRows(site, none, new Set([2]), "", filter), nothing).map((row) => row.at)).toStrictEqual(rows);
    });
});

describe("a shift-range over the shown rows", () => {
    const tree = proposalTree(site, none);
    const clashes = clashesOf(site, none);
    const open = visibleRows(tree, null, new Set());
    const folded = visibleRows(tree, null, new Set([1]));

    it.each([
        { label: "on when the clicked row is off, though the anchor is off", rows: open, kept: [0, 1, 2, 5], anchor: 3, target: 0, ticked: [0, 1, 2, 3, 5] },
        { label: "off when the clicked row is on, though the anchor is on", rows: open, kept: [0, 1, 2, 3, 5], anchor: 1, target: 5, ticked: [0] },
        { label: "on from an off anchor to an off row, past the blocked rows", rows: open, kept: [], anchor: 1, target: 5, ticked: [1, 2, 3, 5] },
        { label: "backwards, to the state the clicked row takes", rows: open, kept: [0, 1, 2, 3], anchor: 5, target: 2, ticked: [0, 1] },
        { label: "into what a folded branch holds", rows: folded, kept: [], anchor: 5, target: 1, ticked: [1, 2, 3, 5] },
        { label: "as a plain click without an anchor", rows: open, kept: [], anchor: null, target: 1, ticked: [1, 2, 3] },
        { label: "as a plain click when the anchor is hidden", rows: folded, kept: [], anchor: 2, target: 1, ticked: [1, 2, 3] },
    ])("ticks $label", ({ rows, kept, anchor, target, ticked }) => {
        expect([...rangeTicked(tree, new Set(kept), clashes, rows, anchor, target)].sort()).toStrictEqual(ticked);
    });
});

describe("ticking one row, a span and what is shown", () => {
    const tree = proposalTree(site, none);
    const clashes = clashesOf(site, none);
    const open = visibleRows(tree, null, new Set());
    const folded = visibleRows(tree, null, new Set([1]));

    it.each([
        { label: "ticks one row without its children", kept: [], at: 1, on: true, deep: false, ticked: [1] },
        { label: "unticks one row and keeps its children", kept: [1, 2, 3], at: 1, on: false, deep: false, ticked: [2, 3] },
        { label: "ticks a row with its whole subtree", kept: [], at: 1, on: true, deep: true, ticked: [1, 2, 3] },
        { label: "leaves a blocked row alone", kept: [], at: 4, on: true, deep: false, ticked: [] },
    ])("$label", ({ kept, at, on, deep, ticked }) => {
        expect([...tickedTo(tree, new Set(kept), clashes, at, on, deep)].sort()).toStrictEqual(ticked);
    });

    it.each([
        { label: "turns a partial node on with its subtree", kept: [2], at: 1, deep: true, on: true },
        { label: "turns a full node off with its subtree", kept: [1, 2, 3], at: 1, deep: true, on: false },
        { label: "turns a ticked row off alone though its children are not ticked", kept: [1], at: 1, deep: false, on: false },
        { label: "turns an unticked row on alone though its children are ticked", kept: [2, 3], at: 1, deep: false, on: true },
    ])("$label", ({ kept, at, deep, on }) => {
        expect(turnsOn(tree, new Set(kept), clashes, at, deep)).toBe(on);
    });

    it.each([
        { label: "paints a span on with each row's subtree", kept: [1], rows: open, from: 1, to: 5, on: true, ticked: [1, 2, 3, 5] },
        { label: "paints a span off with each row's subtree", kept: [0, 1, 2, 3, 5], rows: open, from: 5, to: 1, on: false, ticked: [0] },
        { label: "paints the last row alone when the first is not shown", kept: [], rows: folded, from: 2, to: 5, on: true, ticked: [5] },
    ])("$label", ({ kept, rows, from, to, on, ticked }) => {
        expect([...spanTicked(tree, new Set(kept), clashes, rows, from, to, on)].sort()).toStrictEqual(ticked);
    });

    it.each([
        { label: "a row's own tick, though its children are not ticked", kept: [1], at: 1, on: true },
        { label: "a row's own tick, though its children are ticked", kept: [2, 3], at: 1, on: false },
        { label: "a blocked row as off", kept: [0, 1, 2, 3, 5], at: 4, on: false },
    ])("holds $label", ({ kept, at, on }) => {
        expect(heldOn(tree, new Set(kept), clashes, at)).toBe(on);
    });

    it.each([
        { label: "every shown row and what a fold hides under it", rows: folded, reach: [0, 1, 2, 3, 5] },
        { label: "only the rows a search matched, never their ancestors", rows: visibleRows(tree, matchedRows(site, none, new Set(), "red", "all"), new Set()), reach: [3] },
        { label: "no blocked row", rows: visibleRows(tree, matchedRows(site, none, new Set(), "", "clash"), new Set()), reach: [] },
    ])("reaches $label", ({ rows, reach }) => {
        expect([...shownReach(tree, clashes, rows)].sort()).toStrictEqual(reach);
    });

    it("ticks and unticks the rows it is handed and keeps the rest", () => {
        expect([...tickedAll(new Set([0]), [3, 5], true)].sort()).toStrictEqual([0, 3, 5]);
        expect([...tickedAll(new Set([0, 3, 5]), [3, 5], false)]).toStrictEqual([0]);
    });
});

describe("ticking by the source page's type", () => {
    const clashes = clashesOf(site, none);
    const types = sourceTypesOf({ p0: "hub", p1: "cat", p2: "prod", p3: "prod", p4: "cat", p5: "", p6: undefined });

    it("keeps only the pages a type was named for", () => {
        expect([...types]).toStrictEqual([
            ["p0", "hub"],
            ["p1", "cat"],
            ["p2", "prod"],
            ["p3", "prod"],
            ["p4", "cat"],
        ]);
        expect(sourceTypesOf(null).size).toBe(0);
    });

    it("counts the proposals of each type", () => {
        expect([...typeCounts(site, types)]).toStrictEqual([
            ["hub", 1],
            ["cat", 2],
            ["prod", 2],
        ]);
        expect(typeCounts(words, types).size).toBe(0);
    });

    it.each([
        { label: "ticks every open proposal of a type, never a blocked one", kept: [], typeId: "cat", on: true, ticked: [1] },
        { label: "unticks a type and keeps the rest", kept: [0, 1, 2, 3], typeId: "prod", on: false, ticked: [0, 1] },
        { label: "changes nothing for a type no proposal has", kept: [0], typeId: "none", on: true, ticked: [0] },
    ])("$label", ({ kept, typeId, on, ticked }) => {
        expect([...tickedByType(site, types, new Set(kept), clashes, typeId, on)].sort()).toStrictEqual(ticked);
    });
});

describe("a name shared with another entity", () => {
    const shared: ProposedEntity[] = [
        { pageId: "p0", path: "/shop/", name: "Shop", label: "Shop" },
        { pageId: "p1", path: "/shop/mugs/", name: "Mugs", label: "Shop Mugs", parent: "/shop/" },
        { pageId: "p2", path: "/shop/cups/", name: "Mugs", label: "Other Mugs", parent: "/shop/", clash: "name_in_batch" },
        { pageId: "p3", path: "/cups/", name: "Cups" },
    ];

    it.each<{ label: string; at: number; changes: ProposalEdits; shown: string | null }>([
        { label: "says the label it is shown by", at: 1, changes: none, shown: "Shop Mugs" },
        { label: "says nothing when the label is the name", at: 0, changes: none, shown: null },
        { label: "says nothing without a label", at: 3, changes: none, shown: null },
        { label: "leaves a blocked row to its clash", at: 2, changes: none, shown: null },
        { label: "says nothing once renamed", at: 1, changes: edits([[1, { name: "Mug range" }]]), shown: null },
        { label: "says nothing once moved", at: 1, changes: edits([[1, { parent: { kind: "root" } }]]), shown: null },
        { label: "keeps saying it when only the keywords change", at: 1, changes: edits([[1, { keywords: ["mugs"] }]]), shown: "Shop Mugs" },
    ])("$label", ({ at, changes, shown }) => {
        expect(sharedLabel(shared, changes, clashesOf(shared, changes), at)).toBe(shown);
    });
});

describe("moving through the tree by keyboard", () => {
    const tree = proposalTree(site, none);
    const open = visibleRows(tree, null, new Set());
    const folded = visibleRows(tree, null, new Set([1]));
    const searched = visibleRows(tree, matchedRows(site, none, new Set(), "red", "all"), new Set([0, 1]));

    it.each<{ label: string; rows: typeof open; collapsed: number[]; focus: number | null; key: TreeKey; step: TreeStep | null }>([
        { label: "down from nothing to the first row", rows: open, collapsed: [], focus: null, key: "down", step: { kind: "focus", at: 0 } },
        { label: "up from nothing to the first row", rows: open, collapsed: [], focus: null, key: "up", step: { kind: "focus", at: 0 } },
        { label: "left from nothing nowhere", rows: open, collapsed: [], focus: null, key: "left", step: null },
        { label: "down to the next shown row", rows: open, collapsed: [], focus: 0, key: "down", step: { kind: "focus", at: 1 } },
        { label: "down from the last row nowhere", rows: open, collapsed: [], focus: 5, key: "down", step: null },
        { label: "up to the row above", rows: open, collapsed: [], focus: 1, key: "up", step: { kind: "focus", at: 0 } },
        { label: "up from the first row nowhere", rows: open, collapsed: [], focus: 0, key: "up", step: null },
        { label: "home to the first row", rows: open, collapsed: [], focus: 3, key: "home", step: { kind: "focus", at: 0 } },
        { label: "end to the last row", rows: open, collapsed: [], focus: 0, key: "end", step: { kind: "focus", at: 5 } },
        { label: "left on an open branch folds it", rows: open, collapsed: [], focus: 1, key: "left", step: { kind: "collapse", at: 1 } },
        { label: "left on a leaf to its parent", rows: open, collapsed: [], focus: 2, key: "left", step: { kind: "focus", at: 1 } },
        { label: "left on a top leaf nowhere", rows: open, collapsed: [], focus: 5, key: "left", step: null },
        { label: "right on a folded branch opens it", rows: folded, collapsed: [1], focus: 1, key: "right", step: { kind: "expand", at: 1 } },
        { label: "right on an open branch to its first child", rows: open, collapsed: [], focus: 0, key: "right", step: { kind: "focus", at: 1 } },
        { label: "right on a leaf nowhere", rows: open, collapsed: [], focus: 2, key: "right", step: null },
        { label: "left on a branch a search holds open to its parent", rows: searched, collapsed: [0, 1], focus: 1, key: "left", step: { kind: "focus", at: 0 } },
    ])("goes $label", ({ rows, collapsed, focus, key, step }) => {
        expect(treeStep(rows, tree, new Set(collapsed), focus, key)).toStrictEqual(step);
    });

    it("goes nowhere in an empty tree", () => {
        expect(treeStep([], tree, new Set(), null, "down")).toBeNull();
    });
});
