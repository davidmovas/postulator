import { describe, expect, it } from "vitest";

import type { PageSelection } from "../data/types.js";
import type { ChildRow, ChildrenOf, Selection, SelectionRow, TickState } from "./selection.js";
import {
    between,
    clear,
    clearSubtree,
    count,
    coveredAbove,
    dropWhole,
    exact,
    isEmpty,
    isSelected,
    noSelection,
    range,
    selectAll,
    selectGroup,
    selectRule,
    selectSubtree,
    stateOf,
    toRequest,
    toggle,
} from "./selection.js";

const site: Readonly<Record<string, SelectionRow>> = {
    a: { ancestors: [], typeId: "hub", ruleId: "r-hub" },
    a1: { ancestors: ["a"], typeId: "guide", ruleId: "r-guide" },
    a1x: { ancestors: ["a1", "a"], typeId: "guide", ruleId: "r-guide" },
    a1y: { ancestors: ["a1", "a"], typeId: "guide", ruleId: "r-guide" },
    a2: { ancestors: ["a"], typeId: "info", ruleId: null },
    b: { ancestors: [], typeId: "hub", ruleId: "r-hub" },
    b1: { ancestors: ["b"], typeId: "guide", ruleId: "r-guide" },
};

const rows: readonly string[] = ["a", "a1", "a1x", "a1y", "a2", "b", "b1"];

const sizes: Readonly<Record<string, number>> = { a: 5, a1: 3, a1x: 1, a1y: 1, a2: 1, b: 2, b1: 1 };

const tree: Readonly<Record<string, readonly string[]>> = { a: ["a1", "a2"], a1: ["a1x", "a1y"], a1x: [], a1y: [], a2: [], b: ["b1"], b1: [] };

const rowOf = (id: string): SelectionRow | undefined => site[id];

function row(id: string): SelectionRow {
    const held = site[id];
    if (held === undefined) {
        throw new Error(`no row ${id}`);
    }
    return held;
}

const childrenOf: ChildrenOf = (pageId) =>
    tree[pageId]?.map((id): ChildRow => ({ id, row: row(id), size: sizes[id] ?? null })) ?? null;

function must(state: Selection | null): Selection {
    if (state === null) {
        throw new Error("refused");
    }
    return state;
}

function chosen(state: Selection): string[] {
    return rows.filter((id) => isSelected(state, id, row(id)));
}

function tick(state: Selection, ...ids: string[]): Selection {
    return ids.reduce((held, id) => must(toggle(held, id, row(id), childrenOf)), state);
}

function branch(state: Selection, id: string, typeId?: string): Selection {
    const size = typeId === undefined ? sizes[id] : null;
    return must(selectSubtree(state, typeId === undefined ? { pageId: id, size } : { pageId: id, typeId, size }, row(id), childrenOf));
}

function unbranch(state: Selection, id: string, typeId?: string): Selection {
    const size = typeId === undefined ? sizes[id] : null;
    return clearSubtree(state, typeId === undefined ? { pageId: id, size } : { pageId: id, typeId, size }, row(id));
}

describe("what a selection holds", () => {
    it.each<{ label: string; state: () => Selection; selected: string[] }>([
        { label: "nothing", state: () => noSelection, selected: [] },
        { label: "two picked rows", state: () => tick(noSelection, "a1", "b"), selected: ["a1", "b"] },
        { label: "a page with everything under it", state: () => branch(noSelection, "a"), selected: ["a", "a1", "a1x", "a1y", "a2"] },
        { label: "a branch's pages of one type", state: () => branch(noSelection, "a", "guide"), selected: ["a1", "a1x", "a1y"] },
        { label: "a group of one type", state: () => selectGroup(noSelection, "guide", 4), selected: ["a1", "a1x", "a1y", "b1"] },
        { label: "every row the filter shows", state: () => selectAll(noSelection, { status: "published" }, 40), selected: [...rows] },
        { label: "a rule's matches", state: () => selectRule(noSelection, "r-guide", 4), selected: ["a1", "a1x", "a1y", "b1"] },
    ])("holds $label", ({ state, selected }) => {
        expect(chosen(state())).toStrictEqual(selected);
    });

    it("counts a group's row whose type it was not told as shown, and an untyped page as outside it", () => {
        const group = selectGroup(noSelection, "guide", 4);
        expect(isSelected(group, "z", {})).toBe(true);
        expect(isSelected(group, "z", { typeId: null })).toBe(false);
    });

    it("never holds a rule's row that does not say which rule placed it", () => {
        expect(isSelected(selectRule(noSelection, "r-guide", 4), "z", {})).toBe(false);
    });

    it.each<{ label: string; state: () => Selection; empty: boolean }>([
        { label: "nothing", state: () => noSelection, empty: true },
        { label: "a row picked and dropped", state: () => tick(noSelection, "a", "a"), empty: true },
        { label: "a picked row", state: () => tick(noSelection, "a"), empty: false },
        { label: "a subtree", state: () => branch(noSelection, "a", "guide"), empty: false },
        { label: "a filter", state: () => selectAll(noSelection, {}, 0), empty: false },
        { label: "a cleared one", state: () => clear(), empty: true },
    ])("is empty for $label: $empty", ({ state, empty }) => {
        expect(isEmpty(state())).toBe(empty);
    });
});

describe("taking rows back out", () => {
    it.each<{ label: string; state: () => Selection; inside: string; before: number }>([
        { label: "a subtree", state: () => branch(noSelection, "a"), inside: "a1", before: 5 },
        { label: "a group", state: () => selectGroup(noSelection, "guide", 4), inside: "b1", before: 4 },
        { label: "every row the filter shows", state: () => selectAll(noSelection, { search: "mug" }, 214), inside: "a2", before: 214 },
        { label: "a rule's matches", state: () => selectRule(noSelection, "r-guide", 4), inside: "a1x", before: 4 },
    ])("takes a row out of $label as an exception and puts it back", ({ state, inside, before }) => {
        const out = tick(state(), inside);
        expect(isSelected(out, inside, row(inside))).toBe(false);
        expect([...out.except.keys()]).toStrictEqual([inside]);
        expect(count(out)).toBe(before - 1);

        const back = tick(out, inside);
        expect(isSelected(back, inside, row(inside))).toBe(true);
        expect(back.except.size).toBe(0);
        expect(back.ids.size).toBe(0);
        expect(count(back)).toBe(before);
    });

    it("adds a row outside the group as a picked row beside it", () => {
        const state = tick(selectGroup(noSelection, "guide", 4), "a2");
        expect(chosen(state)).toStrictEqual(["a1", "a1x", "a1y", "a2", "b1"]);
        expect(count(state)).toBe(5);
    });

    it("lets a filter go and keeps the rows picked beside it", () => {
        const state = dropWhole(tick(tick(selectGroup(noSelection, "guide", 4), "a2"), "b1"));
        expect(chosen(state)).toStrictEqual(["a2"]);
        expect(state.except.size).toBe(0);
    });
});

describe("a branch taken out of a ticked branch", () => {
    it("takes the whole branch out, however much of it was loaded", () => {
        const state = unbranch(branch(noSelection, "a"), "a1");
        expect(chosen(state)).toStrictEqual(["a", "a2"]);
        expect(state.outs.map((out) => out.pageId)).toStrictEqual(["a1"]);
        expect(count(state)).toBe(2);
        expect(exact(state)).toBe(true);
        expect(toRequest("s1", state)).toStrictEqual({ siteId: "s1", subtrees: [{ pageId: "a" }], exceptSubtrees: [{ pageId: "a1" }] });
    });

    it("drops the picks, rows and exceptions beneath a branch it takes out", () => {
        const state = unbranch(tick(branch(noSelection, "a"), "a1x"), "a1");
        expect(state.except.size).toBe(0);
        expect(state.outs).toHaveLength(1);
        expect(count(state)).toBe(2);
    });

    it("clears a branch that was its own pick without taking anything out", () => {
        const state = unbranch(branch(tick(noSelection, "b1"), "a"), "a");
        expect(state.subtrees).toHaveLength(0);
        expect(state.outs).toHaveLength(0);
        expect(chosen(state)).toStrictEqual(["b1"]);
    });

    it("puts the whole branch back when it is ticked again", () => {
        const state = branch(unbranch(branch(noSelection, "a"), "a1"), "a1");
        expect(state.outs).toHaveLength(0);
        expect(state.subtrees.map((pick) => pick.pageId)).toStrictEqual(["a"]);
        expect(chosen(state)).toStrictEqual(["a", "a1", "a1x", "a1y", "a2"]);
    });

    it("splits the branch along the path when a page inside it is ticked", () => {
        const state = branch(unbranch(branch(noSelection, "a"), "a1"), "a1x");
        expect(chosen(state)).toStrictEqual(["a", "a1x", "a2"]);
        expect(toRequest("s1", state)).toStrictEqual({
            siteId: "s1",
            subtrees: [{ pageId: "a" }],
            except: ["a1"],
            exceptSubtrees: [{ pageId: "a1y" }],
        });
        expect(count(state)).toBe(3);
        expect(exact(state)).toBe(true);
    });

    it("splits the branch down to a single page ticked alone, and to the branch's own page", () => {
        const removed = unbranch(branch(noSelection, "a"), "a1");
        expect(chosen(tick(removed, "a1y"))).toStrictEqual(["a", "a1y", "a2"]);
        const own = tick(removed, "a1");
        expect(chosen(own)).toStrictEqual(["a", "a1", "a2"]);
        expect(own.outs.map((out) => out.pageId)).toStrictEqual(["a1x", "a1y"]);
    });

    it("refuses to split a branch whose children were not all loaded", () => {
        const removed = unbranch(branch(noSelection, "a"), "a1");
        expect(selectSubtree(removed, { pageId: "a1x", size: 1 }, row("a1x"))).toBeNull();
        expect(toggle(removed, "a1x", row("a1x"))).toBeNull();
    });

    it("refuses to put one type back inside a branch taken out whole", () => {
        const removed = unbranch(branch(noSelection, "a"), "a1");
        expect(selectSubtree(removed, { pageId: "a1x", typeId: "guide" }, row("a1x"), childrenOf)).toBeNull();
    });

    it("takes one type's branch out of a type's pick", () => {
        const state = unbranch(branch(noSelection, "a", "guide"), "a1", "guide");
        expect(chosen(state)).toStrictEqual([]);
        expect(toRequest("s1", state)).toStrictEqual({
            siteId: "s1",
            subtrees: [{ pageId: "a", typeId: "guide" }],
            exceptSubtrees: [{ pageId: "a1", typeId: "guide" }],
        });
        expect(exact(state)).toBe(false);
    });

    it("says when a row is held by a pick above it rather than its own", () => {
        const state = branch(noSelection, "a");
        expect(coveredAbove(state, "a1", row("a1"))).toBe(true);
        expect(coveredAbove(state, "a", row("a"))).toBe(false);
        expect(coveredAbove(selectAll(noSelection, {}, 6), "a", row("a"))).toBe(true);
    });
});

describe("a subtree beside what was already picked", () => {
    it("folds the picked rows and the exceptions beneath it into the subtree", () => {
        const state = branch(tick(noSelection, "a1", "b1"), "a");
        expect([...state.ids.keys()]).toStrictEqual(["b1"]);
        expect(state.subtrees.map((pick) => pick.pageId)).toStrictEqual(["a"]);
        expect(count(state)).toBe(6);

        const again = branch(tick(state, "a1x"), "a1");
        expect(again.except.size).toBe(0);
        expect(again.subtrees.map((pick) => pick.pageId)).toStrictEqual(["a"]);
        expect(count(again)).toBe(6);
    });

    it("folds a smaller subtree into the larger one that holds it", () => {
        const state = branch(branch(noSelection, "a1"), "a");
        expect(state.subtrees.map((pick) => pick.pageId)).toStrictEqual(["a"]);
        expect(count(state)).toBe(5);
    });

    it("keeps a whole subtree beside a type's pick inside it, and says the count needs asking", () => {
        const state = branch(branch(noSelection, "a", "guide"), "a1");
        expect(state.subtrees.map((pick) => [pick.pageId, pick.typeId])).toStrictEqual([
            ["a", "guide"],
            ["a1", undefined],
        ]);
        expect(exact(state)).toBe(false);
    });

    it("selects a subtree once however often it is asked", () => {
        expect(branch(branch(noSelection, "a"), "a").subtrees).toHaveLength(1);
    });

    it("clears only one type's part of a branch", () => {
        const state = tick(noSelection, "a1", "a2");
        expect([...unbranch(state, "a", "guide").ids.keys()]).toStrictEqual(["a2"]);
    });
});

describe("how a row's box shows", () => {
    const untyped = (id: string): TickState => stateOf(work, id, row(id), { leaf: (tree[id] ?? []).length === 0 });
    const work = tick(unbranch(branch(tick(noSelection, "b1"), "a"), "a1"), "a2");

    it.each<{ id: string; shown: TickState }>([
        { id: "a", shown: "some" },
        { id: "a1", shown: "off" },
        { id: "a1x", shown: "off" },
        { id: "a2", shown: "off" },
        { id: "b", shown: "some" },
        { id: "b1", shown: "on" },
    ])("shows $id as $shown", ({ id, shown }) => {
        expect(untyped(id)).toBe(shown);
    });

    it.each<{ label: string; state: () => Selection; id: string; typeId?: string; leaf: boolean; shown: TickState }>([
        { label: "a branch picked whole", state: () => branch(noSelection, "a1"), id: "a1", leaf: false, shown: "on" },
        { label: "a branch ticked by itself", state: () => tick(noSelection, "a1"), id: "a1", leaf: false, shown: "some" },
        { label: "a type's pick in that type's tree", state: () => branch(noSelection, "a", "guide"), id: "a", typeId: "guide", leaf: false, shown: "on" },
        { label: "a group in its own tree", state: () => selectGroup(noSelection, "guide", 4), id: "b", typeId: "guide", leaf: false, shown: "on" },
        { label: "a group in another type's tree", state: () => selectGroup(noSelection, "guide", 4), id: "b", typeId: "info", leaf: false, shown: "off" },
    ])("shows $label as $shown", ({ state, id, typeId, leaf, shown }) => {
        const scope = typeId === undefined ? { leaf } : { typeId, leaf };
        expect(stateOf(state(), id, row(id), scope)).toBe(shown);
    });
});

describe("selecting everything a filter matches", () => {
    it("replaces the picked rows it covers and the earlier filter's exceptions", () => {
        const before = tick(selectAll(tick(noSelection, "a1"), { status: "planned" }, 9), "b");
        const state = selectAll(before, { status: "published" }, 40);
        expect(state.ids.size).toBe(0);
        expect(state.except.size).toBe(0);
        expect(count(state)).toBe(40);
    });

    it("keeps a group's picked rows of another type", () => {
        const state = selectGroup(tick(noSelection, "a1", "a2"), "guide", 4);
        expect([...state.ids.keys()]).toStrictEqual(["a2"]);
        expect(count(state)).toBe(5);
    });

    it("lets a rule's matches replace a filter, as they share the count the person saw", () => {
        const state = selectRule(selectAll(noSelection, {}, 6), "r-hub", 2);
        expect(state.whole).toStrictEqual({ kind: "rule", ruleId: "r-hub", seen: 2 });
        expect(chosen(state)).toStrictEqual(["a", "b"]);
    });
});

describe("a shift-range over the shown rows", () => {
    it("picks every row from the anchor to the target, forwards and backwards, keeping the anchor", () => {
        const forwards = range(tick(noSelection, "a1"), rows, "b", rowOf);
        expect(chosen(forwards)).toStrictEqual(["a1", "a1x", "a1y", "a2", "b"]);
        expect(forwards.anchor).toBe("a1");

        const backwards = range(forwards, rows, "a", rowOf);
        expect(chosen(backwards)).toStrictEqual(["a", "a1", "a1x", "a1y", "a2", "b"]);
        expect(backwards.anchor).toBe("a1");
        expect(count(backwards)).toBe(6);
    });

    it("drops the range when the anchor was dropped", () => {
        const anchored = tick(range(tick(noSelection, "a"), rows, "b1", rowOf), "a1x");
        expect(anchored.anchor).toBe("a1x");
        expect(chosen(range(anchored, rows, "b", rowOf))).toStrictEqual(["a", "a1", "b1"]);
        expect(chosen(range(anchored, rows, "a", rowOf))).toStrictEqual(["a1y", "a2", "b", "b1"]);
    });

    it("takes a range out of a subtree as exceptions", () => {
        const state = range(tick(branch(noSelection, "a"), "a1"), rows, "a2", rowOf);
        expect(chosen(state)).toStrictEqual(["a"]);
        expect([...state.except.keys()]).toStrictEqual(["a1", "a1x", "a1y", "a2"]);
        expect(count(state)).toBe(1);
    });

    it("skips a row inside a branch taken out when its children are not known", () => {
        const removed = tick(unbranch(branch(noSelection, "a"), "a1"), "a2");
        const state = range(tick(removed, "a2"), rows, "a1x", rowOf);
        expect(chosen(state)).toStrictEqual(["a", "a2"]);
    });

    it.each<{ label: string; state: () => Selection }>([
        { label: "no anchor", state: () => noSelection },
        { label: "an anchor no longer shown", state: () => ({ ...tick(noSelection, "a1x"), anchor: "gone" }) },
    ])("acts as a plain toggle and sets the anchor with $label", ({ state }) => {
        const before = state();
        const next = range(before, rows, "a2", rowOf);
        expect(isSelected(next, "a2", row("a2"))).toBe(true);
        expect(next.anchor).toBe("a2");
        expect(next.ids.size).toBe(before.ids.size + 1);
    });

    it.each<{ from: string; to: string; span: string[] }>([
        { from: "a1", to: "a2", span: ["a1", "a1x", "a1y", "a2"] },
        { from: "a2", to: "a1", span: ["a1", "a1x", "a1y", "a2"] },
        { from: "b", to: "b", span: ["b"] },
        { from: "a", to: "gone", span: [] },
    ])("spans $from to $to as $span", ({ from, to, span }) => {
        expect(between(rows, from, to)).toStrictEqual(span);
    });
});

describe("the number shown", () => {
    it.each<{ label: string; state: () => Selection; shown: number; known: boolean }>([
        { label: "two picked rows", state: () => tick(noSelection, "a", "b"), shown: 2, known: true },
        { label: "a subtree by its size", state: () => must(selectSubtree(noSelection, { pageId: "a", size: 212 }, row("a"))), shown: 212, known: true },
        { label: "a subtree with no size as its page", state: () => must(selectSubtree(noSelection, { pageId: "a" }, row("a"))), shown: 1, known: true },
        { label: "a type's subtree, whose size only Go knows", state: () => branch(noSelection, "a", "guide"), shown: 0, known: false },
        { label: "picked rows, subtrees and exceptions together", state: () => tick(branch(branch(tick(noSelection, "b"), "a"), "b1"), "a1"), shown: 6, known: true },
        { label: "a filter by what the person saw", state: () => selectAll(noSelection, {}, 5000), shown: 5000, known: true },
        { label: "a subtree beside a filter", state: () => branch(selectAll(noSelection, { status: "published" }, 9), "a"), shown: 14, known: false },
        { label: "a row whose place is unknown beside a subtree", state: () => branch(must(toggle(noSelection, "z", {})), "a"), shown: 6, known: false },
    ])("counts $label as $shown, known: $known", ({ state, shown, known }) => {
        expect(count(state())).toBe(shown);
        expect(exact(state())).toBe(known);
    });
});

describe("the request Go resolves", () => {
    it.each<{ label: string; state: () => Selection; request: PageSelection }>([
        { label: "nothing", state: () => noSelection, request: { siteId: "s1" } },
        { label: "picked rows", state: () => tick(noSelection, "b", "a1"), request: { siteId: "s1", pageIds: ["b", "a1"] } },
        {
            label: "subtrees with and without a type",
            state: () => branch(branch(noSelection, "a", "guide"), "b"),
            request: { siteId: "s1", subtrees: [{ pageId: "a", typeId: "guide" }, { pageId: "b" }] },
        },
        {
            label: "a filter without its empty fields",
            state: () => tick(selectAll(noSelection, { status: "published", pathPrefix: "", search: undefined, findings: true, body: "without" }, 214), "a2"),
            request: { siteId: "s1", filter: { status: "published", findings: true, body: "without" }, seen: 214, except: ["a2"] },
        },
        { label: "a filter over everything", state: () => selectAll(noSelection, {}, 6), request: { siteId: "s1", filter: {}, seen: 6 } },
        { label: "a group", state: () => selectGroup(noSelection, "guide", 4), request: { siteId: "s1", filter: { pageTypeId: "guide" }, seen: 4 } },
        { label: "a rule's matches", state: () => selectRule(noSelection, "r-guide", 4), request: { siteId: "s1", ruleId: "r-guide", seen: 4 } },
        {
            label: "a subtree with an exception and a branch out beside a picked row",
            state: () => unbranch(tick(branch(tick(noSelection, "b1"), "a"), "a2"), "a1"),
            request: { siteId: "s1", pageIds: ["b1"], subtrees: [{ pageId: "a" }], except: ["a2"], exceptSubtrees: [{ pageId: "a1" }] },
        },
    ])("asks for $label", ({ state, request }) => {
        expect(toRequest("s1", state())).toStrictEqual(request);
    });
});
