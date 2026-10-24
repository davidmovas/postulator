import { describe, expect, it } from "vitest";

import { blockExcerpt, editPlanView, fieldText } from "./diff.js";
import { publishView, revertView } from "./written.js";

const excerptChange = { key: "excerpt", before: "Old", beforeHash: "b1", after: "New", afterHash: "a1" };

const clientBlock = { kind: "paragraph", hash: "h1", raw: "<p>The client's own words.</p>", owner: "client", change: "kept" };

const ourBlock = { kind: "heading", hash: "h2", raw: "<h2>Dosage</h2>", owner: "ours", change: "added" };

describe("the diff a publish result carries", () => {
    it("reads each field's value and hash before and after the write", () => {
        const view = publishView({ wpId: 4, fields: [excerptChange, { key: "" }, 7] });

        expect(view?.fields).toStrictEqual([excerptChange]);
    });

    it("reads the body's blocks in the order the site stored them", () => {
        const view = publishView({ wpId: 4, blocks: [clientBlock, "x", ourBlock] });

        expect(view?.blocks).toStrictEqual([clientBlock, ourBlock]);
    });

    it("names the fields a person left unticked, and none when the result says nothing", () => {
        expect(publishView({ wpId: 4, declined: ["title", "seo_title"] })?.declined).toStrictEqual(["title", "seo_title"]);
        expect(publishView({ wpId: 4 })?.declined).toStrictEqual([]);
        expect(publishView({ wpId: 4 })?.fields).toStrictEqual([]);
        expect(publishView({ wpId: 4 })?.blocks).toStrictEqual([]);
    });
});

describe("the review a draft run holds", () => {
    it("reads every page with its field changes, its blocks and the key its body is written under", () => {
        const view = editPlanView({
            hash: "abc",
            pages: [
                { pageId: "p1", path: "/a/", body: "product_description", fields: [excerptChange], blocks: [ourBlock], wpId: 12 },
                { pageId: "p2", path: "/b/", fields: [], blocks: [], wpId: 13 },
                "x",
            ],
        });

        expect(view?.hash).toBe("abc");
        expect(view?.pages).toStrictEqual([
            { pageId: "p1", path: "/a/", body: "product_description", fields: [excerptChange], blocks: [ourBlock], wpId: 12 },
            { pageId: "p2", path: "/b/", body: "", fields: [], blocks: [], wpId: 13 },
        ]);
    });

    it("answers null for anything that is not a review", () => {
        expect(editPlanView("nope")).toBeNull();
        expect(editPlanView({ hash: "abc" })?.pages).toStrictEqual([]);
    });
});

describe("what a revert did field by field", () => {
    it("reads each field's outcome and its reason", () => {
        const view = revertView({
            path: "/a/",
            outcome: "restored",
            fields: [
                { key: "excerpt", outcome: "restored" },
                { key: "seo_title", outcome: "kept", detail: "it changed on the site since the run wrote it, so the edit is kept" },
                { outcome: "kept" },
            ],
        });

        expect(view?.fields).toStrictEqual([
            { key: "excerpt", outcome: "restored", detail: "" },
            { key: "seo_title", outcome: "kept", detail: "it changed on the site since the run wrote it, so the edit is kept" },
        ]);
        expect(revertView({ path: "/a/" })?.fields).toStrictEqual([]);
    });
});

describe("a block and a field shown as text", () => {
    it("shows a block's markup as its text, cut short", () => {
        expect(blockExcerpt("<p>The client&#39;s <b>own</b> words.</p>")).toBe("The client's own words.");
        expect(blockExcerpt(`<p>${"word ".repeat(80)}</p>`).length).toBeLessThanOrEqual(160);
        expect(blockExcerpt(`<p>${"word ".repeat(80)}</p>`).endsWith("…")).toBe(true);
        expect(blockExcerpt("[entity_products]")).toBe("[entity_products]");
    });

    it("shows an HTML field as its text and a plain field as it is", () => {
        expect(fieldText("product_short_description", "<p>Short &amp; sweet</p>")).toBe("Short & sweet");
        expect(fieldText("variation_description:41", "<p>Blue</p>")).toBe("Blue");
        expect(fieldText("seo_title", "Mugs <b>& cups</b>")).toBe("Mugs <b>& cups</b>");
    });
});
