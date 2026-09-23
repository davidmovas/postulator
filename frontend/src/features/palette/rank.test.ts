import { describe, expect, it } from "vitest";

import { rankBy, scoreOf } from "./rank.js";

describe("scoreOf", () => {
    const cases: readonly { name: string; query: string; text: string; want: number }[] = [
        { name: "an empty query keeps everything", query: "", text: "anything", want: 3 },
        { name: "a prefix wins", query: "mug", text: "Mugs", want: 0 },
        { name: "case is ignored", query: "MUG", text: "mugs", want: 0 },
        { name: "a word boundary is second", query: "mug", text: "Ceramic Mugs", want: 1 },
        { name: "a path separator is a boundary", query: "mug", text: "/collections/mugs/", want: 1 },
        { name: "an inner match is third", query: "ug", text: "Mugs", want: 2 },
        { name: "no match is refused", query: "teapot", text: "Mugs", want: -1 },
    ];

    it.each(cases)("$name", ({ query, text, want }) => {
        expect(scoreOf(query, text)).toBe(want);
    });
});

describe("rankBy", () => {
    const names = ["Ceramic Mugs", "Mug 350ml", "Stoneware", "Smug Teapot", "Mugs"];
    const text = (held: string): string => held;

    it("orders prefix, boundary then inner, keeping the given order inside a band", () => {
        expect(rankBy("mug", names, text, 10)).toEqual(["Mug 350ml", "Mugs", "Ceramic Mugs", "Smug Teapot"]);
    });

    it("drops what does not match", () => {
        expect(rankBy("mug", names, text, 10)).not.toContain("Stoneware");
    });

    it("keeps the given order for an empty query", () => {
        expect(rankBy("", names, text, 3)).toEqual(["Ceramic Mugs", "Mug 350ml", "Stoneware"]);
    });

    it("honours the limit", () => {
        expect(rankBy("mug", names, text, 2)).toEqual(["Mug 350ml", "Mugs"]);
    });
});
