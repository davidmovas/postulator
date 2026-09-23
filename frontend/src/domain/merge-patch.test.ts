import { describe, expect, test } from "vitest";

import { mergePatch } from "./merge-patch.js";

describe("RFC 7396 merge patch", () => {
    test("answers undefined when nothing changed", () => {
        expect(mergePatch({ tone: "plain", length: { min: 600, max: 1200 } }, { tone: "plain", length: { min: 600, max: 1200 } })).toBeUndefined();
    });

    test("carries only the changed leaf", () => {
        expect(
            mergePatch({ tone: "plain", length: { min: 600, max: 1200 } }, { tone: "plain", length: { min: 800, max: 1200 } }),
        ).toEqual({ length: { min: 800 } });
    });

    test("replaces an array wholesale rather than merging it", () => {
        expect(mergePatch({ sections: ["intro", "body", "faq"] }, { sections: ["intro", "faq"] })).toEqual({
            sections: ["intro", "faq"],
        });
    });

    test("replaces an array even when only one element differs", () => {
        expect(mergePatch({ keywords: ["a", "b"] }, { keywords: ["a", "c"] })).toEqual({ keywords: ["a", "c"] });
    });

    test("removes a key with null", () => {
        expect(mergePatch({ tone: "plain", metaRules: { titlePattern: "%s" } }, { tone: "plain" })).toEqual({
            metaRules: null,
        });
    });

    test("removes a leaf inside a nested object", () => {
        expect(
            mergePatch({ images: { featured: true, source: "ai" } }, { images: { featured: true } }),
        ).toEqual({ images: { source: null } });
    });

    test("adds a new key without disturbing the rest", () => {
        expect(mergePatch({ tone: "plain" }, { tone: "plain", images: { featured: true } })).toEqual({
            images: { featured: true },
        });
    });

    test("drops unrepresentable nulls when the base is not an object", () => {
        expect(mergePatch("plain", { featured: true, source: null })).toEqual({ featured: true });
    });

    test("replaces a primitive with the new value", () => {
        expect(mergePatch({ tone: "plain" }, { tone: "expert" })).toEqual({ tone: "expert" });
    });

    test("treats an explicit null value as a removal", () => {
        expect(mergePatch({ templateId: "t1" }, { templateId: null })).toEqual({ templateId: null });
    });
});
