import { describe, expect, it } from "vitest";

import type { Edge, Entity } from "../../../data/types.js";
import { edge, entity, potteryEdges, potteryEntities } from "./fixture.js";
import {
    autoExpandLimit,
    childLimit,
    collapseToDepth,
    defaultFold,
    expandAll,
    liftMore,
    reveal,
    toggle,
    treeOf,
    visibleRows,
} from "./fold.js";
import { buildGraphIndex } from "./index.js";

const small = buildGraphIndex(potteryEntities, potteryEdges);

function wide(rootsCount: number, perRoot: number) {
    const entities: Entity[] = [];
    const edges: Edge[] = [];
    for (let root = 0; root < rootsCount; root += 1) {
        entities.push(entity({ id: `r${root}`, name: `Root ${root}`, kind: "hub", score: 1 - root / 10 }));
        for (let child = 0; child < perRoot; child += 1) {
            const id = `r${root}c${child}`;
            entities.push(entity({ id, name: `Child ${child}`, score: 1 - child / perRoot }));
            edges.push(edge(`${id}-edge`, id, `r${root}`, "parent"));
        }
    }
    return buildGraphIndex(entities, edges);
}

describe("defaultFold", () => {
    it("expands everything on a small site", () => {
        const fold = defaultFold(small);
        expect([...fold.expanded].sort()).toStrictEqual(["glazing", "mugs", "pottery"]);
        expect(fold.lifted.size).toBe(0);
    });

    it("expands only the roots on a large site", () => {
        const big = wide(3, 100);
        expect(big.counts.total).toBeGreaterThan(autoExpandLimit);
        const fold = defaultFold(big);
        expect([...fold.expanded].sort()).toStrictEqual(["r0", "r1", "r2"]);
    });
});

describe("visibleRows", () => {
    it("walks the placement tree in pre-order, children by score", () => {
        const rows = visibleRows(small, defaultFold(small), "score");
        expect(rows.map((row) => row.id)).toStrictEqual([
            "pottery", "mugs", "m350", "m500", "travel", "stone", "glazing", "kiln", "teapots", "care",
        ]);
        expect(rows.map((row) => row.depth)).toStrictEqual([0, 1, 2, 2, 2, 1, 1, 2, 0, 0]);
    });

    it("orders children by name when asked", () => {
        const rows = visibleRows(small, defaultFold(small), "name");
        expect(rows.slice(0, 5).map((row) => row.id)).toStrictEqual(["care", "pottery", "mugs", "m350", "m500"]);
        const roots = rows.filter((row) => row.depth === 0).map((row) => row.id);
        expect(roots).toStrictEqual(["care", "pottery", "teapots"]);
    });

    it("reports a collapsed node's hidden subtree", () => {
        const rows = visibleRows(small, toggle(defaultFold(small), "mugs"), "score");
        const mugs = rows.find((row) => row.id === "mugs");
        expect(mugs).toMatchObject({ kind: "entity", expanded: false, childCount: 3, hiddenChildren: 3 });
        expect(rows.some((row) => row.id === "m350")).toBe(false);
    });

    it("caps siblings and emits a more row", () => {
        const big = wide(1, 120);
        const rows = visibleRows(big, defaultFold(big), "score");
        expect(rows).toHaveLength(1 + childLimit + 1);
        expect(rows[rows.length - 1]).toStrictEqual({
            kind: "more", id: "more:r0", parentId: "r0", depth: 1, hidden: 120 - childLimit, shown: childLimit,
        });
    });

    it("lifts a parent by a count or entirely", () => {
        const big = wide(1, 120);
        const some = visibleRows(big, liftMore(defaultFold(big), "r0", childLimit), "score");
        expect(some).toHaveLength(1 + 100 + 1);
        expect(some[some.length - 1]).toMatchObject({ kind: "more", hidden: 20, shown: 100 });
        const all = visibleRows(big, liftMore(defaultFold(big), "r0", Number.POSITIVE_INFINITY), "score");
        expect(all).toHaveLength(121);
    });
});

describe("reveal", () => {
    it("expands every ancestor of the target", () => {
        const folded = collapseToDepth(small, 0);
        const rows = visibleRows(small, reveal(small, folded, "kiln"), "score");
        expect(rows.map((row) => row.id)).toContain("kiln");
        expect(rows.some((row) => row.id === "m350")).toBe(false);
    });

    it("lifts the siblings when the target sits past the cap", () => {
        const big = wide(1, 120);
        const target = "r0c80";
        const rows = visibleRows(big, reveal(big, defaultFold(big), target), "score");
        expect(rows.some((row) => row.id === target)).toBe(true);
        expect(rows[rows.length - 1]).toMatchObject({ kind: "more", hidden: 20 });
    });

    it("leaves the fold untouched for an unknown id", () => {
        const fold = defaultFold(small);
        expect(reveal(small, fold, "ghost")).toBe(fold);
    });
});

describe("toggle, expandAll and collapseToDepth", () => {
    it("flips one node", () => {
        const fold = defaultFold(small);
        expect(toggle(fold, "mugs").expanded.has("mugs")).toBe(false);
        expect(toggle(toggle(fold, "mugs"), "mugs").expanded.has("mugs")).toBe(true);
    });

    it("collapses to a depth and expands everything", () => {
        expect([...collapseToDepth(small, 0).expanded]).toStrictEqual([]);
        expect([...collapseToDepth(small, 1).expanded].sort()).toStrictEqual(["pottery"]);
        expect([...expandAll(small).expanded].sort()).toStrictEqual(["glazing", "mugs", "pottery"]);
    });
});

describe("treeOf", () => {
    it("nests the visible rows with the widths asked for", () => {
        const rows = visibleRows(small, toggle(defaultFold(small), "glazing"), "score");
        const roots = treeOf(rows, (row) => (row.kind === "entity" ? row.id.length * 10 : 30));
        expect(roots.map((root) => root.id)).toStrictEqual(["pottery", "teapots", "care"]);
        expect(roots[0].children.map((child) => child.id)).toStrictEqual(["mugs", "stone", "glazing"]);
        expect(roots[0].children[0].children.map((child) => child.id)).toStrictEqual(["m350", "m500", "travel"]);
        expect(roots[0].children[2].children).toStrictEqual([]);
        expect(roots[0].width).toBe(70);
    });

    it("keeps a more row as a child node", () => {
        const big = wide(1, 60);
        const roots = treeOf(visibleRows(big, defaultFold(big), "score"), () => 10);
        expect(roots[0].children).toHaveLength(childLimit + 1);
        expect(roots[0].children[childLimit].id).toBe("more:r0");
    });
});
