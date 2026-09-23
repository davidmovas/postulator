import { describe, expect, it } from "vitest";

import type { TreeInput } from "./tree-layout.js";
import { layoutTree } from "./tree-layout.js";

function tree(id: string, width: number, children: TreeInput[] = []): TreeInput {
    return { id, width, children };
}

const options = { rowHeight: 28, nodeHeight: 24, columnGap: 48, rootGap: 1 };

describe("layoutTree", () => {
    it("gives every leaf a row and centres a parent on its children", () => {
        const layout = layoutTree([tree("a", 100, [tree("a1", 60), tree("a2", 60), tree("a3", 60)])], options);
        const rows = layout.nodes.map((node) => [node.id, node.y / options.rowHeight]);
        expect(rows).toStrictEqual([
            ["a", 1],
            ["a1", 0],
            ["a2", 1],
            ["a3", 2],
        ]);
    });

    it("centres a parent between two children", () => {
        const layout = layoutTree([tree("a", 10, [tree("a1", 10), tree("a2", 10)])], options);
        expect(layout.byId.get("a")?.y).toBe(0.5 * options.rowHeight);
    });

    it("aligns columns by depth using the widest node of each depth", () => {
        const layout = layoutTree([tree("a", 100, [tree("a1", 60), tree("a2", 200, [tree("x", 10)])])], options);
        expect(layout.columns).toStrictEqual([0, 148, 396]);
        expect(layout.byId.get("x")?.x).toBe(396);
        expect(layout.byId.get("a1")?.x).toBe(148);
    });

    it("separates root subtrees by the root gap", () => {
        const layout = layoutTree([tree("a", 10, [tree("a1", 10), tree("a2", 10)]), tree("b", 10)], options);
        expect(layout.byId.get("b")?.y).toBe(3 * options.rowHeight);
    });

    it("keeps the nodes in pre-order with their depth and parent", () => {
        const layout = layoutTree([tree("a", 10, [tree("a1", 10, [tree("a1x", 10)]), tree("a2", 10)]), tree("b", 10)], options);
        expect(layout.nodes.map((node) => node.id)).toStrictEqual(["a", "a1", "a1x", "a2", "b"]);
        expect(layout.nodes.map((node) => node.depth)).toStrictEqual([0, 1, 2, 1, 0]);
        expect(layout.nodes.map((node) => node.parentId)).toStrictEqual([null, "a", "a1", "a", null]);
    });

    it("never overlaps two nodes of one column", () => {
        const wide: TreeInput[] = [];
        for (let root = 0; root < 5; root += 1) {
            const children: TreeInput[] = [];
            for (let child = 0; child < 4; child += 1) {
                const leaves: TreeInput[] = [];
                for (let leaf = 0; leaf < (child + root) % 3; leaf += 1) {
                    leaves.push(tree(`r${root}c${child}l${leaf}`, 40));
                }
                children.push(tree(`r${root}c${child}`, 80, leaves));
            }
            wide.push(tree(`r${root}`, 120, children));
        }
        const layout = layoutTree(wide, options);
        const byDepth = new Map<number, number[]>();
        for (const node of layout.nodes) {
            const held = byDepth.get(node.depth) ?? [];
            held.push(node.y);
            byDepth.set(node.depth, held);
        }
        for (const ys of byDepth.values()) {
            ys.sort((left, right) => left - right);
            for (let index = 1; index < ys.length; index += 1) {
                expect(ys[index] - ys[index - 1]).toBeGreaterThanOrEqual(options.rowHeight);
            }
        }
    });

    it("reports the bounds of everything placed", () => {
        const layout = layoutTree([tree("a", 100, [tree("a1", 60), tree("a2", 60)])], options);
        expect(layout.bounds).toStrictEqual({ x: 0, y: 0, width: 148 + 60, height: 2 * options.rowHeight - 4 });
    });

    it("lays out nothing for no roots", () => {
        const layout = layoutTree([], options);
        expect(layout.nodes).toHaveLength(0);
        expect(layout.bounds).toStrictEqual({ x: 0, y: 0, width: 0, height: 0 });
        expect(layout.columns).toStrictEqual([]);
    });
});
