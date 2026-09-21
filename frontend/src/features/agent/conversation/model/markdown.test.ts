import { describe, expect, it } from "vitest";

import { blocksOf, spansOf } from "./markdown.js";

describe("spansOf", () => {
    it("keeps plain prose whole", () => {
        expect(spansOf("a clear sentence")).toEqual([{ kind: "text", text: "a clear sentence" }]);
    });

    it("reads bold, italic and code", () => {
        expect(spansOf("**Ingredient Guide** is *one* `page`")).toEqual([
            { kind: "strong", text: "Ingredient Guide" },
            { kind: "text", text: " is " },
            { kind: "em", text: "one" },
            { kind: "text", text: " " },
            { kind: "code", text: "page" },
        ]);
    });

    it("leaves an unfinished marker as the text it is, which a stream arrives as", () => {
        expect(spansOf("half of **a bold")).toEqual([{ kind: "text", text: "half of **a bold" }]);
    });

    it("never mistakes a snake case identifier for emphasis", () => {
        expect(spansOf("call graph_list_entities first")).toEqual([
            { kind: "text", text: "call graph_list_entities first" },
        ]);
    });

    it("reads a link and falls back to its address when it has no label", () => {
        expect(spansOf("[NIH](https://nih.gov) and [](https://efsa.eu)")).toEqual([
            { kind: "link", text: "NIH", href: "https://nih.gov" },
            { kind: "text", text: " and " },
            { kind: "link", text: "https://efsa.eu", href: "https://efsa.eu" },
        ]);
    });
});

describe("blocksOf", () => {
    it("reads headings at their level", () => {
        expect(blocksOf("### Proposed template")).toEqual([
            { kind: "heading", level: 3, spans: [{ kind: "text", text: "Proposed template" }] },
        ]);
    });

    it("joins the lines of one paragraph and splits on a blank line", () => {
        const blocks = blocksOf("first line\nsecond line\n\nanother paragraph");
        expect(blocks).toHaveLength(2);
        expect(blocks[0]).toEqual({ kind: "paragraph", spans: [{ kind: "text", text: "first line second line" }] });
    });

    it("keeps an ordered list together and remembers its numbers", () => {
        const blocks = blocksOf("1. What it is\n2. How it works");
        expect(blocks).toHaveLength(1);
        expect(blocks[0]).toMatchObject({
            kind: "list",
            ordered: true,
            items: [{ marker: "1.", depth: 0 }, { marker: "2.", depth: 0 }],
        });
    });

    it("indents a nested bullet under its parent", () => {
        const blocks = blocksOf("- top\n    - under it");
        expect(blocks[0]).toMatchObject({ kind: "list", ordered: false, items: [{ depth: 0 }, { depth: 2 }] });
    });

    it("starts a new list when the kind of marker changes", () => {
        const blocks = blocksOf("1. ordered\n- bulleted");
        expect(blocks.map((block) => block.kind)).toEqual(["list", "list"]);
    });

    it("keeps a fenced block verbatim, markers and all", () => {
        expect(blocksOf("```json\n{\"a\": **1**}\n```")).toEqual([
            { kind: "code", language: "json", text: '{"a": **1**}' },
        ]);
    });

    it("reads a table with its heading row", () => {
        const blocks = blocksOf("| Form | Best for |\n| --- | --- |\n| glycinate | sleep |");
        expect(blocks[0]).toMatchObject({ kind: "table" });
        const table = blocks[0] as Extract<(typeof blocks)[number], { kind: "table" }>;
        expect(table.head).toHaveLength(2);
        expect(table.rows).toHaveLength(1);
        expect(table.rows[0][0]).toEqual([{ kind: "text", text: "glycinate" }]);
    });

    it("reads a rule and a quote", () => {
        expect(blocksOf("---").map((block) => block.kind)).toEqual(["rule"]);
        expect(blocksOf("> take care").map((block) => block.kind)).toEqual(["quote"]);
    });

    it("answers nothing for nothing", () => {
        expect(blocksOf("")).toEqual([]);
        expect(blocksOf("   \n\n  ")).toEqual([]);
    });
});
