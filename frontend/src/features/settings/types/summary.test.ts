import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import type { FieldScope, Rule } from "../../../data/types.js";
import { aPageType, aScope, hubLinks } from "../../../testing/pagetypes.js";
import { rulesGiving, typeGroups, writtenSummary } from "./summary.js";

function on(field: string, overrides: Partial<FieldScope> = {}): FieldScope {
    return aScope(field, { enabled: true, ...overrides });
}

function aRule(id: string, typeId: string, enabled = true): Rule {
    return {
        id,
        siteId: "s1",
        kind: "rule",
        number: 1,
        name: id,
        enabled,
        typeId,
        conditions: { slugs: [id] },
        pages: 0,
        createdAt: "2026-10-26T09:00:00Z",
        updatedAt: "2026-10-26T09:00:00Z",
    };
}

describe("what a type writes, in words", () => {
    it.each<{ label: string; scopes: FieldScope[]; text: string }>([
        {
            label: "the SEO title and description as one",
            scopes: [on("seo_title"), on("seo_description"), aScope("body")],
            text: "The SEO title and description.",
        },
        {
            label: "the title, the body with its links and the SEO pair, in the catalog's order",
            scopes: [on("seo_description"), on("body", { links: hubLinks() }), on("title"), on("seo_title")],
            text: "The title, the body and its links, the SEO title and description.",
        },
        {
            label: "one half of the SEO pair alone",
            scopes: [on("seo_description"), aScope("seo_title")],
            text: "The SEO description.",
        },
        {
            label: "a product's fields, the variations once",
            scopes: [
                on("product_description", { links: hubLinks() }),
                on("product_short_description"),
                on("variation_description:12"),
                on("variation_description:13"),
            ],
            text: "The product description and its links, the short description, the variation descriptions.",
        },
        {
            label: "a body without links and a custom field by its label",
            scopes: [on("meta:dosage_note", { label: "Dosage note" }), on("body")],
            text: "The body, Dosage note.",
        },
        {
            label: "nothing when every field is off",
            scopes: [aScope("title"), aScope("body")],
            text: copy.pageTypes.writes.nothing,
        },
    ])("says $label", ({ scopes, text }) => {
        expect(writtenSummary(scopes)).toBe(text);
    });
});

describe("the rules that give a type", () => {
    const rules = [aRule("r1", "t-guide"), aRule("r2", "t-cat"), aRule("r3", "t-guide", false), aRule("r4", "t-guide")];

    it("keeps the enabled rules that give it, in order, and says whether the fallback gives it", () => {
        const fallback = { ...aRule("rf", "t-guide"), kind: "fallback" };
        expect(rulesGiving("t-guide", rules, fallback)).toStrictEqual({ rules: [rules[0], rules[3]], fallback: true });
        expect(rulesGiving("t-cat", rules, fallback)).toStrictEqual({ rules: [rules[1]], fallback: false });
        expect(rulesGiving("t-info", rules, null)).toStrictEqual({ rules: [], fallback: false });
    });
});

describe("the types in two groups", () => {
    it("puts what Postulator writes first and what it never writes apart, each in the usual order", () => {
        const store = aPageType("t-store", "store_category", { name: "Store category" });
        const guide = aPageType("t-guide", "guide", { name: "Guide" });
        const category = aPageType("t-cat", "category", { name: "Category" });
        const product = aPageType("t-product", "product", { name: "Product" });
        const technical = aPageType("t-tech", "technical", { name: "Technical" });

        const grouped = typeGroups([store, guide, technical, product, category]);

        expect(grouped.written.map((type) => type.id)).toStrictEqual(["t-cat", "t-guide", "t-product"]);
        expect(grouped.never.map((type) => type.id)).toStrictEqual(["t-store", "t-tech"]);
    });
});
