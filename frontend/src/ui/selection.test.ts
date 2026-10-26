import { describe, expect, it } from "vitest";

import type { PageSelection, Selection, SelectionRow } from "./selection.js";
import {
    between,
    clear,
    clearSubtree,
    count,
    coveredAbove,
    isEmpty,
    isSelected,
    noSelection,
    range,
    selectAll,
    selectGroup,
    selectRule,
    selectSubtree,
    toRequest,
    toggle,
} from "./selection.js";

const site: Readonly<Record<string, SelectionRow>> = {
    a: { ancestors: [], typeId: "hub", ruleId: "r-hub" },
    a1: { ancestors: ["a"], typeId: "guide", ruleId: "r-guide" },
    a1x: { ancestors: ["a1", "a"], typeId: "guide", ruleId: "r-guide" },
    a2: { ancestors: ["a"], typeId: "info", ruleId: null },
    b: { ancestors: [], typeId: "hub", ruleId: "r-hub" },
    b1: { ancestors: ["b"], typeId: "guide", ruleId: "r-guide" },
};

const rows: readonly string[] = ["a", "a1", "a1x", "a2", "b", "b1"];

const rowOf = (id: string): SelectionRow | undefined => site[id];

function row(id: string): SelectionRow {
    const held = site[id];
    if (held === undefined) {
        throw new Error(`no row ${id}`);
    }
    return held;
}

function chosen(state: Selection): string[] {
    return rows.filter((id) => isSelected(state, id, row(id)));
}

function tick(state: Selection, ...ids: string[]): Selection {
    return ids.reduce((held, id) => toggle(held, id, row(id)), state);
}

describe("what a selection holds", () => {
    it.each<{ label: string; state: () => Selection; selected: string[] }>([
        { label: "nothing", state: () => noSelection, selected: [] },
        { label: "two picked rows", state: () => tick(noSelection, "a1", "b"), selected: ["a1", "b"] },
        {
            label: "a page with everything under it",
            state: () => selectSubtree(noSelection, { pageId: "a", size: 4 }, row("a")),
            selected: ["a", "a1", "a1x", "a2"],
        },
        {
            label: "a branch's pages of one type",
            state: () => selectSubtree(noSelection, { pageId: "a", typeId: "guide", size: 2 }, row("a")),
            selected: ["a1", "a1x"],
        },
        { label: "a group of one type", state: () => selectGroup(noSelection, "guide", 3), selected: ["a1", "a1x", "b1"] },
        { label: "every row the filter shows", state: () => selectAll(noSelection, { status: "publish" }, 40), selected: [...rows] },
        { label: "a rule's matches", state: () => selectRule(noSelection, "r-guide", 3), selected: ["a1", "a1x", "b1"] },
    ])("holds $label", ({ state, selected }) => {
        expect(chosen(state())).toStrictEqual(selected);
    });

    it("counts a group's row whose type it was not told as shown, and an untyped page as outside it", () => {
        const group = selectGroup(noSelection, "guide", 3);
        expect(isSelected(group, "z", { ancestors: [] })).toBe(true);
        expect(isSelected(group, "z", { ancestors: [], typeId: null })).toBe(false);
    });

    it("never holds a rule's row that does not say which rule placed it", () => {
        expect(isSelected(selectRule(noSelection, "r-guide", 3), "z", { ancestors: [] })).toBe(false);
    });

    it.each<{ label: string; state: () => Selection; empty: boolean }>([
        { label: "nothing", state: () => noSelection, empty: true },
        { label: "a row picked and dropped", state: () => tick(noSelection, "a", "a"), empty: true },
        { label: "a picked row", state: () => tick(noSelection, "a"), empty: false },
        { label: "a subtree", state: () => selectSubtree(noSelection, { pageId: "a", typeId: "guide", size: 0 }, row("a")), empty: false },
        { label: "a filter", state: () => selectAll(noSelection, {}, 0), empty: false },
        { label: "a cleared one", state: () => clear(), empty: true },
    ])("is empty for $label: $empty", ({ state, empty }) => {
        expect(isEmpty(state())).toBe(empty);
    });
});

describe("taking rows back out", () => {
    it.each<{ label: string; state: () => Selection; inside: string; before: number }>([
        { label: "a subtree", state: () => selectSubtree(noSelection, { pageId: "a", size: 4 }, row("a")), inside: "a1", before: 4 },
        { label: "a group", state: () => selectGroup(noSelection, "guide", 3), inside: "b1", before: 3 },
        { label: "every row the filter shows", state: () => selectAll(noSelection, { search: "mug" }, 214), inside: "a2", before: 214 },
        { label: "a rule's matches", state: () => selectRule(noSelection, "r-guide", 3), inside: "a1x", before: 3 },
    ])("takes a row out of $label as an exception and puts it back", ({ state, inside, before }) => {
        const whole = state();
        const out = toggle(whole, inside, row(inside));
        expect(isSelected(out, inside, row(inside))).toBe(false);
        expect([...out.except.keys()]).toStrictEqual([inside]);
        expect(count(out)).toBe(before - 1);

        const back = toggle(out, inside, row(inside));
        expect(isSelected(back, inside, row(inside))).toBe(true);
        expect(back.except.size).toBe(0);
        expect(back.ids.size).toBe(0);
        expect(count(back)).toBe(before);
    });

    it("adds a row outside the group as a picked row beside it", () => {
        const state = toggle(selectGroup(noSelection, "guide", 3), "a2", row("a2"));
        expect(chosen(state)).toStrictEqual(["a1", "a1x", "a2", "b1"]);
        expect(count(state)).toBe(4);
    });
});

describe("a subtree beside what was already picked", () => {
    it("folds the picked rows and the exceptions beneath it into the subtree", () => {
        const picked = tick(noSelection, "a1", "b1");
        const state = selectSubtree(picked, { pageId: "a", size: 4 }, row("a"));
        expect([...state.ids.keys()]).toStrictEqual(["b1"]);
        expect(state.subtrees.map((pick) => pick.pageId)).toStrictEqual(["a"]);
        expect(count(state)).toBe(5);

        const excepted = toggle(state, "a1x", row("a1x"));
        const again = selectSubtree(excepted, { pageId: "a1", size: 2 }, row("a1"));
        expect(again.except.size).toBe(0);
        expect(again.subtrees.map((pick) => pick.pageId)).toStrictEqual(["a"]);
        expect(count(again)).toBe(5);
    });

    it("folds a smaller subtree into the larger one that holds it", () => {
        const state = selectSubtree(selectSubtree(noSelection, { pageId: "a1", size: 2 }, row("a1")), { pageId: "a", size: 4 }, row("a"));
        expect(state.subtrees.map((pick) => pick.pageId)).toStrictEqual(["a"]);
        expect(count(state)).toBe(4);
    });

    it("keeps a whole subtree beside a type's pick inside it, since the pick holds only part of it", () => {
        const state = selectSubtree(selectSubtree(noSelection, { pageId: "a", typeId: "guide", size: 2 }, row("a")), { pageId: "a1", size: 2 }, row("a1"));
        expect(state.subtrees.map((pick) => [pick.pageId, pick.typeId])).toStrictEqual([
            ["a", "guide"],
            ["a1", undefined],
        ]);
    });

    it("selects a subtree once however often it is asked", () => {
        const once = selectSubtree(noSelection, { pageId: "a", size: 4 }, row("a"));
        expect(selectSubtree(once, { pageId: "a", size: 4 }, row("a")).subtrees).toHaveLength(1);
    });

    it("clears a subtree with what was picked and taken out beneath it, and keeps the rest", () => {
        const state = tick(selectSubtree(noSelection, { pageId: "a", size: 4 }, row("a")), "a1", "b1");
        const cleared = clearSubtree(state, { pageId: "a" });
        expect(cleared.subtrees).toHaveLength(0);
        expect(cleared.except.size).toBe(0);
        expect([...cleared.ids.keys()]).toStrictEqual(["b1"]);
        expect(count(cleared)).toBe(1);
    });

    it("clears only one type's part of a branch", () => {
        const state = tick(noSelection, "a1", "a2");
        expect([...clearSubtree(state, { pageId: "a", typeId: "guide" }).ids.keys()]).toStrictEqual(["a2"]);
    });

    it("says when a row is held by a pick above it rather than its own", () => {
        const state = selectSubtree(noSelection, { pageId: "a", size: 4 }, row("a"));
        expect(coveredAbove(state, "a1", row("a1"))).toBe(true);
        expect(coveredAbove(state, "a", row("a"))).toBe(false);
        expect(coveredAbove(selectAll(noSelection, {}, 6), "a", row("a"))).toBe(true);
    });
});

describe("selecting everything a filter matches", () => {
    it("replaces the picked rows it covers and the earlier filter's exceptions", () => {
        const before = toggle(selectAll(tick(noSelection, "a1"), { status: "draft" }, 9), "b", row("b"));
        const state = selectAll(before, { status: "publish" }, 40);
        expect(state.ids.size).toBe(0);
        expect(state.except.size).toBe(0);
        expect(count(state)).toBe(40);
    });

    it("keeps a group's picked rows of another type", () => {
        const state = selectGroup(tick(noSelection, "a1", "a2"), "guide", 3);
        expect([...state.ids.keys()]).toStrictEqual(["a2"]);
        expect(count(state)).toBe(4);
    });

    it("lets a rule's matches replace a filter, as they share the count the person saw", () => {
        const state = selectRule(selectAll(noSelection, {}, 6), "r-hub", 2);
        expect(state.whole).toStrictEqual({ kind: "rule", ruleId: "r-hub", seen: 2 });
        expect(chosen(state)).toStrictEqual(["a", "b"]);
    });
});

describe("a shift-range over the shown rows", () => {
    it("picks every row from the anchor to the target, forwards and backwards, keeping the anchor", () => {
        const anchored = tick(noSelection, "a1");
        const forwards = range(anchored, rows, "b", rowOf);
        expect(chosen(forwards)).toStrictEqual(["a1", "a1x", "a2", "b"]);
        expect(forwards.anchor).toBe("a1");

        const backwards = range(forwards, rows, "a", rowOf);
        expect(chosen(backwards)).toStrictEqual(["a", "a1", "a1x", "a2", "b"]);
        expect(backwards.anchor).toBe("a1");
        expect(count(backwards)).toBe(5);
    });

    it("drops the range when the anchor was dropped", () => {
        const all = range(tick(noSelection, "a"), rows, "b1", rowOf);
        const anchored = toggle(all, "a1x", row("a1x"));
        expect(anchored.anchor).toBe("a1x");
        expect(chosen(range(anchored, rows, "b", rowOf))).toStrictEqual(["a", "a1", "b1"]);
        expect(chosen(range(anchored, rows, "a", rowOf))).toStrictEqual(["a2", "b", "b1"]);
    });

    it("takes a range out of a subtree as exceptions", () => {
        const subtree = selectSubtree(noSelection, { pageId: "a", size: 4 }, row("a"));
        const state = range(toggle(subtree, "a1", row("a1")), rows, "a2", rowOf);
        expect(chosen(state)).toStrictEqual(["a"]);
        expect([...state.except.keys()]).toStrictEqual(["a1", "a1x", "a2"]);
        expect(count(state)).toBe(1);
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
        { from: "a1", to: "a2", span: ["a1", "a1x", "a2"] },
        { from: "a2", to: "a1", span: ["a1", "a1x", "a2"] },
        { from: "b", to: "b", span: ["b"] },
        { from: "a", to: "gone", span: [] },
    ])("spans $from to $to as $span", ({ from, to, span }) => {
        expect(between(rows, from, to)).toStrictEqual(span);
    });
});

describe("the number shown", () => {
    it.each<{ label: string; state: () => Selection; shown: number }>([
        { label: "two picked rows", state: () => tick(noSelection, "a", "b"), shown: 2 },
        { label: "a subtree by its size", state: () => selectSubtree(noSelection, { pageId: "a", size: 212 }, row("a")), shown: 212 },
        { label: "a subtree with no size as its page", state: () => selectSubtree(noSelection, { pageId: "a" }, row("a")), shown: 1 },
        {
            label: "a type's subtree with no size as nothing yet",
            state: () => selectSubtree(noSelection, { pageId: "a", typeId: "guide" }, row("a")),
            shown: 0,
        },
        {
            label: "picked rows, subtrees and exceptions together",
            state: () =>
                toggle(
                    selectSubtree(selectSubtree(tick(noSelection, "b"), { pageId: "a", size: 200 }, row("a")), { pageId: "b1", size: 13 }, row("b1")),
                    "a1",
                    row("a1"),
                ),
            shown: 213,
        },
        { label: "a filter by what the person saw", state: () => selectAll(noSelection, {}, 5000), shown: 5000 },
    ])("counts $label as $shown", ({ state, shown }) => {
        expect(count(state())).toBe(shown);
    });
});

describe("the request Go resolves", () => {
    it.each<{ label: string; state: () => Selection; request: PageSelection }>([
        { label: "nothing", state: () => noSelection, request: { siteId: "s1" } },
        { label: "picked rows", state: () => tick(noSelection, "b", "a1"), request: { siteId: "s1", pageIds: ["b", "a1"] } },
        {
            label: "subtrees with and without a type",
            state: () => selectSubtree(selectSubtree(noSelection, { pageId: "a", typeId: "guide", size: 2 }, row("a")), { pageId: "b", size: 2 }, row("b")),
            request: { siteId: "s1", subtrees: [{ pageId: "a", typeId: "guide" }, { pageId: "b" }] },
        },
        {
            label: "a filter without its empty fields",
            state: () => toggle(selectAll(noSelection, { status: "publish", pathPrefix: "", search: undefined, findings: true, body: "without" }, 214), "a2", row("a2")),
            request: { siteId: "s1", filter: { status: "publish", findings: true, body: "without" }, seen: 214, except: ["a2"] },
        },
        { label: "a filter over everything", state: () => selectAll(noSelection, {}, 6), request: { siteId: "s1", filter: {}, seen: 6 } },
        { label: "a group", state: () => selectGroup(noSelection, "guide", 3), request: { siteId: "s1", filter: { pageTypeId: "guide" }, seen: 3 } },
        { label: "a rule's matches", state: () => selectRule(noSelection, "r-guide", 3), request: { siteId: "s1", ruleId: "r-guide", seen: 3 } },
        {
            label: "a subtree with an exception beside a picked row",
            state: () => toggle(selectSubtree(tick(noSelection, "b1"), { pageId: "a", size: 4 }, row("a")), "a1x", row("a1x")),
            request: { siteId: "s1", pageIds: ["b1"], subtrees: [{ pageId: "a" }], except: ["a1x"] },
        },
    ])("asks for $label", ({ state, request }) => {
        expect(toRequest("s1", state())).toStrictEqual(request);
    });
});
