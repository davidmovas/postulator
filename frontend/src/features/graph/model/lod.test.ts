import { describe, expect, it } from "vitest";

import { detailOf, labelled } from "./lod.js";

describe("detailOf", () => {
    it("steps down with the zoom", () => {
        expect(detailOf(1)).toBe("full");
        expect(detailOf(0.6)).toBe("full");
        expect(detailOf(0.59)).toBe("branches");
        expect(detailOf(0.35)).toBe("branches");
        expect(detailOf(0.2)).toBe("roots");
    });
});

describe("labelled", () => {
    it("labels everything at full detail", () => {
        expect(labelled("full", 4, 0, false)).toBe(true);
    });

    it("labels only branches and the selection in between", () => {
        expect(labelled("branches", 3, 2, false)).toBe(true);
        expect(labelled("branches", 3, 0, false)).toBe(false);
        expect(labelled("branches", 3, 0, true)).toBe(true);
    });

    it("labels only the top two depths far out", () => {
        expect(labelled("roots", 1, 0, false)).toBe(true);
        expect(labelled("roots", 2, 5, false)).toBe(false);
        expect(labelled("roots", 2, 5, true)).toBe(true);
    });
});
