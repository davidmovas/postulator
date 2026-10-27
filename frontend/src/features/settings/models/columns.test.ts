import { describe, expect, it } from "vitest";

import { callLayout, modelLayout, purposeLayout } from "./columns.js";

const panelBesideTheDockAt960 = 960 - 68 - 392 - 32 - 2;

describe("the columns the spend tables show at a width", () => {
    it.each([
        { width: 0, columns: ["purpose", "calls", "failed", "spent", "share"] },
        { width: 544, columns: ["purpose", "calls", "failed", "spent", "share"] },
        { width: 543, columns: ["purpose", "calls", "spent", "share"] },
        { width: 460, columns: ["purpose", "calls", "spent", "share"] },
        { width: 459, columns: ["purpose", "calls", "spent"] },
        { width: 120, columns: ["purpose", "calls", "spent"] },
    ])("keeps the purposes that fit $width px", ({ width, columns }) => {
        expect(purposeLayout(width).columns).toStrictEqual(columns);
    });

    it.each([
        { width: 0, columns: ["model", "tier", "calls", "input", "cached", "output", "reasoning", "spent"] },
        { width: 764, columns: ["model", "tier", "calls", "input", "cached", "output", "reasoning", "spent"] },
        { width: 763, columns: ["model", "tier", "calls", "input", "output", "spent"] },
        { width: 596, columns: ["model", "tier", "calls", "input", "output", "spent"] },
        { width: 595, columns: ["model", "calls", "input", "output", "spent"] },
        { width: 504, columns: ["model", "calls", "input", "output", "spent"] },
        { width: 503, columns: ["model", "calls", "spent"] },
        { width: 200, columns: ["model", "calls", "spent"] },
    ])("keeps the model columns that fit $width px", ({ width, columns }) => {
        expect(modelLayout(width).columns).toStrictEqual(columns);
    });

    it.each([
        { width: 0, columns: ["when", "what", "model", "tier", "tokens", "spent"] },
        { width: 756, columns: ["when", "what", "model", "tier", "tokens", "spent"] },
        { width: 755, columns: ["when", "what", "model", "tokens", "spent"] },
        { width: 664, columns: ["when", "what", "model", "tokens", "spent"] },
        { width: 663, columns: ["when", "what", "model", "spent"] },
        { width: 508, columns: ["when", "what", "model", "spent"] },
        { width: 507, columns: ["what", "model", "spent"] },
        { width: 160, columns: ["what", "model", "spent"] },
    ])("keeps the call columns that fit $width px", ({ width, columns }) => {
        expect(callLayout(width).columns).toStrictEqual(columns);
    });

    it("never asks for more than the panel has beside the dock at 960 px", () => {
        for (const layout of [purposeLayout, modelLayout, callLayout]) {
            expect(layout(panelBesideTheDockAt960).least).toBeLessThanOrEqual(panelBesideTheDockAt960);
        }
    });

    it("lets the model name shrink, never the counts, once nothing else fits", () => {
        expect(modelLayout(300).template).toBe("minmax(0,1fr) 4rem 5rem");
        expect(callLayout(300).template).toBe("minmax(0,1.4fr) minmax(0,1fr) 4.5rem");
    });
});
