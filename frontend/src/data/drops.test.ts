import { describe, expect, it, vi } from "vitest";

import { publishDrop, sheetIn, siteImportedOn, subscribeDrop } from "./drops.js";

describe("sheetIn", () => {
    it("takes the first sheet of a drop, whatever its case", () => {
        expect(sheetIn(["C:\sheets\pages.CSV"])).toBe("C:\sheets\pages.CSV");
        expect(sheetIn(["C:\a.png", "C:\b.xlsx", "C:\c.csv"])).toBe("C:\b.xlsx");
    });

    it("answers nothing for a drop this app cannot read", () => {
        expect(sheetIn(["C:\a.png", "C:\notes.txt"])).toBeNull();
        expect(sheetIn([])).toBeNull();
        expect(sheetIn(["   "])).toBeNull();
    });
});

describe("siteImportedOn", () => {
    it("names the site only while the import screen is open", () => {
        expect(siteImportedOn("/s/abc/import")).toBe("abc");
        expect(siteImportedOn("/s/abc/pages")).toBeNull();
        expect(siteImportedOn("/settings/models")).toBeNull();
        expect(siteImportedOn("/s/abc/import/extra")).toBeNull();
    });
});

describe("the drop stream", () => {
    it("hands every listener the paths and stops on unsubscribe", () => {
        const heard = vi.fn();
        const stop = subscribeDrop(heard);
        publishDrop(["C:\a.csv"]);
        expect(heard).toHaveBeenCalledWith(["C:\a.csv"]);
        stop();
        publishDrop(["C:\b.csv"]);
        expect(heard).toHaveBeenCalledTimes(1);
    });
});
