import { describe, expect, it } from "vitest";

import { aPageType, aScope, hubLinks } from "../../../testing/pagetypes.js";
import {
    anchorOf,
    carriesLinks,
    customScope,
    draftOf,
    lastOfRole,
    lengthProblem,
    linkField,
    metaField,
    metaProblem,
    noLinks,
    orderedScopes,
    orderedTypes,
    pickSite,
    reassignTargets,
    requestOf,
    sameDraft,
    usualFor,
} from "./model.js";

describe("a scope's draft", () => {
    const body = aScope("body", { enabled: true, instruction: "  Write for buyers.  ", links: hubLinks(), stored: true });

    it("starts equal to the scope it was read from and sends every field back", () => {
        const draft = draftOf(body);
        expect(sameDraft(draft, draftOf(body))).toBe(true);
        expect(requestOf("t1", "body", draft)).toStrictEqual({
            pageTypeId: "t1",
            field: "body",
            enabled: true,
            aiWrites: true,
            instruction: "Write for buyers.",
            minLength: 0,
            maxLength: 0,
            rewrites: false,
            links: hubLinks(),
            label: "",
            provider: "",
        });
    });

    it("notices a change deep in the link rules", () => {
        const draft = draftOf(body);
        const moved = { ...draft, links: { ...draft.links, rules: { ...draft.links.rules, maxLinks: 3 } } };
        expect(sameDraft(draft, moved)).toBe(false);
        expect(body.links.rules.maxLinks).toBe(20);
    });

    it("refuses a maximum below the minimum and a negative length, and keeps 0 as no limit", () => {
        const draft = draftOf(body);
        expect(lengthProblem({ ...draft, minLength: 120, maxLength: 80 })).toBe("maxBelowMin");
        expect(lengthProblem({ ...draft, minLength: -1 })).toBe("negative");
        expect(lengthProblem({ ...draft, minLength: 120, maxLength: 0 })).toBeNull();
        expect(lengthProblem({ ...draft, minLength: 80, maxLength: 160 })).toBeNull();
    });

    it("keeps the anchor rule it had when a choice it does not know arrives", () => {
        const held = noLinks().anchorStrategy;
        expect(anchorOf("rotate", held)).toBe("rotate");
        expect(anchorOf("shortest", held)).toBe(held);
    });
});

describe("the link-bearing field", () => {
    it("is the product description on a product and the body everywhere else", () => {
        expect(linkField("product")).toBe("product_description");
        expect(linkField("hub")).toBe("body");
        expect(carriesLinks("body", "entity")).toBe(true);
        expect(carriesLinks("body", "product")).toBe(false);
        expect(carriesLinks("seo_title", "hub")).toBe(false);
    });
});

describe("the fields a role uses", () => {
    it("keeps product fields to products and the body and excerpt off them, unless set", () => {
        expect(usualFor(aScope("product_short_description"), "hub")).toBe(false);
        expect(usualFor(aScope("product_short_description"), "product")).toBe(true);
        expect(usualFor(aScope("body"), "product")).toBe(false);
        expect(usualFor(aScope("excerpt"), "product")).toBe(false);
        expect(usualFor(aScope("body", { stored: true }), "product")).toBe(true);
        expect(usualFor(aScope("upsells", { enabled: true }), "entity")).toBe(true);
        expect(usualFor(aScope("meta:dosage", { label: "Dosage" }), "hub")).toBe(true);
    });

    it("orders the fields as the catalog does, the custom ones last by label", () => {
        const scopes = [
            aScope("meta:zinc", { label: "Zinc" }),
            aScope("seo_title"),
            aScope("meta:dose", { label: "Dose" }),
            aScope("title"),
            aScope("body"),
        ];
        expect(orderedScopes(scopes).map((scope) => scope.field)).toStrictEqual([
            "title",
            "body",
            "seo_title",
            "meta:dose",
            "meta:zinc",
        ]);
    });
});

describe("the types of a site", () => {
    const hub = aPageType("t-hub", "hub", { name: "Hub" });
    const category = aPageType("t-cat", "hub", { name: "Category" });
    const guide = aPageType("t-guide", "entity", { name: "Guide" });
    const store = aPageType("t-store", "store_category", { name: "Store category" });
    const types = [store, guide, hub, category];

    it("lists them by role, then by name", () => {
        expect(orderedTypes(types).map((type) => type.name)).toStrictEqual(["Category", "Hub", "Guide", "Store category"]);
    });

    it("knows the last type of a role, which cannot be deleted", () => {
        expect(lastOfRole(types, hub)).toBe(false);
        expect(lastOfRole(types, guide)).toBe(true);
    });

    it("offers the types of the same role first as the place the pages move to", () => {
        expect(reassignTargets(types, hub).map((type) => type.id)).toStrictEqual(["t-cat", "t-store", "t-guide"]);
        expect(reassignTargets(types, guide).map((type) => type.id)).not.toContain("t-guide");
    });

    it("opens the site asked for, else the last one, else the first", () => {
        expect(pickSite("s2", "s1", ["s1", "s2"])).toBe("s2");
        expect(pickSite("gone", "s1", ["s1", "s2"])).toBe("s1");
        expect(pickSite(null, "gone", ["s1", "s2"])).toBe("s1");
        expect(pickSite(null, null, [])).toBeNull();
    });
});

describe("a custom field", () => {
    const scopes = [aScope("meta:dosage", { label: "Dosage" })];

    it("refuses what Go would refuse before anything is sent", () => {
        expect(metaProblem("  ", "Dosage", scopes)).toBe("keyMissing");
        expect(metaProblem("dosage note", "Note", scopes)).toBe("keyShape");
        expect(metaProblem("a,b", "Note", scopes)).toBe("keyShape");
        expect(metaProblem("note", " ", scopes)).toBe("labelMissing");
        expect(metaProblem(" dosage ", "Again", scopes)).toBe("taken");
        expect(metaProblem("strength", "Strength", scopes)).toBeNull();
    });

    it("is sent as a meta field, on, written by a person and carrying no links", () => {
        expect(metaField(" strength ")).toBe("meta:strength");
        expect(customScope("t1", "strength", " Strength ", "store")).toStrictEqual({
            pageTypeId: "t1",
            field: "meta:strength",
            enabled: true,
            aiWrites: false,
            instruction: "",
            minLength: 0,
            maxLength: 0,
            rewrites: false,
            links: noLinks(),
            label: "Strength",
            provider: "store",
        });
        expect(noLinks().links).toBe(false);
    });
});
