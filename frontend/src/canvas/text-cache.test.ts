import { describe, expect, it } from "vitest";

import { TextCache } from "./text-cache.js";

class FakeMeasurer {
    font = "";
    calls = 0;

    measureText(text: string): { width: number } {
        this.calls += 1;
        return { width: [...text].length * 7 };
    }
}

describe("TextCache", () => {
    it("measures once per font and text", () => {
        const measurer = new FakeMeasurer();
        const cache = new TextCache(measurer);
        expect(cache.width("12px sans", "hello")).toBe(35);
        expect(cache.width("12px sans", "hello")).toBe(35);
        expect(measurer.calls).toBe(1);
        expect(cache.width("14px sans", "hello")).toBe(35);
        expect(measurer.calls).toBe(2);
    });

    it("sets the font on the measurer before measuring", () => {
        const measurer = new FakeMeasurer();
        new TextCache(measurer).width("12px mono", "x");
        expect(measurer.font).toBe("12px mono");
    });

    it("returns a label that fits unchanged", () => {
        const cache = new TextCache(new FakeMeasurer());
        expect(cache.ellipsise("12px sans", "short", 100)).toBe("short");
    });

    it("ellipsises a long label to the widest prefix that fits", () => {
        const cache = new TextCache(new FakeMeasurer());
        const clipped = cache.ellipsise("12px sans", "a very long label", 50);
        expect(clipped).toBe("a very…");
        expect(cache.width("12px sans", clipped)).toBeLessThanOrEqual(50);
    });

    it("answers nothing when not even the ellipsis fits", () => {
        const cache = new TextCache(new FakeMeasurer());
        expect(cache.ellipsise("12px sans", "label", 5)).toBe("");
    });
});
