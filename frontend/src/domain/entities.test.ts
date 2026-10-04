import { describe, expect, it } from "vitest";

import type { Entity } from "../data/types.js";
import { entityLabels, trailOf } from "./entities.js";

function entity(id: string, name: string, scopeEntityId: string | null = null): Entity {
    return {
        id,
        siteId: "site",
        name,
        kind: "topic",
        intent: "",
        keywords: [],
        anchors: [],
        scopeEntityId,
        categories: [],
        canonicalPageId: null,
        score: 0,
        source: "user",
        createdAt: null,
        updatedAt: null,
    };
}

describe("trailOf", () => {
    it.each<[string[], string]>([
        [["BPC-157", "Liquid"], "BPC-157 › Liquid"],
        [["", "Liquid"], "Liquid"],
        [["Peptides"], "Peptides"],
        [[], ""],
    ])("joins %j as %s", (names, want) => {
        expect(trailOf(names)).toBe(want);
    });
});

describe("entityLabels", () => {
    it.each<[string, Entity[], Record<string, string>]>([
        [
            "a name used once is its own label",
            [entity("bpc", "BPC-157"), entity("powder", "Powder", "bpc")],
            { bpc: "BPC-157", powder: "Powder" },
        ],
        [
            "a shared name shows the parent it sits under, whatever its case",
            [entity("bpc", "BPC-157"), entity("tb", "TB-500"), entity("bpc-liquid", "Liquid", "bpc"), entity("tb-liquid", "liquid", "tb")],
            { "bpc-liquid": "BPC-157 › Liquid", "tb-liquid": "TB-500 › liquid" },
        ],
        [
            "a shared name at the top has no parent to show",
            [entity("top", "Liquid"), entity("tb", "TB-500"), entity("tb-liquid", "Liquid", "tb")],
            { top: "Liquid", "tb-liquid": "TB-500 › Liquid" },
        ],
        [
            "a parent whose name is shared too shows its own parent",
            [
                entity("bpc", "BPC-157"),
                entity("tb", "TB-500"),
                entity("bpc-generic", "Generic", "bpc"),
                entity("tb-generic", "Generic", "tb"),
                entity("bpc-liquid", "Liquid", "bpc-generic"),
                entity("tb-liquid", "Liquid", "tb-generic"),
            ],
            { "bpc-liquid": "BPC-157 › Generic › Liquid", "tb-liquid": "TB-500 › Generic › Liquid" },
        ],
        [
            "a parent that is not loaded leaves the name alone",
            [entity("one", "Liquid", "elsewhere"), entity("two", "Liquid")],
            { one: "Liquid", two: "Liquid" },
        ],
    ])("%s", (_, entities, want) => {
        const labels = entityLabels(entities);
        for (const [id, label] of Object.entries(want)) {
            expect(labels.get(id)).toBe(label);
        }
    });
});
