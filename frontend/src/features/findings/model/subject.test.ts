import { describe, expect, it } from "vitest";

import { findingCodes } from "../../../generated/vocab.js";
import { findingSentence, rowCandidates } from "./subject.js";

describe("the pages a workbook row matches", () => {
    it("pairs each candidate page with its path", () => {
        expect(rowCandidates({ candidates: ["/lab/guides/", "/shop/guides/"], pageIds: ["p1", "p2"] })).toStrictEqual([
            { pageId: "p1", path: "/lab/guides/" },
            { pageId: "p2", path: "/shop/guides/" },
        ]);
    });

    it("keeps a page whose path is missing under its id and drops what is no id", () => {
        expect(rowCandidates({ candidates: ["/a/"], pageIds: ["p1", "p2", 7, ""] })).toStrictEqual([
            { pageId: "p1", path: "/a/" },
            { pageId: "p2", path: "p2" },
        ]);
    });

    it("answers none for a subject that names no pages", () => {
        expect(rowCandidates(null)).toStrictEqual([]);
        expect(rowCandidates({ pageIds: "p1" })).toStrictEqual([]);
    });
});

describe("a finding's subject in words", () => {
    it.each([
        ["parent_unresolved", { name: "Healing Peptides", path: "" }, "The breadcrumb names “Healing Peptides”, which leads to no page in the model."],
        [
            "parent_unresolved",
            { name: "Healing Peptides", path: "/healing/" },
            "The breadcrumb names “Healing Peptides” at /healing/, which is no page in the model.",
        ],
        [
            "parent_ambiguous",
            { name: "Guides", candidates: ["/lab/guides/", "/shop/guides/"] },
            "The breadcrumb names “Guides”, which 2 pages carry: /lab/guides/, /shop/guides/.",
        ],
        ["parent_cycle", { paths: ["/a/", "/b/"] }, "The parents loop through /a/ › /b/, so the loop is broken here."],
        [
            "parent_name_twin",
            { name: "Peptides", page: "/peptides/", terms: ["/product-category/peptides/"] },
            "The breadcrumb names “Peptides”: the page /peptides/ was taken over the store category /product-category/peptides/.",
        ],
        ["parent_name_twin", { name: "Peptides", page: "", terms: [] }, "A crumb's name is shared by a page and a store category, so the page was taken."],
        [
            "parent_differs",
            { sheet: ["Peptides", "Healing Peptides"], site: ["Peptides"] },
            "The workbook puts this page under Peptides › Healing Peptides; the site shows it under Peptides.",
        ],
        [
            "parent_differs",
            { sheet: ["Peptides"], site: [] },
            "The workbook puts this page under Peptides; the site shows it at the top.",
        ],
        ["parent_differs", { sheet: [], site: ["Peptides"] }, "The workbook places this page under other parents than the site shows."],
        [
            "row_ambiguous",
            { sheet: "Groups", rule: "title", candidates: ["/lab/guides/", "/shop/guides/"], pageIds: ["p1", "p2"] },
            "A row of the sheet Groups matches 2 pages: /lab/guides/, /shop/guides/.",
        ],
        ["row_ambiguous", { sheet: "", candidates: [] }, "A row of the workbook matches several pages."],
        [
            "row_without_page",
            { sheet: "Groups", plannedPath: "/peptides/kpv/", product: false },
            "A row of the sheet Groups has no page on the site yet; it is planned at /peptides/kpv/.",
        ],
        [
            "row_without_page",
            { sheet: "Catalog", plannedPath: "/product/kpv/", product: true },
            "A product row of the sheet Catalog waits for its product in the store.",
        ],
        ["row_without_page", { sheet: "", plannedPath: "" }, "A row of the workbook has no page on the site yet."],
        ["plugin_outdated", { version: "1.4.0", required: "1.4.1" }, "The companion plugin is version 1.4.0 and 1.4.1 is needed."],
        [
            "field_changed_on_site",
            { key: "seo_description", baseHash: "a", currentHash: "b" },
            "The SEO description was edited on the site since it was last read or written.",
        ],
        ["body_changed_on_site", { baseHash: "a", currentHash: "b" }, "The body was edited on the site since it was last read or written."],
        ["render_unreachable", { path: "/x/", status: 403 }, "The site answered 403 when the page was fetched."],
        ["render_unreachable", { path: "/x/", status: 0 }, "The page could not be fetched from the site."],
        ["render_redirected", { target: "/peptides/" }, "Visiting the page landed on /peptides/."],
        ["builder_detected", { builder: "elementor" }, "Made with elementor, so its body is read only."],
        ["technical_detected", { reason: "cart" }, "Read as a technical page: Cart."],
        ["page_title_differs", { planned: "Mugs", site: "Ceramic mugs" }, "The plan says “Mugs”, the site says “Ceramic mugs”."],
        ["url_form_mixed", { absolute: 4, relative: 2 }, "The site's pages mix link forms: 4 absolute, 2 relative."],
        [
            "entity_card_proposed",
            { entityId: "e1", name: "Peptides", base: "3f1a", definition: "Short chains.", facts: [], reason: "" },
            "The agent proposed a card for Peptides; accept or dismiss it on the Graph screen.",
        ],
    ])("words %s from the keys it carries", (code, subject, sentence) => {
        expect(findingSentence(code, subject)).toBe(sentence);
    });

    it("never prints a raw key, a code or JSON, whatever the subject holds", () => {
        for (const code of findingCodes) {
            for (const subject of [null, {}, { name: 42, candidates: "x", paths: [1, null] }]) {
                const sentence = findingSentence(code, subject);
                expect(sentence).not.toBe("");
                expect(sentence).not.toContain("_");
                expect(sentence).not.toContain("{");
            }
        }
    });

    it("falls back to a plain sentence for a code it does not know", () => {
        expect(findingSentence("teleported", { name: "x" })).toBe("The observation flagged something here.");
    });
});
