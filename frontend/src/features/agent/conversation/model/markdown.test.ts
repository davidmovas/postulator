import { describe, expect, it } from "vitest";

import type { Block, Span } from "./markdown.js";
import { blocksOf, spansOf } from "./markdown.js";

function text(text: string): Span {
    return { kind: "text", text };
}

function flat(spans: readonly Span[]): string {
    return spans
        .map((span) => {
            switch (span.kind) {
                case "text":
                case "code":
                    return span.text;
                case "break":
                    return "\n";
                default:
                    return flat(span.spans);
            }
        })
        .join("");
}

function kinds(spans: readonly Span[]): string[] {
    return spans.map((span) => span.kind);
}

describe("spansOf", () => {
    it("keeps plain prose whole", () => {
        expect(spansOf("a clear sentence")).toEqual([text("a clear sentence")]);
    });

    it("reads bold, italic and code", () => {
        expect(spansOf("**Ingredient Guide** is *one* `page`")).toEqual([
            { kind: "strong", spans: [text("Ingredient Guide")] },
            text(" is "),
            { kind: "em", spans: [text("one")] },
            text(" "),
            { kind: "code", text: "page" },
        ]);
    });

    it("reads underscore emphasis the way a model writes it", () => {
        expect(kinds(spansOf("__bold__ and _italic_"))).toEqual(["strong", "text", "em"]);
    });

    it("reads a marker inside a marker", () => {
        const spans = spansOf("**bold with `code` and *more***");
        expect(spans).toHaveLength(1);
        expect(spans[0].kind).toBe("strong");
        expect(kinds((spans[0] as Extract<Span, { kind: "strong" }>).spans)).toEqual([
            "text",
            "code",
            "text",
            "em",
        ]);
    });

    it("reads bold italic written with three markers", () => {
        const spans = spansOf("***both***");
        expect(spans[0].kind).toBe("strong");
        expect(flat(spans)).toBe("both");
    });

    it("reads strikethrough", () => {
        expect(kinds(spansOf("~~gone~~"))).toEqual(["strike"]);
    });

    it("honours a backslash, so an escaped marker stays punctuation", () => {
        expect(spansOf("\\*not italic\\*")).toEqual([text("*not italic*")]);
    });

    it("leaves an unfinished marker as the text it is, which a stream arrives as", () => {
        expect(spansOf("half of **a bold")).toEqual([text("half of **a bold")]);
        expect(spansOf("half of `a code")).toEqual([text("half of `a code")]);
    });

    it("never mistakes a snake case identifier for emphasis", () => {
        expect(spansOf("call graph_list_entities first")).toEqual([text("call graph_list_entities first")]);
        expect(spansOf("max_density and min_words")).toEqual([text("max_density and min_words")]);
    });

    it("reads a link and falls back to its address when it has no label", () => {
        expect(spansOf("[NIH](https://nih.gov) and [](https://efsa.eu)")).toEqual([
            { kind: "link", href: "https://nih.gov", spans: [text("NIH")] },
            text(" and "),
            { kind: "link", href: "https://efsa.eu", spans: [text("https://efsa.eu")] },
        ]);
    });

    it("reads a link whose label carries a marker", () => {
        const spans = spansOf("[**NIH**](https://nih.gov)");
        expect(spans[0].kind).toBe("link");
        expect(kinds((spans[0] as Extract<Span, { kind: "link" }>).spans)).toEqual(["strong"]);
    });

    it("keeps a link title out of the address", () => {
        expect(spansOf('[NIH](https://nih.gov "the source")')).toEqual([
            { kind: "link", href: "https://nih.gov", spans: [text("NIH")] },
        ]);
    });

    it("reads a bare address and an angled one, and leaves the sentence punctuation alone", () => {
        expect(spansOf("see https://nih.gov.")).toEqual([
            text("see "),
            { kind: "link", href: "https://nih.gov", spans: [text("https://nih.gov")] },
            text("."),
        ]);
        expect(spansOf("<https://nih.gov>")).toEqual([
            { kind: "link", href: "https://nih.gov", spans: [text("https://nih.gov")] },
        ]);
    });

    it("renders an image as the words it stands for", () => {
        expect(spansOf("![a magnesium tablet](https://example.com/a.png)")).toEqual([text("a magnesium tablet")]);
    });

    it("reads inline code that carries a backtick", () => {
        expect(spansOf("``a `b` c``")).toEqual([{ kind: "code", text: "a `b` c" }]);
    });
});

describe("blocksOf", () => {
    it("reads headings at their level", () => {
        expect(blocksOf("### Proposed template")).toEqual([
            { kind: "heading", level: 3, spans: [text("Proposed template")] },
        ]);
    });

    it("reads a heading written with a rule under it", () => {
        expect(blocksOf("Ingredient Guide\n====")).toEqual([
            { kind: "heading", level: 1, spans: [text("Ingredient Guide")] },
        ]);
    });

    it("joins the lines of one paragraph and splits on a blank line", () => {
        const blocks = blocksOf("first line\nsecond line\n\nanother paragraph");
        expect(blocks).toHaveLength(2);
        expect(blocks[0]).toEqual({ kind: "paragraph", spans: [text("first line second line")] });
    });

    it("keeps a hard break the writer asked for", () => {
        const blocks = blocksOf("Bergstrasse 12  \n8001 Zurich");
        expect(kinds((blocks[0] as Extract<Block, { kind: "paragraph" }>).spans)).toEqual([
            "text",
            "break",
            "text",
        ]);
    });

    it("keeps an ordered list together and remembers where it starts", () => {
        const blocks = blocksOf("3. What it is\n4. How it works");
        expect(blocks).toHaveLength(1);
        expect(blocks[0]).toMatchObject({ kind: "list", ordered: true, start: 3 });
        const list = blocks[0] as Extract<Block, { kind: "list" }>;
        expect(list.items.map((item) => flat(item.spans))).toEqual(["What it is", "How it works"]);
    });

    it("hangs a nested list inside the item it belongs to", () => {
        const blocks = blocksOf("- top\n    - under it\n    - beside it\n- second");
        expect(blocks).toHaveLength(1);

        const list = blocks[0] as Extract<Block, { kind: "list" }>;
        expect(list.items).toHaveLength(2);
        expect(list.items[0].blocks).toHaveLength(1);
        expect(list.items[0].blocks[0]).toMatchObject({ kind: "list", ordered: false });
        expect(flat(list.items[1].spans)).toBe("second");
    });

    it("keeps a bullet nested under a numbered item in that item", () => {
        const blocks = blocksOf("1. Vitamins\n   - Vitamin D\n2. Minerals");
        expect(blocks).toHaveLength(1);

        const list = blocks[0] as Extract<Block, { kind: "list" }>;
        expect(list.ordered).toBe(true);
        expect(list.items).toHaveLength(2);
        expect(list.items[0].blocks[0]).toMatchObject({ kind: "list", ordered: false });
    });

    it("starts a new list when the kind of marker changes at the same level", () => {
        expect(blocksOf("1. ordered\n- bulleted").map((block) => block.kind)).toEqual(["list", "list"]);
    });

    it("keeps a fenced block verbatim, markers and all", () => {
        expect(blocksOf('```json\n{"a": **1**}\n```')).toEqual([
            { kind: "code", language: "json", text: '{"a": **1**}' },
        ]);
    });

    it("reads a fence whose info string carries more than the language", () => {
        expect(blocksOf('```js title="spec.js"\nconst a = 1;\n```')).toEqual([
            { kind: "code", language: "js", text: "const a = 1;" },
        ]);
    });

    it("keeps a fence that has not closed yet as code, because a stream is still arriving", () => {
        expect(blocksOf("```json\n{")).toEqual([{ kind: "code", language: "json", text: "{" }]);
    });

    it("reads a table with its heading row and how each column is set", () => {
        const blocks = blocksOf("| Form | Dose |\n| :--- | ---: |\n| glycinate | 200 mg |");
        const table = blocks[0] as Extract<Block, { kind: "table" }>;
        expect(table.align).toEqual(["left", "right"]);
        expect(table.head).toHaveLength(2);
        expect(table.rows).toHaveLength(1);
        expect(flat(table.rows[0][1])).toBe("200 mg");
    });

    it("keeps a pipe the writer escaped inside its cell", () => {
        const blocks = blocksOf("| a | b |\n| --- | --- |\n| one \\| two | three |");
        const table = blocks[0] as Extract<Block, { kind: "table" }>;
        expect(flat(table.rows[0][0])).toBe("one | two");
    });

    it("reads a rule", () => {
        expect(blocksOf("---").map((block) => block.kind)).toEqual(["rule"]);
        expect(blocksOf("***").map((block) => block.kind)).toEqual(["rule"]);
    });

    it("keeps a quote of several lines in one block", () => {
        const blocks = blocksOf("> take care\n> and read the label\n\nafter");
        expect(blocks.map((block) => block.kind)).toEqual(["quote", "paragraph"]);

        const quoted = blocks[0] as Extract<Block, { kind: "quote" }>;
        expect(quoted.blocks).toHaveLength(1);
        expect(flat((quoted.blocks[0] as Extract<Block, { kind: "paragraph" }>).spans)).toBe(
            "take care and read the label",
        );
    });

    it("reads a list inside a quote", () => {
        const blocks = blocksOf("> - one\n> - two");
        const quoted = blocks[0] as Extract<Block, { kind: "quote" }>;
        expect(quoted.blocks[0]).toMatchObject({ kind: "list", ordered: false });
    });

    it("answers nothing for nothing", () => {
        expect(blocksOf("")).toEqual([]);
        expect(blocksOf("   \n\n  ")).toEqual([]);
    });
});
