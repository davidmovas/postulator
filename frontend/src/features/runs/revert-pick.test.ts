import { describe, expect, it } from "vitest";

import type { BlockChangeView, FieldChangeView } from "./artifacts.js";
import { publishView } from "./artifacts.js";
import { picked, revertableKeys } from "./revert-pick.js";

const moved: FieldChangeView = { key: "excerpt", before: "Old", beforeHash: "b1", after: "New", afterHash: "a1" };
const still: FieldChangeView = { key: "title", before: "Same", beforeHash: "s1", after: "Same", afterHash: "s1" };
const added: BlockChangeView = { kind: "heading", hash: "h1", raw: "<h2>A</h2>", owner: "ours", change: "added" };
const kept: BlockChangeView = { kind: "paragraph", hash: "h2", raw: "<p>B</p>", owner: "client", change: "kept" };

function viewOf(held: Record<string, unknown>) {
    const view = publishView({ wpId: 4, created: false, ...held });
    if (view === null) {
        throw new Error("no view");
    }
    return view;
}

describe("the keys a person may put back from a publish", () => {
    it("offers each field the write moved and the body when its blocks changed", () => {
        expect(revertableKeys(viewOf({ fields: [moved, still], blocks: [kept, added] }))).toStrictEqual(["excerpt", "body"]);
    });

    it("names a product's body by its description", () => {
        expect(revertableKeys(viewOf({ fields: [], blocks: [added], previousProduct: { shortWritten: false } }))).toStrictEqual([
            "product_description",
        ]);
    });

    it("leaves the body out when every block was kept", () => {
        expect(revertableKeys(viewOf({ fields: [moved], blocks: [kept] }))).toStrictEqual(["excerpt"]);
    });

    it("offers nothing on a page the run created, whose revert takes the whole page down", () => {
        expect(revertableKeys(viewOf({ created: true, fields: [moved], blocks: [added] }))).toStrictEqual([]);
    });

    it("sends the ticked keys in the order they are offered", () => {
        expect(picked(["excerpt", "seo_title", "body"], new Set(["body", "excerpt"]))).toStrictEqual(["excerpt", "body"]);
    });
});
