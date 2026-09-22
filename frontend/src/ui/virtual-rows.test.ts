import { describe, expect, it } from "vitest";

import { nearingEnd } from "./virtual-rows.js";

describe("nearingEnd", () => {
    it("asks for more once the last rendered row is within reach of the end", () => {
        expect(nearingEnd(91, 100, 8)).toBe(false);
        expect(nearingEnd(92, 100, 8)).toBe(true);
        expect(nearingEnd(99, 100, 8)).toBe(true);
    });

    it("asks for more when the whole list is on screen", () => {
        expect(nearingEnd(4, 5, 8)).toBe(true);
    });

    it("asks for nothing while there is nothing", () => {
        expect(nearingEnd(0, 0, 8)).toBe(false);
        expect(nearingEnd(-1, 10, 8)).toBe(false);
    });
});
