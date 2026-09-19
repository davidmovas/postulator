import { describe, expect, it } from "vitest";

import { HitGrid } from "../../../canvas/hit-grid.js";
import { layoutTree } from "../../../canvas/tree-layout.js";
import type { Edge, Entity } from "../../../data/types.js";
import { edge, entity } from "./fixture.js";
import { expandAll, treeOf, visibleRows } from "./fold.js";
import { buildGraphIndex } from "./index.js";
import { rank } from "./search.js";

const roots = 10;
const perRoot = 20;
const perBranch = 50;

function synthetic(): { entities: Entity[]; edges: Edge[] } {
    const entities: Entity[] = [];
    const edges: Edge[] = [];
    for (let root = 0; root < roots; root += 1) {
        const rootId = `hub-${root}`;
        entities.push(entity({ id: rootId, name: `Hub ${root}`, kind: "hub", score: 1 - root / 100 }));
        for (let branch = 0; branch < perRoot; branch += 1) {
            const branchId = `${rootId}-cat-${branch}`;
            entities.push(entity({ id: branchId, name: `Category ${root}.${branch}`, kind: "category", score: 0.9 - branch / 100 }));
            edges.push(edge(`${branchId}-parent`, branchId, rootId, "parent"));
            for (let leaf = 0; leaf < perBranch; leaf += 1) {
                const leafId = `${branchId}-p-${leaf}`;
                entities.push(entity({ id: leafId, name: `Product ${root}.${branch}.${leaf}`, kind: "product", score: 0.5 - leaf / 200, page: leaf % 7 !== 0 }));
                edges.push(edge(`${leafId}-parent`, leafId, branchId, "parent"));
                if (leaf % 5 === 0 && leaf > 0) {
                    edges.push(edge(`${leafId}-rel`, `${branchId}-p-${leaf - 1}`, leafId, "related", leaf % 10 === 0 ? "proposed" : "approved", 0.6));
                }
            }
        }
    }
    return { entities, edges };
}

function timed<T>(work: () => T): { result: T; ms: number } {
    const started = performance.now();
    const result = work();
    return { result, ms: performance.now() - started };
}

describe("a site of ten thousand entities", () => {
    const { entities, edges } = synthetic();

    it("has the size it claims", () => {
        expect(entities.length).toBe(roots + roots * perRoot + roots * perRoot * perBranch);
    });

    it("indexes, folds, lays out, grids and searches inside the budget", () => {
        const indexed = timed(() => buildGraphIndex(entities, edges));
        expect(indexed.result.counts.total).toBe(entities.length);
        expect(indexed.ms).toBeLessThan(400);

        const rows = timed(() => visibleRows(indexed.result, expandAll(indexed.result), "score"));
        expect(rows.result.length).toBe(entities.length);
        expect(rows.ms).toBeLessThan(150);

        const laid = timed(() => layoutTree(treeOf(rows.result, () => 160), { rowHeight: 28, nodeHeight: 24, columnGap: 48, rootGap: 1 }));
        expect(laid.result.nodes.length).toBe(rows.result.length);
        expect(laid.ms).toBeLessThan(250);

        const grid = timed(() => new HitGrid(laid.result.nodes, 64));
        expect(grid.ms).toBeLessThan(150);
        const hit = grid.result.at(laid.result.nodes[5].x + 1, laid.result.nodes[5].y + 1);
        expect(hit?.id).toBe(laid.result.nodes[5].id);

        const searched = timed(() => rank(entities, "product 3.4", 8));
        expect(searched.result.length).toBe(8);
        expect(searched.ms).toBeLessThan(80);
    });
});
