import { describe, expect, it } from "vitest";

import { neighbourOf, positionOf } from "./review-nav.js";

const items = [{ id: "a" }, { id: "b" }, { id: "c" }];

describe("neighbourOf", () => {
    it("walks the loaded items in both directions", () => {
        expect(neighbourOf(items, "a", 1)).toBe("b");
        expect(neighbourOf(items, "b", 1)).toBe("c");
        expect(neighbourOf(items, "c", -1)).toBe("b");
    });

    it("stops at the ends rather than wrapping, because more may not be loaded", () => {
        expect(neighbourOf(items, "a", -1)).toBeNull();
        expect(neighbourOf(items, "c", 1)).toBeNull();
    });

    it("has no neighbour for an item that is not in the loaded page", () => {
        expect(neighbourOf(items, "z", 1)).toBeNull();
        expect(neighbourOf([], "a", 1)).toBeNull();
    });
});

describe("positionOf", () => {
    it("counts from one for a human", () => {
        expect(positionOf(items, "a")).toBe(1);
        expect(positionOf(items, "c")).toBe(3);
    });

    it("is zero when the item is not loaded", () => {
        expect(positionOf(items, "z")).toBe(0);
    });
});
