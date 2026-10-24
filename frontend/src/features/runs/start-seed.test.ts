import { describe, expect, it } from "vitest";

import type { StartForm } from "./start-model.js";
import { startRequestOf } from "./start-model.js";
import { seedOf, seededFields, seededRewrite } from "./start-seed.js";

const form: StartForm = {
    siteId: "s1",
    pageIds: ["p1"],
    publishMode: "draft",
    kind: "generate",
    templateId: "",
    cap: "5.00",
    tokenCap: "0",
    rewrite: [],
    fields: [],
};

describe("the seed a finding's run option hands the drawer", () => {
    it("takes the pages, the field scope and the pages to rewrite", () => {
        const seed = seedOf({ siteId: "s1", pageIds: ["p1"], kind: "", budget: { maxUsd: 0, maxTokens: 0 }, fields: ["seo_description"], rewrite: ["p1"] });
        expect(seed).toStrictEqual({ pageIds: ["p1"], kind: "generate", fields: ["seo_description"], rewrite: ["p1"] });
    });

    it("keeps the kind it names and reads a missing list as none", () => {
        const seed = seedOf({ siteId: "s1", pageIds: ["p1"], kind: "relink", budget: { maxUsd: 0, maxTokens: 0 }, fields: null });
        expect(seed).toStrictEqual({ pageIds: ["p1"], kind: "relink", fields: [], rewrite: [] });
    });

    it("scopes the fields and the rewrite to the kind it was given", () => {
        const seed = { pageIds: ["p1"], kind: "generate", fields: ["title"], rewrite: ["p1"] };
        expect(seededFields(seed, "generate")).toStrictEqual(["title"]);
        expect(seededFields(seed, "relink")).toStrictEqual([]);
        expect(seededFields(null, "generate")).toStrictEqual([]);
        expect(seededRewrite(seed, "generate")).toStrictEqual(["p1"]);
        expect(seededRewrite(seed, "audit")).toStrictEqual([]);
    });
});

describe("the request the drawer sends", () => {
    it("sends the pages, the mode, the kind and the caps, and no template when each page resolves its own", () => {
        expect(startRequestOf(form)).toStrictEqual({
            siteId: "s1",
            pageIds: ["p1"],
            publishMode: "draft",
            kind: "generate",
            budget: { maxUsd: 5, maxTokens: 0 },
        });
    });

    it("sends a picked template, the pages to rewrite and the field scope", () => {
        expect(startRequestOf({ ...form, templateId: "t1", rewrite: ["p1"], fields: ["seo_description"], tokenCap: "9000" })).toStrictEqual({
            siteId: "s1",
            pageIds: ["p1"],
            publishMode: "draft",
            kind: "generate",
            budget: { maxUsd: 5, maxTokens: 9000 },
            templateId: "t1",
            rewrite: ["p1"],
            fields: ["seo_description"],
        });
    });

    it("reads a cap it cannot read as no cap", () => {
        expect(startRequestOf({ ...form, cap: "abc", tokenCap: "-3" }).budget).toStrictEqual({ maxUsd: 0, maxTokens: 0 });
    });
});
