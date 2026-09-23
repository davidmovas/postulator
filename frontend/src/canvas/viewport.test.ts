import { describe, expect, it } from "vitest";

import {
    centerOn,
    fit,
    identity,
    maxZoom,
    minZoom,
    panBy,
    toScreen,
    toWorld,
    visibleWorld,
    zoomAt,
} from "./viewport.js";

describe("toScreen and toWorld", () => {
    it("round-trips a point through the transform", () => {
        const view = { x: 100, y: 50, k: 2 };
        const world = { x: 10, y: -5 };
        const screen = toScreen(view, world);
        expect(screen).toStrictEqual({ x: 120, y: 40 });
        expect(toWorld(view, screen)).toStrictEqual(world);
    });
});

describe("zoomAt", () => {
    it("keeps the world point under the anchor fixed", () => {
        const view = { x: 30, y: 30, k: 1 };
        const anchor = { x: 200, y: 120 };
        const before = toWorld(view, anchor);
        const zoomed = zoomAt(view, anchor, 1.5);
        expect(zoomed.k).toBeCloseTo(1.5);
        const after = toWorld(zoomed, anchor);
        expect(after.x).toBeCloseTo(before.x);
        expect(after.y).toBeCloseTo(before.y);
    });

    it("clamps the scale to the allowed range", () => {
        expect(zoomAt(identity, { x: 0, y: 0 }, 100).k).toBe(maxZoom);
        expect(zoomAt(identity, { x: 0, y: 0 }, 0.0001).k).toBe(minZoom);
    });

    it("returns the same view when the scale cannot change", () => {
        const view = { x: 5, y: 5, k: maxZoom };
        expect(zoomAt(view, { x: 40, y: 40 }, 2)).toStrictEqual(view);
    });
});

describe("panBy", () => {
    it("shifts the origin in screen pixels", () => {
        expect(panBy({ x: 1, y: 2, k: 2 }, 10, -5)).toStrictEqual({ x: 11, y: -3, k: 2 });
    });
});

describe("fit", () => {
    it("scales the bounds into the size with padding and centres them", () => {
        const view = fit({ x: 0, y: 0, width: 400, height: 100 }, { width: 220, height: 220 }, 10);
        expect(view.k).toBeCloseTo(0.5);
        const centre = toScreen(view, { x: 200, y: 50 });
        expect(centre.x).toBeCloseTo(110);
        expect(centre.y).toBeCloseTo(110);
    });

    it("never magnifies past one to one", () => {
        const view = fit({ x: 0, y: 0, width: 50, height: 50 }, { width: 1000, height: 1000 }, 0);
        expect(view.k).toBe(1);
    });

    it("centres empty bounds at one to one", () => {
        const view = fit({ x: 0, y: 0, width: 0, height: 0 }, { width: 100, height: 100 }, 0);
        expect(view.k).toBe(1);
        expect(toScreen(view, { x: 0, y: 0 })).toStrictEqual({ x: 50, y: 50 });
    });

    it("never scales below the minimum", () => {
        const view = fit({ x: 0, y: 0, width: 100000, height: 10 }, { width: 100, height: 100 }, 0);
        expect(view.k).toBe(minZoom);
    });
});

describe("visibleWorld", () => {
    it("is the size mapped into world space", () => {
        expect(visibleWorld({ x: 100, y: 0, k: 2 }, { width: 300, height: 200 })).toStrictEqual({
            x: -50,
            y: 0,
            width: 150,
            height: 100,
        });
    });
});

describe("centerOn", () => {
    it("puts the world point at the middle of the size", () => {
        const view = centerOn({ x: 0, y: 0, k: 2 }, { x: 10, y: 20 }, { width: 100, height: 60 });
        expect(toScreen(view, { x: 10, y: 20 })).toStrictEqual({ x: 50, y: 30 });
    });
});
