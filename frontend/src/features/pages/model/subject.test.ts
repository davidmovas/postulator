import { describe, expect, it } from "vitest";

import { findingCodes } from "../../../generated/vocab.js";
import { findingSentence } from "./subject.js";

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
            "parent_differs",
            { workbook: "p1", derived: "p2", reason: "jsonld_breadcrumb" },
            "The workbook names a different parent from the one above, which comes from the JSON-LD breadcrumb.",
        ],
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
