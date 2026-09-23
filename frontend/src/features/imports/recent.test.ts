import { describe, expect, it } from "vitest";

import { fileName, forget, recentCap, remember } from "./recent.js";
import type { RecentFile } from "./recent.js";

function entry(path: string, rows = 10): RecentFile {
    return { path, rows, at: "2026-09-21T08:00:00Z" };
}

describe("fileName", () => {
    it.each([
        ["C:\\sheets\\mugs.csv", "mugs.csv"],
        ["C:/sheets/mugs.csv", "mugs.csv"],
        ["mugs.csv", "mugs.csv"],
        ["C:\\sheets\\", "C:\\sheets\\"],
        ["", ""],
    ])("reads %s as %s", (path, want) => {
        expect(fileName(path)).toBe(want);
    });
});

describe("remember", () => {
    it("puts the newest file first", () => {
        expect(remember([entry("a")], entry("b")).map((row) => row.path)).toEqual(["b", "a"]);
    });

    it("keeps one row per file", () => {
        const held = remember([entry("a", 1), entry("b")], entry("a", 99));
        expect(held.map((row) => row.path)).toEqual(["a", "b"]);
        expect(held[0]?.rows).toBe(99);
    });

    it("holds five files at most", () => {
        let held: RecentFile[] = [];
        for (const path of ["a", "b", "c", "d", "e", "f", "g"]) {
            held = remember(held, entry(path));
        }
        expect(held).toHaveLength(recentCap);
        expect(held.map((row) => row.path)).toEqual(["g", "f", "e", "d", "c"]);
    });

    it("has nothing to remember about no file", () => {
        expect(remember([entry("a")], entry(""))).toEqual([entry("a")]);
    });
});

describe("forget", () => {
    it("drops the file it names and nothing else", () => {
        expect(forget([entry("a"), entry("b")], "a").map((row) => row.path)).toEqual(["b"]);
        expect(forget([entry("a")], "z")).toHaveLength(1);
    });
});
