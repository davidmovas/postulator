import { describe, expect, it } from "vitest";

import { laidOut, shown, widestFitting } from "./collapse.js";

type Column = "name" | "count" | "when";

const tracks = {
    name: { track: "minmax(100px,1fr)", least: 100 },
    count: { track: "60px", least: 60 },
    when: { track: "80px", least: 80 },
};

const shrunk = { ...tracks, name: { track: "minmax(0,1fr)", least: 0 } };

const every = laidOut<Column>(["name", "count", "when"], tracks, 24);
const lean = laidOut<Column>(["name", "count"], tracks, 24);
const narrowest = laidOut<Column>(["name", "count"], shrunk, 24);

describe("collapsing columns by measured width", () => {
    it("joins the tracks and adds the row's chrome and a gap between each pair", () => {
        expect(every.template).toBe("minmax(100px,1fr) 60px 80px");
        expect(every.least).toBe(100 + 60 + 80 + 24 + 24);
        expect(lean.least).toBe(100 + 60 + 24 + 12);
        expect(narrowest.least).toBe(60 + 24 + 12);
    });

    it.each([
        { width: 0, columns: ["name", "count", "when"] },
        { width: 2000, columns: ["name", "count", "when"] },
        { width: 288, columns: ["name", "count", "when"] },
        { width: 287, columns: ["name", "count"] },
        { width: 196, columns: ["name", "count"] },
        { width: 195, columns: ["name", "count"] },
        { width: 40, columns: ["name", "count"] },
    ])("keeps the widest layout that fits $width px", ({ width, columns }) => {
        expect(widestFitting(width, [every, lean], narrowest).columns).toStrictEqual(columns);
    });

    it("falls to the shrinking layout once nothing else fits", () => {
        expect(widestFitting(195, [every, lean], narrowest)).toBe(narrowest);
        expect(widestFitting(196, [every, lean], narrowest)).toBe(lean);
    });

    it("says whether a column is shown", () => {
        expect(shown(lean, "count")).toBe(true);
        expect(shown(lean, "when")).toBe(false);
    });
});
