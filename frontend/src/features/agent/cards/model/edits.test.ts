import { describe, expect, it } from "vitest";

import { copy } from "../../../../copy/index.js";
import { aFinding } from "../../../../testing/findings.js";
import { optionDoes, optionLabel } from "../../../findings/labels.js";
import { findingSentence } from "../../../findings/model/subject.js";
import { findingCodeLabel } from "../../../pages/model/labels.js";
import type { Part } from "./card.js";
import { describeAction } from "./describe.js";
import { argsOf } from "./fixture.js";

const none = { currentTemplate: null, finding: null, summary: null };

function texts(parts: readonly Part[]): string[] {
    return parts.filter((part): part is string => typeof part === "string");
}

function refs(parts: readonly Part[]): string[] {
    return parts.filter((part): part is Exclude<Part, string> => typeof part !== "string").map((part) => `${part.kind}:${part.id}`);
}

describe("the agent's edits to a page", () => {
    const linesOf = (tool: string, args: Record<string, unknown>): string[] =>
        describeAction(tool, args, null, none).lines.map((line) => texts(line.parts).join(""));
    const said = copy.agent.describe;

    it("names the field it sets, the page and the new value", () => {
        const view = describeAction("pages_fields", argsOf("pages_fields", "set"), null, none);
        expect(texts(view.title).join("")).toBe("Set the SEO description of ");
        expect(refs(view.title)).toStrictEqual(["page:p1"]);
        expect(linesOf("pages_fields", argsOf("pages_fields", "set"))).toContain(
            said.fields.value("Hand-thrown mugs that keep coffee hot."),
        );
    });

    it("names a page by its path when the call gave no id, and says an empty value clears the field", () => {
        const args = { op: "set", path: "/mugs/", key: "title", value: "", hash: "" };
        const view = describeAction("pages_fields", args, null, none);
        expect(texts(view.title).join("")).toBe("Set the title of /mugs/");
        expect(view.lines.map((line) => [line.tone, texts(line.parts).join("")])).toContainEqual(["warn", said.fields.clears]);
    });

    it("shows a replaced block's new html as its text, never as markup", () => {
        const view = describeAction("pages_blocks", argsOf("pages_blocks", "replace"), null, none);
        expect(texts(view.title).join("")).toBe("Replace block 11 of ");
        expect(refs(view.title)).toStrictEqual(["page:p1"]);
        const written = linesOf("pages_blocks", argsOf("pages_blocks", "replace"));
        expect(written).toContain(said.blocks.text("Fired at 1240 °C & glazed."));
        expect(written.join("\n")).not.toMatch(/[<>]/);
    });

    it("inserts at the start of the body or after a block", () => {
        expect(texts(describeAction("pages_blocks", { op: "insert_after", pageId: "p1", index: -1, html: "<p>A</p>" }, null, none).title).join("")).toBe(
            "Insert a block at the start of ",
        );
        expect(texts(describeAction("pages_blocks", { op: "insert_after", pageId: "p1", index: 3, html: "<p>A</p>" }, null, none).title).join("")).toBe(
            "Insert a block after block 3 of ",
        );
    });

    it("reads a deleted block as a danger line", () => {
        const view = describeAction("pages_blocks", { op: "delete", path: "/mugs/", index: 4, hash: "b1" }, null, none);
        expect(texts(view.title).join("")).toBe("Delete block 4 of /mugs/");
        expect(view.lines.map((line) => [line.tone, texts(line.parts).join("")])).toContainEqual(["danger", said.blocks.deleteBody]);
    });

    it("names the anchor a placed link takes and the page it points to", () => {
        const args = { op: "place_link", pageId: "p1", index: 2, hash: "b1", targetId: "p2", anchor: "travel mugs" };
        const view = describeAction("pages_blocks", args, null, none);
        expect(texts(view.title).join("")).toBe(said.blocks.placeLink("travel mugs", 2));
        expect(refs(view.title)).toStrictEqual(["page:p1"]);
        expect(view.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["page:p2"]);
    });

    it("keeps no card of its own for a read", () => {
        expect(describeAction("pages_blocks", { op: "list", pageId: "p1" }, null, none).generic).toBe(true);
        expect(describeAction("pages_fields", { op: "get", pageId: "p1" }, null, none).generic).toBe(true);
        expect(describeAction("findings", { op: "list" }, null, none).generic).toBe(true);
    });
});

describe("the finding card", () => {
    const finding = aFinding();
    const context = { currentTemplate: null, finding, summary: null };

    it("names the code, the page and the subject in words, then what the option does", () => {
        const view = describeAction("findings", argsOf("findings", "resolve"), null, context);
        expect(texts(view.title).join("")).toBe(copy.agent.describe.findings.resolve(findingCodeLabel(finding.code)));
        expect(refs(view.title)).toStrictEqual(["page:p1"]);
        const written = view.lines.map((line) => texts(line.parts).join(""));
        expect(written).toContain(findingSentence(finding.code, finding.subject));
        expect(written).toContain(copy.agent.describe.findings.option(optionLabel("accept_site"), optionDoes("accept_site")));
    });

    it("names a site-wide finding without a page", () => {
        const site = aFinding({ code: "url_form_mixed", pageId: undefined, subject: { absolute: 4, relative: 2 } });
        const view = describeAction("findings", { op: "resolve", findingId: "f1", option: "ignore" }, null, { currentTemplate: null, finding: site, summary: null });
        expect(texts(view.title).join("")).toBe(copy.agent.describe.findings.resolveSite(findingCodeLabel("url_form_mixed")));
        expect(refs(view.title)).toStrictEqual([]);
    });

    it("says only the option while the finding is not read yet", () => {
        const view = describeAction("findings", argsOf("findings", "resolve"), null, none);
        expect(texts(view.title).join("")).toBe(copy.agent.describe.findings.resolveAny);
        expect(view.lines).toHaveLength(1);
    });

    it("names the page a row maps to and the type a page is given", () => {
        const mapped = describeAction("findings", { op: "resolve", findingId: "f2", option: "map_to_page", pageId: "p3" }, null, none);
        expect(mapped.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["page:p3"]);
        const typed = describeAction("findings", { op: "resolve", findingId: "f3", option: "set_type", pageTypeId: "pt-hub" }, null, none);
        expect(typed.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["pagetype:pt-hub"]);
    });
});
