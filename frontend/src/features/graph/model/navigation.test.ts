import { describe, expect, it } from "vitest";

import { potteryEdges, potteryEntities } from "./fixture.js";
import { defaultFold, toggle, visibleRows } from "./fold.js";
import { buildGraphIndex } from "./index.js";
import { move } from "./navigation.js";

const index = buildGraphIndex(potteryEntities, potteryEdges);
const rows = visibleRows(index, defaultFold(index), "score");

describe("move", () => {
    it("steps up and down the visible rows", () => {
        expect(move(rows, "pottery", "down")).toStrictEqual({ id: "mugs", expand: false });
        expect(move(rows, "mugs", "up")).toStrictEqual({ id: "pottery", expand: false });
        expect(move(rows, "pottery", "up")).toBeNull();
        expect(move(rows, "care", "down")).toBeNull();
    });

    it("starts at the first row when nothing is selected", () => {
        expect(move(rows, null, "down")).toStrictEqual({ id: "pottery", expand: false });
        expect(move(rows, null, "up")).toStrictEqual({ id: "pottery", expand: false });
    });

    it("goes left to the parent and right to the first child", () => {
        expect(move(rows, "m350", "left")).toStrictEqual({ id: "mugs", expand: false });
        expect(move(rows, "pottery", "left")).toBeNull();
        expect(move(rows, "mugs", "right")).toStrictEqual({ id: "m350", expand: false });
        expect(move(rows, "m350", "right")).toBeNull();
    });

    it("asks to expand a collapsed node on right", () => {
        const collapsed = visibleRows(index, toggle(defaultFold(index), "mugs"), "score");
        expect(move(collapsed, "mugs", "right")).toStrictEqual({ id: "mugs", expand: true });
    });

    it("jumps home and end", () => {
        expect(move(rows, "stone", "home")).toStrictEqual({ id: "pottery", expand: false });
        expect(move(rows, "stone", "end")).toStrictEqual({ id: "care", expand: false });
    });

    it("skips more rows", () => {
        const withMore = [
            ...rows.slice(0, 5),
            { kind: "more" as const, id: "more:mugs", parentId: "mugs", depth: 2, hidden: 3, shown: 3 },
            ...rows.slice(5),
        ];
        expect(move(withMore, "travel", "down")).toStrictEqual({ id: "stone", expand: false });
        expect(move(withMore, "stone", "up")).toStrictEqual({ id: "travel", expand: false });
    });

    it("answers nothing for an unknown selection", () => {
        expect(move(rows, "ghost", "down")).toBeNull();
    });
});
