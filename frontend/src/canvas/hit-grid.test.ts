import { describe, expect, it } from "vitest";

import { HitGrid } from "./hit-grid.js";

const items = [
    { id: "a", x: 0, y: 0, width: 100, height: 20 },
    { id: "b", x: 50, y: 10, width: 100, height: 20 },
    { id: "far", x: 1000, y: 1000, width: 10, height: 10 },
];

describe("HitGrid", () => {
    it("finds the item under a point, the later one winning where they overlap", () => {
        const grid = new HitGrid(items, 32);
        expect(grid.at(10, 10)?.id).toBe("a");
        expect(grid.at(60, 15)?.id).toBe("b");
        expect(grid.at(100, 10)?.id).toBe("b");
        expect(grid.at(500, 500)).toBeNull();
    });

    it("treats the right and bottom edges as outside", () => {
        const grid = new HitGrid(items, 32);
        expect(grid.at(150, 15)).toBeNull();
        expect(grid.at(10, 20)).toBeNull();
    });

    it("lists the items touching a rectangle once each", () => {
        const grid = new HitGrid(items, 32);
        expect(grid.within({ x: 40, y: 0, width: 20, height: 40 }).map((item) => item.id)).toStrictEqual(["a", "b"]);
        expect(grid.within({ x: 200, y: 0, width: 50, height: 50 })).toStrictEqual([]);
        expect(grid.within({ x: 0, y: 0, width: 2000, height: 2000 })).toHaveLength(3);
    });

    it("is empty without items", () => {
        const grid = new HitGrid([], 32);
        expect(grid.at(0, 0)).toBeNull();
        expect(grid.within({ x: 0, y: 0, width: 10, height: 10 })).toStrictEqual([]);
    });
});
