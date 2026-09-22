import { describe, expect, it } from "vitest";

import { row } from "./fixture.js";
import { relinkCap, relinkSelection } from "./relink.js";

describe("the pages a relink run takes", () => {
    it("takes only the pages that owe a link", () => {
        const selection = relinkSelection([
            row({ pageId: "owes", path: "/owes/", missing: 2 }),
            row({ pageId: "settled", path: "/settled/", missing: 0 }),
            row({ pageId: "off-graph", path: "/off/", missing: 0, offGraph: 3 }),
        ]);

        expect(selection.pageIds).toStrictEqual(["owes"]);
        expect(selection.total).toBe(1);
        expect(selection.capped).toBe(false);
    });

    it("leaves out a page the audit could not judge", () => {
        const selection = relinkSelection([
            row({ pageId: "unmapped", path: "/unmapped/", missing: 4, skipReason: "not_mapped" }),
            row({ pageId: "owes", path: "/owes/", missing: 1 }),
        ]);

        expect(selection.pageIds).toStrictEqual(["owes"]);
    });

    it("keeps the order the table is in", () => {
        const selection = relinkSelection([
            row({ pageId: "second", path: "/b/", missing: 1 }),
            row({ pageId: "first", path: "/a/", missing: 9 }),
        ]);

        expect(selection.pageIds).toStrictEqual(["second", "first"]);
    });

    it("stops at the cap and says the run is not the whole list", () => {
        const rows = Array.from({ length: relinkCap + 7 }, (_unused, index) =>
            row({ pageId: `page-${index}`, path: `/p${index}/`, missing: 1 }),
        );

        const selection = relinkSelection(rows);

        expect(selection.pageIds).toHaveLength(relinkCap);
        expect(selection.total).toBe(relinkCap + 7);
        expect(selection.capped).toBe(true);
        expect(selection.pageIds[relinkCap - 1]).toBe(`page-${relinkCap - 1}`);
    });

    it("asks for nothing when nothing owes a link", () => {
        const selection = relinkSelection([]);

        expect(selection.pageIds).toStrictEqual([]);
        expect(selection.total).toBe(0);
        expect(selection.capped).toBe(false);
    });
});
