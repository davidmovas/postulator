import { describe, expect, it } from "vitest";

import { keywordList, keywordsLine, keywordTexts, mainKeyword, sameKeywords, volumeLabel } from "./keywords.js";

describe("keywordList", () => {
    it("orders by volume and keeps the unmeasured in the order given", () => {
        expect(
            keywordList([
                { text: "bpc-157" },
                { text: "bpc 157 dosage", volume: 1900 },
                { text: "what is bpc" },
                { text: "bpc 157", volume: 12000 },
            ]),
        ).toStrictEqual([
            { text: "bpc 157", volume: 12000 },
            { text: "bpc 157 dosage", volume: 1900 },
            { text: "bpc-157" },
            { text: "what is bpc" },
        ]);
    });

    it("trims, drops a blank phrase and keeps a repeat once with the first known volume", () => {
        expect(
            keywordList([{ text: " BPC 157 " }, { text: "  " }, { text: "bpc 157", volume: 800 }, { text: "tb 500", volume: -2 }]),
        ).toStrictEqual([{ text: "BPC 157", volume: 800 }, { text: "tb 500" }]);
    });

    it("reads no list as an empty one", () => {
        expect(keywordList(null)).toStrictEqual([]);
        expect(keywordList(undefined)).toStrictEqual([]);
    });
});

describe("reading a list", () => {
    const list = [{ text: "bpc 157", volume: 12000 }, { text: "bpc-157" }];

    it("names the main keyword and the phrases in order", () => {
        expect(mainKeyword(list)).toBe("bpc 157");
        expect(mainKeyword([])).toBe("");
        expect(keywordTexts(list)).toStrictEqual(["bpc 157", "bpc-157"]);
    });

    it("compares phrases, volumes and order", () => {
        expect(sameKeywords(list, [{ text: "bpc 157", volume: 12000 }, { text: "bpc-157", volume: null }])).toBe(true);
        expect(sameKeywords(list, [{ text: "bpc 157", volume: 9000 }, { text: "bpc-157" }])).toBe(false);
        expect(sameKeywords(list, [{ text: "bpc 157" }, { text: "bpc-157" }])).toBe(false);
        expect(sameKeywords(list, list.slice(0, 1))).toBe(false);
    });

    it("reads a list on one line, as the sheet writes it", () => {
        expect(keywordsLine([{ text: "bpc-157" }, { text: "bpc 157", volume: 12000 }])).toBe("bpc 157 (12,000), bpc-157");
        expect(keywordsLine(null)).toBe("");
    });

    it("writes a volume for a reader", () => {
        expect(volumeLabel({ text: "bpc 157", volume: 12000 })).toBe("12,000");
        expect(volumeLabel({ text: "rare", volume: 0 })).toBe("0");
        expect(volumeLabel({ text: "bpc-157" })).toBe("");
    });
});
