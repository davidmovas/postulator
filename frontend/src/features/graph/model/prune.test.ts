import { describe, expect, it } from "vitest";

import { potteryEdges, potteryEntities } from "./fixture.js";
import { defaultFold, visibleRows } from "./fold.js";
import { buildGraphIndex } from "./index.js";
import { isolate } from "./lens.js";
import { pruneRows } from "./prune.js";

const index = buildGraphIndex(potteryEntities, potteryEdges);
const rows = visibleRows(index, defaultFold(index), "score");

describe("pruneRows", () => {
    it("keeps everything when nothing is isolated", () => {
        expect(pruneRows(rows, null)).toStrictEqual(rows);
    });

    it("drops a subtree whose root is not kept", () => {
        const kept = isolate(index, new Set(["kiln"]));
        expect(pruneRows(rows, kept).map((row) => row.id)).toStrictEqual(["pottery", "glazing", "kiln"]);
    });

    it("keeps a more row only under a kept parent", () => {
        const withMore = [
            ...rows.slice(0, 5),
            { kind: "more" as const, id: "more:mugs", parentId: "mugs", depth: 2, hidden: 3, shown: 3 },
            ...rows.slice(5),
        ];
        const kept = isolate(index, new Set(["travel"]));
        expect(pruneRows(withMore, kept).map((row) => row.id)).toStrictEqual(["pottery", "mugs", "travel", "more:mugs"]);
        expect(pruneRows(withMore, isolate(index, new Set(["care"]))).map((row) => row.id)).toStrictEqual(["care"]);
    });
});
