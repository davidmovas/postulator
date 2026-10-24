import { describe, expect, it } from "vitest";

import type { PreviewPage } from "../../data/types.js";
import { choicesOf, chosenFor, needsUrl, noChoices, pickable, withChoice } from "./choices.js";

function row(part: Partial<PreviewPage>): PreviewPage {
    return { path: "/guides/", title: "Guides", keywords: null, wpType: "page", action: "ambiguous", ...part };
}

const candidates = [
    { pageId: "p-lab", path: "/lab/guides/", title: "Lab guides" },
    { pageId: "p-shop", path: "/shop/guides/", title: "Shop guides" },
];

describe("the pages a person picks for ambiguous rows", () => {
    it("starts with no pick and sends none", () => {
        expect(choicesOf(noChoices)).toStrictEqual([]);
    });

    it("keeps one pick per row, the last one winning, and sends them in row order", () => {
        let choices = withChoice(noChoices, "Groups#7", "p-lab");
        choices = withChoice(choices, "Groups#3", "p-shop");
        choices = withChoice(choices, "Groups#7", "p-shop");
        expect(choicesOf(choices)).toStrictEqual([
            { rowKey: "Groups#3", pageId: "p-shop" },
            { rowKey: "Groups#7", pageId: "p-shop" },
        ]);
    });

    it("leaves the picks it was given as they were", () => {
        const before = withChoice(noChoices, "a", "p1");
        withChoice(before, "b", "p2");
        expect(choicesOf(before)).toStrictEqual([{ rowKey: "a", pageId: "p1" }]);
    });

    it("reads which page a row was given", () => {
        const choices = withChoice(noChoices, "Groups#7", "p-lab");
        expect(chosenFor(choices, row({ rowKey: "Groups#7", candidates }))).toBe("p-lab");
        expect(chosenFor(choices, row({ rowKey: "Groups#8", candidates }))).toBeNull();
        expect(chosenFor(choices, row({ candidates }))).toBeNull();
    });
});

describe("which rows a person can pick a page for", () => {
    it("offers a pick on a row with a key and the pages it matches, ambiguous or already picked", () => {
        expect(pickable(row({ rowKey: "Groups#7", candidates }))).toBe(true);
        expect(pickable(row({ rowKey: "Groups#7", candidates, action: "update", matchedBy: "mapped" }))).toBe(true);
    });

    it("offers none on a row without a key, which needs a URL instead", () => {
        const keyless = row({ candidates });
        expect(pickable(keyless)).toBe(false);
        expect(needsUrl(keyless)).toBe(true);
        expect(needsUrl(row({ rowKey: "Groups#7", candidates }))).toBe(false);
        expect(needsUrl(row({ action: "update" }))).toBe(false);
    });

    it("offers none on a row that matched no page or one page alone", () => {
        expect(pickable(row({ rowKey: "Groups#7", action: "create" }))).toBe(false);
        expect(pickable(row({ rowKey: "Groups#7", action: "update", candidates: [] }))).toBe(false);
    });
});
