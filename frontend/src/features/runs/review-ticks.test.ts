import { describe, expect, it } from "vitest";

import type { EditPlanView, FieldChangeView, PageEditView } from "./artifacts.js";
import { acceptRequest, offeredKeys } from "./review-ticks.js";

function change(key: string): FieldChangeView {
    return { key, before: "", beforeHash: "", after: "x", afterHash: "h" };
}

function page(overrides: Partial<PageEditView>): PageEditView {
    return { pageId: "p1", path: "/a/", body: "", fields: [], blocks: [], wpId: 1, ...overrides };
}

function planOf(...pages: PageEditView[]): EditPlanView {
    return { pages, hash: "h" };
}

describe("the keys a review offers to tick", () => {
    it("offers every field change and the body key of each page whose blocks change", () => {
        const offered = offeredKeys(
            planOf(page({ fields: [change("excerpt"), change("seo_title")], body: "product_description" }), page({ body: "body" })),
        );
        expect(offered).toStrictEqual(["excerpt", "seo_title", "product_description", "body"]);
    });

    it("offers nothing on a review that changes bodies only, which is accepted whole", () => {
        expect(offeredKeys(planOf(page({ body: "body" }), page({ pageId: "p2", body: "body" })))).toStrictEqual([]);
        expect(offeredKeys(null)).toStrictEqual([]);
    });

    it("offers a key once though two pages carry it", () => {
        expect(offeredKeys(planOf(page({ fields: [change("excerpt")], body: "body" }), page({ body: "body" })))).toStrictEqual([
            "excerpt",
            "body",
        ]);
    });
});

describe("the acceptance a person sends", () => {
    it("accepts whole when nothing can be ticked or everything is", () => {
        expect(acceptRequest("i1", null)).toStrictEqual({ itemId: "i1", acceptFindings: true });
        expect(acceptRequest("i1", { offered: ["excerpt", "body"], unticked: new Set() })).toStrictEqual({
            itemId: "i1",
            acceptFindings: true,
        });
    });

    it("names the ticked keys when some are unticked", () => {
        expect(acceptRequest("i1", { offered: ["excerpt", "seo_title", "body"], unticked: new Set(["seo_title"]) })).toStrictEqual({
            itemId: "i1",
            acceptFindings: true,
            fields: ["excerpt", "body"],
        });
    });

    it("sends nothing when every key is unticked", () => {
        expect(acceptRequest("i1", { offered: ["excerpt"], unticked: new Set(["excerpt"]) })).toBeNull();
    });
});
